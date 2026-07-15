package tui

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/anthropics/lingtai-tui/i18n"
	"github.com/anthropics/lingtai-tui/internal/fs"
)

func TestPR5Stage5StaleAcceptedRailAddressKeepsMainRescansAndSurfacesLocalizedStatus(t *testing.T) {
	installationTestStart(t)

	previousLanguage := i18n.Lang()
	if err := i18n.SetLang("zh"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := i18n.SetLang(previousLanguage); err != nil {
			t.Errorf("restore i18n language: %v", err)
		}
	})

	app, mailboxScanner, _ := installationNewApp(t, 0)
	targetDir := filepath.Join(app.projectDir, "agent-a")
	const (
		originalAddress    = "agent-a"
		replacementAddress = "agent-a-replacement"
	)
	installationWriteAgent(t, targetDir, originalAddress, "Agent A", "Agent A")

	initial := pr5RailLifecycleSnapshot(app, originalAddress, "Agent A", 7401)
	replacement := pr5RailLifecycleSnapshot(app, originalAddress, "Agent A", 7401)
	replacement.Records[0].Address = replacementAddress
	displayInventory := &pr5RailInventoryScanScript{steps: []pr5RailInventoryScanStep{
		{snapshot: initial},
		{snapshot: replacement},
	}}
	app.setAgentRailInventoryScanner(displayInventory.Scan)

	initialInventoryResult := pr5RunTrailingRailInventoryScan(t, app.Init(), displayInventory)
	app, _ = installationDeliverApp(t, app, initialInventoryResult)
	pr5RequireRailLifecycleRows(t, app, []string{i18n.T("rail.main"), "Agent A"})

	app, _ = installationAcceptInitial(t, app)
	if got := mailboxScanner.scans.Load(); got != 1 {
		t.Fatalf("initial root mailbox scans = %d, want 1", got)
	}

	app = pr5UpdateRailFocusApp(t, app, tea.WindowSizeMsg{Width: 84, Height: 24})
	app = pr5UpdateRailFocusApp(t, app, tea.KeyPressMsg{Code: tea.KeyTab})
	app = pr5UpdateRailFocusApp(t, app, tea.KeyPressMsg{Code: tea.KeyDown})
	app.mail.input.SetValue("keep Main draft")
	app.mail.statusFlash = ""
	app.mail.statusExpiry = time.Time{}

	selected, ok := app.agentRail.selectedRow()
	if !ok || selected.originalMain || selected.target.policy != asyncTargetHomeAgentRail ||
		selected.target.addressFingerprint != fs.AddressFingerprint(originalAddress) {
		t.Fatalf("selected row = %#v ok=%v, want accepted ordinary Agent A identity", selected, ok)
	}
	if app.mailFocus != mailFocusRail || app.mail.input.Focused() {
		t.Fatalf("fixture focus = (%v,input=%v), want rail only", app.mailFocus, app.mail.input.Focused())
	}

	beforeOwnerState := pr5StaleActivationOwnerState(app, mailboxScanner)
	beforeRows := append([]railRow(nil), app.agentRail.rows...)
	beforeCursor := app.agentRail.cursor
	beforeDisplayScans := displayInventory.calls

	// Keep the accepted row visible while its real manifest identity changes. The
	// accepted-record gate still approves the last root inventory, but the fresh
	// manifest reread at activation must reject this address incarnation.
	installationWriteAgent(t, targetDir, replacementAddress, "Agent A", "Agent A")

	rejected, rescanCmd := installationDeliverApp(t, app, tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := pr5StaleActivationOwnerState(rejected, mailboxScanner); !reflect.DeepEqual(got, beforeOwnerState) {
		t.Fatalf("stale Enter mutated active Main owner state\n got: %#v\nwant: %#v", got, beforeOwnerState)
	}
	if !reflect.DeepEqual(rejected.agentRail.rows, beforeRows) || rejected.agentRail.cursor != beforeCursor {
		t.Fatalf("stale Enter synchronously mutated rows/cursor: rows=%#v cursor=%d, want rows=%#v cursor=%d",
			rejected.agentRail.rows, rejected.agentRail.cursor, beforeRows, beforeCursor)
	}
	if displayInventory.calls != beforeDisplayScans {
		t.Fatalf("stale Enter executed display inventory synchronously: calls=%d want %d", displayInventory.calls, beforeDisplayScans)
	}
	if rejected.mail.statusFlash != "" || !rejected.mail.statusExpiry.IsZero() {
		t.Fatalf("stale Enter claimed a completed rescan before command execution: status=%q expiry=%v",
			rejected.mail.statusFlash, rejected.mail.statusExpiry)
	}
	if rescanCmd == nil {
		t.Fatal("stale accepted rail address rescan command = nil; want one display-inventory rescan while Main remains active")
	}

	rescanRaw := rescanCmd()
	rescanResult, ok := rescanRaw.(agentRailInventoryResultMsg)
	if !ok {
		t.Fatalf("stale accepted rail address command returned %T, want one agentRailInventoryResultMsg", rescanRaw)
	}
	if displayInventory.calls != beforeDisplayScans+1 {
		t.Fatalf("stale accepted rail address display scans = %d, want %d", displayInventory.calls, beforeDisplayScans+1)
	}
	if got := mailboxScanner.scans.Load(); got != 1 {
		t.Fatalf("stale accepted rail address mailbox scans = %d, want unchanged 1", got)
	}

	refreshed, followup := installationDeliverApp(t, rejected, rescanResult)
	if followup != nil {
		t.Fatalf("accepted stale-address display result returned follow-up %T, want nil", runCmd(followup))
	}
	if got := pr5StaleActivationOwnerState(refreshed, mailboxScanner); !reflect.DeepEqual(got, beforeOwnerState) {
		t.Fatalf("accepted stale-address rescan mutated active Main owner state\n got: %#v\nwant: %#v", got, beforeOwnerState)
	}
	pr5RequireRailLifecycleRows(t, refreshed, []string{i18n.T("rail.main"), "Agent A"})
	updatedRow, ok := refreshed.agentRail.selectedRow()
	if !ok || updatedRow.originalMain || updatedRow.target.directory != selected.target.directory ||
		updatedRow.target.addressFingerprint != fs.AddressFingerprint(replacementAddress) ||
		updatedRow.target.addressFingerprint == selected.target.addressFingerprint {
		t.Fatalf("accepted replacement row = %#v ok=%v, want same directory with new address identity", updatedRow, ok)
	}
	if displayInventory.calls != beforeDisplayScans+1 || mailboxScanner.scans.Load() != 1 {
		t.Fatalf("accepted stale-address scan counts display=%d mailbox=%d, want %d/1",
			displayInventory.calls, mailboxScanner.scans.Load(), beforeDisplayScans+1)
	}
	wantStatus := i18n.TIn("zh", "projects.target_changed")
	if refreshed.mail.statusFlash != wantStatus || !refreshed.mail.statusExpiry.After(time.Now()) {
		t.Fatalf("accepted stale-address status=%q expiry=%v, want localized %q with future expiry",
			refreshed.mail.statusFlash, refreshed.mail.statusExpiry, wantStatus)
	}
}

func pr5StaleActivationOwnerState(app App, mailboxScanner *installationScriptedScanner) []any {
	return []any{
		app.currentView,
		app.mailStore.binding,
		app.currentThread,
		app.mailGeneration,
		app.mail.generation,
		app.mailStore.id,
		app.mailStore.version,
		app.mailStore.snapshot,
		app.mailStore.cache,
		app.mailStore.tickChain,
		app.mailStore.tickRunning,
		app.mail.orchestrator,
		app.mail.orchAddr,
		app.mail.orchName,
		app.mail.sessionCache,
		app.mail.acceptedSnapshot,
		app.mail.input.Value(),
		app.mailFocus,
		app.mail.input.Focused(),
		app.threadLoads.Counters(),
		mailboxScanner.scans.Load(),
	}
}
