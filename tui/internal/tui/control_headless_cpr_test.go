// control_headless_cpr_test.go is the RED commit for the headless CPR seam.
// It freezes requestCPRHeadlessWithDeps, the private narrow seam for the
// selected-contained-Agent /cpr control, before any production implementation
// exists. It reuses the existing clear-context owners verbatim
// (clearContextDeps.revive, clearContextDeps.isAlive) so CPR delegates the
// existing revive owner and waits for the existing fresh-heartbeat observation;
// green implements the seam against these same owners.
package tui

import (
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// TestRequestCPRHeadlessWithDepsDelegatesReviveOwnerAndWaitsForFreshHeartbeat
// freezes the contained-Agent CPR seam: it must resolve the selected Agent,
// delegate to the existing revive owner with the resolved Agent directory, and
// then wait for the existing fresh-heartbeat observation instead of proceeding
// on a stale heartbeat.
func TestRequestCPRHeadlessWithDepsDelegatesReviveOwnerAndWaitsForFreshHeartbeat(t *testing.T) {
	projectDir, agentDir := writeHeadlessCPRTestAgent(t, "agent")

	var revived atomic.Bool
	var revivedAgentDir atomic.Value
	var alive atomic.Bool
	var staleObs, freshObs atomic.Int64

	err := requestCPRHeadlessWithDeps(projectDir, "agent", clearContextDeps{
		revive: func(lingtaiCmd, gotDir string) error {
			revivedAgentDir.Store(gotDir)
			revived.Store(true)
			alive.Store(true)
			return nil
		},
		isAlive: func(gotDir string, timeoutSec float64) bool {
			a := alive.Load()
			if a {
				freshObs.Add(1)
			} else {
				staleObs.Add(1)
			}
			return a
		},
		sleep: time.Sleep,
	})
	if err != nil {
		t.Fatalf("requestCPRHeadlessWithDeps: %v", err)
	}
	if !revived.Load() {
		t.Fatal("CPR did not delegate to the existing revive owner")
	}
	if got := revivedAgentDir.Load().(string); got != agentDir {
		t.Fatalf("revive owner received dir %q, want %q", got, agentDir)
	}
	if staleObs.Load() < 1 || freshObs.Load() < 1 {
		t.Fatalf("fresh-heartbeat observation saw stale=%d fresh=%d; want both >= 1 (must wait for a fresh heartbeat after revive, not proceed on a stale one)", staleObs.Load(), freshObs.Load())
	}
}

// TestRequestCPRHeadlessWithDepsRejectsNonAgentWithoutRevive freezes the
// smallest invalid/non-Agent guard directly required by the current owner
// contract (controlAgentSignal rejects the reserved "human" key): a non-Agent
// target must be rejected with ErrInvalidControlAgent before the revive owner
// is ever delegated to.
func TestRequestCPRHeadlessWithDepsRejectsNonAgentWithoutRevive(t *testing.T) {
	projectDir, _ := writeHeadlessCPRTestAgent(t, "agent")

	var revived atomic.Bool
	err := requestCPRHeadlessWithDeps(projectDir, "human", clearContextDeps{
		revive:  func(string, string) error { revived.Store(true); return nil },
		isAlive: func(string, float64) bool { return true },
		sleep:   time.Sleep,
	})
	if !errors.Is(err, ErrInvalidControlAgent) {
		t.Fatalf("err = %v, want ErrInvalidControlAgent", err)
	}
	if revived.Load() {
		t.Fatal("non-Agent guard must reject before delegating to the revive owner")
	}
}

// writeHeadlessCPRTestAgent creates a contained Agent (a real .lingtai project
// with a single non-human Agent subdirectory carrying a minimal manifest), the
// shape the current owner contract requires.
func writeHeadlessCPRTestAgent(t *testing.T, agentKey string) (projectDir, agentDir string) {
	t.Helper()
	projectDir = filepath.Join(t.TempDir(), ".lingtai")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	agentDir = filepath.Join(projectDir, agentKey)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, ".agent.json"), []byte("{\"molt_count\":1}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return projectDir, agentDir
}