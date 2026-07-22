package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// DirectTarget identifies one inventory target without coupling target identity
// to unread persistence or any future async acceptance store. Directory is the
// caller's canonical working directory; Address is the target's network address.
type DirectTarget struct {
	Directory string
	Address   string
}

// AddressFingerprint returns a stable, nickname-independent identity for an
// address. Whitespace surrounding a serialized address is not identity.
func AddressFingerprint(address string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(address)))
	return hex.EncodeToString(sum[:])
}

// NormalizeMailEndpoints converts the mailbox schema's string-or-list endpoint
// value into one trimmed, order-preserving, deduplicated list. CC is deliberately
// not part of this input.
func NormalizeMailEndpoints(value interface{}) []string {
	var raw []string
	switch endpoints := value.(type) {
	case string:
		raw = []string{endpoints}
	case []string:
		raw = endpoints
	case []interface{}:
		raw = make([]string, 0, len(endpoints))
		for _, endpoint := range endpoints {
			if text, ok := endpoint.(string); ok {
				raw = append(raw, text)
			}
		}
	default:
		return nil
	}

	seen := make(map[string]struct{}, len(raw))
	result := make([]string, 0, len(raw))
	for _, endpoint := range raw {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" {
			continue
		}
		if _, exists := seen[endpoint]; exists {
			continue
		}
		seen[endpoint] = struct{}{}
		result = append(result, endpoint)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// IsDirectMail reports whether msg belongs to the strict human-target thread.
// Direct mail has exactly one primary recipient and no CC participants; group or
// copied mail must not leak into either endpoint's one-to-one transcript.
func IsDirectMail(msg MailMessage, humanAddress, targetAddress string) bool {
	humanAddress = strings.TrimSpace(humanAddress)
	targetAddress = strings.TrimSpace(targetAddress)
	from := strings.TrimSpace(msg.From)
	if humanAddress == "" || targetAddress == "" || len(msg.CC) != 0 {
		return false
	}
	to := NormalizeMailEndpoints(msg.To)
	if len(to) != 1 {
		return false
	}
	if from == humanAddress {
		return to[0] == targetAddress
	}
	return from == targetAddress && to[0] == humanAddress
}
