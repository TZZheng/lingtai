package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/anthropics/lingtai-tui/i18n"
)

// codexEntryIndex finds the entry index for a Codex account by its token-file
// path. Fails the test when absent.
func codexEntryIndex(t *testing.T, m LoginModel, absPath string) int {
	t.Helper()
	for i := range m.entries {
		if m.entries[i].Provider == "codex" && m.entries[i].CodexPath == absPath {
			return i
		}
	}
	t.Fatalf("could not find codex entry for %q; entries=%#v", absPath, m.entries)
	return -1
}

// TestLoginModel_NoPoolFileShowsNotInPool verifies that with NO pool file, a
// valid Codex account renders as absent from the pool rather than inventing a
// phantom default weight. This keeps the display aligned with the kernel, which
// cannot load-balance onto accounts absent from codex-auth-pool.json.
func TestLoginModel_NoPoolFileShowsNotInPool(t *testing.T) {
	t.Setenv("LINGTAI_TUI_DIR", "")
	_, globalDir := withTempCodexHome(t)

	acctPath := filepath.Join(globalDir, codexAuthSubdir, "work.json")
	writeCodexTokenAt(t, acctPath, "work@example.com")

	m := NewLoginModel("", globalDir)
	idx := codexEntryIndex(t, m, acctPath)
	m.entries[idx].Status = loginValid

	if inPool, weight := m.codexEntryMembership(m.entries[idx]); inPool || weight != 0 {
		t.Errorf("valid account with no pool file should be absent from pool; inPool=%v weight=%d", inPool, weight)
	}

	m.width = 100
	view := m.View()
	if !strings.Contains(view, i18n.T("login.codex_pool_not_member")) {
		t.Errorf("view should show the not-in-pool label; view=%q", view)
	}
	if strings.Contains(view, strings.TrimSpace(strings.Split(i18n.T("login.codex_pool_weight"), "%")[0])) {
		t.Errorf("view must not show a phantom pool weight when no pool file exists; view=%q", view)
	}
}

// TestLoginModel_PlusAddsMissingAccountAtWeightOne verifies the "+" key joins
// an absent Codex account to the pool at weight 1 and persists it
// (lazy-writing the pool file), without touching any preset binding.
func TestLoginModel_PlusAddsMissingAccountAtWeightOne(t *testing.T) {
	t.Setenv("LINGTAI_TUI_DIR", "")
	_, globalDir := withTempCodexHome(t)

	acctPath := filepath.Join(globalDir, codexAuthSubdir, "work.json")
	writeCodexTokenAt(t, acctPath, "work@example.com")

	m := NewLoginModel("", globalDir)
	idx := codexEntryIndex(t, m, acctPath)
	m.entries[idx].Status = loginValid
	m.cursor = idx

	// Absent from pool; "+" should join at weight 1.
	m, cmd := m.Update(tea.KeyPressMsg{Text: "+", Code: '+'})
	if cmd != nil {
		t.Fatal("editing pool weight must not start a command")
	}
	if got := m.poolWeights[acctPath]; got != 1 {
		t.Fatalf("in-memory weight after + = %d, want 1", got)
	}
	// Persisted to disk with the relative ref.
	if got := codexPoolWeights(globalDir)[acctPath]; got != 1 {
		t.Fatalf("persisted weight after + = %d, want 1", got)
	}
}

// TestLoginModel_MinusClampsAtZero verifies "-" decrements and never goes below
// 0 (disabled).
func TestLoginModel_MinusClampsAtZero(t *testing.T) {
	t.Setenv("LINGTAI_TUI_DIR", "")
	_, globalDir := withTempCodexHome(t)

	acctPath := filepath.Join(globalDir, codexAuthSubdir, "work.json")
	writeCodexTokenAt(t, acctPath, "work@example.com")

	m := NewLoginModel("", globalDir)
	idx := codexEntryIndex(t, m, acctPath)
	m.entries[idx].Status = loginValid
	m.cursor = idx

	// Absent accounts stay absent on "-"; "+" first joins at 1, then "-"
	// disables to 0 and further "-" stays 0.
	m, _ = m.Update(tea.KeyPressMsg{Text: "-", Code: '-'})
	if _, ok := m.poolWeights[acctPath]; ok {
		t.Fatalf("- on absent account must not add it to pool; weights=%v", m.poolWeights)
	}
	m, _ = m.Update(tea.KeyPressMsg{Text: "+", Code: '+'})
	m, _ = m.Update(tea.KeyPressMsg{Text: "-", Code: '-'})
	if got := m.poolWeights[acctPath]; got != 0 {
		t.Fatalf("weight after + then - = %d, want 0", got)
	}
	m, _ = m.Update(tea.KeyPressMsg{Text: "-", Code: '-'})
	if got := m.poolWeights[acctPath]; got != 0 {
		t.Fatalf("weight after second - should clamp at 0; got %d", got)
	}
}

// TestLoginModel_ZeroDisablesAccount verifies "0" sets the weight straight to 0
// (disabled) and the row renders the disabled label.
func TestLoginModel_ZeroDisablesAccount(t *testing.T) {
	t.Setenv("LINGTAI_TUI_DIR", "")
	_, globalDir := withTempCodexHome(t)

	acctPath := filepath.Join(globalDir, codexAuthSubdir, "work.json")
	writeCodexTokenAt(t, acctPath, "work@example.com")

	m := NewLoginModel("", globalDir)
	idx := codexEntryIndex(t, m, acctPath)
	m.entries[idx].Status = loginValid
	m.cursor = idx

	// Bump to 2 first so we can prove "0" is absolute, not a decrement.
	m, _ = m.Update(tea.KeyPressMsg{Text: "+", Code: '+'})
	m, _ = m.Update(tea.KeyPressMsg{Text: "+", Code: '+'})
	if got := m.poolWeights[acctPath]; got != 2 {
		t.Fatalf("precondition: weight should be 2; got %d", got)
	}
	m, _ = m.Update(tea.KeyPressMsg{Text: "0", Code: '0'})
	if got := m.poolWeights[acctPath]; got != 0 {
		t.Fatalf("weight after 0 = %d, want 0", got)
	}

	m.width = 100
	view := m.View()
	if !strings.Contains(view, i18n.T("login.codex_pool_disabled")) {
		t.Errorf("disabled account row should show the disabled label; view=%q", view)
	}
}

// TestLoginModel_PoolEditDoesNotRewritePresets guards the core separation-of-
// concerns rule: editing a pool weight touches ONLY the pool file, never the
// active-account preset binding.
func TestLoginModel_PoolEditDoesNotRewritePresets(t *testing.T) {
	t.Setenv("LINGTAI_TUI_DIR", "")
	_, globalDir := withTempCodexHome(t)

	// A legacy-bound saved codex preset (no codex_auth_path).
	saveCodexPresetForTest(t, "codex-a", "")

	acctPath := filepath.Join(globalDir, codexAuthSubdir, "work.json")
	writeCodexTokenAt(t, acctPath, "work@example.com")

	m := NewLoginModel("", globalDir)
	idx := codexEntryIndex(t, m, acctPath)
	m.entries[idx].Status = loginValid
	m.cursor = idx

	m, _ = m.Update(tea.KeyPressMsg{Text: "+", Code: '+'})

	// The preset must remain legacy-bound — pool editing never rewrites it.
	if ref, ok := reloadSavedRef(t, "codex-a"); ok {
		t.Errorf("pool weight edit must not bind the preset; got codex_auth_path=%q", ref)
	}
}

// TestLoginModel_PoolWeightIgnoredOnVirtualRow verifies the pool keys are inert
// on the virtual "add account" row (no crash, no pool write).
func TestLoginModel_PoolWeightIgnoredOnVirtualRow(t *testing.T) {
	t.Setenv("LINGTAI_TUI_DIR", "")
	_, globalDir := withTempCodexHome(t)

	acctPath := filepath.Join(globalDir, codexAuthSubdir, "work.json")
	writeCodexTokenAt(t, acctPath, "work@example.com")

	m := NewLoginModel("", globalDir)
	// Put the cursor on the virtual add row (index == len(entries)).
	m.cursor = len(m.entries)
	if !m.cursorOnVirtualRow() {
		t.Fatalf("precondition: cursor should be on the virtual add row")
	}
	m, cmd := m.Update(tea.KeyPressMsg{Text: "+", Code: '+'})
	if cmd != nil {
		t.Fatal("pool key on virtual row must not start a command")
	}
	// No pool file should have been created.
	if w := codexPoolWeights(globalDir); len(w) != 0 {
		t.Errorf("virtual-row pool edit must not write the pool file; got %v", w)
	}
}

// TestLoginModel_CorruptPoolFileWarns guards N5: a malformed codex-auth-pool.json
// must surface a visible warning in the credentials view (and set the model's
// poolCorrupt flag) instead of silently rendering every account as "not in
// pool". The bad file must be left untouched.
func TestLoginModel_CorruptPoolFileWarns(t *testing.T) {
	t.Setenv("LINGTAI_TUI_DIR", "")
	_, globalDir := withTempCodexHome(t)

	acctPath := filepath.Join(globalDir, codexAuthSubdir, "work.json")
	writeCodexTokenAt(t, acctPath, "work@example.com")

	// Write a malformed pool file.
	poolFile := codexPoolPath(globalDir)
	badContent := []byte("{not valid json")
	if err := os.WriteFile(poolFile, badContent, 0o644); err != nil {
		t.Fatalf("seed malformed pool: %v", err)
	}

	m := NewLoginModel("", globalDir)
	if !m.poolCorrupt {
		t.Fatal("model should flag a malformed pool file as corrupt")
	}

	m.width = 100
	view := m.View()
	if !strings.Contains(view, i18n.T("login.codex_pool_corrupt")) {
		t.Errorf("view should warn about the corrupt pool file; view=%q", view)
	}

	// The corrupt file must not have been clobbered.
	got, err := os.ReadFile(poolFile)
	if err != nil {
		t.Fatalf("read pool file back: %v", err)
	}
	if string(got) != string(badContent) {
		t.Errorf("corrupt pool file must be preserved verbatim; got %q", string(got))
	}
}

// TestLoginModel_ModelClassifiedPoolRendersClassified guards login honesty for
// a v2 (model-classified) pool: rows must render the classified state — never a
// flat membership label the kernel would not honor — the note explaining that
// flat edits are off must appear, and the +/-/0 footer hint must be suppressed
// (the keys are inert, so advertising them would be a lie).
func TestLoginModel_ModelClassifiedPoolRendersClassified(t *testing.T) {
	t.Setenv("LINGTAI_TUI_DIR", "")
	_, globalDir := withTempCodexHome(t)

	acctPath := filepath.Join(globalDir, codexAuthSubdir, "work.json")
	writeCodexTokenAt(t, acctPath, "work@example.com")
	seedModelClassifiedPool(t, globalDir)

	m := NewLoginModel("", globalDir)
	if !m.poolModelClassified {
		t.Fatal("model should detect the model-classified pool")
	}
	if m.poolModelCount != 2 {
		t.Fatalf("poolModelCount = %d, want 2", m.poolModelCount)
	}

	idx := codexEntryIndex(t, m, acctPath)
	m.entries[idx].Status = loginValid
	m.cursor = idx
	m.width = 120
	view := m.View()

	if !strings.Contains(view, i18n.T("login.codex_pool_model_classified")) {
		t.Errorf("view should show the model-classified row label; view=%q", view)
	}
	if strings.Contains(view, i18n.T("login.codex_pool_not_member")) {
		t.Errorf("view must not show a flat not-in-pool label for a classified pool; view=%q", view)
	}
	note := fmt.Sprintf(i18n.T("login.codex_pool_model_classified_note"), 2)
	if !strings.Contains(view, note) {
		t.Errorf("view should show the classified-pool note; want %q in view=%q", note, view)
	}
	if strings.Contains(view, i18n.T("login.codex_pool_hint")) {
		t.Errorf("footer must not advertise the inert +/-/0 pool keys on a classified pool; view=%q", view)
	}
}

// TestLoginModel_ModelClassifiedPoolRefusesFlatEdit guards the edit path: on a
// model-classified pool the +/-/0 keys must produce the informational
// hand-edit message and leave both the pool file bytes and the in-memory
// weights untouched — never write a flat entry the kernel would ignore.
func TestLoginModel_ModelClassifiedPoolRefusesFlatEdit(t *testing.T) {
	t.Setenv("LINGTAI_TUI_DIR", "")
	_, globalDir := withTempCodexHome(t)

	acctPath := filepath.Join(globalDir, codexAuthSubdir, "work.json")
	writeCodexTokenAt(t, acctPath, "work@example.com")
	raw := seedModelClassifiedPool(t, globalDir)

	m := NewLoginModel("", globalDir)
	idx := codexEntryIndex(t, m, acctPath)
	m.entries[idx].Status = loginValid
	m.cursor = idx

	note := fmt.Sprintf(i18n.T("login.codex_pool_model_classified_note"), 2)
	for _, key := range []tea.KeyPressMsg{
		{Text: "+", Code: '+'},
		{Text: "-", Code: '-'},
		{Text: "0", Code: '0'},
	} {
		var cmd tea.Cmd
		m, cmd = m.Update(key)
		if cmd != nil {
			t.Fatalf("pool key %q on classified pool must not start a command", key.Text)
		}
		if m.message != note {
			t.Errorf("key %q: message = %q, want the classified-pool note %q", key.Text, m.message, note)
		}
		if m.messageOK {
			t.Errorf("key %q: the refusal message must not style as success", key.Text)
		}
		if _, ok := m.poolWeights[acctPath]; ok {
			t.Errorf("key %q: in-memory weights must stay untouched; weights=%v", key.Text, m.poolWeights)
		}
	}

	got, err := os.ReadFile(codexPoolPath(globalDir))
	if err != nil {
		t.Fatalf("read pool file back: %v", err)
	}
	if string(got) != string(raw) {
		t.Errorf("classified pool file must stay byte-identical after refused edits;\n got: %s\nwant: %s", got, raw)
	}
}

// TestLoginModel_EmptyModelsClassifiedPoolTruthful pins login honesty for a v2
// pool whose `models` map is present but EMPTY, with a sibling flat `accounts`
// entry for this very account. Kernel semantics: any `models` dict — even `{}`
// — classifies the pool and the sibling accounts are ignored. The UI must
// therefore render the classified state (never the sibling's flat weight,
// which the kernel would not honor), show the note with the truthful count of
// 0, suppress the +/-/0 footer hint, and keep those keys inert — no write, no
// in-memory weight, file bytes untouched.
func TestLoginModel_EmptyModelsClassifiedPoolTruthful(t *testing.T) {
	t.Setenv("LINGTAI_TUI_DIR", "")
	_, globalDir := withTempCodexHome(t)

	acctPath := filepath.Join(globalDir, codexAuthSubdir, "work.json")
	writeCodexTokenAt(t, acctPath, "work@example.com")
	raw := seedEmptyModelsClassifiedPool(t, globalDir) // sibling accounts entry: work.json @ weight 3

	m := NewLoginModel("", globalDir)
	if !m.poolModelClassified {
		t.Fatal("an empty-but-present models map must classify the pool")
	}
	if m.poolModelCount != 0 {
		t.Fatalf("poolModelCount = %d, want 0", m.poolModelCount)
	}

	idx := codexEntryIndex(t, m, acctPath)
	m.entries[idx].Status = loginValid
	m.cursor = idx
	m.width = 120
	view := m.View()

	if !strings.Contains(view, i18n.T("login.codex_pool_model_classified")) {
		t.Errorf("view should show the model-classified row label; view=%q", view)
	}
	if flat := fmt.Sprintf(i18n.T("login.codex_pool_weight"), 3); strings.Contains(view, flat) {
		t.Errorf("view must not render the sibling flat weight the kernel ignores; view=%q", view)
	}
	if strings.Contains(view, i18n.T("login.codex_pool_not_member")) {
		t.Errorf("view must not show a flat not-in-pool label for a classified pool; view=%q", view)
	}
	note := fmt.Sprintf(i18n.T("login.codex_pool_model_classified_note"), 0)
	if !strings.Contains(view, note) {
		t.Errorf("view should show the classified-pool note with count 0; want %q in view=%q", note, view)
	}
	if strings.Contains(view, i18n.T("login.codex_pool_hint")) {
		t.Errorf("footer must not advertise the inert +/-/0 pool keys on a classified pool; view=%q", view)
	}

	// The in-memory flat map still carries the sibling entry (display is what
	// is gated on classification) — the inert keys must not mutate it.
	if w := m.poolWeights[acctPath]; w != 3 {
		t.Fatalf("precondition: sibling flat weight should load as 3; got %d", w)
	}
	for _, key := range []tea.KeyPressMsg{
		{Text: "+", Code: '+'},
		{Text: "-", Code: '-'},
		{Text: "0", Code: '0'},
	} {
		var cmd tea.Cmd
		m, cmd = m.Update(key)
		if cmd != nil {
			t.Fatalf("pool key %q on classified pool must not start a command", key.Text)
		}
		if m.message != note {
			t.Errorf("key %q: message = %q, want the classified-pool note %q", key.Text, m.message, note)
		}
		if w := m.poolWeights[acctPath]; w != 3 {
			t.Errorf("key %q: in-memory weights must stay untouched; weights=%v", key.Text, m.poolWeights)
		}
	}

	got, err := os.ReadFile(codexPoolPath(globalDir))
	if err != nil {
		t.Fatalf("read pool file back: %v", err)
	}
	if string(got) != string(raw) {
		t.Errorf("empty-models pool file must stay byte-identical after inert keys;\n got: %s\nwant: %s", got, raw)
	}
}

// TestLoginModel_MissingPoolFileNoWarn pins that the N5 warning is scoped to
// CORRUPT files: a missing pool file stays quiet (no warning, poolCorrupt false),
// preserving the existing no-pool UX.
func TestLoginModel_MissingPoolFileNoWarn(t *testing.T) {
	t.Setenv("LINGTAI_TUI_DIR", "")
	_, globalDir := withTempCodexHome(t)

	acctPath := filepath.Join(globalDir, codexAuthSubdir, "work.json")
	writeCodexTokenAt(t, acctPath, "work@example.com")

	m := NewLoginModel("", globalDir)
	if m.poolCorrupt {
		t.Fatal("a missing pool file must not be flagged corrupt")
	}

	m.width = 100
	view := m.View()
	if strings.Contains(view, i18n.T("login.codex_pool_corrupt")) {
		t.Errorf("no corrupt warning should show for a missing pool file; view=%q", view)
	}
}
