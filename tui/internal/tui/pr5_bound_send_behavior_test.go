package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/lingtai-tui/internal/fs"
)

func TestPR5Stage2OrdinarySendDefersBoundWriteAndPreservesMainSentinel(t *testing.T) {
	app, target := pr5OrdinarySendApp(t, "agent-a", "Agent A", 4101, 1)
	sentinel := pr5WriteMainInsightSentinel(t, app.projectDir)
	app.mail.input.SetValue("hello exact A1")

	prepared, cmd := installationDeliverApp(t, app, SendMsg{})
	pr5RequireInboxBodies(t, target)
	if got := prepared.mail.input.Value(); got != "hello exact A1" {
		t.Fatalf("draft changed before the bound send request reached its publication gate: got %q", got)
	}
	request := runCmd(cmd)
	if request == nil {
		t.Fatal("ordinary SendMsg returned no bound request command")
	}

	got, refreshCmd := installationDeliverApp(t, prepared, request)
	pr5RequireInboxBodies(t, target, "hello exact A1")
	if got.mail.input.Value() != "" || got.mail.pendingMessage != "" {
		t.Fatalf("successful bound send did not clear the exact draft: pending=%q input=%q", got.mail.pendingMessage, got.mail.input.Value())
	}
	if refreshCmd == nil {
		t.Fatal("successful bound send returned no steady refresh request")
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("ordinary send re-armed Main's shared insight sentinel: %v", err)
	}
}

func TestPR5Stage2StaleA1SendRequestCannotLandOnColdA2(t *testing.T) {
	app, targetA := pr5OrdinarySendApp(t, "agent-a", "Agent A", 4101, 1)
	sentinel := pr5WriteMainInsightSentinel(t, app.projectDir)
	app.mail.input.SetValue("stale A1 body")

	preparedA1, cmd := installationDeliverApp(t, app, SendMsg{})
	pr5RequireInboxBodies(t, targetA)
	request := runCmd(cmd)
	if request == nil {
		t.Fatal("A1 SendMsg returned no bound request command")
	}

	targetB := filepath.Join(preparedA1.projectDir, "agent-b")
	installationWriteAgent(t, targetB, "agent-b", "Agent B", "Agent B")
	pr5BindCoordinatorRailTarget(t, &preparedA1, targetB, "Agent B", 4201, 2)
	pr5BindCoordinatorRailTarget(t, &preparedA1, targetA, "Agent A", 4101, 3)
	preparedA1.mail.pendingMessage = "fresh A2 pending"
	preparedA1.mail.input.SetValue("fresh A2 input")

	got, followup := installationDeliverApp(t, preparedA1, request)
	if followup != nil {
		t.Fatalf("stale A1 send request returned effect %T", runCmd(followup))
	}
	pr5RequireInboxBodies(t, targetA)
	pr5RequireInboxBodies(t, targetB)
	if got.mail.pendingMessage != "fresh A2 pending" || got.mail.input.Value() != "fresh A2 input" {
		t.Fatalf("stale A1 send changed cold A2 draft: pending=%q input=%q", got.mail.pendingMessage, got.mail.input.Value())
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("stale A1 send re-armed Main's shared insight sentinel: %v", err)
	}
}

func TestPR5Stage2SendWriteErrorRetainsDraftStatusAndSentinel(t *testing.T) {
	app, target := pr5OrdinarySendApp(t, "agent-a", "Agent A", 4101, 1)
	sentinel := pr5WriteMainInsightSentinel(t, app.projectDir)
	// A mailbox path that is a regular file preserves a fully valid target
	// binding/revalidation while making the actual write fail deterministically.
	if err := os.WriteFile(filepath.Join(target, "mailbox"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	app.mail.pendingMessage = "unsent editor body"
	app.mail.input.SetValue("unsent editor body")

	prepared, cmd := installationDeliverApp(t, app, SendMsg{})
	if prepared.mail.pendingMessage != "unsent editor body" || prepared.mail.input.Value() != "unsent editor body" {
		t.Fatalf("draft cleared before attempted bound write: pending=%q input=%q", prepared.mail.pendingMessage, prepared.mail.input.Value())
	}
	request := runCmd(cmd)
	if request == nil {
		t.Fatal("failing SendMsg returned no bound request command")
	}
	got, followup := installationDeliverApp(t, prepared, request)
	if followup != nil {
		t.Fatalf("failed bound write requested refresh/effect %T", runCmd(followup))
	}
	if got.mail.pendingMessage != "unsent editor body" || got.mail.input.Value() != "unsent editor body" {
		t.Fatalf("failed bound write lost draft: pending=%q input=%q", got.mail.pendingMessage, got.mail.input.Value())
	}
	if got.mail.statusFlash == "" {
		t.Fatal("failed bound write did not expose a status error")
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("failed bound write removed Main's shared insight sentinel: %v", err)
	}
}

func pr5OrdinarySendApp(t *testing.T, address, name string, pid int, generation uint64) (App, string) {
	t.Helper()
	app, _, _ := installationNewApp(t, 0)
	// Give the fixture human a non-pseudo identity so a successful local send
	// lands in the exact target inbox (and a target-mailbox failure is observable).
	if err := os.MkdirAll(app.mail.humanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(app.mail.humanDir, ".agent.json"),
		[]byte(`{"address":"human","agent_name":"human","admin":{}}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	app.mailStore.setAsyncTargetRevalidator(func(asyncOwner, asyncTarget) bool { return true })
	target := filepath.Join(app.projectDir, address)
	installationWriteAgent(t, target, address, name, name)
	pr5BindCoordinatorRailTarget(t, &app, target, name, pid, generation)
	return app, target
}

func pr5WriteMainInsightSentinel(t *testing.T, projectDir string) string {
	t.Helper()
	assetDir := filepath.Join(projectDir, ".tui-asset")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(assetDir, ".insight.done")
	if err := os.WriteFile(sentinel, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return sentinel
}

func pr5RequireInboxBodies(t *testing.T, target string, want ...string) {
	t.Helper()
	messages, err := fs.ReadInbox(target)
	if err != nil {
		t.Fatalf("read inbox for %s: %v", target, err)
	}
	if len(messages) != len(want) {
		t.Fatalf("inbox bodies for %s: got %d messages, want %d (%v)", target, len(messages), len(want), want)
	}
	for i := range want {
		if messages[i].Message != want[i] {
			t.Fatalf("inbox body %d for %s = %q, want %q", i, target, messages[i].Message, want[i])
		}
	}
}
