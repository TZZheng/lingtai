package tui

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/anthropics/lingtai-tui/internal/fs"
)

func TestPR5Stage4InventoryErrorPreservesAndRemovalRejectsVisibleOrdinaryCompletion(t *testing.T) {
	installationTestStart(t)

	app, scanner, _ := installationNewApp(t, 0)
	projectRoot := filepath.Dir(app.projectDir)
	targetADir := filepath.Join(app.projectDir, "agent-a")
	targetBDir := filepath.Join(app.projectDir, "agent-b")
	installationWriteAgent(t, targetADir, "agent-a", "Agent A", "Agent A")
	installationWriteAgent(t, targetBDir, "agent-b", "Agent B", "Agent B")

	acceptedInventory := pr5RailLifecycleSnapshot(app, "agent-a", "Agent A", 7301)
	agentBInventory := pr5RailLifecycleSnapshot(app, "agent-b", "Agent B", 7302)
	acceptedInventory.Records = append(acceptedInventory.Records, agentBInventory.Records...)
	removedAInventory := agentBInventory
	readdedInventory := acceptedInventory
	inventoryScript := &pr5RailInventoryScanScript{steps: []pr5RailInventoryScanStep{
		{snapshot: acceptedInventory},
		{err: errors.New("inventory temporarily unavailable before removal")},
		{snapshot: removedAInventory},
		{err: errors.New("inventory temporarily unavailable after removal")},
		{snapshot: readdedInventory},
	}}
	setter, ok := any(&app).(pr5AgentRailInventoryScannerSetter)
	if !ok {
		t.Fatal("App has no root-owned agent-rail inventory lifecycle boundary")
	}
	setter.setAgentRailInventoryScanner(inventoryScript.Scan)
	inventoryResult := pr5RunTrailingRailInventoryScan(t, app.Init(), inventoryScript)
	app, _ = installationDeliverApp(t, app, inventoryResult)
	pr5RequireRailLifecycleRows(t, app, []string{"Main", "Agent A", "Agent B"})

	historyA := pr5ProjectionMail(
		"inventory-a-0", "agent-a", "human", nil,
		"historical Agent A mail before inventory removal", "2026-07-15T00:00:00Z",
	)
	historyB := pr5ProjectionMail(
		"inventory-b-0", "agent-b", "human", nil,
		"historical Agent B mail before inventory removal", "2026-07-15T00:00:00Z",
	)
	scanner.messages = []fs.MailMessage{historyA, historyB}
	app, _ = installationAcceptInitial(t, app)

	initialTargets, ready := app.agentRail.acceptedDirectTargets(app.mailStore.binding.owner)
	if !ready || len(initialTargets) != 3 || app.railUnreadStore == nil {
		t.Fatalf("inventory-interaction baseline targets: ready=%v count=%d unreadStore=%v, want true/3/live",
			ready, len(initialTargets), app.railUnreadStore != nil)
	}
	laterA1 := pr5ProjectionMail(
		"inventory-a-1", "agent-a", "human", nil,
		"first later Agent A mail before inventory removal", "2026-07-15T00:01:00Z",
	)
	laterB1 := pr5ProjectionMail(
		"inventory-b-1", "agent-b", "human", nil,
		"first later Agent B mail before inventory removal", "2026-07-15T00:01:00Z",
	)
	scanner.messages = []fs.MailMessage{historyA, historyB, laterA1, laterB1}
	steady := installationRefreshResult(t, &app, false)
	app, _ = installationDeliverApp(t, app, steady)
	rootNMessages := append([]fs.MailMessage(nil), app.mailStore.snapshot.cache.Messages...)
	pr5RequireSameTimestampUnread(t, app, initialTargets, rootNMessages, 1, 1)

	app.threadLoads = newThreadLoadCoordinator(&pr5RailActivationWorker{})
	app.mailStore.pollRate = time.Nanosecond
	app.agentRail.cursor = 1
	rowA := app.agentRail.rows[1]
	visibleA, activationCmd := app.activateOrdinaryRailRow(rowA)
	firstAResult, ok := pr5FindRailThreadLoadResult(activationCmd)
	if activationCmd == nil || !ok {
		t.Fatalf("inventory-interaction A activation returned cmd/result = %v/%v, want both", activationCmd != nil, ok)
	}
	app, followup := installationDeliverApp(t, visibleA, firstAResult)
	if followup != nil || app.mail.initialLoading || app.currentThread.sessionCache != firstAResult.sessionCache {
		t.Fatalf("inventory-interaction visible A@N publication: followup=%v loading=%v installed=%v",
			followup != nil, app.mail.initialLoading, app.currentThread.sessionCache == firstAResult.sessionCache)
	}
	rootN := app.mailStore.snapshot
	rootNVersion := rootN.Version()
	renderedNBodies := pr5SortedVisibleBodies(app.mail.messages)
	if !reflect.DeepEqual(renderedNBodies, []string{
		"first later Agent A mail before inventory removal",
		"historical Agent A mail before inventory removal",
	}) {
		t.Fatalf("visible A@N bodies before inventory interactions = %v, want exact A-only history", renderedNBodies)
	}
	pr5RequireSameTimestampUnread(t, app, initialTargets, rootNMessages, 0, 1)
	seenABytes, err := os.ReadFile(fs.RailUnreadStatePath(projectRoot))
	if err != nil {
		t.Fatalf("read A@N seen state before inventory interactions: %v", err)
	}

	laterA2 := pr5ProjectionMail(
		"inventory-a-2", "agent-a", "human", nil,
		"second later Agent A mail while inventory changes", "2026-07-15T00:02:00Z",
	)
	laterB2 := pr5ProjectionMail(
		"inventory-b-2", "agent-b", "human", nil,
		"second later Agent B mail while inventory changes", "2026-07-15T00:02:00Z",
	)
	worker := newPR5BlockingThreadLoadWorker(t)
	app.threadLoads = newThreadLoadCoordinator(worker)
	scanner.messages = []fs.MailMessage{historyA, historyB, laterA1, laterB1, laterA2, laterB2}
	advancedRefresh := installationRefreshResult(t, &app, false)
	staged, stagedCmd := installationDeliverApp(t, app, advancedRefresh)
	if stagedCmd == nil || staged.mailStore.snapshot == rootN || staged.mailStore.version <= rootNVersion ||
		staged.mail.acceptedSnapshot != rootN || staged.mail.asyncStoreVersion != rootNVersion ||
		staged.currentThread.acceptedSnapshotVersion != rootNVersion ||
		!reflect.DeepEqual(pr5SortedVisibleBodies(staged.mail.messages), renderedNBodies) {
		t.Fatalf("inventory-interaction staged A@N coordinate: cmd=%v rootAdvanced=%v storeVersion=%d old=%d mailRootN=%v versions=%d/%d bodies=%v",
			stagedCmd != nil, staged.mailStore.snapshot != rootN, staged.mailStore.version, rootNVersion,
			staged.mail.acceptedSnapshot == rootN, staged.mail.asyncStoreVersion,
			staged.currentThread.acceptedSnapshotVersion, pr5SortedVisibleBodies(staged.mail.messages))
	}
	rootNPlusOneMessages := append([]fs.MailMessage(nil), staged.mailStore.snapshot.cache.Messages...)
	pr5RequireSameTimestampUnread(t, staged, initialTargets, rootNPlusOneMessages, 1, 2)
	if stagedBytes, err := os.ReadFile(fs.RailUnreadStatePath(projectRoot)); err != nil {
		t.Fatalf("read staged A@N durable state: %v", err)
	} else if !bytes.Equal(stagedBytes, seenABytes) {
		t.Fatal("root N+1 advanced durable A before exact direct publication")
	}
	stagedResults := pr5StartBatchCommands(t, stagedCmd, "inventory-interaction A@N+1 projection")
	flight := pr5AwaitThreadLoadFlight(t, worker, "inventory-interaction A@N+1")
	if flight.request.envelope.target != rowA.target ||
		flight.request.envelope.storeVersion != staged.mailStore.version ||
		flight.request.envelope.generation.thread != staged.mailStore.binding.generation {
		t.Fatalf("inventory-interaction A@N+1 flight coordinates = %#v, want exact A target generation=%d version=%d",
			flight.request.envelope, staged.mailStore.binding.generation, staged.mailStore.version)
	}

	beforeErrorRows := pr5RailLifecycleRows(staged)
	beforeErrorBytes, err := os.ReadFile(fs.RailUnreadStatePath(projectRoot))
	if err != nil {
		t.Fatalf("read unread state before inventory error: %v", err)
	}
	errorResult := pr5RunTrailingRailInventoryScan(t, staged.resumeProjectMail(false), inventoryScript)
	afterError, cmd := installationDeliverApp(t, staged, errorResult)
	if cmd != nil || !reflect.DeepEqual(pr5RailLifecycleRows(afterError), beforeErrorRows) ||
		!reflect.DeepEqual(pr5SortedVisibleBodies(afterError.mail.messages), renderedNBodies) {
		t.Fatalf("inventory error changed accepted rows/content: cmd=%v rows=%v want=%v bodies=%v want=%v",
			cmd != nil, pr5RailLifecycleRows(afterError), beforeErrorRows,
			pr5SortedVisibleBodies(afterError.mail.messages), renderedNBodies)
	}
	pr5RequireSameTimestampUnread(t, afterError, initialTargets, rootNPlusOneMessages, 1, 2)
	if bytesAfterError, err := os.ReadFile(fs.RailUnreadStatePath(projectRoot)); err != nil {
		t.Fatalf("read unread state after inventory error: %v", err)
	} else if !bytes.Equal(bytesAfterError, beforeErrorBytes) {
		t.Fatal("inventory error changed durable unread state")
	}

	removalResult := pr5RunTrailingRailInventoryScan(t, afterError.resumeProjectMail(false), inventoryScript)
	removed, cmd := installationDeliverApp(t, afterError, removalResult)
	if cmd != nil {
		t.Fatal("successful inventory removal returned an unexpected command")
	}
	pr5RequireRailLifecycleRows(t, removed, []string{"Main", "Agent B"})
	removedTargets, ready := removed.agentRail.acceptedDirectTargets(removed.mailStore.binding.owner)
	if !ready || len(removedTargets) != 2 || removedTargets[1] != initialTargets[2] {
		t.Fatalf("targets after A removal: ready=%v targets=%#v, want exact Main+B", ready, removedTargets)
	}
	if removed.agentRail.cursor != 1 || removed.agentRail.rows[1].directTarget != initialTargets[2] {
		t.Fatalf("cursor after active A removal = %d target=%#v, want fallback B at row 1",
			removed.agentRail.cursor, removed.agentRail.rows[removed.agentRail.cursor].directTarget)
	}
	if removed.agentRail.rows[0].unread != 0 || removed.agentRail.rows[1].unread != 2 ||
		removed.railUnreadStore.UnreadCount(removedTargets[1], rootNPlusOneMessages, removed.mail.humanAddr) != 2 {
		t.Fatalf("Main/B unread after A removal = cached %d/%d live B=%d, want 0/2/2",
			removed.agentRail.rows[0].unread, removed.agentRail.rows[1].unread,
			removed.railUnreadStore.UnreadCount(removedTargets[1], rootNPlusOneMessages, removed.mail.humanAddr))
	}
	if removed.currentThread.target != rowA.target || removed.mail.acceptedSnapshot != rootN ||
		removed.mail.asyncStoreVersion != rootNVersion ||
		!reflect.DeepEqual(pr5SortedVisibleBodies(removed.mail.messages), renderedNBodies) {
		t.Fatalf("A removal changed retained staged A@N content/coordinate: target=%#v snapshotN=%v version=%d want=%d bodies=%v",
			removed.currentThread.target, removed.mail.acceptedSnapshot == rootN,
			removed.mail.asyncStoreVersion, rootNVersion, pr5SortedVisibleBodies(removed.mail.messages))
	}
	removedBytes, err := os.ReadFile(fs.RailUnreadStatePath(projectRoot))
	if err != nil {
		t.Fatalf("read durable unread state after A removal: %v", err)
	}
	if bytes.Equal(removedBytes, beforeErrorBytes) {
		t.Fatal("successful A removal did not transactionally remove its durable target state")
	}
	restartedRemoved, err := fs.OpenRailUnreadStore(projectRoot, removedTargets, rootNPlusOneMessages, removed.mail.humanAddr)
	if err != nil {
		t.Fatalf("restart unread state after A removal: %v", err)
	}
	if got := restartedRemoved.UnreadCount(removedTargets[1], rootNPlusOneMessages, removed.mail.humanAddr); got != 2 {
		t.Fatalf("restart Agent B unread after A removal = %d, want preserved 2", got)
	}

	completionCache, err := (directThreadLoadWorker{}).Load(flight.request)
	if err != nil {
		t.Fatalf("build removed-A controlled completion: %v", err)
	}
	flight.release <- pr5ThreadLoadReply{sessionCache: completionCache}
	completion := pr5AwaitVisibleRefreshThreadLoadResult(t, stagedResults, "removed A@N+1 completion")
	rejected, followup := installationDeliverApp(t, removed, completion)
	if followup != nil || rejected.currentThread.sessionCache == completionCache ||
		rejected.mail.sessionCache == completionCache || rejected.mail.acceptedSnapshot != rootN ||
		rejected.mail.asyncStoreVersion != rootNVersion ||
		!reflect.DeepEqual(pr5SortedVisibleBodies(rejected.mail.messages), renderedNBodies) {
		t.Fatalf("removed-A completion escaped publication gate: followup=%v threadInstalled=%v mailInstalled=%v snapshotN=%v version=%d bodies=%v",
			followup != nil, rejected.currentThread.sessionCache == completionCache,
			rejected.mail.sessionCache == completionCache, rejected.mail.acceptedSnapshot == rootN,
			rejected.mail.asyncStoreVersion, pr5SortedVisibleBodies(rejected.mail.messages))
	}
	pr5RequireThreadLoadCounters(t, rejected.threadLoads.Counters(), ThreadLoadCounters{
		Started:       1,
		Coalesced:     0,
		Completed:     1,
		TrueCancelled: 0,
		StaleDropped:  1,
	})
	if bytesAfterRejected, err := os.ReadFile(fs.RailUnreadStatePath(projectRoot)); err != nil {
		t.Fatalf("read durable unread after removed-A completion: %v", err)
	} else if !bytes.Equal(bytesAfterRejected, removedBytes) {
		t.Fatal("rejected removed-A completion changed durable unread state")
	}

	postRemovalErrorResult := pr5RunTrailingRailInventoryScan(t, rejected.resumeProjectMail(false), inventoryScript)
	postRemovalError, cmd := installationDeliverApp(t, rejected, postRemovalErrorResult)
	if cmd != nil {
		t.Fatal("post-removal inventory error returned an unexpected command")
	}
	pr5RequireRailLifecycleRows(t, postRemovalError, []string{"Main", "Agent B"})
	if bytesAfterPostRemovalError, err := os.ReadFile(fs.RailUnreadStatePath(projectRoot)); err != nil {
		t.Fatalf("read durable unread after post-removal inventory error: %v", err)
	} else if !bytes.Equal(bytesAfterPostRemovalError, removedBytes) {
		t.Fatal("post-removal inventory error changed last-good durable unread state")
	}

	readdResult := pr5RunTrailingRailInventoryScan(t, postRemovalError.resumeProjectMail(false), inventoryScript)
	readded, cmd := installationDeliverApp(t, postRemovalError, readdResult)
	if cmd != nil {
		t.Fatal("successful A re-add returned an unexpected command")
	}
	pr5RequireRailLifecycleRows(t, readded, []string{"Main", "Agent A", "Agent B"})
	readdedTargets, ready := readded.agentRail.acceptedDirectTargets(readded.mailStore.binding.owner)
	if !ready || len(readdedTargets) != 3 || readdedTargets[1] != initialTargets[1] || readdedTargets[2] != initialTargets[2] {
		t.Fatalf("targets after A re-add: ready=%v targets=%#v, want exact Main+A+B", ready, readdedTargets)
	}
	if readded.agentRail.cursor != 2 || readded.agentRail.rows[2].directTarget != initialTargets[2] {
		t.Fatalf("cursor after A re-add = %d target=%#v, want retained selected B at row 2",
			readded.agentRail.cursor, readded.agentRail.rows[readded.agentRail.cursor].directTarget)
	}
	if readded.agentRail.rows[0].unread != 0 || readded.agentRail.rows[1].unread != 0 || readded.agentRail.rows[2].unread != 2 {
		t.Fatalf("cached Main/A/B unread after re-add = %d/%d/%d, want new A baseline 0 and preserved B 2",
			readded.agentRail.rows[0].unread, readded.agentRail.rows[1].unread, readded.agentRail.rows[2].unread)
	}
	if readded.currentThread.target != rowA.target || readded.mail.acceptedSnapshot != rootN ||
		!reflect.DeepEqual(pr5SortedVisibleBodies(readded.mail.messages), renderedNBodies) {
		t.Fatalf("A re-add silently published rejected N+1 content: target=%#v snapshotN=%v bodies=%v",
			readded.currentThread.target, readded.mail.acceptedSnapshot == rootN,
			pr5SortedVisibleBodies(readded.mail.messages))
	}
	finalRestart, err := fs.OpenRailUnreadStore(projectRoot, readdedTargets, rootNPlusOneMessages, readded.mail.humanAddr)
	if err != nil {
		t.Fatalf("final restart unread state after A re-add: %v", err)
	}
	for i, want := range []int{0, 0, 2} {
		if unread := finalRestart.UnreadCount(readdedTargets[i], rootNPlusOneMessages, readded.mail.humanAddr); unread != want {
			t.Fatalf("final restarted Main/A/B unread[%d] = %d, want %d", i, unread, want)
		}
	}

	if got := scanner.scans.Load(); got != 3 {
		t.Fatalf("inventory-interaction mail scans = %d, want exactly 3", got)
	}
	if inventoryScript.calls != 5 {
		t.Fatalf("inventory-interaction inventory scans = %d, want exactly 5", inventoryScript.calls)
	}
	select {
	case extra := <-worker.started:
		t.Fatalf("inventory interaction started an extra physical worker for target=%#v", extra.request.envelope.target)
	default:
	}
}
