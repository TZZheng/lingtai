package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/lingtai-tui/internal/processscan"
)

func TestParseListArgsModesAndDir(t *testing.T) {
	opts, err := parseListArgs([]string{"--detailed", "--admin", "--json", "./project"})
	if err != nil {
		t.Fatalf("parseListArgs returned error: %v", err)
	}
	if !opts.Detailed || !opts.Admin || !opts.JSON {
		t.Fatalf("expected detailed/admin/json true, got detailed=%v admin=%v json=%v", opts.Detailed, opts.Admin, opts.JSON)
	}
	want, _ := filepath.Abs("./project")
	if opts.FilterDir != want {
		t.Fatalf("FilterDir=%q, want %q", opts.FilterDir, want)
	}
}

func TestListProcsFromAgentProcessesPreservesSpacesAndFilter(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "Project With Spaces")
	agentDir := filepath.Join(project, ".lingtai", "agent A")
	otherProject := filepath.Join(root, "Other Project")
	otherAgentDir := filepath.Join(otherProject, ".lingtai", "agent B")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got := listProcsFromAgentProcesses([]processscan.AgentProcess{
		{PID: 111, Uptime: "00:01:02", AgentDir: agentDir},
		{PID: 222, Uptime: "00:02:03", AgentDir: otherAgentDir},
		{PID: 333, Uptime: "00:03:04", AgentDir: agentDir},
	}, project, 333)

	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %+v", len(got), got)
	}
	proc := got[0]
	if proc.PID != "111" || proc.Uptime != "00:01:02" {
		t.Fatalf("unexpected pid/uptime: %+v", proc)
	}
	if proc.Agent != "agent A" || proc.Project != project || proc.Dir != agentDir {
		t.Fatalf("space-containing path was not preserved: %+v", proc)
	}
	if phantoms := detectPhantomDirs(got, ""); phantoms[project] {
		t.Fatalf("existing project with spaces should not be phantom: %+v", phantoms)
	}
}

func TestCollapseListProcsKeepsDistinctSpacePaths(t *testing.T) {
	root := t.TempDir()
	dirA := filepath.Join(root, "Project With Spaces", ".lingtai", "agent A")
	dirB := filepath.Join(root, "Project With Spaces", ".lingtai", "agent B")
	got := collapseListProcsByAgentDir([]listProc{
		{PID: "111", Agent: "agent A", Dir: dirA},
		{PID: "222", Agent: "agent B", Dir: dirB},
	})
	if len(got) != 2 {
		t.Fatalf("distinct full dirs should not collapse, got %+v", got)
	}
}

func TestPrintListJSONIncludesHeartbeatAndLock(t *testing.T) {
	project := t.TempDir()
	agentDir := filepath.Join(project, ".lingtai", "agent-a")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	heartbeat := fmt.Sprintf("%.6f", float64(time.Now().UnixNano())/1e9)
	if err := os.WriteFile(filepath.Join(agentDir, ".agent.heartbeat"), []byte(heartbeat), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, ".agent.lock"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	procs := []listProc{{
		PID:     "123",
		Uptime:  "1s",
		Agent:   "agent-a",
		Project: project,
		Dir:     agentDir,
		Info: listAgentInfo{
			Address:   "agent-a",
			AgentName: "agent-a",
			State:     "asleep",
		},
	}}

	var buf bytes.Buffer
	printListJSON(&buf, procs, nil, listOptions{JSON: true})
	var parsed listJSONOutput
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if parsed.Count != 1 || len(parsed.Processes) != 1 {
		t.Fatalf("unexpected process count: %+v", parsed)
	}
	got := parsed.Processes[0]
	if !got.Heartbeat.Fresh {
		t.Fatalf("heartbeat should be fresh: %+v", got.Heartbeat)
	}
	if !got.LockExists {
		t.Fatal("lock_exists should be true")
	}
}

func TestCollapseListProcsPrefersRuntimeStatusPID(t *testing.T) {
	project := t.TempDir()
	agentDir := filepath.Join(project, ".lingtai", "agent-a")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	status := []byte(`{"runtime":{"pid":222,"running":true}}`)
	if err := os.WriteFile(filepath.Join(agentDir, ".status.json"), status, 0o644); err != nil {
		t.Fatal(err)
	}

	got := collapseListProcsByAgentDir([]listProc{
		{PID: "111", Agent: "agent-a", Project: project, Dir: agentDir},
		{PID: "222", Agent: "agent-a", Project: project, Dir: agentDir},
	})
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %+v", len(got), got)
	}
	if got[0].PID != "222" {
		t.Fatalf("PID = %q, want status pid 222", got[0].PID)
	}
}

func TestParseListArgsRejectsUnknownFlag(t *testing.T) {
	if _, err := parseListArgs([]string{"--verbose"}); err == nil {
		t.Fatal("expected unknown flag error")
	}
}

func adminRawFromJSON(raw string) interface{} {
	var manifest map[string]interface{}
	_ = json.Unmarshal([]byte(raw), &manifest)
	return manifest["admin"]
}

func TestSummarizeAdmin(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"null", `{"admin": null}`, "admin=null"},
		{"empty", `{"admin": {}}`, "admin={}"},
		{"sorted", `{"admin": {"nirvana": false, "karma": true}}`, "karma=true,nirvana=false"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := summarizeAdmin(adminRawFromJSON(tc.raw)); got != tc.want {
				t.Fatalf("summarizeAdmin=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestRoleLabel(t *testing.T) {
	if got := roleLabel(listAgentInfo{IsOrchestrator: true}); got != "MAIN" {
		t.Fatalf("orchestrator role=%q", got)
	}
	if got := roleLabel(listAgentInfo{IsHuman: true}); got != "HUMAN" {
		t.Fatalf("human role=%q", got)
	}
	if got := roleLabel(listAgentInfo{}); got != "AGENT" {
		t.Fatalf("agent role=%q", got)
	}
}

func TestPrintListDetailedAndAdmin(t *testing.T) {
	procs := []listProc{
		{
			PID:     "123",
			Uptime:  "1m 0s",
			Agent:   "mimo-1",
			Project: "/tmp/project",
			Dir:     "/tmp/project/.lingtai/mimo-1",
			Info: listAgentInfo{
				Address:        "mimo-1",
				AgentName:      "mimo-1",
				Nickname:       "Mimo",
				State:          "IDLE",
				IsOrchestrator: true,
				AdminSummary:   "karma=true",
				IMHandles:      "telegram:@Lingtaidev1bot",
			},
		},
	}

	var detailed bytes.Buffer
	printList(&detailed, procs, nil, listOptions{Detailed: true}, true)
	detailedOut := detailed.String()
	for _, want := range []string{"ROLE", "MAIN", "IDLE", "mimo-1", "Mimo", "telegram:@Lingtaidev1bot"} {
		if !strings.Contains(detailedOut, want) {
			t.Fatalf("detailed output missing %q:\n%s", want, detailedOut)
		}
	}

	var admin bytes.Buffer
	printList(&admin, procs, nil, listOptions{Admin: true, Detailed: true}, true)
	adminOut := admin.String()
	for _, want := range []string{"ADMIN", "karma=true", "MAIN"} {
		if !strings.Contains(adminOut, want) {
			t.Fatalf("admin output missing %q:\n%s", want, adminOut)
		}
	}
}
