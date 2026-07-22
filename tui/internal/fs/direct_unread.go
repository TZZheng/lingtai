package fs

// DirectUnreadStore is a compile-only RED scaffold for durable direct-thread
// unread state. It intentionally owns no functional state yet.
type DirectUnreadStore struct{}

// OpenDirectUnreadStore is a compile-only RED scaffold. Persistence and
// baselining deliberately do not exist yet.
func OpenDirectUnreadStore(projectDirectory, humanAddress string, targets []DirectTarget, accepted []MailMessage) (*DirectUnreadStore, error) {
	return &DirectUnreadStore{}, nil
}

// SyncTargets is a compile-only RED scaffold. It deliberately does nothing.
func (s *DirectUnreadStore) SyncTargets(targets []DirectTarget, accepted []MailMessage) error {
	return nil
}

// UnreadCount is a compile-only RED scaffold. It deliberately reports no unread
// messages until the durable-state implementation is added.
func (s *DirectUnreadStore) UnreadCount(target DirectTarget, accepted []MailMessage) (int, error) {
	return 0, nil
}

// MarkSeen is a compile-only RED scaffold. It deliberately does nothing.
func (s *DirectUnreadStore) MarkSeen(target DirectTarget, accepted []MailMessage) error {
	return nil
}
