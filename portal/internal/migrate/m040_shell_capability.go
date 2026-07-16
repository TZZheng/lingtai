package migrate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
)

type portalShellCapabilityRewrite struct {
	path string
	data []byte
}

func portalDecodeJSONUseNumber(data []byte, dst interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("invalid trailing data after JSON document")
		}
		return err
	}
	return nil
}

// migrateShellCapability mirrors TUI m040 because portal and TUI share
// .lingtai/meta.json and per-agent init.json state. It canonicalizes the legacy
// bash capability to shell while preserving the configuration object unchanged.
// Conflicting keys and unknown init shapes fail closed before any file rewrite.
func migrateShellCapability(lingtaiDir string) error {
	entries, err := os.ReadDir(lingtaiDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read .lingtai dir: %w", err)
	}

	var rewrites []portalShellCapabilityRewrite
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

		doc, err := portalShellCapabilityDocument(data)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			continue
		}
		changed, err := canonicalizePortalShellCapabilityDocument(doc)
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
		rewrites = append(rewrites, portalShellCapabilityRewrite{path: path, data: updated})
	}
	if len(errs) > 0 {
		return fmt.Errorf("shell capability migration preflight failed: %w", errors.Join(errs...))
	}

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

func portalShellCapabilityDocument(data []byte) (map[string]interface{}, error) {
	var doc map[string]interface{}
	if err := portalDecodeJSONUseNumber(data, &doc); err != nil {
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

func canonicalizePortalShellCapabilityDocument(doc map[string]interface{}) (bool, error) {
	manifest, ok := doc["manifest"].(map[string]interface{})
	if !ok {
		return false, nil
	}
	caps, ok := manifest["capabilities"].(map[string]interface{})
	if !ok {
		return false, nil
	}
	legacy, hasLegacy := caps["bash"]
	canonical, hasCanonical := caps["shell"]
	if !hasLegacy {
		return false, nil
	}
	if hasCanonical && !reflect.DeepEqual(legacy, canonical) {
		return false, fmt.Errorf("conflicting capability configuration: %q and %q differ", "bash", "shell")
	}
	if !hasCanonical {
		caps["shell"] = legacy
	}
	delete(caps, "bash")
	return true, nil
}
