package tui

import (
	"embed"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/anthropics/lingtai-tui/i18n"
)

// helpDocs holds the embedded markdown reference pages: one overview.md plus
// one <command>.md per slash command. Bodies are English-only — they are
// procedural reference content, not localized UI strings.
//
//go:embed help/*.md
var helpDocs embed.FS

// helpMissingDoc is shown when a command has a palette entry but no embedded
// markdown page. The help_test.go test guards against this in practice, but a
// graceful fallback keeps the viewer usable if a doc is ever removed.
func helpMissingDoc(name string) string {
	return fmt.Sprintf("# /%s\n\n%s", name, i18n.T("help.missing_doc"))
}

// readHelpDoc returns the embedded markdown for the named doc (without the
// "help/" prefix or ".md" suffix), or "" if it is absent.
func readHelpDoc(stem string) string {
	data, err := helpDocs.ReadFile("help/" + stem + ".md")
	if err != nil {
		return ""
	}
	return string(data)
}

// buildHelpEntries assembles the markdown viewer entries for the /help view:
// an "Overview" group with the intro, then a "Commands" group with one entry
// per slash command in DefaultCommands() order. Each command entry carries the
// same short palette description the user already sees, and the embedded
// per-command markdown as content.
func buildHelpEntries() []MarkdownEntry {
	overviewGroup := i18n.T("help.group_overview")
	commandsGroup := i18n.T("help.group_commands")

	var entries []MarkdownEntry
	entries = append(entries, MarkdownEntry{
		Label:   i18n.T("help.overview_label"),
		Group:   overviewGroup,
		Content: readHelpDoc("overview"),
	})

	for _, cmd := range DefaultCommands() {
		content := readHelpDoc(cmd.Name)
		if content == "" {
			content = helpMissingDoc(cmd.Name)
		}
		entries = append(entries, MarkdownEntry{
			Label:       "/" + cmd.Name,
			Description: i18n.T(cmd.Description),
			Group:       commandsGroup,
			Content:     content,
		})
	}
	return entries
}

// HelpModel is the /help view. It is a thin wrapper around MarkdownViewerModel:
// an overview page plus a browsable per-command reference. Esc/q close the inner
// viewer, which emits MarkdownViewerCloseMsg — App.Update already routes that
// back to the mail view, so no extra close plumbing is needed here.
type HelpModel struct {
	inner MarkdownViewerModel
}

// NewHelpModel builds the help view with all command pages loaded from the
// embedded markdown.
func NewHelpModel() HelpModel {
	inner := NewMarkdownViewer(buildHelpEntries(), i18n.T("help.title"))
	return HelpModel{inner: inner}
}

func (m HelpModel) Init() tea.Cmd { return m.inner.Init() }

func (m HelpModel) Update(msg tea.Msg) (HelpModel, tea.Cmd) {
	var cmd tea.Cmd
	m.inner, cmd = m.inner.Update(msg)
	return m, cmd
}

func (m HelpModel) View() string { return m.inner.View() }
