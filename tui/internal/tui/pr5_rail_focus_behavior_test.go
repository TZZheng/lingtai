package tui

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func pr5UpdateRailFocusApp(t *testing.T, a App, msg tea.Msg) App {
	t.Helper()
	model, _ := a.Update(msg)
	updated, ok := model.(App)
	if !ok {
		t.Fatalf("Update(%T) returned %T, want App", msg, model)
	}
	return updated
}

func pr5ContentLocalX(t *testing.T, budget LayoutBudget, terminalX int) (int, bool) {
	t.Helper()
	method := reflect.ValueOf(budget).MethodByName("ContentLocalX")
	if !method.IsValid() {
		t.Fatal("LayoutBudget.ContentLocalX is missing; root input must not invent a second geometry calculation")
	}
	out := method.Call([]reflect.Value{reflect.ValueOf(terminalX)})
	if len(out) != 2 || out[0].Kind() != reflect.Int || out[1].Kind() != reflect.Bool {
		t.Fatalf("LayoutBudget.ContentLocalX signature = %#v, want (int, bool)", out)
	}
	return int(out[0].Int()), out[1].Bool()
}

func TestPR5Stage3LayoutBudgetTranslatesTerminalXOnce(t *testing.T) {
	a := mailLayoutApp(t)
	a.width = 84
	visible := a.layoutBudget()

	for _, tc := range []struct {
		terminalX int
		localX    int
		ok        bool
	}{
		{terminalX: 23, localX: 0, ok: false},
		{terminalX: 24, localX: 0, ok: true},
		{terminalX: 83, localX: 59, ok: true},
		{terminalX: 84, localX: 0, ok: false},
	} {
		gotX, gotOK := pr5ContentLocalX(t, visible, tc.terminalX)
		if gotX != tc.localX || gotOK != tc.ok {
			t.Fatalf("visible ContentLocalX(%d) = (%d, %v), want (%d, %v)", tc.terminalX, gotX, gotOK, tc.localX, tc.ok)
		}
	}

	a.width = 83
	hidden := a.layoutBudget()
	for _, tc := range []struct {
		terminalX int
		localX    int
		ok        bool
	}{
		{terminalX: 0, localX: 0, ok: true},
		{terminalX: 82, localX: 82, ok: true},
		{terminalX: 83, localX: 0, ok: false},
	} {
		gotX, gotOK := pr5ContentLocalX(t, hidden, tc.terminalX)
		if gotX != tc.localX || gotOK != tc.ok {
			t.Fatalf("hidden ContentLocalX(%d) = (%d, %v), want (%d, %v)", tc.terminalX, gotX, gotOK, tc.localX, tc.ok)
		}
	}
}

func TestPR5Stage3TabMouseAndResizeOwnOneRailChatFocus(t *testing.T) {
	a := mailLayoutApp(t)
	a.agentRail.rows = append(a.agentRail.rows, railRow{label: "worker"})
	a = pr5UpdateRailFocusApp(t, a, tea.WindowSizeMsg{Width: 84, Height: 24})

	if !a.mail.input.Focused() {
		t.Fatal("chat must own default focus")
	}

	a = pr5UpdateRailFocusApp(t, a, tea.KeyPressMsg{Code: tea.KeyTab})
	if a.mail.input.Focused() {
		t.Fatal("Tab with a visible rail must move the single focus owner to the rail and blur chat")
	}

	a = pr5UpdateRailFocusApp(t, a, tea.MouseClickMsg(tea.Mouse{X: 24, Y: 2, Button: tea.MouseLeft}))
	if !a.mail.input.Focused() {
		t.Fatal("x=ContentX must route to chat-local x=0 and restore chat focus")
	}

	a = pr5UpdateRailFocusApp(t, a, tea.MouseClickMsg(tea.Mouse{X: 23, Y: 2, Button: tea.MouseLeft}))
	if a.mail.input.Focused() {
		t.Fatal("x=ContentX-1 must route to the rail and blur chat")
	}

	a = pr5UpdateRailFocusApp(t, a, tea.KeyPressMsg{Code: tea.KeyEscape})
	if !a.mail.input.Focused() {
		t.Fatal("Esc while rail-focused must return the single focus owner to chat")
	}

	a = pr5UpdateRailFocusApp(t, a, tea.KeyPressMsg{Code: tea.KeyTab})
	if a.mail.input.Focused() {
		t.Fatal("precondition: visible rail must be focused before it hides")
	}
	a = pr5UpdateRailFocusApp(t, a, tea.WindowSizeMsg{Width: 83, Height: 24})
	if !a.mail.input.Focused() {
		t.Fatal("hiding the rail below minimum width must force chat focus")
	}
	a = pr5UpdateRailFocusApp(t, a, tea.KeyPressMsg{Code: tea.KeyTab})
	if !a.mail.input.Focused() {
		t.Fatal("Tab must be inert for rail focus while the rail is hidden")
	}

	a.visiting = true
	a = pr5UpdateRailFocusApp(t, a, tea.WindowSizeMsg{Width: 84, Height: 24})
	a = pr5UpdateRailFocusApp(t, a, tea.KeyPressMsg{Code: tea.KeyTab})
	if !a.mail.input.Focused() {
		t.Fatal("Tab must be inert for the retained home rail during a cross-project visit")
	}
}
