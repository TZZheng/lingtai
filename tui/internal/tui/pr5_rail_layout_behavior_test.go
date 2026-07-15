package tui

import (
	"reflect"
	"testing"
)

func TestPR5Stage3RailLayoutUsesOneMinimumWidthBudgetAndVisitFence(t *testing.T) {
	const railWidth = 24

	tests := []struct {
		name         string
		width        int
		visiting     bool
		wantRail     int
		wantContent  int
		wantVisible  bool
		wantRailX    int
		wantContentX int
	}{
		{
			name:         "exact minimum shows rail",
			width:        minimumChatWidth + railWidth,
			wantRail:     railWidth,
			wantContent:  minimumChatWidth,
			wantVisible:  true,
			wantRailX:    0,
			wantContentX: railWidth,
		},
		{
			name:         "one above minimum grows chat only",
			width:        minimumChatWidth + railWidth + 1,
			wantRail:     railWidth,
			wantContent:  minimumChatWidth + 1,
			wantVisible:  true,
			wantRailX:    0,
			wantContentX: railWidth,
		},
		{
			name:         "one below minimum hides rail without a ghost column",
			width:        minimumChatWidth + railWidth - 1,
			wantRail:     0,
			wantContent:  minimumChatWidth + railWidth - 1,
			wantVisible:  false,
			wantRailX:    0,
			wantContentX: 0,
		},
		{
			name:         "cross-project visit hides home rail",
			width:        minimumChatWidth + railWidth,
			visiting:     true,
			wantRail:     0,
			wantContent:  minimumChatWidth + railWidth,
			wantVisible:  false,
			wantRailX:    0,
			wantContentX: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := App{
				width:       test.width,
				height:      30,
				currentView: appViewMail,
				visiting:    test.visiting,
			}
			budget := app.layoutBudget()
			if budget.TerminalWidth != test.width ||
				budget.RailWidth != test.wantRail ||
				budget.ContentWidth != test.wantContent ||
				budget.RailVisible != test.wantVisible {
				t.Fatalf(
					"width=%d visiting=%t got terminal/rail/content/visible=%d/%d/%d/%t, want %d/%d/%d/%t",
					test.width,
					test.visiting,
					budget.TerminalWidth,
					budget.RailWidth,
					budget.ContentWidth,
					budget.RailVisible,
					test.width,
					test.wantRail,
					test.wantContent,
					test.wantVisible,
				)
			}
			railX, contentX := pr5LayoutOrigins(t, budget)
			if railX != test.wantRailX || contentX != test.wantContentX {
				t.Fatalf(
					"width=%d visiting=%t got rail/content origins=%d/%d, want %d/%d",
					test.width,
					test.visiting,
					railX,
					contentX,
					test.wantRailX,
					test.wantContentX,
				)
			}
		})
	}
}

func pr5LayoutOrigins(t *testing.T, budget LayoutBudget) (int, int) {
	t.Helper()
	value := reflect.ValueOf(budget)
	fields := []string{"RailX", "ContentX"}
	origins := make([]int, len(fields))
	for i, name := range fields {
		field := value.FieldByName(name)
		if !field.IsValid() || field.Kind() != reflect.Int {
			t.Fatalf("LayoutBudget missing integer %s; render, resize, and X hit testing cannot share one root geometry", name)
		}
		origins[i] = int(field.Int())
	}
	return origins[0], origins[1]
}
