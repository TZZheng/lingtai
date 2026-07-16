package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRailUnreadMarkSeen_DurableWriteFailureDoesNotAdvanceMemory(t *testing.T) {
	projectDir := t.TempDir()
	human := "project/human"
	target := DirectTarget{
		Directory: filepath.Join(projectDir, ".lingtai", "agent-b"),
		Address:   "project/agent-b",
	}
	old := MailMessage{
		ID:         "old",
		MailboxID:  "old",
		From:       target.Address,
		To:         human,
		ReceivedAt: "2026-07-13T10:00:00Z",
	}

	store, err := OpenRailUnreadStore(projectDir, []DirectTarget{target}, []MailMessage{old}, human)
	if err != nil {
		t.Fatal(err)
	}
	if got := store.UnreadCount(target, []MailMessage{old}, human); got != 0 {
		t.Fatalf("baseline unread = %d, want 0", got)
	}

	later := MailMessage{
		ID:         "later",
		MailboxID:  "later",
		From:       target.Address,
		To:         human,
		ReceivedAt: "2026-07-13T10:01:00Z",
	}
	snapshot := []MailMessage{old, later}
	if got := store.UnreadCount(target, snapshot, human); got != 1 {
		t.Fatalf("pre-mark unread = %d, want 1", got)
	}

	blocker := RailUnreadStatePath(projectDir) + ".tmp"
	if err := os.Mkdir(blocker, 0o755); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(blocker) }()

	if err := store.MarkSeen(target, snapshot, human); err == nil {
		t.Fatal("MarkSeen succeeded despite the atomic-write temp-path blocker")
	}
	if got := store.UnreadCount(target, snapshot, human); got != 1 {
		t.Fatalf("live unread after failed MarkSeen = %d, want 1", got)
	}

	if err := os.Remove(blocker); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenRailUnreadStore(projectDir, []DirectTarget{target}, snapshot, human)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.UnreadCount(target, snapshot, human); got != 1 {
		t.Fatalf("reopened unread after failed MarkSeen = %d, want 1", got)
	}
}

func incomingMail(id, from, human, timestamp string) MailMessage {
	return MailMessage{ID: id, MailboxID: id, From: from, To: human, ReceivedAt: timestamp}
}

func TestRailUnreadSyncTargets_DurableWriteFailureDoesNotAdvanceMemoryOrDisk(t *testing.T) {
	projectDir := t.TempDir()
	human := "project/human"
	directory := filepath.Join(projectDir, ".lingtai", "agent-b")
	targetA := DirectTarget{Directory: directory, Address: "project/agent-a"}
	old := incomingMail("old", targetA.Address, human, "2026-07-13T10:00:00Z")
	store, err := OpenRailUnreadStore(projectDir, []DirectTarget{targetA}, []MailMessage{old}, human)
	if err != nil {
		t.Fatal(err)
	}
	later := incomingMail("later", targetA.Address, human, "2026-07-13T10:01:00Z")
	aSnapshot := []MailMessage{old, later}
	if got := store.UnreadCount(targetA, aSnapshot, human); got != 1 {
		t.Fatalf("pre-sync unread = %d, want 1", got)
	}

	targetB := DirectTarget{Directory: directory, Address: "project/agent-b"}
	bSnapshot := []MailMessage{incomingMail("b-history", targetB.Address, human, "2026-07-13T10:02:00Z")}
	blocker := RailUnreadStatePath(projectDir) + ".tmp"
	if err := os.Mkdir(blocker, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := store.SyncTargets([]DirectTarget{targetB}, bSnapshot, human); err == nil {
		t.Fatal("SyncTargets succeeded despite the atomic-write temp-path blocker")
	}
	if got := store.UnreadCount(targetA, aSnapshot, human); got != 1 {
		t.Fatalf("live unread after failed SyncTargets = %d, want 1", got)
	}
	if got := store.UnreadCount(targetB, bSnapshot, human); got != 0 {
		t.Fatalf("failed SyncTargets recognized replacement identity with unread = %d, want 0", got)
	}
	if err := os.Remove(blocker); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenRailUnreadStore(projectDir, []DirectTarget{targetA}, aSnapshot, human)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.UnreadCount(targetA, aSnapshot, human); got != 1 {
		t.Fatalf("reopened unread after failed SyncTargets = %d, want 1", got)
	}
}

func TestRailUnreadSameTimestampMaxIDSetAcrossRestart(t *testing.T) {
	projectDir := t.TempDir()
	human := "project/human"
	target := DirectTarget{Directory: filepath.Join(projectDir, ".lingtai", "agent-b"), Address: "project/agent-b"}
	old := incomingMail("old", target.Address, human, "2026-07-13T09:00:00Z")
	maxOne := incomingMail("max-1", target.Address, human, "2026-07-13T10:00:00Z")
	store, err := OpenRailUnreadStore(projectDir, []DirectTarget{target}, []MailMessage{old, maxOne}, human)
	if err != nil {
		t.Fatal(err)
	}

	maxTwo := incomingMail("max-2", target.Address, human, maxOne.ReceivedAt)
	accepted := []MailMessage{old, maxOne, maxTwo}
	if got := store.UnreadCount(target, accepted, human); got != 1 {
		t.Fatalf("same-timestamp late ID unread = %d, want 1", got)
	}
	if err := store.MarkSeen(target, accepted, human); err != nil {
		t.Fatal(err)
	}
	restarted, err := OpenRailUnreadStore(projectDir, []DirectTarget{target}, accepted, human)
	if err != nil {
		t.Fatal(err)
	}
	if got := restarted.UnreadCount(target, accepted, human); got != 0 {
		t.Fatalf("restart unread after max-ID set = %d, want 0", got)
	}

	maxThree := incomingMail("max-3", target.Address, human, maxOne.ReceivedAt)
	if got := restarted.UnreadCount(target, append(accepted, maxThree), human); got != 1 {
		t.Fatalf("post-restart same-timestamp late ID unread = %d, want 1", got)
	}
}

func TestRailUnreadMissingCorruptAndUnsupportedStateBaselineCurrentSnapshot(t *testing.T) {
	projectDir := t.TempDir()
	human := "project/human"
	target := DirectTarget{Directory: filepath.Join(projectDir, ".lingtai", "agent-b"), Address: "project/agent-b"}
	history := []MailMessage{incomingMail("old", target.Address, human, "2026-07-13T09:00:00Z")}

	store, err := OpenRailUnreadStore(projectDir, []DirectTarget{target}, history, human)
	if err != nil {
		t.Fatal(err)
	}
	if got := store.UnreadCount(target, history, human); got != 0 {
		t.Fatalf("missing-state baseline unread = %d, want 0", got)
	}

	path := RailUnreadStatePath(projectDir)
	if err := os.WriteFile(path, []byte("not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err = OpenRailUnreadStore(projectDir, []DirectTarget{target}, history, human)
	if err != nil {
		t.Fatal(err)
	}
	if got := store.UnreadCount(target, history, human); got != 0 {
		t.Fatalf("corrupt-state baseline unread = %d, want 0", got)
	}

	if err := os.WriteFile(path, []byte(`{"version":999,"targets":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err = OpenRailUnreadStore(projectDir, []DirectTarget{target}, history, human)
	if err != nil {
		t.Fatal(err)
	}
	if got := store.UnreadCount(target, history, human); got != 0 {
		t.Fatalf("unsupported-version baseline unread = %d, want 0", got)
	}
}

func TestRailUnreadReadErrorIsNotTreatedAsMissingOrCorrupt(t *testing.T) {
	projectDir := t.TempDir()
	path := RailUnreadStatePath(projectDir)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := OpenRailUnreadStore(projectDir, nil, nil, "project/human"); err == nil {
		t.Fatal("OpenRailUnreadStore succeeded when the state path was unreadable")
	}
}

func TestRailUnreadIncomingAndOutgoingFiltering(t *testing.T) {
	projectDir := t.TempDir()
	human := "project/human"
	target := DirectTarget{Directory: filepath.Join(projectDir, ".lingtai", "agent-b"), Address: "project/agent-b"}
	store, err := OpenRailUnreadStore(projectDir, []DirectTarget{target}, nil, human)
	if err != nil {
		t.Fatal(err)
	}

	outgoing := MailMessage{ID: "out", MailboxID: "out", From: human, To: target.Address, ReceivedAt: "2026-07-13T10:02:00Z"}
	if got := store.UnreadCount(target, []MailMessage{outgoing}, human); got != 0 {
		t.Fatalf("outgoing unread = %d, want 0", got)
	}
	if err := store.MarkSeen(target, []MailMessage{outgoing}, human); err != nil {
		t.Fatal(err)
	}

	incoming := incomingMail("in", target.Address, human, "2026-07-13T10:01:00Z")
	notDirect := MailMessage{ID: "not-direct", MailboxID: "not-direct", From: target.Address, To: "project/other", ReceivedAt: "2026-07-13T10:03:00Z"}
	if got := store.UnreadCount(target, []MailMessage{outgoing, incoming, notDirect}, human); got != 1 {
		t.Fatalf("incoming filtering unread = %d, want 1", got)
	}
}

func TestRailUnreadAddressChangeAndDirectoryReuseRebaseline(t *testing.T) {
	projectDir := t.TempDir()
	human := "project/human"
	directory := filepath.Join(projectDir, ".lingtai", "agent-b")
	target := DirectTarget{Directory: directory, Address: "project/agent-b"}
	store, err := OpenRailUnreadStore(projectDir, []DirectTarget{target}, nil, human)
	if err != nil {
		t.Fatal(err)
	}

	changed := DirectTarget{Directory: directory, Address: "project/agent-b-v2"}
	changedHistory := []MailMessage{incomingMail("new-address-history", changed.Address, human, "2026-07-13T10:02:00Z")}
	if err := store.SyncTargets([]DirectTarget{changed}, changedHistory, human); err != nil {
		t.Fatal(err)
	}
	if got := store.UnreadCount(changed, changedHistory, human); got != 0 {
		t.Fatalf("address-change unread = %d, want re-baselined 0", got)
	}

	if err := store.SyncTargets(nil, changedHistory, human); err != nil {
		t.Fatal(err)
	}
	reusedHistory := append(changedHistory, incomingMail("reused", changed.Address, human, "2026-07-13T10:03:00Z"))
	if err := store.SyncTargets([]DirectTarget{changed}, reusedHistory, human); err != nil {
		t.Fatal(err)
	}
	if got := store.UnreadCount(changed, reusedHistory, human); got != 0 {
		t.Fatalf("directory-reuse unread = %d, want re-baselined 0", got)
	}
}

func TestRailUnreadNicknameRenameRetainsBoundary(t *testing.T) {
	projectDir := t.TempDir()
	human := "project/human"
	before := AgentNode{
		WorkingDir: filepath.Join(projectDir, ".lingtai", "agent-b"),
		Address:    "project/agent-b",
		Nickname:   "Before",
	}
	directTarget := func(node AgentNode) DirectTarget {
		return DirectTarget{Directory: node.WorkingDir, Address: node.Address}
	}
	target := directTarget(before)
	store, err := OpenRailUnreadStore(projectDir, []DirectTarget{target}, nil, human)
	if err != nil {
		t.Fatal(err)
	}
	incoming := incomingMail("in", target.Address, human, "2026-07-13T10:00:00Z")
	snapshot := []MailMessage{incoming}
	if got := store.UnreadCount(target, snapshot, human); got != 1 {
		t.Fatalf("pre-rename unread = %d, want 1", got)
	}

	after := before
	after.Nickname = "After"
	renamedTarget := directTarget(after)
	if err := store.SyncTargets([]DirectTarget{renamedTarget}, snapshot, human); err != nil {
		t.Fatal(err)
	}
	if got := store.UnreadCount(renamedTarget, snapshot, human); got != 1 {
		t.Fatalf("nickname-only rename unread = %d, want retained 1", got)
	}
}

func TestRailUnreadMarkSeenRequiresSyncedIdentity(t *testing.T) {
	projectDir := t.TempDir()
	human := "project/human"
	target := DirectTarget{Directory: filepath.Join(projectDir, ".lingtai", "agent-b"), Address: "project/agent-b"}
	store, err := OpenRailUnreadStore(projectDir, []DirectTarget{target}, nil, human)
	if err != nil {
		t.Fatal(err)
	}

	changed := DirectTarget{Directory: target.Directory, Address: "project/agent-b-v2"}
	if err := store.MarkSeen(changed, nil, human); err == nil {
		t.Fatal("MarkSeen accepted an address identity that had not been synchronized")
	}
	if err := store.SyncTargets([]DirectTarget{changed}, nil, human); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkSeen(changed, nil, human); err != nil {
		t.Fatalf("MarkSeen rejected synchronized identity: %v", err)
	}

	unsynced := DirectTarget{Directory: filepath.Join(projectDir, ".lingtai", "agent-c"), Address: "project/agent-c"}
	if err := store.MarkSeen(unsynced, nil, human); err == nil {
		t.Fatal("MarkSeen accepted a target that had not been synchronized")
	}
}
