package tui

// Headless tests freezing the future private refresh facade:
//
//	hardRefreshDirWithArgs(lingtaiCmd, dir, args string) error
//
// Contract:
//   - empty args delegates to the strong-refresh default (hardRefreshDir).
//   - one arg is resolved through resolvePresetInAllowed and the canonical
//     allowed[] entry is passed to the preset owner (hardRefreshDirWithPreset).
//   - a disallowed or ambiguous arg fails before any mutation: neither
//     owner runs and the agent dir is left untouched.
//
// The two owners are reached only through unexported delegate vars
// (refreshStrong / refreshStrongWithPreset), mirroring the existing
// launchHeartbeat* override seam, so this test fakes private deps only —
// no real agent is launched, no signal is approximated, and nothing is
// injected through a public flag.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withRefreshSeams swaps the private owner delegates for fakes and restores
// them when the test ends.
func withRefreshSeams(t *testing.T, deps refreshDelegates) {
	t.Helper()
	origStrong, origPreset := refreshStrong, refreshStrongWithPreset
	refreshStrong, refreshStrongWithPreset = deps.strong, deps.preset
	t.Cleanup(func() { refreshStrong, refreshStrongWithPreset = origStrong, origPreset })
}

// refreshDelegates captures the two private owners the facade must route to.
type refreshDelegates struct {
	strong func(lingtaiCmd, dir string) error
	preset func(lingtaiCmd, dir, presetPath string) error
}

// TestHardRefreshDirWithArgs_NoArg_DelegatesStrongDefault verifies the
// zero-arg form routes to the strong-refresh owner (hardRefreshDir) with
// the exact command and directory, and never touches the preset owner.
func TestHardRefreshDirWithArgs_NoArg_DelegatesStrongDefault(t *testing.T) {
	var strongCalls, presetCalls []string
	withRefreshSeams(t, refreshDelegates{
		strong: func(cmd, dir string) error {
			strongCalls = append(strongCalls, cmd+"|"+dir)
			return errors.New("strong boom")
		},
		preset: func(cmd, dir, presetPath string) error {
			presetCalls = append(presetCalls, presetPath)
			return nil
		},
	})

	dir := t.TempDir()
	err := hardRefreshDirWithArgs("lingtai", dir, "")

	if err == nil || !contains(err.Error(), "strong boom") {
		t.Fatalf("err = %v, want the strong owner's error", err)
	}
	if len(strongCalls) != 1 {
		t.Fatalf("strong calls = %d, want exactly 1", len(strongCalls))
	}
	if got := strongCalls[0]; got != "lingtai|"+dir {
		t.Errorf("strong owner got %q, want %q", got, "lingtai|"+dir)
	}
	if len(presetCalls) != 0 {
		t.Errorf("preset owner called %d times for no-arg refresh", len(presetCalls))
	}
}

// TestHardRefreshDirWithArgs_AllowedStem_ResolvesThenDelegatesPreset
// verifies a bare preset name is resolved against manifest.preset.allowed
// and the canonical path is handed to the preset owner.
func TestHardRefreshDirWithArgs_AllowedStem_ResolvesThenDelegatesPreset(t *testing.T) {
	var strongCalls, presetCalls []string
	withRefreshSeams(t, refreshDelegates{
		strong: func(cmd, dir string) error {
			strongCalls = append(strongCalls, cmd)
			return nil
		},
		preset: func(cmd, dir, presetPath string) error {
			presetCalls = append(presetCalls, cmd+"|"+dir+"|"+presetPath)
			return nil
		},
	})

	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "init.json"), map[string]interface{}{
		"manifest": map[string]interface{}{
			"preset": map[string]interface{}{
				"allowed": []interface{}{
					"~/.lingtai-tui/presets/templates/minimax.json",
					"~/.lingtai-tui/presets/saved/zhipu-1.json",
				},
			},
		},
	})

	if err := hardRefreshDirWithArgs("lingtai", dir, "zhipu-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(strongCalls) != 0 {
		t.Errorf("strong owner called %d times for preset refresh", len(strongCalls))
	}
	if len(presetCalls) != 1 {
		t.Fatalf("preset calls = %d, want exactly 1", len(presetCalls))
	}
	want := "lingtai|" + dir + "|~/.lingtai-tui/presets/saved/zhipu-1.json"
	if got := presetCalls[0]; got != want {
		t.Errorf("preset owner got %q, want %q", got, want)
	}
}

// TestHardRefreshDirWithArgs_AllowedExactPath_ResolvesThenDelegatesPreset
// verifies the full allowed[] entry also resolves and is forwarded verbatim.
func TestHardRefreshDirWithArgs_AllowedExactPath_ResolvesThenDelegatesPreset(t *testing.T) {
	ref := "~/.lingtai-tui/presets/templates/minimax.json"
	var presetCalls []string
	withRefreshSeams(t, refreshDelegates{
		strong: func(cmd, dir string) error {
			t.Fatal("strong owner called for exact-path preset refresh")
			return nil
		},
		preset: func(cmd, dir, presetPath string) error {
			presetCalls = append(presetCalls, presetPath)
			return nil
		},
	})

	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "init.json"), map[string]interface{}{
		"manifest": map[string]interface{}{
			"preset": map[string]interface{}{"allowed": []interface{}{ref}},
		},
	})

	if err := hardRefreshDirWithArgs("lingtai", dir, ref); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(presetCalls) != 1 || presetCalls[0] != ref {
		t.Errorf("preset calls = %v, want exactly [%s]", presetCalls, ref)
	}
}

// TestHardRefreshDirWithArgs_DisallowedPreset_FailsBeforeMutation verifies
// a non-allowed name fails before either owner runs and before the agent
// dir gains any refresh artifacts.
func TestHardRefreshDirWithArgs_DisallowedPreset_FailsBeforeMutation(t *testing.T) {
	var strongCalls, presetCalls int
	withRefreshSeams(t, refreshDelegates{
		strong: func(cmd, dir string) error {
			strongCalls++
			return nil
		},
		preset: func(cmd, dir, presetPath string) error {
			presetCalls++
			return nil
		},
	})

	dir := t.TempDir()
	initPath := filepath.Join(dir, "init.json")
	writeJSON(t, initPath, map[string]interface{}{
		"manifest": map[string]interface{}{
			"preset": map[string]interface{}{
				"allowed": []interface{}{"~/.lingtai-tui/presets/templates/minimax.json"},
			},
		},
	})

	err := hardRefreshDirWithArgs("lingtai", dir, "ghost-preset")

	if err == nil {
		t.Fatal("expected error for disallowed preset")
	}
	if !contains(err.Error(), "ghost-preset") {
		t.Errorf("error %q missing requested name", err.Error())
	}
	if strongCalls != 0 || presetCalls != 0 {
		t.Errorf("owners ran despite disallowed preset (strong=%d preset=%d)", strongCalls, presetCalls)
	}
	assertDirUnmutated(t, dir, initPath)
}

// TestHardRefreshDirWithArgs_AmbiguousPreset_FailsBeforeMutation verifies a
// basename shared by two allowed entries fails before either owner runs and
// before any mutation.
func TestHardRefreshDirWithArgs_AmbiguousPreset_FailsBeforeMutation(t *testing.T) {
	var strongCalls, presetCalls int
	withRefreshSeams(t, refreshDelegates{
		strong: func(cmd, dir string) error {
			strongCalls++
			return nil
		},
		preset: func(cmd, dir, presetPath string) error {
			presetCalls++
			return nil
		},
	})

	dir := t.TempDir()
	initPath := filepath.Join(dir, "init.json")
	writeJSON(t, initPath, map[string]interface{}{
		"manifest": map[string]interface{}{
			"preset": map[string]interface{}{
				"allowed": []interface{}{
					"~/.lingtai-tui/presets/templates/mimo.json",
					"~/.lingtai-tui/presets/saved/mimo.json",
				},
			},
		},
	})

	err := hardRefreshDirWithArgs("lingtai", dir, "mimo")

	if err == nil {
		t.Fatal("expected ambiguity error")
	}
	if !contains(err.Error(), "ambiguous") {
		t.Errorf("error %q missing 'ambiguous'", err.Error())
	}
	if strongCalls != 0 || presetCalls != 0 {
		t.Errorf("owners ran despite ambiguous preset (strong=%d preset=%d)", strongCalls, presetCalls)
	}
	assertDirUnmutated(t, dir, initPath)
}

// assertDirUnmutated fails if the agent dir contains anything beyond the
// given init.json after a refused refresh.
func assertDirUnmutated(t *testing.T, dir, initPath string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(initPath) {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("agent dir mutated by refused refresh: %v", names)
	}
}