package fs

import (
	"crypto/sha256"
	"encoding/hex"
)

// ProjectHash returns the first 12 hex chars of SHA-256(projectPath).
// Used by registry.jsonl to give each project a stable filesystem-safe id.
func ProjectHash(projectPath string) string {
	sum := sha256.Sum256([]byte(projectPath))
	return hex.EncodeToString(sum[:])[:12]
}
