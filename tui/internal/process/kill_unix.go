//go:build !windows

package process

import (
	"syscall"
	"time"
)

// TerminateAgentProcesses sends SIGTERM to every `lingtai run <agentDir>`
// process, waits briefly for them to exit, then escalates to SIGKILL for any
// stragglers. Returns nil on success or when no matching processes exist; a
// non-nil error indicates the final SIGKILL pass still saw live processes.
//
// Used by /refresh to forcibly clear an agent that did not honor `.suspend`.
// Normal launch paths must NOT call this — duplicate-launch protection in
// LaunchAgent assumes the prior agent shut down gracefully.
func TerminateAgentProcesses(agentDir string) error {
	procs := FindAgentProcesses(agentDir)
	if len(procs) == 0 {
		return nil
	}
	for _, p := range procs {
		_ = syscall.Kill(p.PID, syscall.SIGTERM)
	}
	// Poll up to ~2s for graceful exit before escalating.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(FindAgentProcesses(agentDir)) == 0 {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	// Escalate: SIGKILL anything still alive.
	for _, p := range FindAgentProcesses(agentDir) {
		_ = syscall.Kill(p.PID, syscall.SIGKILL)
	}
	// Brief final wait so callers observe a clean ps.
	deadline = time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if len(FindAgentProcesses(agentDir)) == 0 {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if remaining := FindAgentProcesses(agentDir); len(remaining) > 0 {
		return &terminateError{remaining: len(remaining)}
	}
	return nil
}

type terminateError struct{ remaining int }

func (e *terminateError) Error() string {
	return "agent processes still alive after SIGKILL"
}
