package main

// Regression tests for issue #488: `clean` must refuse to delete .lingtai/
// while agents are still alive after the suspend timeout, unless --force.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeCleanAgent creates a minimal agent dir under lingtaiDir. When alive
// is true, a fresh .agent.heartbeat makes fs.IsAlive report it as running.
func writeCleanAgent(t *testing.T, lingtaiDir, name string, alive bool) string {
	t.Helper()
	agentDir := filepath.Join(lingtaiDir, name)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf(`{"address":"%s","agent_name":"%s","admin":false}`, name, name)
	if err := os.WriteFile(filepath.Join(agentDir, ".agent.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if alive {
		ts := fmt.Sprintf("%.3f", float64(time.Now().UnixNano())/1e9)
		if err := os.WriteFile(filepath.Join(agentDir, ".agent.heartbeat"), []byte(ts), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return agentDir
}

func TestCleanRefusesWhileAgentAlive(t *testing.T) {
	lingtaiDir := filepath.Join(t.TempDir(), ".lingtai")
	agentDir := writeCleanAgent(t, lingtaiDir, "alice", true)

	err := cleanProject(lingtaiDir, false, 300*time.Millisecond)
	if err == nil {
		t.Fatal("expected error while agent heartbeat is fresh, got nil")
	}
	if !strings.Contains(err.Error(), agentDir) {
		t.Errorf("error should list the surviving agent dir, got: %v", err)
	}
	if _, statErr := os.Stat(lingtaiDir); statErr != nil {
		t.Errorf(".lingtai/ must not be removed while an agent is alive: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(agentDir, ".suspend")); statErr != nil {
		t.Errorf(".suspend signal should still have been written: %v", statErr)
	}
}

func TestCleanForceRemovesDespiteLiveAgent(t *testing.T) {
	lingtaiDir := filepath.Join(t.TempDir(), ".lingtai")
	writeCleanAgent(t, lingtaiDir, "alice", true)

	if err := cleanProject(lingtaiDir, true, 300*time.Millisecond); err != nil {
		t.Fatalf("force clean should proceed, got: %v", err)
	}
	if _, statErr := os.Stat(lingtaiDir); !os.IsNotExist(statErr) {
		t.Error(".lingtai/ should be removed with --force")
	}
}

func TestCleanRefusesWhenDiscoverFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root — unlistable dirs are still listable")
	}
	lingtaiDir := filepath.Join(t.TempDir(), ".lingtai")
	writeCleanAgent(t, lingtaiDir, "alice", false)
	if err := os.Chmod(lingtaiDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(lingtaiDir, 0o755) })

	err := cleanProject(lingtaiDir, false, 300*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when agents cannot be listed, got nil")
	}
	if _, statErr := os.Stat(lingtaiDir); statErr != nil {
		t.Errorf(".lingtai/ must not be removed when discovery fails: %v", statErr)
	}
}

func TestCleanGuardsAgentAppearingDuringWait(t *testing.T) {
	lingtaiDir := filepath.Join(t.TempDir(), ".lingtai")
	aliceDir := writeCleanAgent(t, lingtaiDir, "alice", true)

	// While cleanProject waits for alice to suspend, alice dies and bob
	// appears. Only re-discovering before deletion can catch bob.
	done := make(chan struct{})
	go func() {
		defer close(done)
		time.Sleep(400 * time.Millisecond)
		stale := fmt.Sprintf("%.3f", float64(time.Now().Add(-time.Minute).UnixNano())/1e9)
		if err := os.WriteFile(filepath.Join(aliceDir, ".agent.heartbeat"), []byte(stale), 0o644); err != nil {
			t.Error(err)
			return
		}
		bobDir := filepath.Join(lingtaiDir, "bob")
		if err := os.MkdirAll(bobDir, 0o755); err != nil {
			t.Error(err)
			return
		}
		manifest := `{"address":"bob","agent_name":"bob","admin":false}`
		if err := os.WriteFile(filepath.Join(bobDir, ".agent.json"), []byte(manifest), 0o644); err != nil {
			t.Error(err)
			return
		}
		ts := fmt.Sprintf("%.3f", float64(time.Now().UnixNano())/1e9)
		if err := os.WriteFile(filepath.Join(bobDir, ".agent.heartbeat"), []byte(ts), 0o644); err != nil {
			t.Error(err)
		}
	}()

	err := cleanProject(lingtaiDir, false, 5*time.Second)
	<-done
	if err == nil {
		t.Fatal("expected error for the agent that appeared during the wait, got nil")
	}
	if !strings.Contains(err.Error(), "bob") {
		t.Errorf("error should list the late-appearing agent, got: %v", err)
	}
	if _, statErr := os.Stat(lingtaiDir); statErr != nil {
		t.Errorf(".lingtai/ must not be removed while bob is alive: %v", statErr)
	}
}

func TestCleanRemovesWhenAgentsDead(t *testing.T) {
	lingtaiDir := filepath.Join(t.TempDir(), ".lingtai")
	writeCleanAgent(t, lingtaiDir, "alice", false)

	if err := cleanProject(lingtaiDir, false, 300*time.Millisecond); err != nil {
		t.Fatalf("clean of a dead network should succeed, got: %v", err)
	}
	if _, statErr := os.Stat(lingtaiDir); !os.IsNotExist(statErr) {
		t.Error(".lingtai/ should be removed when no agent is alive")
	}
}
