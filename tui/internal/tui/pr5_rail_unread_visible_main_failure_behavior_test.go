package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/anthropics/lingtai-tui/internal/fs"
)

func TestPR5Stage4VisibleMainMarkSeenFailurePreservesUnreadAndSurfacesStatus(t *testing.T) {
	installationTestStart(t)

	app, scanner, _ := installationNewApp(t, 0)
	projectRoot := filepath.Dir(app.projectDir)
	targetADir := filepath.Join(app.projectDir, "agent-a")
	installationWriteAgent(t, targetADir, "agent-a", "Agent A", "Agent A")

	acceptedInventory := pr5RailLifecycleSnapshot(app, "agent-a", "Agent A", 6502)
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

	historyMain := pr5ProjectionMail(
		"historical-main", app.mail.orchAddr, "human", nil,
		"historical Main mail", "2026-07-15T00:00:00Z",
	)
	historyA := pr5ProjectionMail(
		"historical-a", "agent-a", "human", nil,
		"historical Agent A mail", "2026-07-15T00:00:00Z",
	)
	scanner.messages = []fs.MailMessage{historyMain, historyA}
	app, _ = installationAcceptInitial(t, app)
	app, _ = app.updateMailChildWindowSize(app.layoutBudget().ChildWindowSize())

	targets, ready := app.agentRail.acceptedDirectTargets(app.mailStore.binding.owner)
	if !ready || len(targets) != 2 || app.railUnreadStore == nil || app.mailStore.snapshot == nil {
		t.Fatalf(
			"visible Main failure baseline: ready=%v targets=%d store=%v snapshot=%v",
			ready, len(targets), app.railUnreadStore != nil, app.mailStore.snapshot != nil,
		)
	}
	if app.currentView != appViewMail || !app.mail.ready || app.mail.initialLoading ||
		app.mailStore.binding.target.policy != asyncTargetHomeMain {
		t.Fatalf(
			"visible Main failure readiness: view=%v ready=%v loading=%v policy=%v",
			app.currentView, app.mail.ready, app.mail.initialLoading, app.mailStore.binding.target.policy,
		)
	}
	if got := app.railUnreadStore.UnreadCount(targets[0], app.mailStore.snapshot.cache.Messages, app.mail.humanAddr); got != 0 {
		t.Fatalf("historical Main unread after baseline = %d, want 0", got)
	}

	statePath := fs.RailUnreadStatePath(projectRoot)
	baselineState, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read baseline unread state: %v", err)
	}
	pr5BlockRailUnreadStateParent(t, projectRoot)
	app.mail.statusFlash = ""
	app.mail.statusExpiry = time.Time{}

	laterMain := pr5ProjectionMail(
		"later-main", app.mail.orchAddr, "human", nil,
		"later Main mail", "2026-07-15T00:01:00Z",
	)
	acceptedLater := []fs.MailMessage{historyMain, historyA, laterMain}
	scanner.messages = acceptedLater
	beforeMailScans := scanner.scans.Load()
	beforeInventoryScans := inventoryScript.calls
	steady := installationRefreshResult(t, &app, false)
	app, _ = installationDeliverApp(t, app, steady)

	if got := scanner.scans.Load(); got != beforeMailScans+1 {
		t.Fatalf("failed visible Main advancement mailbox scans = %d, want %d", got, beforeMailScans+1)
	}
	if got := inventoryScript.calls; got != beforeInventoryScans {
		t.Fatalf("failed visible Main advancement inventory scans = %d, want unchanged %d", got, beforeInventoryScans)
	}
	if app.mail.acceptedSnapshot != app.mailStore.snapshot ||
		app.currentThread.acceptedSnapshotVersion != app.mailStore.snapshot.Version() ||
		app.currentView != appViewMail || !app.mail.ready || app.mail.initialLoading {
		t.Fatalf(
			"failed visible Main accepted coordinate: sameSnapshot=%v threadVersion=%d storeVersion=%d view=%v ready=%v loading=%v",
			app.mail.acceptedSnapshot == app.mailStore.snapshot,
			app.currentThread.acceptedSnapshotVersion,
			app.mailStore.snapshot.Version(),
			app.currentView, app.mail.ready, app.mail.initialLoading,
		)
	}

	acceptedMessages := app.mailStore.snapshot.cache.Messages
	if got := app.railUnreadStore.UnreadCount(targets[0], acceptedMessages, app.mail.humanAddr); got != 1 {
		t.Fatalf("live Main unread after failed MarkSeen = %d, want preserved 1", got)
	}
	if got := app.railUnreadStore.UnreadCount(targets[1], acceptedMessages, app.mail.humanAddr); got != 0 {
		t.Fatalf("inactive Agent A unread after failed Main MarkSeen = %d, want 0", got)
	}
	if app.agentRail.rows[0].unread != 1 || app.agentRail.rows[1].unread != 0 {
		t.Fatalf("cached Main/Agent A unread after failed MarkSeen = %d/%d, want 1/0", app.agentRail.rows[0].unread, app.agentRail.rows[1].unread)
	}
	railView := ansi.Strip(app.agentRail.View(24, 8, app.mail.orchDisplayName()))
	pr5RequireRenderedUnreadCount(t, pr5RailRenderedLine(t, railView, app.mail.orchDisplayName()), 1)
	pr5RequireNoRenderedUnreadCount(t, pr5RailRenderedLine(t, railView, "Agent A"), 1)

	// Restore the exact pre-mark durable bytes after removing the deterministic
	// blocker, then prove a restart still sees the unadvanced Main boundary.
	stateDir := filepath.Dir(statePath)
	if err := os.Remove(stateDir); err != nil {
		t.Fatalf("remove unread state blocker: %v", err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("restore unread state directory: %v", err)
	}
	if err := os.WriteFile(statePath, baselineState, 0o644); err != nil {
		t.Fatalf("restore pre-mark unread state: %v", err)
	}
	reopened, err := fs.OpenRailUnreadStore(projectRoot, targets, acceptedMessages, app.mail.humanAddr)
	if err != nil {
		t.Fatalf("reopen pre-mark unread state: %v", err)
	}
	if got := reopened.UnreadCount(targets[0], acceptedMessages, app.mail.humanAddr); got != 1 {
		t.Fatalf("restart-visible Main unread after failed MarkSeen = %d, want 1", got)
	}
	if got := reopened.UnreadCount(targets[1], acceptedMessages, app.mail.humanAddr); got != 0 {
		t.Fatalf("restart-visible Agent A unread after failed Main MarkSeen = %d, want 0", got)
	}

	pr5RequireUnreadPersistenceStatus(t, app)
}
