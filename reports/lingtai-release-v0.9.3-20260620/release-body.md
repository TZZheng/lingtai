## LingTai TUI/Portal v0.9.3

This release ships the post-v0.9.2 TUI/portal update set from `v0.9.2..29d25fa`.

### Highlights
- Adds `/notification` history backed by the sqlite event log so older notification blocks can be inspected after the live notification surface changes.
- Fixes a migration version collision in the TUI metadata/state schema.
- Updates the notification block viewer to display persisted canonical notification-block snapshots from the kernel.
- Shows local timezone information in `/kanban` timestamps.
- Shows local timezone and CLI token usage in the daemons view.
- Adds runtime refresh verification lessons to the developer guide.

### Validation
- `git diff --check` passed.
- `tui: go test ./...` passed.
- `tui: make build` passed.
- `npm --prefix portal/web ci` completed.
- `npm --prefix portal/web run build` passed.
- `portal: go test ./...` passed.
- `portal: make build` passed.

Note: npm reported existing audit warnings (`4 vulnerabilities: 1 low, 2 moderate, 1 high`) and an `fsevents` allow-scripts warning; these did not block build/test gates and should be tracked separately if desired.
