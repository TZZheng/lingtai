package tui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/anthropics/lingtai-tui/internal/fs"
)

func TestPR5Stage4RailRowsRenderAcceptedUnreadCountsWithoutScanning(t *testing.T) {
	installationTestStart(t)

	app, scanner, _ := installationNewApp(t, 0)
	projectRoot := filepath.Dir(app.projectDir)
	targetADir := filepath.Join(app.projectDir, "agent-a")
	targetBDir := filepath.Join(app.projectDir, "agent-b")
	installationWriteAgent(t, targetADir, "agent-a", "Agent A", "Agent A")
	installationWriteAgent(t, targetBDir, "agent-b", "Agent B", "Agent B")

	acceptedInventory := pr5RailLifecycleSnapshot(app, "agent-a", "Agent A", 6401)
	agentB := pr5RailLifecycleSnapshot(app, "agent-b", "Agent B", 6402)
	acceptedInventory.Records = append(acceptedInventory.Records, agentB.Records...)
	inventoryScript := &pr5RailInventoryScanScript{steps: []pr5RailInventoryScanStep{{
		snapshot: acceptedInventory,
	}}}
	setter, ok := any(&app).(pr5AgentRailInventoryScannerSetter)
	if !ok {
		t.Fatal("App has no root-owned agent-rail inventory lifecycle boundary")
	}
	setter.setAgentRailInventoryScanner(inventoryScript.Scan)
	inventoryResult := pr5RunTrailingRailInventoryScan(t, app.Init(), inventoryScript)
	app, _ = installationDeliverApp(t, app, inventoryResult)
	pr5RequireRailLifecycleRows(t, app, []string{"Main", "Agent A", "Agent B"})

	historyA := pr5ProjectionMail(
		"historical-a", "agent-a", "human", nil,
		"historical Agent A mail", "2026-07-15T00:00:00Z",
	)
	historyB := pr5ProjectionMail(
		"historical-b", "agent-b", "human", nil,
		"historical Agent B mail", "2026-07-15T00:00:00Z",
	)
	scanner.messages = []fs.MailMessage{historyA, historyB}
	app, _ = installationAcceptInitial(t, app)

	baselineView := ansi.Strip(app.agentRail.View(24, 10))
	baselineA := pr5RailRenderedLine(t, baselineView, "Agent A")
	baselineB := pr5RailRenderedLine(t, baselineView, "Agent B")
	pr5RequireNoRenderedUnreadCount(t, baselineA, 1)
	pr5RequireNoRenderedUnreadCount(t, baselineB, 2)

	laterA := pr5ProjectionMail(
		"later-a", "agent-a", "human", nil,
		"later Agent A mail", "2026-07-15T00:01:00Z",
	)
	laterB1 := pr5ProjectionMail(
		"later-b-1", "agent-b", "human", nil,
		"later Agent B mail one", "2026-07-15T00:01:00Z",
	)
	laterB2 := pr5ProjectionMail(
		"later-b-2", "agent-b", "human", nil,
		"later Agent B mail two", "2026-07-15T00:02:00Z",
	)
	outgoingB := pr5ProjectionMail(
		"outgoing-b", "human", "agent-b", nil,
		"outgoing human mail", "2026-07-15T00:03:00Z",
	)
	ccOnlyA := pr5ProjectionMail(
		"cc-only-a", "agent-a", "someone-else", []string{"human"},
		"CC-only Agent A mail", "2026-07-15T00:04:00Z",
	)
	acceptedLater := []fs.MailMessage{historyA, historyB, laterA, laterB1, laterB2, outgoingB, ccOnlyA}
	scanner.messages = acceptedLater
	steady := installationRefreshResult(t, &app, false)
	app, _ = installationDeliverApp(t, app, steady)

	if app.railUnreadStore == nil || app.mailStore.snapshot == nil {
		t.Fatalf("accepted steady snapshot unread state: store=%v snapshot=%v, want both live", app.railUnreadStore != nil, app.mailStore.snapshot != nil)
	}
	targetA := fs.DirectTarget{Directory: targetADir, Address: "agent-a"}
	targetB := fs.DirectTarget{Directory: targetBDir, Address: "agent-b"}
	acceptedMessages := app.mailStore.snapshot.cache.Messages
	if got := app.railUnreadStore.UnreadCount(targetA, acceptedMessages, app.mail.humanAddr); got != 1 {
		t.Fatalf("accepted Agent A unread source count = %d, want 1", got)
	}
	if got := app.railUnreadStore.UnreadCount(targetB, acceptedMessages, app.mail.humanAddr); got != 2 {
		t.Fatalf("accepted Agent B unread source count = %d, want 2", got)
	}

	statePath := fs.RailUnreadStatePath(projectRoot)
	beforeState, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read unread state before rendering: %v", err)
	}
	beforeMailScans := scanner.scans.Load()
	beforeInventoryScans := inventoryScript.calls
	beforeSnapshot := app.mailStore.snapshot
	beforeVersion := app.mailStore.version

	// A future scanner value is not accepted state. Pure rendering must neither
	// scan it nor replace the counts projected from the accepted snapshot above.
	scanner.messages = append(append([]fs.MailMessage(nil), acceptedLater...), pr5ProjectionMail(
		"future-unaccepted-b", "agent-b", "human", nil,
		"future unaccepted Agent B mail", "2026-07-15T00:05:00Z",
	))
	for i := 0; i < 5; i++ {
		view := ansi.Strip(app.agentRail.View(24, 10))
		pr5RequireRenderedUnreadCount(t, pr5RailRenderedLine(t, view, "Agent A"), 1)
		pr5RequireRenderedUnreadCount(t, pr5RailRenderedLine(t, view, "Agent B"), 2)
		pr5RequireNoRenderedUnreadCount(t, pr5RailRenderedLine(t, view, "Agent B"), 3)
	}

	if got := scanner.scans.Load(); got != beforeMailScans {
		t.Fatalf("repeated rail View mailbox scans = %d, want unchanged %d", got, beforeMailScans)
	}
	if got := inventoryScript.calls; got != beforeInventoryScans {
		t.Fatalf("repeated rail View inventory scans = %d, want unchanged %d", got, beforeInventoryScans)
	}
	if app.mailStore.snapshot != beforeSnapshot || app.mailStore.version != beforeVersion {
		t.Fatalf(
			"repeated rail View changed accepted snapshot/version: snapshotChanged=%v version=%d want=%d",
			app.mailStore.snapshot != beforeSnapshot, app.mailStore.version, beforeVersion,
		)
	}
	afterState, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read unread state after rendering: %v", err)
	}
	if !bytes.Equal(afterState, beforeState) {
		t.Fatal("repeated rail View mutated durable unread state")
	}
}

func pr5RailRenderedLine(t *testing.T, view, label string) string {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, label) {
			return line
		}
	}
	t.Fatalf("rendered rail has no %q row:\n%s", label, view)
	return ""
}

func pr5RequireRenderedUnreadCount(t *testing.T, line string, want int) {
	t.Helper()
	pattern := regexp.MustCompile(fmt.Sprintf(`(^|[^0-9])%d([^0-9]|$)`, want))
	if !pattern.MatchString(line) {
		t.Fatalf("rendered rail row %q has no unread count %d", strings.TrimSpace(line), want)
	}
}

func pr5RequireNoRenderedUnreadCount(t *testing.T, line string, unwanted int) {
	t.Helper()
	pattern := regexp.MustCompile(fmt.Sprintf(`(^|[^0-9])%d([^0-9]|$)`, unwanted))
	if pattern.MatchString(line) {
		t.Fatalf("rendered rail row %q unexpectedly has unread count %d", strings.TrimSpace(line), unwanted)
	}
}
