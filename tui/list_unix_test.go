//go:build !windows

package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestHumanUptimeFromEtime(t *testing.T) {
	cases := []struct{ etime, want string }{
		{"04:09", "4m 9s"},
		{"00:05", "0m 5s"},
		{"01:02:03", "1h 2m"},
		{"23:59:59", "23h 59m"},
		{"1-00:00:01", "1d 0h"},
		{"2-03:04:05", "2d 3h"},
		{" 12:34 ", "12m 34s"},
		// Unparseable values pass through untouched rather than being guessed.
		{"", ""},
		{"?", "?"},
		{"1:2:3:4", "1:2:3:4"},
		{"x-00:01", "x-00:01"},
	}
	for _, c := range cases {
		if got := humanUptimeFromEtime(c.etime); got != c.want {
			t.Errorf("humanUptimeFromEtime(%q) = %q, want %q", c.etime, got, c.want)
		}
	}
}

func TestListMainFailsLoudOnScanError(t *testing.T) {
	if os.Getenv("LINGTAI_TEST_LIST_SCAN_FAIL") == "1" {
		os.Args = []string{"lingtai-tui", "list"}
		listMain()
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run", "^TestListMainFailsLoudOnScanError$")
	cmd.Env = append(os.Environ(),
		"LINGTAI_TEST_LIST_SCAN_FAIL=1",
		"PATH="+t.TempDir(), // ps is unreachable, so the scan command fails
	)
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
	if strings.Contains(stdout.String(), "No lingtai processes running") {
		t.Fatalf("stdout = %q, scan failure must not masquerade as an empty process list", stdout.String())
	}
}
