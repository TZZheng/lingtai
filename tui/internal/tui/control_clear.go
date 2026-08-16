package tui

func requestClearContextHeadlessWithDeps(lingtaiCmd, dir string, cfg clearWaitConfig, deps clearContextDeps) (string, error) {
	completed, err := requestClearContextWithDeps(lingtaiCmd, dir, cfg, deps)
	if err != nil {
		return "", err
	}
	if completed {
		return "cleared", nil
	}
	return "clear_requested", nil
}
