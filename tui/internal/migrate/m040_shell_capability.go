package migrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/lingtai-tui/internal/preset"
)

type shellCapabilityRewrite struct {
	path string
	data []byte
}

// migrateShellCapability canonicalizes the legacy bash capability in existing
// per-agent init.json files. The configuration object is moved unchanged to
// shell. A conflicting bash/shell pair or an unknown init shape is a schema
// error: leave all files untouched and return an error so the migration version
// is not advanced.
func migrateShellCapability(lingtaiDir string) error {
	entries, err := os.ReadDir(lingtaiDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read .lingtai dir: %w", err)
	}

	// Preflight every candidate before rewriting any init.json. This matters
	// when one agent is valid and a later agent has a shape we cannot interpret:
	// content errors must not leave a partially migrated network behind.
	var rewrites []shellCapabilityRewrite
	var errs []error
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "" || name[0] == '.' || name == "human" {
			continue
		}
		path := filepath.Join(lingtaiDir, name, "init.json")
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				errs = append(errs, fmt.Errorf("%s: read init.json: %w", name, err))
			}
			continue
		}

		doc, err := shellCapabilityDocument(data)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			continue
		}
		changed, err := canonicalizeShellCapabilityDocument(doc)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			continue
		}
		if !changed {
			continue
		}
		updated, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: marshal init.json: %w", name, err))
			continue
		}
		rewrites = append(rewrites, shellCapabilityRewrite{path: path, data: updated})
	}
	if len(errs) > 0 {
		return fmt.Errorf("shell capability migration preflight failed: %w", errors.Join(errs...))
	}

	// Only the write phase follows a successful content preflight. Each file is
	// replaced atomically, matching the surrounding migration convention.
	for _, rewrite := range rewrites {
		tmp := rewrite.path + ".tmp"
		if err := os.WriteFile(tmp, rewrite.data, 0o644); err != nil {
			errs = append(errs, fmt.Errorf("%s: write init.json.tmp: %w", rewrite.path, err))
			continue
		}
		if err := os.Rename(tmp, rewrite.path); err != nil {
			errs = append(errs, fmt.Errorf("%s: rename init.json: %w", rewrite.path, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("shell capability migration incomplete: %w", errors.Join(errs...))
	}
	return nil
}

// shellCapabilityDocument decodes the agent-init shape that m040 understands.
// Missing manifest/capabilities fields are valid no-ops; present non-object
// fields are not, because silently skipping them would stamp an unknown shape.
func shellCapabilityDocument(data []byte) (map[string]interface{}, error) {
	var doc map[string]interface{}
	if err := preset.DecodeJSONUseNumber(data, &doc); err != nil {
		return nil, fmt.Errorf("malformed init.json: %w", err)
	}
	if doc == nil {
		return nil, fmt.Errorf("init.json top level must be an object")
	}
	if rawManifest, exists := doc["manifest"]; exists {
		manifest, ok := rawManifest.(map[string]interface{})
		if !ok || manifest == nil {
			return nil, fmt.Errorf("manifest must be an object")
		}
		if rawCaps, exists := manifest["capabilities"]; exists {
			caps, ok := rawCaps.(map[string]interface{})
			if !ok || caps == nil {
				return nil, fmt.Errorf("manifest.capabilities must be an object")
			}
		}
	}
	return doc, nil
}

func canonicalizeShellCapabilityDocument(doc map[string]interface{}) (bool, error) {
	manifest, ok := doc["manifest"].(map[string]interface{})
	if !ok {
		return false, nil
	}
	caps, ok := manifest["capabilities"].(map[string]interface{})
	if !ok {
		return false, nil
	}
	return preset.CanonicalizeCapabilities(caps)
}
