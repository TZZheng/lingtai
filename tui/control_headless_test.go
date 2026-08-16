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

func TestHeadlessControlRegistry(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "lingtai-tui")
	build := exec.Command("go", "build", "-o", bin, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build lingtai-tui: %v\n%s", err, output)
	}

	project := filepath.Join(t.TempDir(), ".lingtai")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	workerDir := filepath.Join(project, "worker")
	if err := os.Mkdir(workerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workerDir, ".agent.json"), []byte(`{"agent_id":"private-worker-id","agent_name":"Private Worker","admin":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	assertQuietFailure := func(t *testing.T, code string, args ...string) {
		t.Helper()
		cmd := exec.Command(bin, append([]string{"control", "--project", project, "--agent", "worker"}, args...)...)
		var outBuf, errBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf
		err := cmd.Run()
		exitErr, ok := err.(*exec.ExitError)
		if !ok || exitErr.ExitCode() != 1 {
			t.Fatalf("exit = %v, want code 1; stdout=%q stderr=%q", err, outBuf.String(), errBuf.String())
		}
		if outBuf.Len() != 0 {
			t.Fatalf("stdout = %q, want empty on failure", outBuf.String())
		}
		var failure struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		if err := json.Unmarshal(errBuf.Bytes(), &failure); err != nil {
			t.Fatalf("decode error JSON %q: %v", errBuf.String(), err)
		}
		if failure.Code != code || failure.Error == "" {
			t.Fatalf("failure = %+v, want code %q with nonempty error", failure, code)
		}
		entries, err := os.ReadDir(workerDir)
		if err != nil {
			t.Fatalf("read worker dir: %v", err)
		}
		if len(entries) != 1 || entries[0].Name() != ".agent.json" {
			t.Fatalf("worker dir = %v, want no lifecycle signal written", entries)
		}
	}

	for _, tc := range []struct {
		name string
		args []string
		code string
	}{
		{name: "unknown command", args: []string{"bogus"}, code: "invalid_command"},
		{name: "sleep with unexpected argument", args: []string{"sleep", "now"}, code: "invalid_args"},
		{name: "suspend with unexpected argument", args: []string{"suspend", "now"}, code: "invalid_args"},
		{name: "cpr with unexpected argument", args: []string{"cpr", "all"}, code: "invalid_args"},
		{name: "clear with unexpected argument", args: []string{"clear", "all"}, code: "invalid_args"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertQuietFailure(t, tc.code, tc.args...)
		})
	}
}

func TestHeadlessControlSuspend(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "lingtai-tui")
	build := exec.Command("go", "build", "-o", bin, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build lingtai-tui: %v\n%s", err, output)
	}

	project := filepath.Join(t.TempDir(), ".lingtai")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	writeManifest := func(agent, body string) string {
		t.Helper()
		dir := filepath.Join(project, agent)
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".agent.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return dir
	}

	workerDir := writeManifest("worker", `{"agent_id":"private-worker-id","agent_name":"Private Worker","admin":{}}`)
	humanDir := writeManifest("human", `{"agent_id":"private-human-id","agent_name":"Private Human","admin":null}`)

	run := func(agent string) (stdout, stderr string, err error) {
		t.Helper()
		cmd := exec.Command(bin, "control", "--project", project, "--agent", agent, "suspend")
		var outBuf, errBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf
		err = cmd.Run()
		return outBuf.String(), errBuf.String(), err
	}

	assertWorkerSuccess := func(attempt string) {
		t.Helper()
		stdout, stderr, err := run("worker")
		if err != nil {
			t.Fatalf("%s suspend failed: %v\nstdout: %s\nstderr: %s", attempt, err, stdout, stderr)
		}
		if stderr != "" {
			t.Fatalf("%s suspend stderr = %q, want empty", attempt, stderr)
		}
		var success struct {
			Command   string          `json:"command"`
			Agent     string          `json:"agent"`
			Status    string          `json:"status"`
			AgentID   json.RawMessage `json:"agent_id"`
			AgentName json.RawMessage `json:"agent_name"`
		}
		if err := json.Unmarshal([]byte(stdout), &success); err != nil {
			t.Fatalf("decode %s success JSON %q: %v", attempt, stdout, err)
		}
		if success.Command != "suspend" || success.Agent != "worker" || success.Status != "signaled" {
			t.Fatalf("%s success = %+v, want command=suspend agent=worker status=signaled", attempt, success)
		}
		if len(success.AgentID) != 0 || len(success.AgentName) != 0 || strings.Contains(stdout, "private-worker-id") || strings.Contains(stdout, "Private Worker") {
			t.Fatalf("%s success output exposes private agent identity: %s", attempt, stdout)
		}
		if data, err := os.ReadFile(filepath.Join(workerDir, ".suspend")); err != nil {
			t.Fatalf("%s suspend did not write .suspend: %v", attempt, err)
		} else if len(data) != 0 {
			t.Fatalf("%s .suspend contains %d bytes, want empty", attempt, len(data))
		}
	}

	assertWorkerSuccess("initial")
	assertWorkerSuccess("repeated")

	stdout, stderr, err := run("human")
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 1 {
		t.Fatalf("human suspend exit = %v, want code 1; stdout=%q stderr=%q", err, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("human suspend stdout = %q, want empty", stdout)
	}
	var failure struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := json.Unmarshal([]byte(stderr), &failure); err != nil {
		t.Fatalf("decode human error JSON %q: %v", stderr, err)
	}
	if failure.Code != "invalid_agent" {
		t.Fatalf("human suspend failure = %+v, want code invalid_agent", failure)
	}
	if _, err := os.Stat(filepath.Join(humanDir, ".suspend")); !os.IsNotExist(err) {
		t.Fatalf("human suspend wrote .suspend: %v", err)
	}
}
