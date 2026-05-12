package migrate

// migrateSessionBackfill is a no-op since v0.9.x: it was added to backfill
// the secretary's hourly session-history dumps at ~/.lingtai-tui/brief/
// projects/<hash>/history/. The secretary agent has been removed; the
// version slot is preserved so existing meta.json files don't trip the
// "newer than this binary supports" guard.
func migrateSessionBackfill(_ string) error {
	return nil
}
