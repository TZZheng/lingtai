package config

import (
	"errors"
	"testing"
)

func TestSelectTUIUpdaterRoutesInstallMethods(t *testing.T) {
	tests := []struct {
		name   string
		method TUIInstallMethod
		want   TUIInstallMethod
	}{
		{name: "homebrew", method: TUIInstallMethodHomebrew, want: TUIInstallMethodHomebrew},
		{name: "source", method: TUIInstallMethodSource, want: TUIInstallMethodSource},
		{name: "unknown", method: TUIInstallMethodUnknown, want: TUIInstallMethodUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			updater := SelectTUIUpdater(TUIInstallInfo{Method: tc.method})
			if got := updater.InstallMethod(); got != tc.want {
				t.Fatalf("InstallMethod() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestHomebrewTUIUpdaterDoctorRunsUpdateThenUpgrade(t *testing.T) {
	runner := &fakeRunner{}
	result := RunTUIUpdate(TUIInstallInfo{Method: TUIInstallMethodHomebrew}, TUIUpdateOptions{
		LatestVersion:         "v0.8.1",
		Runner:                runner,
		LookPath:              func(string) (string, error) { return "/opt/homebrew/bin/brew", nil },
		IncludeHomebrewUpdate: true,
		ResolveHomebrewPath:   true,
	})

	if !result.Healthy || !result.Updated {
		t.Fatalf("expected healthy updated result: %+v", result)
	}
	update := "/opt/homebrew/bin/brew update"
	upgrade := "/opt/homebrew/bin/brew upgrade lingtai-ai/lingtai/lingtai-tui"
	if !containsCall(runner.calls, update) || !containsCall(runner.calls, upgrade) {
		t.Fatalf("expected doctor brew calls, got %#v", runner.calls)
	}
	if indexOfCall(runner.calls, update) > indexOfCall(runner.calls, upgrade) {
		t.Fatalf("expected brew update before brew upgrade, got %#v", runner.calls)
	}
	if !containsLine(result.Lines, "Brew upgrade finished") {
		t.Fatalf("expected restart guidance line: %+v", result.Lines)
	}
}

func TestHomebrewTUIUpdaterStartupRunsUpgradeOnlyWithoutLookPath(t *testing.T) {
	runner := &fakeRunner{}
	lookedUp := false
	result := RunTUIUpdate(TUIInstallInfo{Method: TUIInstallMethodHomebrew}, TUIUpdateOptions{
		LatestVersion: "v0.8.1",
		Runner:        runner,
		LookPath: func(string) (string, error) {
			lookedUp = true
			return "", errors.New("should not be called")
		},
	})

	if !result.Healthy || !result.Updated {
		t.Fatalf("expected healthy updated result: %+v", result)
	}
	if lookedUp {
		t.Fatalf("startup-style updater should execute brew by name without resolving PATH")
	}
	if len(runner.calls) != 1 || runner.calls[0] != "brew upgrade lingtai-ai/lingtai/lingtai-tui" {
		t.Fatalf("expected only brew upgrade call, got %#v", runner.calls)
	}
}

func TestTUIUpdaterSourceAndUnknownDoNotRunBrew(t *testing.T) {
	tests := []struct {
		name       string
		method     TUIInstallMethod
		wantLine   string
		wantHealth bool
	}{
		{name: "source", method: TUIInstallMethodSource, wantLine: "Source/user-local TUI update is not automated yet", wantHealth: true},
		{name: "unknown", method: TUIInstallMethodUnknown, wantLine: "TUI install method is unknown", wantHealth: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runner := &fakeRunner{}
			result := RunTUIUpdate(TUIInstallInfo{Method: tc.method}, TUIUpdateOptions{
				LatestVersion: "v0.8.1",
				Runner:        runner,
				LookPath:      func(string) (string, error) { return "/opt/homebrew/bin/brew", nil },
			})
			if result.Healthy != tc.wantHealth {
				t.Fatalf("Healthy = %v, want %v: %+v", result.Healthy, tc.wantHealth, result.Lines)
			}
			if !containsLine(result.Lines, tc.wantLine) {
				t.Fatalf("expected guidance line %q, got %+v", tc.wantLine, result.Lines)
			}
			if containsCall(runner.calls, "brew") {
				t.Fatalf("%s updater must not run brew, got %#v", tc.name, runner.calls)
			}
		})
	}
}
