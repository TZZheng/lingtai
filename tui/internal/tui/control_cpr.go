package tui

import (
	"fmt"

	"github.com/anthropics/lingtai-tui/internal/config"
)

// requestCPRHeadlessWithDeps is the selected-Agent /cpr headless seam. It
// resolves the contained real Agent through the existing control containment
// owner, captures the pre-revive heartbeat baseline through the existing
// fresh-heartbeat observation, delegates the existing revive owner with the
// resolved Agent directory, and then waits for a heartbeat that is genuinely
// fresh after the revive — never proceeding on a stale pre-existing heartbeat.
//
// The Python launch command is derived read-only from the machine's global
// TUI dir (the same source the interactive owner uses at startup), so a pure
// headless control call creates no side-effect global state.
func requestCPRHeadlessWithDeps(projectDir, agentKey string, deps clearContextDeps) error {
	deps = normalizeClearContextDeps(deps)

	var resolvedAgent string
	// Reuse the existing containment owner: project resolution, safe-name and
	// human rejection, real-directory/traversal checks, and manifest/real-Agent
	// checks all complete before any revive side effect is possible.
	if err := controlAgentSignal(projectDir, agentKey, "cpr", func(agentDir string) error {
		resolvedAgent = agentDir
		return nil
	}); err != nil {
		return err
	}

	// Pre-revive heartbeat baseline: only a fresh observation means the Agent
	// is already alive, in which case CPR is a no-op like the interactive
	// owner. A stale heartbeat must not be accepted as aliveness.
	if deps.isAlive(resolvedAgent, 3.0) {
		return nil
	}

	globalDir, err := config.GlobalDirPath()
	if err != nil {
		return fmt.Errorf("resolve global TUI dir for revive: %w", err)
	}
	lingtaiCmd := config.LingtaiCmd(globalDir)

	if err := deps.revive(lingtaiCmd, resolvedAgent); err != nil {
		return fmt.Errorf("revive agent %q: %w", agentKey, err)
	}

	// Wait for a fresh post-revive heartbeat through the existing
	// fresh-heartbeat observation owner; a stale pre-existing heartbeat is not
	// accepted as success.
	if err := waitForAgentAlive(resolvedAgent, defaultClearWaitConfig.aliveTimeout, defaultClearWaitConfig.pollInterval, deps); err != nil {
		return fmt.Errorf("wait for fresh heartbeat after reviving agent %q: %w", agentKey, err)
	}
	return nil
}
