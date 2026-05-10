package tui

// UsePresetMsg is emitted when a preset is selected for use.
type UsePresetMsg struct {
	Name string
}

// AllCapabilities is the list of all available capability names.
// email and psyche are kernel intrinsics (always loaded), not capabilities.
// codex and library are mandatory — always injected at save time, not shown as toggleable.
var AllCapabilities = []string{
	"file", "bash", "web_search",
	"vision",
	"avatar", "daemon",
}

// AllAddons is the list of available addon names.
var AllAddons = []string{"imap", "telegram", "feishu", "wechat"}
