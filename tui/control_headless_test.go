package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHeadlessControlSleep(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "lingtai-tui")
	build := exec.Command("go", "build", "-o", bin, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build lingtai-tui: %v\n%s", err, output)
	}

	root := t.TempDir()
	project := filepath.Join(root, ".lingtai")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	writeManifest := func(dir, body string) {
		t.Helper()
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".agent.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	agentDir := filepath.Join(project, "worker")
	writeManifest(agentDir, `{"agent_id":"private-worker-id","agent_name":"Worker","admin":{}}`)
	humanDir := filepath.Join(project, "human")
	writeManifest(humanDir, `{"agent_id":"private-human-id","agent_name":"Human","admin":null}`)
	nonAgentDir := filepath.Join(project, "notes")
	if err := os.Mkdir(nonAgentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outsideDir := filepath.Join(root, "outside-agent")
	writeManifest(outsideDir, `{"agent_id":"private-outside-id","agent_name":"Outside","admin":{}}`)
	linkDir := filepath.Join(project, "linked-agent")
	if err := os.Symlink(outsideDir, linkDir); err != nil {
		t.Fatalf("create symlink escape fixture: %v", err)
	}

	run := func(agent string) (stdout, stderr string, err error) {
		t.Helper()
		cmd := exec.Command(bin, "control", "--project", project, "--agent", agent, "sleep")
		var outBuf, errBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf
		err = cmd.Run()
		return outBuf.String(), errBuf.String(), err
	}

	stdout, stderr, err := run("worker")
	if err != nil {
		t.Fatalf("safe control failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("safe control stderr = %q, want empty", stderr)
	}
	var success struct {
		Command string `json:"command"`
		Agent   string `json:"agent"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &success); err != nil {
		t.Fatalf("decode success JSON %q: %v", stdout, err)
	}
	if success.Command != "sleep" || success.Agent != "worker" || success.Status != "signaled" {
		t.Fatalf("success = %+v, want command=sleep agent=worker status=signaled", success)
	}
	if strings.Contains(stdout, "private-worker-id") {
		t.Fatalf("success output exposes private agent ID: %s", stdout)
	}
	if data, err := os.ReadFile(filepath.Join(agentDir, ".sleep")); err != nil {
		t.Fatalf("safe control did not write .sleep: %v", err)
	} else if len(data) != 0 {
		t.Fatalf(".sleep contains %d bytes, want empty", len(data))
	}

	for _, tc := range []struct {
		name   string
		agent  string
		target string
	}{
		{name: "traversal", agent: "../outside-agent", target: outsideDir},
		{name: "human", agent: "human", target: humanDir},
		{name: "non-Agent", agent: "notes", target: nonAgentDir},
		{name: "symlink escape", agent: "linked-agent", target: outsideDir},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := run(tc.agent)
			exitErr, ok := err.(*exec.ExitError)
			if !ok || exitErr.ExitCode() != 1 {
				t.Fatalf("exit = %v, want code 1; stdout=%q stderr=%q", err, stdout, stderr)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty on failure", stdout)
			}
			var failure struct {
				Error string `json:"error"`
				Code  string `json:"code"`
			}
			if err := json.Unmarshal([]byte(stderr), &failure); err != nil {
				t.Fatalf("decode error JSON %q: %v", stderr, err)
			}
			if failure.Code != "invalid_agent" || failure.Error == "" {
				t.Fatalf("failure = %+v, want nonempty error and code invalid_agent", failure)
			}
			if _, err := os.Stat(filepath.Join(tc.target, ".sleep")); !os.IsNotExist(err) {
				t.Fatalf("rejected target wrote .sleep: %v", err)
			}
		})
	}
}
