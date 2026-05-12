package fs

import (
	"bufio"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TapeFrame is a single timestamped snapshot of the network topology.
type TapeFrame struct {
	T   int64   `json:"t"`
	Net Network `json:"net"`
}

// eventRecord is one line from logs/events.jsonl.
type eventRecord struct {
	Type      string  `json:"type"`
	Ts        float64 `json:"ts"`
	Address   string  `json:"address"`
	AgentName string  `json:"agent_name"`
	Old       string  `json:"old"`
	New       string  `json:"new"`
}

// timestampedMail pairs a mail message with its parsed unix timestamp.
type timestampedMail struct {
	msg MailMessage
	ts  float64 // unix seconds
}

// ReconstructTape scans agent directories under baseDir, reads events.jsonl
// and mailbox contents, and reconstructs the full topology tape as a sequence
// of TapeFrame snapshots at 3-second intervals.
func ReconstructTape(baseDir string) ([]TapeFrame, error) {
	// 1. Discover all agents
	agents, err := DiscoverAgents(baseDir)
	if err != nil {
		return nil, err
	}
	if len(agents) == 0 {
		return nil, nil
	}

	// Normalize states to uppercase
	for i := range agents {
		agents[i].State = strings.ToUpper(agents[i].State)
	}

	// 2. Read all events across all agents
	var allEvents []eventRecord
	for _, a := range agents {
		events := readEventsJSONL(a.WorkingDir)
		allEvents = append(allEvents, events...)
	}

	// 3. Read all mail across all agents (inbox + archive)
	var allMail []timestampedMail
	for _, a := range agents {
		inbox, _ := readMailFolder(filepath.Join(a.WorkingDir, "mailbox", "inbox"))
		archive, _ := readMailFolder(filepath.Join(a.WorkingDir, "mailbox", "archive"))
		for _, msg := range append(inbox, archive...) {
			ts := mailTimestamp(msg)
			if ts > 0 {
				allMail = append(allMail, timestampedMail{msg: msg, ts: ts})
			}
		}
	}

	// 4. Determine time range.
	// minTs uses ALL events so an agent visible from heartbeats appears at its
	// real first-seen time. maxTs uses only mutation-causing events (state
	// changes + mail), so heartbeats during long-idle tails don't push the
	// timeline up to "now" with nothing happening.
	minTs := math.MaxFloat64
	maxTs := 0.0
	for _, e := range allEvents {
		if e.Ts < minTs {
			minTs = e.Ts
		}
		if e.Type == "agent_state" && e.Ts > maxTs {
			maxTs = e.Ts
		}
	}
	for _, m := range allMail {
		if m.ts < minTs {
			minTs = m.ts
		}
		if m.ts > maxTs {
			maxTs = m.ts
		}
	}

	// No events and no mail → no frames
	if minTs == math.MaxFloat64 {
		return nil, nil
	}
	// No mutation-causing events at all → use minTs so we still emit at least
	// one frame (initial visibility snapshot).
	if maxTs == 0.0 {
		maxTs = minTs
	}

	// Sort events by timestamp for replay
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Ts < allEvents[j].Ts
	})

	// Sort mail by timestamp
	sort.Slice(allMail, func(i, j int) bool {
		return allMail[i].ts < allMail[j].ts
	})

	// Determine first-event time per agent (for visibility: agent appears when its first event <= t)
	firstEventTs := make(map[string]float64)
	for _, e := range allEvents {
		if e.Address == "" {
			continue
		}
		resolved := ResolveAddress(e.Address, baseDir)
		if _, ok := firstEventTs[resolved]; !ok {
			firstEventTs[resolved] = e.Ts
		}
	}
	// Agents with mail but no events: use their first mail timestamp
	for _, m := range allMail {
		from := m.msg.From
		if from != "" {
			resolved := ResolveAddress(from, baseDir)
			if _, ok := firstEventTs[resolved]; !ok {
				firstEventTs[resolved] = m.ts
			}
		}
	}

	// Pre-read static edges (avatar + contact) once — they don't change per-frame
	var avatarEdges []AvatarEdge
	var contactEdges []ContactEdge
	for _, a := range agents {
		edges, _ := ReadLedger(a.WorkingDir)
		avatarEdges = append(avatarEdges, edges...)
		contactEdges = append(contactEdges, ReadContacts(a.WorkingDir)...)
	}
	if avatarEdges == nil {
		avatarEdges = []AvatarEdge{}
	}
	if contactEdges == nil {
		contactEdges = []ContactEdge{}
	}
	// Relativize static edges so they match AgentNode.Address format
	for i := range avatarEdges {
		avatarEdges[i].Parent = RelativizeAddress(avatarEdges[i].Parent, baseDir)
		avatarEdges[i].Child = RelativizeAddress(avatarEdges[i].Child, baseDir)
	}
	for i := range contactEdges {
		contactEdges[i].Owner = RelativizeAddress(contactEdges[i].Owner, baseDir)
		contactEdges[i].Target = RelativizeAddress(contactEdges[i].Target, baseDir)
	}

	// 5. Build frames using activity-driven sampling.
	//
	// Sample points (snapped to a 3s grid for cache friendliness):
	//   - The first frame (startMs aligned).
	//   - Every agent_state event timestamp.
	//   - Every mail timestamp.
	//   - Every firstEventTs (so an agent appears the moment it becomes visible,
	//     even if its first event is a heartbeat).
	//   - One "heartbeat" sample per maxGapMs during long idle stretches.
	//   - The final frame (endMs aligned, +1 interval if needed).
	//
	// This produces O(events + mail + duration/maxGapMs) frames instead of
	// O(duration/3s). For a 2-week-old project with a few hundred events that
	// drops frame count by roughly 100x.
	const intervalMs int64 = 3000
	const maxGapMs int64 = 60 * 1000 // emit at least one frame per minute
	startMs := int64(minTs * 1000)
	endMs := int64(maxTs * 1000)

	// Align start down (floor) and end up (ceil) so the final frame's
	// effective time is at or after the latest mutation event. Snapping the
	// end down can place all events into a single bucket whose tSec is
	// earlier than the actual event timestamps, leaving mail unprocessed.
	startMs = (startMs / intervalMs) * intervalMs
	endMs = ((endMs + intervalMs - 1) / intervalMs) * intervalMs
	if endMs < startMs {
		endMs = startMs
	}

	// Build the sorted set of activity-driven sample timestamps. Each
	// interesting event is sampled at the next 3s tick (ceil) so the sample
	// occurs strictly at-or-after the event, ensuring the event-cursor
	// advances when we visit that sample.
	sampleSet := make(map[int64]struct{})
	snapUp := func(tMs int64) int64 { return ((tMs + intervalMs - 1) / intervalMs) * intervalMs }
	snapDown := func(tMs int64) int64 { return (tMs / intervalMs) * intervalMs }
	sampleSet[startMs] = struct{}{}
	sampleSet[endMs] = struct{}{}
	for _, e := range allEvents {
		if e.Type == "agent_state" {
			sampleSet[snapUp(int64(e.Ts*1000))] = struct{}{}
		}
	}
	for _, m := range allMail {
		sampleSet[snapUp(int64(m.ts*1000))] = struct{}{}
	}
	for _, ft := range firstEventTs {
		sampleSet[snapUp(int64(ft*1000))] = struct{}{}
	}
	// Add heartbeat samples during long gaps so the scrubber feels alive.
	sorted := make([]int64, 0, len(sampleSet))
	for t := range sampleSet {
		if t < startMs || t > endMs {
			continue
		}
		sorted = append(sorted, t)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	if len(sorted) == 0 || sorted[0] != startMs {
		sorted = append([]int64{startMs}, sorted...)
	}
	gapFilled := make([]int64, 0, len(sorted))
	for i, t := range sorted {
		gapFilled = append(gapFilled, t)
		if i+1 >= len(sorted) {
			continue
		}
		next := sorted[i+1]
		for h := t + maxGapMs; h < next; h += maxGapMs {
			gapFilled = append(gapFilled, snapDown(h))
		}
	}
	sorted = gapFilled

	type edgeKey struct{ sender, recipient string }
	type edgeCounts struct{ direct, cc, bcc int }

	var frames []TapeFrame
	eventIdx := 0
	mailIdx := 0

	// Track agent states via replay. All agents start as ASLEEP.
	agentState := make(map[string]string)
	for _, a := range agents {
		if !a.IsHuman {
			agentState[a.WorkingDir] = "ASLEEP"
		}
	}

	// Cumulative mail counts — updated incrementally each frame
	mailCounts := make(map[edgeKey]*edgeCounts)

	ensure := func(k edgeKey) *edgeCounts {
		if c, ok := mailCounts[k]; ok {
			return c
		}
		c := &edgeCounts{}
		mailCounts[k] = c
		return c
	}

	for _, t := range sorted {
		tSec := float64(t) / 1000.0

		// Advance events up to this time
		for eventIdx < len(allEvents) && allEvents[eventIdx].Ts <= tSec {
			ev := allEvents[eventIdx]
			if ev.Type == "agent_state" && ev.Address != "" {
				resolved := ResolveAddress(ev.Address, baseDir)
				agentState[resolved] = strings.ToUpper(ev.New)
			}
			eventIdx++
		}

		// Advance mail frontier — only process new messages
		for mailIdx < len(allMail) && allMail[mailIdx].ts <= tSec {
			msg := allMail[mailIdx].msg
			from := ResolveAddress(msg.From, baseDir)
			recipients := resolveRecipients(msg.To)
			for _, r := range recipients {
				ensure(edgeKey{from, ResolveAddress(r, baseDir)}).direct++
			}
			for _, r := range msg.CC {
				ensure(edgeKey{from, ResolveAddress(r, baseDir)}).cc++
			}
			for _, r := range msg.BCC {
				ensure(edgeKey{from, ResolveAddress(r, baseDir)}).bcc++
			}
			mailIdx++
		}

		// Snapshot current cumulative mail counts into edges (relativized)
		var mailEdges []MailEdge
		for k, c := range mailCounts {
			mailEdges = append(mailEdges, MailEdge{
				Sender:    RelativizeAddress(k.sender, baseDir),
				Recipient: RelativizeAddress(k.recipient, baseDir),
				Count:     c.direct + c.cc + c.bcc,
				Direct:    c.direct,
				CC:        c.cc,
				BCC:       c.bcc,
			})
		}
		if mailEdges == nil {
			mailEdges = []MailEdge{}
		}

		// Build node list: human always present, agents visible after their first event
		var nodes []AgentNode
		for _, a := range agents {
			if a.IsHuman {
				node := a
				node.Alive = true
				node.State = "ACTIVE"
				nodes = append(nodes, node)
				continue
			}
			// Agent visible only if its first event is <= t
			ft, hasFirst := firstEventTs[a.WorkingDir]
			if !hasFirst || ft > tSec {
				continue
			}
			node := a
			if state, ok := agentState[a.WorkingDir]; ok {
				node.State = state
			} else {
				node.State = "ASLEEP"
			}
			node.Alive = node.State == "ACTIVE" || node.State == "IDLE"
			nodes = append(nodes, node)
		}
		if nodes == nil {
			nodes = []AgentNode{}
		}

		stats := computeStats(nodes, mailEdges)

		frames = append(frames, TapeFrame{
			T: t,
			Net: Network{
				Nodes:        nodes,
				AvatarEdges:  avatarEdges,
				ContactEdges: contactEdges,
				MailEdges:    mailEdges,
				Stats:        stats,
			},
		})
	}

	return frames, nil
}

// readEventsJSONL reads logs/events.jsonl and returns parsed event records
// for event types we care about: agent_state, heartbeat_start, refresh_start.
func readEventsJSONL(agentDir string) []eventRecord {
	path := filepath.Join(agentDir, "logs", "events.jsonl")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var events []eventRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev eventRecord
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		switch ev.Type {
		case "agent_state", "heartbeat_start", "refresh_start":
			events = append(events, ev)
		}
	}
	return events
}

// mailTimestamp extracts the best timestamp from a mail message as unix seconds.
// Prefers SentAt, falls back to ReceivedAt.
func mailTimestamp(msg MailMessage) float64 {
	for _, raw := range []string{msg.SentAt, msg.ReceivedAt} {
		if raw == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			t, err = time.Parse(time.RFC3339Nano, raw)
		}
		if err == nil {
			return float64(t.UnixMilli()) / 1000.0
		}
	}
	return 0
}
