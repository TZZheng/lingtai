package migrate

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// migrateRemoveBrief — see tui/internal/migrate/m035_remove_brief.go.
// Kept in sync because both binaries can be the first to open a project
// post-secretary-removal, and either one needs to clear the brief.md
// fallback so the kernel stops injecting stale content.
func migrateRemoveBrief(lingtaiDir string) error {
	entries, err := os.ReadDir(lingtaiDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "human" {
			continue
		}
		if len(entry.Name()) > 0 && entry.Name()[0] == '.' {
			continue
		}
		agentDir := filepath.Join(lingtaiDir, entry.Name())
		stripAgentBrief(agentDir)
	}
	stripHumanBriefSetting(filepath.Join(lingtaiDir, "human"))
	return nil
}

func stripAgentBrief(agentDir string) {
	os.Remove(filepath.Join(agentDir, "system", "brief.md"))

	initPath := filepath.Join(agentDir, "init.json")
	data, err := os.ReadFile(initPath)
	if err != nil {
		return
	}
	var initData map[string]interface{}
	if err := json.Unmarshal(data, &initData); err != nil {
		return
	}
	_, hadBrief := initData["brief"]
	_, hadBriefFile := initData["brief_file"]
	if !hadBrief && !hadBriefFile {
		return
	}
	delete(initData, "brief")
	delete(initData, "brief_file")
	out, err := json.MarshalIndent(initData, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(initPath, out, 0o644)
}

func stripHumanBriefSetting(humanDir string) {
	settingsPath := filepath.Join(humanDir, "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return
	}
	if _, ok := settings["brief"]; !ok {
		return
	}
	delete(settings, "brief")
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(settingsPath, out, 0o644)
}
