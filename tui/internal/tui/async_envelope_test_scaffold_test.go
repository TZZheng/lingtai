package tui

import "github.com/anthropics/lingtai-tui/internal/fs"

// TEMPORARY TEST-ONLY PRODUCTION-SHAPED SCAFFOLD.
//
// This file exists only so the direct PR4 predicate contract can compile before
// async_envelope.go exists. Delete it when the production protocol lands. In
// particular, acceptAsync is deliberately fail-closed: even a fully current,
// well-formed capture returns false so the positive contract assertions stay RED.

type asyncKind uint8

const (
	asyncInitialRebuild asyncKind = iota + 1
	asyncSteadyRefresh
	asyncSessionPersist
	asyncOlderPage
	asyncExactHistoryCount
	asyncRefreshTick
	asyncLivenessPulse
	asyncEditorDone
)

type asyncFieldMask uint16

const (
	asyncHasOwner asyncFieldMask = 1 << iota
	asyncHasTarget
	asyncHasGeneration
	asyncHasEpoch
	asyncHasSourceCache
	asyncHasStoreVersion
)

type asyncOwner struct {
	projectID  string
	storeID    uint64
	activation uint64
}

type asyncTarget struct {
	directory          string
	addressFingerprint string
	inventoryBound     bool
}

type asyncGeneration struct {
	thread uint64
	epoch  uint64
}

type asyncSourceCache struct {
	cache    *fs.SessionCache
	identity string
}

type asyncBinding struct {
	owner      asyncOwner
	target     asyncTarget
	generation uint64
}

type asyncEnvelope struct {
	kind         asyncKind
	fields       asyncFieldMask
	owner        asyncOwner
	target       asyncTarget
	generation   asyncGeneration
	source       asyncSourceCache
	storeVersion uint64
}

type asyncCurrent struct {
	binding          asyncBinding
	sessionSource    asyncSourceCache
	outstandingCount asyncSourceCache
	storeVersion     uint64
	tickEpoch        uint64
	pulseEpoch       uint64
	revalidateTarget func(asyncOwner, asyncTarget) bool
}

func temporaryAsyncRequiredMask(kind asyncKind) (asyncFieldMask, bool) {
	base := asyncHasOwner | asyncHasTarget | asyncHasGeneration
	switch kind {
	case asyncInitialRebuild, asyncSteadyRefresh:
		return base | asyncHasStoreVersion, true
	case asyncSessionPersist, asyncOlderPage:
		return base | asyncHasSourceCache | asyncHasStoreVersion, true
	case asyncExactHistoryCount:
		return base | asyncHasSourceCache, true
	case asyncRefreshTick, asyncLivenessPulse:
		return base | asyncHasEpoch, true
	case asyncEditorDone:
		return base, true
	default:
		return 0, false
	}
}

func captureAsync(kind asyncKind, current asyncCurrent) asyncEnvelope {
	fields, known := temporaryAsyncRequiredMask(kind)
	if !known {
		return asyncEnvelope{kind: kind}
	}
	envelope := asyncEnvelope{
		kind:       kind,
		fields:     fields,
		owner:      current.binding.owner,
		target:     current.binding.target,
		generation: asyncGeneration{thread: current.binding.generation},
	}
	if fields&asyncHasStoreVersion != 0 {
		envelope.storeVersion = current.storeVersion
	}
	switch kind {
	case asyncSessionPersist, asyncOlderPage:
		envelope.source = current.sessionSource
	case asyncExactHistoryCount:
		envelope.source = current.outstandingCount
	case asyncRefreshTick:
		envelope.generation.epoch = current.tickEpoch
	case asyncLivenessPulse:
		envelope.generation.epoch = current.pulseEpoch
	}
	return envelope
}

func temporaryAsyncNeedsInventoryRevalidation(kind asyncKind) bool {
	switch kind {
	case asyncInitialRebuild,
		asyncSteadyRefresh,
		asyncSessionPersist,
		asyncOlderPage,
		asyncExactHistoryCount,
		asyncRefreshTick,
		asyncEditorDone:
		return true
	default:
		return false
	}
}

func acceptAsync(current asyncCurrent, got asyncEnvelope) bool {
	// Exercise the injectable seam so resolver-call assertions can already pin
	// the seven substantive paths. The liveness pulse intentionally performs no
	// process-inventory scan. No resolver result can make this scaffold accept.
	if current.binding.target.inventoryBound &&
		temporaryAsyncNeedsInventoryRevalidation(got.kind) &&
		current.revalidateTarget != nil {
		_ = current.revalidateTarget(current.binding.owner, current.binding.target)
	}
	return false
}
