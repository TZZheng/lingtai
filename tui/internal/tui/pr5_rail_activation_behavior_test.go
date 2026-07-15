package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/anthropics/lingtai-tui/internal/fs"
	"github.com/anthropics/lingtai-tui/internal/inventory"
)

type pr5RailActivationWorker struct {
	requests []threadLoadRequest
}

func (w *pr5RailActivationWorker) Load(request threadLoadRequest) (*fs.SessionCache, error) {
	w.requests = append(w.requests, request)
	return directThreadLoadWorker{}.Load(request)
}

func TestPR5Stage3RailEnterRevalidatesBeforeColdActivationAndPublishesOneDirectThread(t *testing.T) {
	app, scanner, _ := installationNewApp(t, 0)
	targetDir := filepath.Join(app.projectDir, "agent-a")
	installationWriteAgent(t, targetDir, "agent-a", "Agent A", "Agent A")
	installationWriteEvents(t, targetDir, 1, "event-a")
	scanner.messages = []fs.MailMessage{
		pr5ProjectionMail("a-in", "human", "agent-a", nil, "mail-a-in", "2026-07-15T01:00:00Z"),
	}

	app, _ = installationAcceptInitial(t, app)
	rootSnapshot := app.mailStore.snapshot
	if rootSnapshot == nil || app.mail.acceptedSnapshot != rootSnapshot {
		t.Fatalf("initial root snapshot store=%p mail=%p, want one accepted root snapshot", rootSnapshot, app.mail.acceptedSnapshot)
	}

	owner := app.asyncCurrent().binding.owner
	app.agentRail.installInventory(owner, inventory.Snapshot{
		FilterDir: filepath.Dir(app.projectDir),
		Records: []inventory.Record{{
			PID:                     4101,
			Agent:                   "agent-a",
			Project:                 filepath.Dir(app.projectDir),
			AgentDir:                targetDir,
			Address:                 "agent-a",
			AgentName:               "Agent A",
			Nickname:                "Agent A",
			ManifestAddressVerified: true,
			Role:                    inventory.RoleAgent,
			Enterable:               false,
		}},
	})
	app = pr5UpdateRailFocusApp(t, app, tea.WindowSizeMsg{Width: 84, Height: 24})
	app = pr5UpdateRailFocusApp(t, app, tea.KeyPressMsg{Code: tea.KeyTab})
	app = pr5UpdateRailFocusApp(t, app, tea.KeyPressMsg{Code: tea.KeyDown})
	selected, ok := app.agentRail.selectedRow()
	if !ok || selected.originalMain || selected.target.policy != asyncTargetHomeAgentRail {
		t.Fatalf("selected rail row = %#v ok=%v, want one ordinary home-Agent target", selected, ok)
	}

	beforeBinding := app.mailStore.binding
	beforeThread := app.currentThread
	beforeMailGeneration := app.mailGeneration
	beforeStoreID := app.mailStore.id
	beforeStoreVersion := app.mailStore.version
	beforeStoreSnapshot := app.mailStore.snapshot
	beforeStoreCache := app.mailStore.cache
	beforeTickChain := app.mailStore.tickChain
	beforeTickRunning := app.mailStore.tickRunning
	beforeScans := scanner.scans.Load()
	beforeRows := append([]railRow(nil), app.agentRail.rows...)
	sharedCurrent := app.mailStore.asyncState
	if sharedCurrent == nil {
		t.Fatal("fixture root store has no shared async-current holder")
	}

	var staleRevalidations int
	var staleGateSawMutation bool
	var staleBefore asyncCurrent
	app.setAsyncTargetRevalidator(func(gotOwner asyncOwner, gotTarget asyncTarget) bool {
		staleRevalidations++
		live := sharedCurrent.load()
		if live.binding != staleBefore.binding || live.storeVersion != staleBefore.storeVersion || live.tickEpoch != staleBefore.tickEpoch {
			staleGateSawMutation = true
		}
		if gotOwner != owner || gotTarget != selected.target {
			t.Errorf("stale prospective revalidation owner=%#v target=%#v, want owner=%#v selected=%#v", gotOwner, gotTarget, owner, selected.target)
		}
		return false
	})
	staleBefore = sharedCurrent.load()

	stale, _ := installationDeliverApp(t, app, tea.KeyPressMsg{Code: tea.KeyEnter})
	if staleRevalidations != 1 {
		t.Fatalf("focused ordinary Enter stale revalidations = %d, want one prospective asyncColdThreadLoad gate", staleRevalidations)
	}
	if staleGateSawMutation {
		t.Fatal("stale prospective gate observed a target/generation/tick mutation before revalidation")
	}
	if stale.mailStore.binding != beforeBinding || stale.currentThread != beforeThread ||
		stale.mailGeneration != beforeMailGeneration || stale.mail.generation != beforeBinding.generation ||
		stale.mailStore.id != beforeStoreID || stale.mailStore.version != beforeStoreVersion ||
		stale.mailStore.snapshot != beforeStoreSnapshot || !reflect.DeepEqual(stale.mailStore.cache, beforeStoreCache) ||
		stale.mailStore.tickChain != beforeTickChain || stale.mailStore.tickRunning != beforeTickRunning ||
		!reflect.DeepEqual(stale.agentRail.rows, beforeRows) || scanner.scans.Load() != beforeScans {
		t.Fatalf("stale ordinary Enter visibly mutated App/store/thread/rail or started a root scan: binding=%#v thread=%#v generation=%d store=(%d,%d,%p) tick=(%d,%v) scans=%d",
			stale.mailStore.binding, stale.currentThread, stale.mailGeneration,
			stale.mailStore.id, stale.mailStore.version, stale.mailStore.snapshot,
			stale.mailStore.tickChain, stale.mailStore.tickRunning, scanner.scans.Load())
	}

	worker := &pr5RailActivationWorker{}
	stale.threadLoads = newThreadLoadCoordinator(worker)
	stale.mailStore.pollRate = time.Nanosecond
	var acceptedRevalidations int
	var acceptedGateSawMutation bool
	var acceptedBefore asyncCurrent
	stale.setAsyncTargetRevalidator(func(gotOwner asyncOwner, gotTarget asyncTarget) bool {
		acceptedRevalidations++
		if acceptedRevalidations == 1 {
			live := sharedCurrent.load()
			if live.binding != acceptedBefore.binding || live.storeVersion != acceptedBefore.storeVersion || live.tickEpoch != acceptedBefore.tickEpoch {
				acceptedGateSawMutation = true
			}
		}
		if gotOwner != owner || gotTarget != selected.target {
			t.Errorf("accepted revalidation owner=%#v target=%#v, want owner=%#v selected=%#v", gotOwner, gotTarget, owner, selected.target)
		}
		return true
	})
	acceptedBefore = sharedCurrent.load()

	activated, activationCmd := installationDeliverApp(t, stale, tea.KeyPressMsg{Code: tea.KeyEnter})
	if acceptedRevalidations != 1 {
		t.Fatalf("accepted ordinary Enter revalidations before physical load = %d, want exactly one prospective gate", acceptedRevalidations)
	}
	if acceptedGateSawMutation {
		t.Fatal("accepted prospective gate observed a target/generation/tick mutation before revalidation")
	}
	if activationCmd == nil {
		t.Fatal("accepted focused ordinary Enter returned nil, want one cold direct-load command")
	}
	if activated.mailStore.id != beforeStoreID || activated.mailStore.version != beforeStoreVersion ||
		activated.mailStore.snapshot != beforeStoreSnapshot || !reflect.DeepEqual(activated.mailStore.cache, beforeStoreCache) {
		t.Fatalf("ordinary activation replaced/mutated the root store: id=%d/%d version=%d/%d snapshot=%p/%p",
			activated.mailStore.id, beforeStoreID, activated.mailStore.version, beforeStoreVersion,
			activated.mailStore.snapshot, beforeStoreSnapshot)
	}
	if activated.mailStore.binding.target != selected.target || activated.mailStore.binding.generation <= beforeBinding.generation ||
		activated.mailGeneration != activated.mailStore.binding.generation || activated.mail.generation != activated.mailGeneration {
		t.Fatalf("activated binding=%#v mailGeneration=%d mail.generation=%d, want selected target and one fresh generation > %d",
			activated.mailStore.binding, activated.mailGeneration, activated.mail.generation, beforeBinding.generation)
	}
	if activated.mailStore.tickChain == beforeTickChain || !activated.mailStore.tickRunning {
		t.Fatalf("ordinary activation tick=(%d,%v), want the same store's chain rotated from %d and resumed", activated.mailStore.tickChain, activated.mailStore.tickRunning, beforeTickChain)
	}
	if scanner.scans.Load() != beforeScans {
		t.Fatalf("ordinary activation root scans = %d, want unchanged %d (no immediate global refresh)", scanner.scans.Load(), beforeScans)
	}

	completion, ok := pr5FindRailThreadLoadResult(activationCmd)
	if !ok {
		t.Fatal("ordinary activation command produced no threadLoadResultMsg")
	}
	if completion.err != nil || completion.sessionCache == nil {
		t.Fatalf("ordinary cold completion cache=%p err=%v, want nonnil/no error", completion.sessionCache, completion.err)
	}
	if len(worker.requests) != 1 {
		t.Fatalf("physical cold-load requests = %d, want exactly 1", len(worker.requests))
	}
	request := worker.requests[0]
	if request.eventWindow <= 0 || request.inquiryWindow <= 0 ||
		request.eventWindow > activated.mail.pageSize || request.inquiryWindow > activated.mail.pageSize {
		t.Fatalf("cold-load windows event=%d inquiry=%d page=%d, want positive bounded windows", request.eventWindow, request.inquiryWindow, activated.mail.pageSize)
	}
	if !reflect.DeepEqual(request.acceptedMessages, rootSnapshot.cache.Messages) {
		t.Fatalf("detached accepted mailbox rows = %#v, want exact root snapshot rows %#v", request.acceptedMessages, rootSnapshot.cache.Messages)
	}
	if len(request.acceptedMessages) > 0 && &request.acceptedMessages[0] == &rootSnapshot.cache.Messages[0] {
		t.Fatal("cold-load request retained the root snapshot message slice instead of a detached copy")
	}

	published, followup := installationDeliverApp(t, activated, completion)
	if followup != nil {
		t.Fatalf("accepted cold-load publication returned unexpected follow-up %T", runCmd(followup))
	}
	if acceptedRevalidations != 2 {
		t.Fatalf("ordinary activation total revalidations = %d, want prospective plus publication gates", acceptedRevalidations)
	}
	generation := activated.mailStore.binding.generation
	pr5RequireColdDirectProjection(t, published, rootSnapshot, targetDir, generation,
		[]string{"event-a-000", "mail-a-in"}, []string{"mail-a-in"})
	if published.currentThread.sessionCache != completion.sessionCache ||
		published.currentThread.acceptedSnapshotVersion != rootSnapshot.Version() {
		t.Fatalf("published thread cache=%p snapshot=%d, want completion cache=%p root version=%d",
			published.currentThread.sessionCache, published.currentThread.acceptedSnapshotVersion,
			completion.sessionCache, rootSnapshot.Version())
	}
	if published.mailStore.id != beforeStoreID || published.mailStore.version != beforeStoreVersion ||
		published.mailStore.snapshot != rootSnapshot || !reflect.DeepEqual(published.mailStore.cache, beforeStoreCache) ||
		scanner.scans.Load() != beforeScans {
		t.Fatalf("cold publication changed root ownership/cache/scan count: id=%d version=%d snapshot=%p scans=%d",
			published.mailStore.id, published.mailStore.version, published.mailStore.snapshot, scanner.scans.Load())
	}
	pr5RequireThreadLoadCounters(t, published.threadLoads.Counters(), ThreadLoadCounters{Started: 1, Completed: 1})

	sessionPath := filepath.Join(published.mail.humanDir, "logs", "session.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
		t.Fatal(err)
	}
	const sentinel = "aggregate-session-sentinel\n"
	if err := os.WriteFile(sessionPath, []byte(sentinel), 0o644); err != nil {
		t.Fatal(err)
	}
	published.currentThread.sessionCache.Persist()
	got, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != sentinel {
		t.Fatalf("cold direct cache persisted into aggregate session.jsonl: got %q want untouched sentinel", got)
	}
}

func pr5FindRailThreadLoadResult(cmd tea.Cmd) (threadLoadResultMsg, bool) {
	if cmd == nil {
		return threadLoadResultMsg{}, false
	}
	msg := cmd()
	if result, ok := msg.(threadLoadResultMsg); ok {
		return result, true
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, child := range batch {
			if result, ok := pr5FindRailThreadLoadResult(child); ok {
				return result, true
			}
		}
	}
	return threadLoadResultMsg{}, false
}
