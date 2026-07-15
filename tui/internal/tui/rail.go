package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/anthropics/lingtai-tui/i18n"
)

// railRow is display-only rail state. It intentionally owns no MailModel,
// ProjectMailStore, MailCache, scanner, tick, or thread projection.
type railRow struct {
	label        string
	originalMain bool
}

// AgentRailState is the root-owned display projection for the home Agent rail.
// Inventory-backed ordinary rows are installed into this projection separately;
// rendering never scans the filesystem or process table.
type AgentRailState struct {
	rows []railRow
}

type mailPaneFocus uint8

const (
	mailFocusChat mailPaneFocus = iota
	mailFocusRail
)

func (a App) updateMailChildWindowSize(size tea.WindowSizeMsg) (App, tea.Cmd) {
	var focusCmd tea.Cmd
	if a.currentView == appViewMail && !a.layoutBudget().RailVisible && a.mailFocus == mailFocusRail {
		focusCmd = a.focusMailChat()
	}
	updated, sizeCmd := a.updateChildWindowSize(size)
	return updated, tea.Batch(focusCmd, sizeCmd)
}

func (a *App) focusMailChat() tea.Cmd {
	if a == nil {
		return nil
	}
	a.mailFocus = mailFocusChat
	return a.mail.input.Focus()
}

func (a *App) focusMailRail() {
	if a == nil {
		return
	}
	a.mailFocus = mailFocusRail
	a.mail.input.Blur()
}

func (a App) handleMailFocusKey(msg tea.KeyPressMsg) (App, tea.Cmd, bool) {
	if a.currentView != appViewMail {
		return a, nil, false
	}

	key := msg.String()
	if a.mail.copyMode {
		return a, nil, false
	}
	if key == "tab" || key == "shift+tab" {
		if !a.layoutBudget().RailVisible {
			cmd := a.focusMailChat()
			return a, cmd, true
		}
		if a.mailFocus == mailFocusRail {
			cmd := a.focusMailChat()
			return a, cmd, true
		}
		a.focusMailRail()
		return a, nil, true
	}

	if key == "esc" && a.mailFocus == mailFocusRail && !a.mail.copyMode {
		cmd := a.focusMailChat()
		return a, cmd, true
	}

	if a.mailFocus == mailFocusRail && key != copyModeToggleKey {
		return a, nil, true
	}
	return a, nil, false
}

func (a App) handleMailMouseClick(msg tea.MouseClickMsg) (App, tea.Cmd, bool) {
	if a.currentView != appViewMail {
		return a, nil, false
	}

	budget := a.layoutBudget()
	if msg.Y < budget.TopChromeRows || msg.Y >= budget.TopChromeRows+budget.ChildHeight {
		return a, nil, true
	}
	if localX, ok := budget.ContentLocalX(msg.X); ok {
		focusCmd := a.focusMailChat()
		mouse := msg.Mouse()
		mouse.X = localX
		mouse.Y -= budget.TopChromeRows
		updated, mailCmd := a.mail.Update(tea.MouseClickMsg(mouse))
		a.mail = updated
		return a, tea.Batch(focusCmd, mailCmd), true
	}
	if _, ok := budget.RailLocalX(msg.X); ok {
		a.focusMailRail()
		return a, nil, true
	}
	return a, nil, true
}

func (s *AgentRailState) installMain(label string) {
	if s == nil {
		return
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = "Main"
	}
	for i := range s.rows {
		if s.rows[i].originalMain {
			s.rows[i].label = label
			return
		}
	}
	s.rows = append([]railRow{{label: label, originalMain: true}}, s.rows...)
}

func (s AgentRailState) rowsForView(mainLabel string) []railRow {
	rows := append([]railRow(nil), s.rows...)
	for i := range rows {
		if rows[i].originalMain {
			if current := strings.TrimSpace(mainLabel); current != "" {
				rows[i].label = current
			}
			return rows
		}
	}
	fallback := AgentRailState{}
	fallback.installMain(mainLabel)
	return append(fallback.rows, rows...)
}

func fixedRailLine(text string, width int) string {
	if width <= 0 {
		return ""
	}
	text = ansi.Truncate(text, width, "…")
	return lipgloss.NewStyle().Width(width).Render(text)
}

// View renders exactly the rectangle supplied by the root LayoutBudget. It does
// not choose a width or height of its own.
func (s AgentRailState) View(width, height int, mainLabel string) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	logical := []string{
		StyleTitle.Render("  " + i18n.T("props.network_agents")),
		"",
	}
	for _, row := range s.rowsForView(mainLabel) {
		marker := StyleAccent.Render("› ")
		logical = append(logical, marker+StyleTitle.Render(row.label))
	}

	lines := make([]string, height)
	for i := range lines {
		if i < len(logical) {
			lines[i] = fixedRailLine(logical[i], width)
		} else {
			lines[i] = fixedRailLine("", width)
		}
	}
	return strings.Join(lines, "\n")
}

func fixedMailBlock(content string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(content)
}

// composeMailWithRail consumes the same root budget used for child sizing. The
// horizontal join happens before existing root-owned vertical chrome.
func (a App) composeMailWithRail(mailContent string) string {
	budget := a.layoutBudget()
	if !budget.RailVisible {
		return mailContent
	}

	rail := a.agentRail.View(budget.RailWidth, budget.ChildHeight, a.mail.orchDisplayName())
	chat := fixedMailBlock(mailContent, budget.ContentWidth, budget.ChildHeight)
	return lipgloss.JoinHorizontal(lipgloss.Top, rail, chat)
}
