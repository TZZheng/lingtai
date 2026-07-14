package tui

import "github.com/anthropics/lingtai-tui/internal/fs"

// ThreadState is the one active, cold thread coordinate owned by App in PR5.
// It references the currently installed target-local session projection; it does
// not own a project store, mailbox cache, scanner, tick, or inactive-state map.
// PR7 owns any future bounded retention of inactive warm states.
type ThreadState struct {
	target                  asyncTarget
	generation              uint64
	acceptedSnapshotVersion uint64
	sessionCache            *fs.SessionCache
}

func newColdThreadState(target asyncTarget, generation, acceptedSnapshotVersion uint64, sessionCache *fs.SessionCache) ThreadState {
	return ThreadState{
		target:                  target,
		generation:              generation,
		acceptedSnapshotVersion: acceptedSnapshotVersion,
		sessionCache:            sessionCache,
	}
}

// ThreadLoadCounters classifies physical and logical cold-load work without
// calling completed filesystem work cancellation. TrueCancelled remains zero in
// this slice because no cancellation reaches the filesystem loops.
type ThreadLoadCounters struct {
	Started       uint64
	Coalesced     uint64
	Completed     uint64
	TrueCancelled uint64
	StaleDropped  uint64
}

// ThreadLoadCoordinator is the App-owned resource-accounting surface for the
// forthcoming behavioral cold loader. Stage 1 intentionally installs ownership
// and honest counters only; coalescing/loading behavior follows its own RED.
type ThreadLoadCoordinator struct {
	counters ThreadLoadCounters
}

func (c ThreadLoadCoordinator) Counters() ThreadLoadCounters {
	return c.counters
}
