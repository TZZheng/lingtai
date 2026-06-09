package tui

import (
	"strings"
	"testing"
)

// TestEveryCommandHasHelpDoc guards the real maintenance risk: a slash command
// is added to DefaultCommands() but ships without a help page. Every command
// must have a non-empty embedded help/<name>.md.
func TestEveryCommandHasHelpDoc(t *testing.T) {
	for _, cmd := range DefaultCommands() {
		doc := readHelpDoc(cmd.Name)
		if strings.TrimSpace(doc) == "" {
			t.Errorf("command %q has no embedded help/%s.md", cmd.Name, cmd.Name)
		}
	}
}

// TestOverviewDocEmbedded ensures the overview intro is embedded and non-empty.
func TestOverviewDocEmbedded(t *testing.T) {
	if strings.TrimSpace(readHelpDoc("overview")) == "" {
		t.Fatal("help/overview.md is missing or empty")
	}
}

// TestBuildHelpEntries verifies the viewer entries cover the overview plus one
// entry per command, all with non-empty content.
func TestBuildHelpEntries(t *testing.T) {
	entries := buildHelpEntries()

	cmds := DefaultCommands()
	wantCount := len(cmds) + 1 // overview + one per command
	if len(entries) != wantCount {
		t.Fatalf("buildHelpEntries() returned %d entries, want %d", len(entries), wantCount)
	}

	// First entry is the overview.
	if entries[0].Content == "" {
		t.Error("overview entry has empty content")
	}

	// Remaining entries map 1:1 to commands, in order, with content.
	for i, cmd := range cmds {
		e := entries[i+1]
		wantLabel := "/" + cmd.Name
		if e.Label != wantLabel {
			t.Errorf("entry %d: label = %q, want %q", i+1, e.Label, wantLabel)
		}
		if strings.TrimSpace(e.Content) == "" {
			t.Errorf("entry %d (%s): empty content", i+1, wantLabel)
		}
	}
}

// TestHelpDocsHaveNoStrayFiles ensures no embedded help/*.md is orphaned — every
// non-overview doc corresponds to a real command, so docs don't drift out of
// sync after a command is removed.
func TestHelpDocsHaveNoStrayFiles(t *testing.T) {
	known := map[string]bool{"overview": true}
	for _, cmd := range DefaultCommands() {
		known[cmd.Name] = true
	}

	dirents, err := helpDocs.ReadDir("help")
	if err != nil {
		t.Fatalf("reading embedded help dir: %v", err)
	}
	for _, de := range dirents {
		name := de.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		stem := strings.TrimSuffix(name, ".md")
		if !known[stem] {
			t.Errorf("orphan help doc help/%s — no matching command", name)
		}
	}
}

// TestNewHelpModelTitle sanity-checks the wrapper constructs with a title.
func TestNewHelpModelTitle(t *testing.T) {
	m := NewHelpModel()
	if m.inner.title == "" {
		t.Error("HelpModel inner viewer has empty title")
	}
}
