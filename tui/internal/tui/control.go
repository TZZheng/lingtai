package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/lingtai-tui/internal/fs"
	"github.com/anthropics/lingtai-tui/internal/preset"
)

var (
	// ErrInvalidControlProject means a headless control project is not an
	// existing absolute .lingtai directory.
	ErrInvalidControlProject = errors.New("invalid control project")
	// ErrInvalidControlAgent means a headless control target is not a real,
	// contained Agent directory.
	ErrInvalidControlAgent = errors.New("invalid control agent")
)

// sleepAgent is the selected-Agent /sleep owner shared by the interactive
// palette and the narrow headless facade.
func sleepAgent(agentDir string) error {
	return fs.TouchSignal(agentDir, fs.SignalSleep)
}

// ControlAgentSleep validates an explicit selected Agent beneath projectDir
// and performs the same empty .sleep signal write as interactive /sleep.
func ControlAgentSleep(projectDir, agentKey string) error {
	resolvedProject, err := resolveControlProject(projectDir)
	if err != nil {
		return err
	}
	if err := preset.ValidateSafeName(agentKey); err != nil {
		return fmt.Errorf("%w: agent key %q %s", ErrInvalidControlAgent, agentKey, err)
	}
	if agentKey == "human" {
		return fmt.Errorf("%w: human is not an Agent", ErrInvalidControlAgent)
	}

	agentDir := filepath.Join(resolvedProject, agentKey)
	info, err := os.Lstat(agentDir)
	if err != nil {
		return fmt.Errorf("%w: agent %q does not exist", ErrInvalidControlAgent, agentKey)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%w: agent %q is not a real directory", ErrInvalidControlAgent, agentKey)
	}
	resolvedAgent, err := filepath.EvalSymlinks(agentDir)
	if err != nil || filepath.Dir(resolvedAgent) != resolvedProject {
		return fmt.Errorf("%w: agent %q escapes the project", ErrInvalidControlAgent, agentKey)
	}

	manifestInfo, err := os.Lstat(filepath.Join(resolvedAgent, ".agent.json"))
	if err != nil || manifestInfo.Mode()&os.ModeSymlink != 0 || !manifestInfo.Mode().IsRegular() {
		return fmt.Errorf("%w: agent %q has no real manifest", ErrInvalidControlAgent, agentKey)
	}
	agent, err := fs.ReadAgent(resolvedAgent)
	if err != nil || agent.IsHuman {
		return fmt.Errorf("%w: agent %q is not a real Agent", ErrInvalidControlAgent, agentKey)
	}

	if err := sleepAgent(resolvedAgent); err != nil {
		return fmt.Errorf("write sleep signal for agent %q: %w", agentKey, err)
	}
	return nil
}

func resolveControlProject(projectDir string) (string, error) {
	if !filepath.IsAbs(projectDir) {
		return "", fmt.Errorf("%w: --project must be absolute", ErrInvalidControlProject)
	}
	projectDir = filepath.Clean(projectDir)
	if filepath.Base(projectDir) != ".lingtai" {
		return "", fmt.Errorf("%w: --project must name a .lingtai directory", ErrInvalidControlProject)
	}
	resolved, err := filepath.EvalSymlinks(projectDir)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidControlProject, err)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() || filepath.Base(resolved) != ".lingtai" {
		return "", fmt.Errorf("%w: --project must resolve to an existing .lingtai directory", ErrInvalidControlProject)
	}
	return resolved, nil
}
