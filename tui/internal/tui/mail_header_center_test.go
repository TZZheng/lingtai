package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// centerStart returns the column index of the first rune of center within line
// (both ANSI-stripped), or -1 if center is not found.
func centerStart(line, center string) int {
	plain := ansi.Strip(line)
	c := ansi.Strip(center)
	idx := strings.Index(plain, c)
	if idx < 0 {
		return -1
	}
	// Index is a byte offset; convert to a rune/cell column. The header is
	// ASCII-padded on the left, but the brand/center may contain multibyte
	// runes, so measure display width up to the match.
	return lipgloss.Width(plain[:idx])
}

// TestComposeCenteredHeaderAbsoluteMidpoint proves that, given ample width, the
// center block is placed at the absolute terminal center: its midpoint equals
// the terminal midpoint (within one cell for parity), independent of the left
// and right block widths.
func TestComposeCenteredHeaderAbsoluteMidpoint(t *testing.T) {
	left := "  LingTai"
	center := "✹ 菩提本无树 ✹"
	right := "orch ◉ active"
	width := 80

	line := composeCenteredHeader(left, center, right, width)

	start := centerStart(line, center)
	if start < 0 {
		t.Fatalf("center block not found in line: %q", ansi.Strip(line))
	}
	centerW := lipgloss.Width(center)
	gotMid := start + centerW/2
	wantMid := width / 2
	if diff := gotMid - wantMid; diff < -1 || diff > 1 {
		t.Errorf("center midpoint = %d, want terminal midpoint %d (start=%d centerW=%d)", gotMid, wantMid, start, centerW)
	}
}

// TestComposeCenteredHeaderAnchorsLeftAndRight verifies the left block stays at
// column 0 and the right block ends flush with the terminal width.
func TestComposeCenteredHeaderAnchorsLeftAndRight(t *testing.T) {
	left := "  LingTai"
	center := "✹ hi ✹"
	right := "orch ◉ active"
	width := 80

	line := composeCenteredHeader(left, center, right, width)
	plain := ansi.Strip(line)

	if !strings.HasPrefix(plain, ansi.Strip(left)) {
		t.Errorf("line should start with left block %q; got %q", left, plain)
	}
	if !strings.HasSuffix(plain, ansi.Strip(right)) {
		t.Errorf("line should end with right block %q; got %q", right, plain)
	}
}

// TestComposeCenteredHeaderNarrowFallback verifies that when the three blocks
// cannot be laid out without overlap, the helper still returns a non-empty line
// containing all three blocks (graceful compact fallback, no panic).
func TestComposeCenteredHeaderNarrowFallback(t *testing.T) {
	left := "  LingTai brand block"
	center := "✹ 菩提本无树 ✹"
	right := "orchestrator ◉ active network[3]"
	width := 40 // too narrow for absolute centering

	line := composeCenteredHeader(left, center, right, width)
	plain := ansi.Strip(line)

	if plain == "" {
		t.Fatal("narrow fallback produced an empty line")
	}
	for _, block := range []string{left, center, right} {
		if !strings.Contains(plain, ansi.Strip(block)) {
			t.Errorf("narrow fallback line missing block %q; got %q", block, plain)
		}
	}
}

// TestComposeCenteredHeaderNoOverlap verifies the center block never overlaps
// the left or right blocks when absolute centering is used.
func TestComposeCenteredHeaderNoOverlap(t *testing.T) {
	left := "  LingTai"
	center := "✹ 菩提本无树 ✹"
	right := "orch ◉ active"
	width := 80

	line := composeCenteredHeader(left, center, right, width)
	start := centerStart(line, center)
	if start < 0 {
		t.Fatalf("center block not found: %q", ansi.Strip(line))
	}
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	centerW := lipgloss.Width(center)
	if start < leftW {
		t.Errorf("center starts at %d, overlaps left block of width %d", start, leftW)
	}
	if end := start + centerW; end > width-rightW {
		t.Errorf("center ends at %d, overlaps right block (width-rightW=%d)", end, width-rightW)
	}
}
