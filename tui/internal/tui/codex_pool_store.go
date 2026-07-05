package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Codex pool (load-balancing) storage.
//
// The kernel's `codex-pool` provider reads a NON-SECRET pool file that lists the
// Codex OAuth token files to load-balance across, each with an integer weight.
// The pool file is the single source of truth for load balancing — presets do
// NOT encode weights, and enabling the pool never rewrites saved presets. This
// file owns loading/saving that pool file; the token files themselves stay put
// (legacy ~/.lingtai-tui/codex-auth.json and per-account
// ~/.lingtai-tui/codex-auth/<slug>.json) and are never read here.
//
// Contract (must match the kernel):
//
//	Default path:
//	  $LINGTAI_TUI_DIR/codex-auth-pool.json  when LINGTAI_TUI_DIR is set;
//	  otherwise ~/.lingtai-tui/codex-auth-pool.json.
//
//	Schema:
//	  {"version": 1, "accounts": [{"path": "codex-auth.json", "weight": 1}, ...]}
//
//	Rules:
//	  - `path` is TUI-dir-relative for token files under the TUI dir; the legacy
//	    file serializes as "codex-auth.json", per-account as
//	    "codex-auth/<slug>.json". Files outside the TUI dir fall back to a
//	    "~/"-prefixed or absolute ref.
//	  - Store only paths/refs and integer weights — NEVER token contents.
//	  - Weight 0 means the account is disabled (present but not balanced onto).
//
// Nothing here reads, logs, or writes token material.

// codexPoolFileName is the non-secret pool file's basename, shared with the
// kernel's reader.
const codexPoolFileName = "codex-auth-pool.json"

// codexPoolVersion is the schema version written into the pool file.
const codexPoolVersion = 1

// codexPoolAccount is one balanced account: a stable ref to its token file plus
// an integer weight. Weight 0 disables the account without dropping it.
type codexPoolAccount struct {
	Path   string `json:"path"`
	Weight int    `json:"weight"`
}

// codexPool is the on-disk pool file shape.
type codexPool struct {
	Version  int                `json:"version"`
	Accounts []codexPoolAccount `json:"accounts"`
}

// codexPoolPath returns the absolute path of the pool file. LINGTAI_TUI_DIR
// wins when set (matching the kernel reader); otherwise the file lives directly
// under globalDir (~/.lingtai-tui). globalDir is only consulted as the fallback
// base, so the two readers agree on the location.
func codexPoolPath(globalDir string) string {
	if base := strings.TrimSpace(os.Getenv("LINGTAI_TUI_DIR")); base != "" {
		return filepath.Join(base, codexPoolFileName)
	}
	return filepath.Join(globalDir, codexPoolFileName)
}

// codexPoolBaseDir returns the directory the pool file lives in — the same base
// that relative `path` entries resolve against. LINGTAI_TUI_DIR wins when set,
// mirroring codexPoolPath, so refs written here round-trip through the kernel.
func codexPoolBaseDir(globalDir string) string {
	if base := strings.TrimSpace(os.Getenv("LINGTAI_TUI_DIR")); base != "" {
		return base
	}
	return globalDir
}

// codexPoolRefForPath maps an absolute token-file path to the stable ref stored
// in the pool file. Unlike codexAuthRefForPath (which maps the legacy file to ""
// to preserve preset fallback semantics), the pool wants an EXPLICIT, stable
// relative ref for every account:
//   - a token file under the TUI dir → its TUI-dir-relative path
//     ("codex-auth.json" for the legacy file, "codex-auth/<slug>.json" per-account);
//   - a file under the user's home but outside the TUI dir → "~/"-prefixed;
//   - anything else → the absolute path unchanged.
func codexPoolRefForPath(globalDir, absPath string) string {
	if absPath == "" {
		return ""
	}
	base := codexPoolBaseDir(globalDir)
	if rel, err := filepath.Rel(base, absPath); err == nil && rel != "" &&
		!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
		return filepath.ToSlash(rel)
	}
	if home, err := os.UserHomeDir(); err == nil {
		if rel, err := filepath.Rel(home, absPath); err == nil && !strings.HasPrefix(rel, "..") {
			return "~/" + filepath.ToSlash(rel)
		}
	}
	return absPath
}

// resolveCodexPoolRef expands a pool `path` entry to an absolute path — the
// inverse of codexPoolRefForPath. A bare relative ref resolves under the pool
// base dir (so "codex-auth.json" lands on the legacy file); "~/" and absolute
// refs are honored. Empty refs resolve to "" so callers can skip them.
func resolveCodexPoolRef(globalDir, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if ref == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return ref
	}
	if strings.HasPrefix(ref, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ref[2:])
		}
		return ref
	}
	if filepath.IsAbs(ref) {
		return ref
	}
	return filepath.Join(codexPoolBaseDir(globalDir), ref)
}

// loadCodexPool reads the pool file. A missing file is NOT an error: it returns
// an empty pool (Version defaulted, no accounts) so callers can treat "no pool
// yet" and "empty pool" identically. A malformed file returns the parse error so
// the caller can surface it rather than silently clobbering the user's edits.
func loadCodexPool(globalDir string) (codexPool, error) {
	path := codexPoolPath(globalDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return codexPool{Version: codexPoolVersion}, nil
		}
		return codexPool{}, err
	}
	var pool codexPool
	if err := json.Unmarshal(data, &pool); err != nil {
		return codexPool{}, err
	}
	if pool.Version == 0 {
		pool.Version = codexPoolVersion
	}
	return pool, nil
}

// saveCodexPool writes the pool file (version stamped, parent created). The file
// is non-secret — it holds only refs and weights — so it is written 0644, unlike
// the 0600 token files. Callers build the accounts list from
// codexPoolRefForPath so only stable relative refs land on disk.
func saveCodexPool(globalDir string, pool codexPool) error {
	pool.Version = codexPoolVersion
	if pool.Accounts == nil {
		pool.Accounts = []codexPoolAccount{}
	}
	path := codexPoolPath(globalDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(pool, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// codexPoolWeights returns a map from resolved ABSOLUTE token-file path to the
// weight recorded in the pool file. Entries whose ref can't be resolved are
// skipped. Used by the credentials UI to look up each Codex account row's weight
// without re-parsing the pool per row. A missing pool file yields an empty map
// (callers apply the default-weight policy on top).
func codexPoolWeights(globalDir string) map[string]int {
	pool, err := loadCodexPool(globalDir)
	if err != nil {
		return map[string]int{}
	}
	out := make(map[string]int, len(pool.Accounts))
	for _, acct := range pool.Accounts {
		abs := resolveCodexPoolRef(globalDir, acct.Path)
		if abs == "" {
			continue
		}
		out[abs] = acct.Weight
	}
	return out
}

// codexPoolFileCorrupt reports whether the pool file exists but fails to parse.
// A missing file is NOT corrupt (returns false) — that is the normal "no pool
// yet" state. Used by the credentials UI to warn the user that a malformed pool
// file is being ignored for display (and won't be overwritten), instead of
// silently rendering every account as "not in pool".
func codexPoolFileCorrupt(globalDir string) bool {
	if _, err := os.Stat(codexPoolPath(globalDir)); err != nil {
		return false // missing (or unstattable) — not a parse-corruption case
	}
	_, err := loadCodexPool(globalDir)
	return err != nil
}

// codexPoolMembership reports an account's real pool state for absPath: inPool
// is true only when the pool file actually records the account, and weight is
// its stored weight (meaningful only when inPool). An account absent from the
// pool file is NOT in the pool — the UI must say so rather than inventing a
// default weight, because the kernel runtime has no such entry either and would
// fall back to the legacy single token. This truthfulness is the whole point of
// GLM's display/runtime-mismatch fix: never render a phantom "pool weight: 1"
// for an account the pool doesn't contain.
func codexPoolMembership(weights map[string]int, absPath string) (inPool bool, weight int) {
	w, ok := weights[absPath]
	return ok, w
}

// setCodexPoolWeight records weight for the token file at absPath and persists
// the pool file, creating it on first edit (the lazy-write policy). The account
// is added if absent, updated in place if present. Other accounts and their
// weights are preserved in their original order.
//
// Entries are matched by RESOLVED ABSOLUTE token path, not by ref string, so a
// weight edit lands on the right account even when the stored ref used a
// different style (relative vs "~/"/absolute) — e.g. because LINGTAI_TUI_DIR
// changed between edits. To stay robust against a pool that already contains
// more than one entry resolving to the same token (a duplicate produced before
// this fix), the save collapses all such entries into a single account: the
// first occurrence keeps its slot but adopts the current output ref style
// (codexPoolRefForPath) and the new weight; any later duplicates are dropped.
// The result always holds at most one account per resolved absolute token.
func setCodexPoolWeight(globalDir, absPath string, weight int) error {
	if weight < 0 {
		weight = 0
	}
	pool, err := loadCodexPool(globalDir)
	if err != nil {
		// A malformed pool file must not be silently overwritten — surface it.
		return err
	}

	ref := codexPoolRefForPath(globalDir, absPath)
	deduped := make([]codexPoolAccount, 0, len(pool.Accounts))
	applied := false
	for _, acct := range pool.Accounts {
		if resolveCodexPoolRef(globalDir, acct.Path) == absPath {
			if applied {
				// A second (or later) entry for the same token — drop it so the
				// pool holds a single account for this abs path.
				continue
			}
			// First match: normalize its ref style and set the new weight.
			deduped = append(deduped, codexPoolAccount{Path: ref, Weight: weight})
			applied = true
			continue
		}
		deduped = append(deduped, acct)
	}
	if !applied {
		deduped = append(deduped, codexPoolAccount{Path: ref, Weight: weight})
	}
	pool.Accounts = deduped
	return saveCodexPool(globalDir, pool)
}
