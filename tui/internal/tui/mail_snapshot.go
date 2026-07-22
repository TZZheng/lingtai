package tui

import "github.com/anthropics/lingtai-tui/internal/fs"

// acceptedMailSnapshot is Main MailModel's private publication boundary.
// Producers keep refreshing MailModel.cache, but render and older-history
// consumers see a deeply detached cache only after the refresh completion has
// passed MailModel's existing generation gate.
type acceptedMailSnapshot struct {
	cache fs.MailCache
	ready bool
}

func newAcceptedMailSnapshot(candidate fs.MailCache) acceptedMailSnapshot {
	return acceptedMailSnapshot{cache: candidate.Clone(), ready: true}
}

func (s acceptedMailSnapshot) cacheCopy(humanDir string) fs.MailCache {
	if !s.ready {
		return fs.NewMailCache(humanDir)
	}
	return s.cache.Clone()
}

// messagesForUnread is a compile-only RED scaffold. It intentionally exposes
// no accepted messages until the detached unread accessor is implemented.
func (s acceptedMailSnapshot) messagesForUnread(humanDir string) []fs.MailMessage {
	return nil
}
