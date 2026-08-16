package tui

import "strings"

// refreshStrong / refreshStrongWithPreset are the private refresh owners,
// reached only through these unexported delegate vars. They mirror the
// launchHeartbeat* override seam so headless tests can fake the owners
// without launching a real agent. The strong owner (hardRefreshDir) and the
// preset owner (hardRefreshDirWithPreset) are never substituted with a
// `.refresh` signal — the hard relaunch is preserved.
var (
	refreshStrong           = hardRefreshDir
	refreshStrongWithPreset = hardRefreshDirWithPreset
)

// hardRefreshDirWithArgs is the private refresh facade taking the optional
// preset argument of the /refresh control. Contract:
//
//   - empty args delegates to the strong-refresh default (hardRefreshDir).
//   - one arg is resolved through resolvePresetInAllowed and the canonical
//     allowed[] entry is passed to the preset owner
//     (hardRefreshDirWithPreset).
//   - a disallowed or ambiguous arg fails before any mutation: neither owner
//     runs and the agent dir is left untouched.
func hardRefreshDirWithArgs(lingtaiCmd, dir, args string) error {
	args = strings.TrimSpace(args)
	if args == "" {
		return refreshStrong(lingtaiCmd, dir)
	}
	resolved, err := resolvePresetInAllowed(dir, args)
	if err != nil {
		return err
	}
	return refreshStrongWithPreset(lingtaiCmd, dir, resolved)
}
