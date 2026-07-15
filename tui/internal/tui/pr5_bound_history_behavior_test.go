package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/anthropics/lingtai-tui/internal/fs"
)

func TestPR5Stage2OrdinaryHistoryStaysBoundToCurrentDirectTarget(t *testing.T) {
	app, scanner, _ := installationNewApp(t, 0)
	targetA := filepath.Join(app.projectDir, "agent-a")
	targetB := filepath.Join(app.projectDir, "agent-b")
	installationWriteAgent(t, targetA, "agent-a", "Agent A", "Agent A")
	installationWriteAgent(t, targetB, "agent-b", "Agent B", "Agent B")
	installationWriteEvents(t, targetA, 150, "event-a")
	installationWriteEvents(t, targetB, 150, "event-b")

	scanner.messages = []fs.MailMessage{
		pr5ProjectionMail("a-in", "human", "agent-a", nil, "mail-a-in", "2026-07-14T01:00:00Z"),
		pr5ProjectionMail("a-out", "agent-a", "human", nil, "mail-a-out", "2026-07-14T01:00:01Z"),
		pr5ProjectionMail("b-in", "human", "agent-b", nil, "mail-b-in", "2026-07-14T01:00:02Z"),
		pr5ProjectionMail("b-out", "agent-b", "human", nil, "mail-b-out", "2026-07-14T01:00:03Z"),
		pr5ProjectionMail("b-cc-a", "human", "agent-b", []string{"agent-a"}, "mail-b-cc-a", "2026-07-14T01:00:04Z"),
	}

	initial := installationRefreshResult(t, &app, true)
	app, _ = installationDeliverApp(t, app, initial)
	rootSnapshot := app.mail.acceptedSnapshot
	if rootSnapshot == nil {
		t.Fatal("initial Main refresh did not install the accepted root snapshot")
	}
	beforeStoreID := app.mailStore.id
	beforeStoreVersion := app.mailStore.version
	beforeStoreSnapshot := app.mailStore.snapshot
	beforeTick := app.mailStore.tickChain
	beforeScans := scanner.scans.Load()
	app.mailStore.setAsyncTargetRevalidator(func(asyncOwner, asyncTarget) bool { return true })
	app.threadLoads = newThreadLoadCoordinator(directThreadLoadWorker{})

	sentinelPath := filepath.Join(app.mail.humanDir, "logs", "session.jsonl")
	if err := os.MkdirAll(filepath.Dir(sentinelPath), 0o755); err != nil {
		t.Fatal(err)
	}
	const sentinel = "main-aggregate-history-sentinel\n"
	if err := os.WriteFile(sentinelPath, []byte(sentinel), 0o644); err != nil {
		t.Fatal(err)
	}

	app = pr5ProjectColdDirectTarget(t, app, rootSnapshot, targetA, "Agent A", 4101, 1, "A1")
	if app.mail.sessionCache.Complete() {
		t.Fatal("A1 one-event cold window over 150 events must remain partial")
	}
	var requestCmd tea.Cmd
	app.mail, requestCmd = app.mail.requestOlderPage()
	if requestCmd == nil || !app.mail.olderLoadInFlight {
		t.Fatal("A1 partial direct history did not start one older-page command")
	}
	staleRaw := requestCmd()
	stalePage, ok := staleRaw.(mailOlderPageMsg)
	if !ok {
		t.Fatalf("A1 older-page command returned %T, want mailOlderPageMsg", staleRaw)
	}
	if stalePage.envelope.target.directory != targetA || stalePage.envelope.generation.thread != 1 || stalePage.envelope.source.cache != app.currentThread.sessionCache {
		t.Fatalf("A1 history envelope = %#v, want exact A1 target/generation/source", stalePage.envelope)
	}

	app = pr5ProjectColdDirectTarget(t, app, rootSnapshot, targetB, "Agent B", 4201, 2, "B1")
	app = pr5ProjectColdDirectTarget(t, app, rootSnapshot, targetA, "Agent A", 4101, 3, "A2")
	a2ColdCache := app.currentThread.sessionCache
	a2ColdBodies := pr5SortedSessionBodies(a2ColdCache.Entries())

	var followup tea.Cmd
	app, followup = installationDeliverApp(t, app, stalePage)
	if followup != nil {
		t.Fatalf("stale A1 history completion scheduled unexpected follow-up %T", followup)
	}
	if app.currentThread.sessionCache != a2ColdCache || app.mail.sessionCache != a2ColdCache ||
		!reflect.DeepEqual(pr5SortedSessionBodies(app.mail.sessionCache.Entries()), a2ColdBodies) {
		t.Fatal("stale A1 history completion mutated fresh A2 cold state")
	}

	app.mail, requestCmd = app.mail.requestOlderPage()
	if requestCmd == nil || !app.mail.olderLoadInFlight {
		t.Fatal("fresh A2 partial direct history did not start one older-page command")
	}
	freshRaw := requestCmd()
	freshPage, ok := freshRaw.(mailOlderPageMsg)
	if !ok {
		t.Fatalf("A2 older-page command returned %T, want mailOlderPageMsg", freshRaw)
	}
	if freshPage.envelope.target.directory != targetA || freshPage.envelope.generation.thread != 3 || freshPage.envelope.source.cache != a2ColdCache {
		t.Fatalf("A2 history envelope = %#v, want exact fresh A2 target/generation/source", freshPage.envelope)
	}
	app, _ = installationDeliverApp(t, app, freshPage)

	if app.mail.sessionCache == a2ColdCache || app.mail.sessionCache != app.currentThread.sessionCache {
		t.Fatalf("accepted A2 older page cache=%p cold=%p thread=%p, want one distinct installed current cache", app.mail.sessionCache, a2ColdCache, app.currentThread.sessionCache)
	}
	if !app.mail.sessionCache.Complete() {
		t.Fatal("A2 expanded older window over 150 target events must be complete")
	}
	if got, want := pr5HistoryMailBodies(app.mail.sessionCache.Entries()), []string{"mail-a-in", "mail-a-out"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("A2 direct history mail bodies = %v, want %v; another target or CC-only row leaked", got, want)
	}
	if got := pr5HistoryBodyPrefixCount(app.mail.sessionCache.Entries(), "event-a-"); got != 150 {
		t.Fatalf("A2 direct history event count = %d, want all 150 A events", got)
	}
	if got := pr5HistoryBodyPrefixCount(app.mail.sessionCache.Entries(), "event-b-"); got != 0 {
		t.Fatalf("A2 direct history included %d B events", got)
	}

	identity := app.mail.sessionCache.HistoryCountIdentity()
	wantSource := filepath.Clean(filepath.Join(targetA, "logs", "events.jsonl"))
	if !strings.Contains(identity, wantSource) || strings.Contains(identity, filepath.Clean(filepath.Join(targetB, "logs", "events.jsonl"))) {
		t.Fatalf("A2 history count identity = %q, want only source %q", identity, wantSource)
	}
	if !app.mail.historyCountLoading || app.mail.historyCountCache != app.mail.sessionCache || app.mail.historyCountIdentity != identity {
		t.Fatalf("A2 exact-count origin loading=%v cache=%p current=%p identity=%q/%q", app.mail.historyCountLoading, app.mail.historyCountCache, app.mail.sessionCache, app.mail.historyCountIdentity, identity)
	}
	count := app.mail.historyCountCmd(app.mail.historyCountCache)().(mailHistoryCountMsg)
	app, _ = installationDeliverApp(t, app, count)
	if !app.mail.historyCountLoaded || app.mail.historyStats.Detailed != 150 {
		t.Fatalf("A2 exact history count loaded=%v stats=%+v, want 150 target events", app.mail.historyCountLoaded, app.mail.historyStats)
	}

	app.mail.sessionCache.Persist()
	gotSentinel, err := os.ReadFile(sentinelPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotSentinel) != sentinel {
		t.Fatalf("ordinary A2 history used MainAggregateWriter: got session.jsonl %q want sentinel %q", gotSentinel, sentinel)
	}
	if app.mailStore.id != beforeStoreID || app.mailStore.version != beforeStoreVersion ||
		app.mailStore.snapshot != beforeStoreSnapshot || app.mailStore.tickChain != beforeTick || scanner.scans.Load() != beforeScans {
		t.Fatalf("ordinary A/B/A history mutated/scanned root store: id=%d/%d version=%d/%d snapshot=%p/%p tick=%d/%d scans=%d/%d",
			app.mailStore.id, beforeStoreID, app.mailStore.version, beforeStoreVersion,
			app.mailStore.snapshot, beforeStoreSnapshot, app.mailStore.tickChain, beforeTick,
			scanner.scans.Load(), beforeScans)
	}
}

func pr5HistoryMailBodies(entries []fs.SessionEntry) []string {
	bodies := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Type == "mail" {
			bodies = append(bodies, entry.Body)
		}
	}
	sort.Strings(bodies)
	return bodies
}

func pr5HistoryBodyPrefixCount(entries []fs.SessionEntry, prefix string) int {
	count := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Body, prefix) {
			count++
		}
	}
	return count
}
