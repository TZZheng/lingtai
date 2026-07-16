package migrate

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestPortalM040CanonicalizesLegacyShellAndPreservesConfig(t *testing.T) {
	lingtaiDir := t.TempDir()
	path := writeAgentInitPortal(t, lingtaiDir, "alice", `{
  "manifest": {
    "capabilities": {
      "bash": {"yolo": true, "paths": ["./scripts"], "policy": {"mode": "user"}},
      "web_search": {"provider": "duckduckgo"}
    }
  },
  "comment_file": "keep-me.md"
}`)

	if err := migrateShellCapability(lingtaiDir); err != nil {
		t.Fatalf("migrateShellCapability: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	manifest := doc["manifest"].(map[string]interface{})
	caps := manifest["capabilities"].(map[string]interface{})
	if _, ok := caps["bash"]; ok {
		t.Fatalf("legacy bash key survived: %#v", caps)
	}
	shell := caps["shell"].(map[string]interface{})
	if shell["yolo"] != true || !reflect.DeepEqual(shell["paths"], []interface{}{"./scripts"}) {
		t.Fatalf("shell configuration changed: %#v", shell)
	}
	if shell["policy"].(map[string]interface{})["mode"] != "user" || doc["comment_file"] != "keep-me.md" {
		t.Fatalf("unrelated init configuration changed: %#v", doc)
	}
}

func TestPortalM040FailsClosedBeforeRewritingSiblings(t *testing.T) {
	lingtaiDir := t.TempDir()
	goodPath := writeAgentInitPortal(t, lingtaiDir, "alice", `{"manifest":{"capabilities":{"bash":{"yolo":true}}}}`)
	conflictPath := writeAgentInitPortal(t, lingtaiDir, "bob", `{"manifest":{"capabilities":{"bash":{"yolo":true},"shell":{"yolo":false}}}}`)
	malformedPath := writeAgentInitPortal(t, lingtaiDir, "carol", `{"manifest":`)
	nonObjectManifestPath := writeAgentInitPortal(t, lingtaiDir, "dave", `{"manifest":[]}`)
	nonObjectCapabilitiesPath := writeAgentInitPortal(t, lingtaiDir, "erin", `{"manifest":{"capabilities":null}}`)

	if err := migrateShellCapability(lingtaiDir); err == nil {
		t.Fatal("unknown/conflicting init shapes returned nil error")
	} else {
		for _, want := range []string{"bob", "carol", "dave", "erin", "conflicting", "malformed", "manifest", "capabilities"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error %q does not identify %q", err, want)
			}
		}
	}
	for name, path := range map[string]string{
		"valid sibling": goodPath, "conflict": conflictPath,
		"malformed": malformedPath, "non-object manifest": nonObjectManifestPath,
		"non-object capabilities": nonObjectCapabilitiesPath,
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if name == "valid sibling" && !strings.Contains(string(data), `"bash"`) {
			t.Fatalf("%s was rewritten despite preflight failure: %s", name, data)
		}
	}
}

func TestPortalM040PreservesLargeLegacyShellNumberAndAdvancesVersion(t *testing.T) {
	lingtaiDir := t.TempDir()
	path := writeAgentInitPortal(t, lingtaiDir, "alice", `{"manifest":{"capabilities":{"bash":{"tenant_id":9007199254740993}}}}`)
	writeMeta(t, lingtaiDir, 39)

	if err := Run(lingtaiDir); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := readMeta(t, lingtaiDir).Version; got != CurrentVersion {
		t.Fatalf("version = %d, want %d", got, CurrentVersion)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := portalShellCapabilityDocument(data)
	if err != nil {
		t.Fatalf("decode rewritten init.json: %v", err)
	}
	manifest := doc["manifest"].(map[string]interface{})
	caps := manifest["capabilities"].(map[string]interface{})
	shell := caps["shell"].(map[string]interface{})
	if got := shell["tenant_id"].(json.Number); got != json.Number("9007199254740993") {
		t.Fatalf("tenant_id = %q, want exact token 9007199254740993", got)
	}
}

func TestPortalM040LargeNumberConflictFailsClosedWithoutRewriteOrVersionAdvance(t *testing.T) {
	lingtaiDir := t.TempDir()
	content := `{"manifest":{"capabilities":{"bash":{"tenant_id":9007199254740993},"shell":{"tenant_id":9007199254740992}}}}`
	path := writeAgentInitPortal(t, lingtaiDir, "alice", content)
	writeMeta(t, lingtaiDir, 39)

	err := Run(lingtaiDir)
	if err == nil || !strings.Contains(err.Error(), "bash") || !strings.Contains(err.Error(), "shell") {
		t.Fatalf("conflict error = %v, want explicit bash/shell error", err)
	}
	if got := readMeta(t, lingtaiDir).Version; got != 39 {
		t.Fatalf("version advanced to %d after conflict", got)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Fatalf("conflicting init.json was rewritten: %s", got)
	}
}

func TestPortalM040UpgradeFromBelow39PreservesLargeLegacyCapabilityThroughM038AndM039(t *testing.T) {
	lingtaiDir := t.TempDir()
	path := writeAgentInitPortal(t, lingtaiDir, "alice", `{"manifest":{"context_limit":4096,"capabilities":{"bash":{"tenant_id":9007199254740993}}}}`)
	writeMeta(t, lingtaiDir, 37)

	if err := Run(lingtaiDir); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := readMeta(t, lingtaiDir).Version; got != CurrentVersion {
		t.Fatalf("version = %d, want %d", got, CurrentVersion)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"tenant_id": 9007199254740993`) {
		t.Fatalf("large capability number changed during m038/m039/m040 upgrade: %s", data)
	}
}

func TestPortalM040UpgradeFromBelow39PreflightsLargeNumberConflictBeforeM038(t *testing.T) {
	lingtaiDir := t.TempDir()
	content := `{"manifest":{"capabilities":{"bash":{"tenant_id":9007199254740993},"shell":{"tenant_id":9007199254740992}}}}`
	path := writeAgentInitPortal(t, lingtaiDir, "alice", content)
	writeMeta(t, lingtaiDir, 37)

	err := Run(lingtaiDir)
	if err == nil || !strings.Contains(err.Error(), "bash") || !strings.Contains(err.Error(), "shell") {
		t.Fatalf("conflict error = %v, want explicit bash/shell error", err)
	}
	if got := readMeta(t, lingtaiDir).Version; got != 37 {
		t.Fatalf("version advanced to %d after preflight conflict", got)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Fatalf("conflicting init.json was rewritten before m038: %s", got)
	}
}
