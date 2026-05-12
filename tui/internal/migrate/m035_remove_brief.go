package migrate

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// migrateRemoveBrief strips the secretary-era brief plumbing from every
// agent in the project: deletes system/brief.md, and removes the "brief",
// "brief_file" keys from each init.json. The brief section used to inject
// a secretary-maintained summary into the agent's system prompt; with the
// secretary agent removed, leaving the file in place would keep stale
// content visible to the kernel until the user wiped .lingtai/ by hand.
//
// Per-project settings.json's optional `brief` toggle is also dropped — the
// field was only consulted by the (now-deleted) /settings brief switch.
//
// Best-effort: any single agent that fails to migrate is skipped silently.
// Returning an error would block startup for a benign cleanup.
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
