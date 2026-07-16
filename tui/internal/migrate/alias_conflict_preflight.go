package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/anthropics/lingtai-tui/internal/preset"
)

// preflightShellCapabilityConflicts checks the legacy/canonical capability pair
// before any pending migration gets a chance to rewrite init.json. It only
// rejects a valid, object-shaped pair whose decoded values differ; malformed or
// unrelated shapes retain the best-effort behavior of the older migrations.
func preflightShellCapabilityConflicts(lingtaiDir string) error {
	entries, err := os.ReadDir(lingtaiDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read .lingtai dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "" || name[0] == '.' || name == "human" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(lingtaiDir, name, "init.json"))
		if err != nil {
			continue
		}

		var doc map[string]interface{}
		if err := preset.DecodeJSONUseNumber(data, &doc); err != nil {
			continue
		}
		manifest, ok := doc["manifest"].(map[string]interface{})
		if !ok {
			continue
		}
		caps, ok := manifest["capabilities"].(map[string]interface{})
		if !ok {
			continue
		}
		legacy, hasLegacy := caps["bash"]
		canonical, hasCanonical := caps["shell"]
		if hasLegacy && hasCanonical && !reflect.DeepEqual(legacy, canonical) {
			return fmt.Errorf("%s: conflicting capability configuration: %q and %q differ", name, "bash", "shell")
		}
	}
	return nil
}
