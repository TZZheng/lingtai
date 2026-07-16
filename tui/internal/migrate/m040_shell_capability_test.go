package migrate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestM040CanonicalizesLegacyShellCapabilityAndPreservesConfig(t *testing.T) {
	lingtaiDir := t.TempDir()
	initPath := writeAgentInit(t, lingtaiDir, "alice", `{
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
	var doc map[string]interface{}
	data, err := os.ReadFile(initPath)
	if err != nil {
		t.Fatal(err)
	}
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
	if shell["policy"].(map[string]interface{})["mode"] != "user" {
		t.Fatalf("nested shell configuration changed: %#v", shell)
	}
	if doc["comment_file"] != "keep-me.md" || caps["web_search"].(map[string]interface{})["provider"] != "duckduckgo" {
		t.Fatalf("unrelated init configuration changed: %#v", doc)
	}
}

func TestM040ConflictFailsClosedAndDoesNotRewrite(t *testing.T) {
	lingtaiDir := t.TempDir()
	content := `{"manifest":{"capabilities":{"bash":{"yolo":true},"shell":{"yolo":false}}}}`
	path := writeAgentInit(t, lingtaiDir, "alice", content)

	err := migrateShellCapability(lingtaiDir)
	if err == nil || !strings.Contains(err.Error(), "bash") || !strings.Contains(err.Error(), "shell") {
		t.Fatalf("conflict error = %v, want explicit bash/shell error", err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != content {
		t.Fatalf("conflicting init.json was rewritten: %s", got)
	}
}

func TestM040RejectsUnknownShapesBeforeAnyRewrite(t *testing.T) {
	lingtaiDir := t.TempDir()
	goodPath := writeAgentInit(t, lingtaiDir, "alice", `{"manifest":{"capabilities":{"bash":{"yolo":true}}}}`)
	malformedPath := writeAgentInit(t, lingtaiDir, "bob", `{"manifest":`)
	nonObjectManifestPath := writeAgentInit(t, lingtaiDir, "carol", `{"manifest":[]}`)
	nonObjectCapabilitiesPath := writeAgentInit(t, lingtaiDir, "dave", `{"manifest":{"capabilities":null}}`)

	writeMeta(t, lingtaiDir, 39)
	err := Run(lingtaiDir)
	if err == nil {
		t.Fatal("unknown init shapes returned nil error")
	}
	if meta := readMeta(t, lingtaiDir); meta.Version != 39 {
		t.Fatalf("failed m040 advanced meta version to %d", meta.Version)
	}
	for _, want := range []string{"bob", "carol", "dave", "malformed", "manifest", "capabilities"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not identify %q", err, want)
		}
	}
	for name, path := range map[string]string{
		"valid sibling": goodPath, "malformed": malformedPath,
		"non-object manifest": nonObjectManifestPath, "non-object capabilities": nonObjectCapabilitiesPath,
	} {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if name == "valid sibling" && !strings.Contains(string(data), `"bash"`) {
			t.Fatalf("%s was rewritten despite preflight failure: %s", name, data)
		}
		if name != "valid sibling" && string(data) == "" {
			t.Fatalf("%s unexpectedly became empty", name)
		}
	}
}

func TestM040Idempotent(t *testing.T) {
	lingtaiDir := t.TempDir()
	path := writeAgentInit(t, lingtaiDir, "alice", `{"manifest":{"capabilities":{"shell":{"yolo":true}}}}`)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrateShellCapability(lingtaiDir); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("canonical init.json was rewritten: %s", after)
	}
}

func TestM040SkipsNonAgentDirectories(t *testing.T) {
	lingtaiDir := t.TempDir()
	humanPath := writeAgentInit(t, lingtaiDir, "human", `{"manifest":{"capabilities":{"bash":{}}}}`)
	dotPath := writeAgentInit(t, lingtaiDir, ".portal", `{"manifest":{"capabilities":{"bash":{}}}}`)
	if err := migrateShellCapability(lingtaiDir); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{humanPath, dotPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), `"bash"`) {
			t.Fatalf("non-agent init.json was changed: %s", filepath.Base(path))
		}
	}
}

func TestM040PreservesLargeLegacyShellNumberAndAdvancesVersion(t *testing.T) {
	lingtaiDir := t.TempDir()
	path := writeAgentInit(t, lingtaiDir, "alice", `{"manifest":{"capabilities":{"bash":{"tenant_id":9007199254740993}}}}`)
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
	doc, err := shellCapabilityDocument(data)
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

func TestM040LargeNumberConflictFailsClosedWithoutRewriteOrVersionAdvance(t *testing.T) {
	lingtaiDir := t.TempDir()
	content := `{"manifest":{"capabilities":{"bash":{"tenant_id":9007199254740993},"shell":{"tenant_id":9007199254740992}}}}`
	path := writeAgentInit(t, lingtaiDir, "alice", content)
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

func TestM040UpgradeFromBelow39PreservesLargeLegacyCapabilityThroughM038AndM039(t *testing.T) {
	lingtaiDir := t.TempDir()
	path := writeAgentInit(t, lingtaiDir, "alice", `{"manifest":{"context_limit":4096,"capabilities":{"bash":{"tenant_id":9007199254740993}}}}`)
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

func TestM040UpgradeFromBelow39PreflightsLargeNumberConflictBeforeM038(t *testing.T) {
	lingtaiDir := t.TempDir()
	content := `{"manifest":{"capabilities":{"bash":{"tenant_id":9007199254740993},"shell":{"tenant_id":9007199254740992}}}}`
	path := writeAgentInit(t, lingtaiDir, "alice", content)
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
