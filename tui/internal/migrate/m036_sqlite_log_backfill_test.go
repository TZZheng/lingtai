package migrate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSQLiteDiscoverBackfillCandidatesMissingSidecar(t *testing.T) {
	lingtaiDir := t.TempDir()
	agentDir := makeSQLiteBackfillAgentWithEvents(t, lingtaiDir, "agent-a")

	candidates, skipped, err := sqliteDiscoverBackfillCandidates("/definitely/not/used", lingtaiDir)
	if err != nil {
		t.Fatalf("sqliteDiscoverBackfillCandidates: %v", err)
	}
	if skipped != 0 {
		t.Fatalf("skipped running = %d, want 0", skipped)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(candidates))
	}
	if candidates[0].AgentDir != agentDir {
		t.Fatalf("candidate agent dir = %q, want %q", candidates[0].AgentDir, agentDir)
	}
	if !strings.Contains(candidates[0].Reason, "missing") {
		t.Fatalf("reason = %q, want missing sidecar", candidates[0].Reason)
	}
}

func TestSQLiteDiscoverBackfillCandidatesExistingSidecarWithoutCursor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake python shell script is POSIX-only")
	}
	lingtaiDir := t.TempDir()
	agentDir := makeSQLiteBackfillAgentWithEvents(t, lingtaiDir, "agent-a")
	if err := os.WriteFile(filepath.Join(agentDir, "logs", "log.sqlite"), []byte("placeholder"), 0o644); err != nil {
		t.Fatal(err)
	}
	python := fakeSQLitePython(t, "printf '[]\\n'")

	candidates, skipped, err := sqliteDiscoverBackfillCandidates(python, lingtaiDir)
	if err != nil {
		t.Fatalf("sqliteDiscoverBackfillCandidates: %v", err)
	}
	if skipped != 0 {
		t.Fatalf("skipped running = %d, want 0", skipped)
	}
	if len(candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(candidates))
	}
	if !strings.Contains(candidates[0].Reason, "not been backfilled") {
		t.Fatalf("reason = %q, want missing backfill cursor", candidates[0].Reason)
	}
}

func TestSQLiteDiscoverBackfillCandidatesExistingSidecarWithCursor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake python shell script is POSIX-only")
	}
	lingtaiDir := t.TempDir()
	agentDir := makeSQLiteBackfillAgentWithEvents(t, lingtaiDir, "agent-a")
	if err := os.WriteFile(filepath.Join(agentDir, "logs", "log.sqlite"), []byte("placeholder"), 0o644); err != nil {
		t.Fatal(err)
	}
	python := fakeSQLitePython(t, "printf '[{\"byte_offset\":12,\"line_no\":1}]\\n'")

	candidates, skipped, err := sqliteDiscoverBackfillCandidates(python, lingtaiDir)
	if err != nil {
		t.Fatalf("sqliteDiscoverBackfillCandidates: %v", err)
	}
	if skipped != 0 {
		t.Fatalf("skipped running = %d, want 0", skipped)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidates = %d, want 0", len(candidates))
	}
}

func TestSQLiteRebuildProgressScriptNamespaceFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PYTHONPATH shell setup is POSIX-only")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available: ", err)
	}

	block := extractSQLiteImportBlock(sqliteRebuildProgressScript)
	if block == "" {
		t.Fatal("could not extract cross-version import block from sqliteRebuildProgressScript")
	}
	if !strings.Contains(block, "except ModuleNotFoundError as exc:") {
		t.Fatalf("import block must catch ModuleNotFoundError, got:\n%s", block)
	}

	t.Run("new-only", func(t *testing.T) {
		dir := t.TempDir()
		writeFakeLoggingPackage(t, dir, "new")
		out, _, err := runSQLiteImportBlock(t, python, dir, block)
		if err != nil {
			t.Fatalf("new-only namespace failed: %v\n%s", err, out)
		}
		if !strings.Contains(out, `"namespace": "new"`) {
			t.Fatalf("expected new namespace, got:\n%s", out)
		}
	})

	t.Run("old-only", func(t *testing.T) {
		dir := t.TempDir()
		writeFakeLoggingPackage(t, dir, "old")
		out, _, err := runSQLiteImportBlock(t, python, dir, block)
		if err != nil {
			t.Fatalf("old-only fallback failed: %v\n%s", err, out)
		}
		if !strings.Contains(out, `"namespace": "old"`) {
			t.Fatalf("expected old namespace fallback, got:\n%s", out)
		}
	})

	t.Run("lingtai-present-kernel-absent", func(t *testing.T) {
		dir := t.TempDir()
		writePartialNewKernelPackage(t, dir)
		out, _, err := runSQLiteImportBlock(t, python, dir, block)
		if err != nil {
			t.Fatalf("fallback with lingtai present but kernel absent failed: %v\n%s", err, out)
		}
		if !strings.Contains(out, `"namespace": "old"`) {
			t.Fatalf("expected old namespace fallback, got:\n%s", out)
		}
	})

	t.Run("broken-new-runtime-surfaces", func(t *testing.T) {
		dir := t.TempDir()
		writeBrokenNewKernelPackage(t, dir)
		out, _, err := runSQLiteImportBlock(t, python, dir, block)
		if err == nil {
			t.Fatalf("expected error from broken new runtime, got:\n%s", out)
		}
		if !strings.Contains(out, "lingtai.kernel.services.logging") {
			t.Fatalf("expected internal ModuleNotFoundError to surface, got:\n%s", out)
		}
		if strings.Contains(out, `"namespace": "old"`) {
			t.Fatalf("old namespace was loaded despite broken new runtime:\n%s", out)
		}
	})
}

func extractSQLiteImportBlock(script string) string {
	lines := strings.Split(script, "\n")
	start := -1
	end := -1
	for i, line := range lines {
		if strings.Contains(line, "Cross-version compatibility:") {
			start = i
		}
		if start != -1 && strings.TrimSpace(line) == "from lingtai_kernel.services import logging as logmod" {
			end = i
			break
		}
	}
	if start == -1 || end == -1 {
		return ""
	}
	return strings.Join(lines[start:end+1], "\n")
}

func runSQLiteImportBlock(t *testing.T, python, packagesDir, block string) (string, *exec.Cmd, error) {
	t.Helper()
	py := filepath.Join(t.TempDir(), "probe.py")
	script := block + "\nimport json\nprint(json.dumps({\"namespace\": logmod.__namespace__, \"file\": logmod.__file__}, ensure_ascii=False))\n"
	if err := os.WriteFile(py, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	// -S skips site-packages so an editable-installed lingtai_kernel cannot
	// shadow the fake packages; PYTHONPATH is limited to the temp package root.
	cmd := exec.Command(python, "-S", py)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "PYTHONPATH=") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	cmd.Env = append(cmd.Env, "PYTHONPATH="+packagesDir)
	out, err := cmd.CombinedOutput()
	return string(out), cmd, err
}

func writeFakeLoggingPackage(t *testing.T, root, namespace string) {
	t.Helper()
	var path string
	if namespace == "new" {
		path = filepath.Join(root, "lingtai", "kernel", "services", "logging.py")
	} else {
		path = filepath.Join(root, "lingtai_kernel", "services", "logging.py")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := fmt.Sprintf("__namespace__ = %q\n__file__ = %q\n", namespace, path)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writePartialNewKernelPackage(t *testing.T, root string) {
	t.Helper()
	lingtaiInit := filepath.Join(root, "lingtai", "__init__.py")
	if err := os.MkdirAll(filepath.Dir(lingtaiInit), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lingtaiInit, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	writeFakeLoggingPackage(t, root, "old")
}

func writeBrokenNewKernelPackage(t *testing.T, root string) {
	t.Helper()
	for _, p := range []string{"lingtai/__init__.py", "lingtai/kernel/__init__.py"} {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	servicesInit := filepath.Join(root, "lingtai", "kernel", "services", "__init__.py")
	if err := os.MkdirAll(filepath.Dir(servicesInit), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `raise ModuleNotFoundError("simulated broken logging", name="lingtai.kernel.services.logging")` + "\n"
	if err := os.WriteFile(servicesInit, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	writeFakeLoggingPackage(t, root, "old")
}

func TestSQLiteBackfillPromptCopyEmphasizesCostProgressAndSafeSkip(t *testing.T) {
	var b strings.Builder
	sqlitePrintBackfillPrompt(&b, []sqliteBackfillCandidate{{Name: "agent-a", EventsBytes: 2048, Reason: "SQLite sidecar is missing"}}, 1)
	out := b.String()
	for _, want := range []string{
		"can take a long time",
		"Skipping is safe",
		"does not affect normal LingTai use",
		"Backfill historical logs now? [y/N]",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("prompt missing %q:\n%s", want, out)
		}
	}
}

func makeSQLiteBackfillAgentWithEvents(t *testing.T, lingtaiDir, name string) string {
	t.Helper()
	agentDir := filepath.Join(lingtaiDir, name)
	logsDir := filepath.Join(agentDir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "init.json"), []byte(`{"manifest":{"agent_name":"`+name+`"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logsDir, "events.jsonl"), []byte(`{"type":"test","ts":1}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return agentDir
}

func fakeSQLitePython(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "python")
	content := "#!/bin/sh\nset -eu\n" + body + "\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
