package tui

import "testing"

// manifestContextLimit must read context_limit from EITHER nesting, because the
// two artifacts fs.ReadInitManifest returns disagree on its location:
//   - the kernel-resolved manifest.resolved.json puts it at the TOP LEVEL
//     (verified across every live agent on this machine); llm.context_limit is
//     absent there. PR #441 read only the llm path and so always showed no limit.
//   - the raw init.json fallback keeps the saved-preset shape llm.context_limit.
//
// Top level wins when both are present. This pins the fix so a future refactor
// can't silently regress to the single-nesting assumption.
func TestManifestContextLimit(t *testing.T) {
	tests := []struct {
		name     string
		manifest map[string]interface{}
		want     int64
	}{
		{
			name:     "resolved manifest — top-level context_limit",
			manifest: map[string]interface{}{"context_limit": float64(250000), "llm": map[string]interface{}{"model": "x"}},
			want:     250000,
		},
		{
			name:     "raw init.json fallback — llm.context_limit",
			manifest: map[string]interface{}{"llm": map[string]interface{}{"context_limit": float64(128000)}},
			want:     128000,
		},
		{
			name:     "both present — top level wins",
			manifest: map[string]interface{}{"context_limit": float64(300000), "llm": map[string]interface{}{"context_limit": float64(128000)}},
			want:     300000,
		},
		{
			name:     "neither present — unknown (0)",
			manifest: map[string]interface{}{"llm": map[string]interface{}{"model": "x"}},
			want:     0,
		},
		{
			name:     "zero is treated as unknown",
			manifest: map[string]interface{}{"context_limit": float64(0)},
			want:     0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := manifestContextLimit(tt.manifest); got != tt.want {
				t.Errorf("manifestContextLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}
