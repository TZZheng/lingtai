package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestHeadlessClearLiveTargetReportsRequestedOnlyAfterOwnerAcceptance(t *testing.T) {
	dir := t.TempDir()
	historyPath, contextPath := writeClearTestHistory(t, dir)
	writeJSON(t, filepath.Join(dir, ".agent.json"), map[string]interface{}{"molt_count": 7})

	reviveCalled := false
	status, err := requestClearContextHeadlessWithDeps("python", dir, shortClearTestConfig(), clearContextDeps{
		isAlive: func(string, float64) bool { return true },
		revive: func(string, string) error {
			reviveCalled = true
			return nil
		},
		sleep: time.Sleep,
	})
	if err != nil {
		t.Fatalf("requestClearContextHeadlessWithDeps: %v", err)
	}
	if status != "clear_requested" {
		t.Fatalf("status = %q, want %q", status, "clear_requested")
	}
	if reviveCalled {
		t.Fatal("live clear facade should not revive")
	}
	assertFileContent(t, filepath.Join(dir, ".clear"), "tui\n")
	assertFileContent(t, historyPath, "old chat\n")
	assertFileContent(t, contextPath, "old context\n")
	if _, err := os.Stat(filepath.Join(dir, ".suspend")); !os.IsNotExist(err) {
		t.Fatalf("live clear facade wrote .suspend; err=%v", err)
	}
}

func TestHeadlessClearRevivedTargetReportsClearedOnlyAfterCompletionAndResuspend(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, ".agent.json"), map[string]interface{}{"molt_count": 1})

	var alive atomic.Bool
	var revived atomic.Bool
	kernelDone := make(chan error, 1)
	go func() {
		kernelDone <- simulateClearKernel(dir, &alive)
	}()

	status, err := requestClearContextHeadlessWithDeps("python", dir, shortClearTestConfig(), clearContextDeps{
		isAlive: func(string, float64) bool { return alive.Load() },
		revive: func(cmd, gotDir string) error {
			if cmd != "python" {
				return fmt.Errorf("lingtaiCmd = %q", cmd)
			}
			if gotDir != dir {
				return fmt.Errorf("dir = %q, want %q", gotDir, dir)
			}
			revived.Store(true)
			alive.Store(true)
			return nil
		},
		sleep: time.Sleep,
	})
	if err != nil {
		t.Fatalf("requestClearContextHeadlessWithDeps: %v", err)
	}
	if status != "cleared" {
		t.Fatalf("status = %q, want %q", status, "cleared")
	}
	if !revived.Load() {
		t.Fatal("revived clear facade did not revive the agent")
	}
	if err := <-kernelDone; err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, filepath.Join(dir, ".suspend"), "")
	if count, ok := readMoltCount(dir); !ok || count != 2 {
		t.Fatalf("molt_count = %d, ok=%v; want 2,true", count, ok)
	}
}

func TestHeadlessClearNeverReportsStatusWhenOwnerRejects(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	status, err := requestClearContextHeadlessWithDeps("python", dir, shortClearTestConfig(), clearContextDeps{
		isAlive: func(string, float64) bool { return true },
		sleep:   time.Sleep,
	})
	if err == nil {
		t.Fatal("clear facade should surface the owner's rejection")
	}
	if status != "" {
		t.Fatalf("status = %q, want empty when owner rejects", status)
	}
}

func TestHeadlessClearNeverClaimsClearedWithoutOwnerCompletion(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, ".agent.json"), map[string]interface{}{"molt_count": 1})

	var alive atomic.Bool
	status, err := requestClearContextHeadlessWithDeps("python", dir, shortClearTestConfig(), clearContextDeps{
		isAlive: func(string, float64) bool { return alive.Load() },
		revive: func(string, string) error {
			alive.Store(true)
			return nil
		},
		sleep: time.Sleep,
	})
	if err == nil {
		t.Fatal("clear facade must not claim cleared without owner-observed completion")
	}
	if status != "" {
		t.Fatalf("status = %q, want empty on owner failure", status)
	}
}