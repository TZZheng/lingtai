package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestPR5Stage3PasteRoutesOnlyToChatFocus(t *testing.T) {
	a := mailLayoutApp(t)
	a = pr5UpdateRailFocusApp(t, a, tea.WindowSizeMsg{Width: 84, Height: 24})
	a.mail.input.SetValue("seed")

	a = pr5UpdateRailFocusApp(t, a, tea.PasteMsg{Content: "-chat"})
	if got := a.mail.input.Value(); got != "seed-chat" {
		t.Fatalf("chat-focused paste value = %q, want one exact insertion %q", got, "seed-chat")
	}

	a = pr5UpdateRailFocusApp(t, a, tea.KeyPressMsg{Code: tea.KeyTab})
	if a.mailFocus != mailFocusRail || a.mail.input.Focused() {
		t.Fatal("precondition: visible rail must own focus and blur chat")
	}
	a = pr5UpdateRailFocusApp(t, a, tea.PasteMsg{Content: "-rail"})
	if got := a.mail.input.Value(); got != "seed-chat" {
		t.Fatalf("rail-focused paste mutated chat value: got %q", got)
	}

	a = pr5UpdateRailFocusApp(t, a, tea.WindowSizeMsg{Width: 83, Height: 24})
	if a.mailFocus != mailFocusChat || !a.mail.input.Focused() {
		t.Fatal("hiding the rail must restore chat focus before paste")
	}
	a = pr5UpdateRailFocusApp(t, a, tea.PasteMsg{Content: "-hidden"})
	if got := a.mail.input.Value(); got != "seed-chat-hidden" {
		t.Fatalf("hidden-rail paste value = %q, want one exact insertion %q", got, "seed-chat-hidden")
	}

	a.visiting = true
	a = pr5UpdateRailFocusApp(t, a, tea.WindowSizeMsg{Width: 84, Height: 24})
	if a.layoutBudget().RailVisible || a.mailFocus != mailFocusChat || !a.mail.input.Focused() {
		t.Fatal("cross-project visit must hide the home rail and keep chat focus")
	}
	a = pr5UpdateRailFocusApp(t, a, tea.PasteMsg{Content: "-visit"})
	if got := a.mail.input.Value(); got != "seed-chat-hidden-visit" {
		t.Fatalf("visit paste value = %q, want one exact insertion %q", got, "seed-chat-hidden-visit")
	}
}

func TestPR5Stage3CopyModeKeepsMailKeyPrecedenceWithRetainedRailFocus(t *testing.T) {
	a := mailLayoutApp(t)
	a = pr5UpdateRailFocusApp(t, a, tea.WindowSizeMsg{Width: 84, Height: 24})
	a = pr5UpdateRailFocusApp(t, a, tea.KeyPressMsg{Code: tea.KeyTab})
	if a.mailFocus != mailFocusRail || a.mail.input.Focused() {
		t.Fatal("precondition: visible rail must own focus and blur chat")
	}

	a = pr5UpdateRailFocusApp(t, a, ctrlYKey(t))
	if !a.mail.copyMode {
		t.Fatal("ctrl+y must reach Mail and enable native copy mode even while rail focus is retained")
	}
	if a.mailFocus != mailFocusRail || a.mail.input.Focused() {
		t.Fatal("native copy mode must not silently transfer the retained rail focus")
	}
	if got := a.View().MouseMode; got != tea.MouseModeNone {
		t.Fatalf("copy-mode MouseMode = %v, want MouseModeNone for terminal-native selection", got)
	}

	a = pr5UpdateRailFocusApp(t, a, tea.KeyPressMsg{Code: tea.KeyEsc})
	if a.mail.copyMode {
		t.Fatal("first Esc must reach Mail and exit copy mode before the rail handles Esc")
	}
	if a.mailFocus != mailFocusRail || a.mail.input.Focused() {
		t.Fatal("copy-mode Esc must preserve the retained rail focus")
	}
	if got := a.View().MouseMode; got != tea.MouseModeCellMotion {
		t.Fatalf("post-copy MouseMode = %v, want MouseModeCellMotion", got)
	}

	a = pr5UpdateRailFocusApp(t, a, tea.KeyPressMsg{Code: tea.KeyEsc})
	if a.mailFocus != mailFocusChat || !a.mail.input.Focused() {
		t.Fatal("second Esc must return rail focus to chat after Mail has exited copy mode")
	}
}
