package tui

import (
	"testing"

	"github.com/anthropics/lingtai-tui/internal/fs"
)

// Issue #40: the notification footer condenses the kernel's vital-signs
// meta block into a single line for the TUI banner.
func TestFormatNotificationMetaFooter(t *testing.T) {
	tests := []struct {
		name string
		meta *fs.NotificationMeta
		want string
	}{
		{
			name: "nil meta — older event pre-dating issue #40",
			meta: nil,
			want: "",
		},
		{
			name: "all sentinels — kernel computed nothing yet",
			meta: &fs.NotificationMeta{
				CurrentTime: "",
				Context: &fs.NotificationMetaContext{
					SystemTokens:  -1,
					HistoryTokens: -1,
					Usage:         -1.0,
				},
				InjectionSeq: 0,
			},
			want: "",
		},
		{
			name: "full meta",
			meta: &fs.NotificationMeta{
				CurrentTime: "2026-05-05T21:10:48-07:00",
				Context: &fs.NotificationMetaContext{
					SystemTokens:  38398,
					HistoryTokens: 109121,
					Usage:         0.147519,
				},
				InjectionSeq: 2,
			},
			// Time format depends on the local TZ database — check the
			// non-time fragments directly via substring checks below.
			want: "ctx 14.8%",
		},
		{
			name: "ctx only — time/seq dropped",
			meta: &fs.NotificationMeta{
				Context: &fs.NotificationMetaContext{Usage: 0.5},
			},
			want: "ctx 50.0%",
		},
		{
			name: "seq only",
			meta: &fs.NotificationMeta{
				InjectionSeq: 7,
			},
			want: "seq 7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatNotificationMetaFooter(tt.meta)
			if tt.name == "full meta" {
				// Time fragment is TZ-dependent; verify the prefix is
				// stable and that the time fragment + seq landed.
				if !contains(got, "ctx 14.8%") {
					t.Errorf("missing ctx fragment in %q", got)
				}
				if !contains(got, "seq 2") {
					t.Errorf("missing seq fragment in %q", got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
