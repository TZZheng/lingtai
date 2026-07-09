//go:build !windows

package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestPurgeMainFailsLoudOnScanError(t *testing.T) {
	if os.Getenv("LINGTAI_TEST_PURGE_SCAN_FAIL") == "1" {
		os.Args = []string{"lingtai-tui", "purge"}
		purgeMain()
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run", "^TestPurgeMainFailsLoudOnScanError$")
	cmd.Env = append(os.Environ(),
		"LINGTAI_TEST_PURGE_SCAN_FAIL=1",
		"PATH="+t.TempDir(), // ps is unreachable, so the scan command fails
	)
	cmd.Stdin = strings.NewReader("n\n") // never reached; keeps a regression from hanging on the kill prompt
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected nonzero exit when ps is unavailable, got err=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	if code := exitErr.ExitCode(); code != 1 {
		t.Fatalf("exit code = %d, want 1 (stderr=%q)", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "error running ps") {
		t.Fatalf("stderr = %q, want it to report the ps failure", stderr.String())
	}
	if strings.Contains(stdout.String(), "No lingtai processes found") {
		t.Fatalf("stdout = %q, scan failure must not masquerade as an empty process list", stdout.String())
	}
}
