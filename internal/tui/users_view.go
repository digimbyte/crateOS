package tui

import (
	"fmt"
)

func (m model) viewUsers() string {
	if m.userFormOpen {
		return renderSplitView(m, "Users", renderUserSelectionPanel(m), renderUserFocusPanel(m), "  tab / ↑↓ cycle fields · [enter] save user form · [esc] cancel form")
	}
	return renderSplitView(m, "Users", renderUserSelectionPanel(m), renderUserFocusPanel(m), "  ↑↓ select user · [a] add · [enter] edit · [r] cycle role · [p] perms preset · [s] set current · [d] delete · [esc] back · [:] command")
}

func renderUserSelectionPanel(m model) string {
	lines := []string{}
	if len(m.users) == 0 {
		lines = append(lines, dim.Render("  No users found."))
		return renderSelectionPanel("USER DIRECTORY", 34, lines)
	}
	for i, u := range m.users {
		cursor := m.currentUserIndex()
		marker := ""
		if u.Name == m.currentUser {
			marker = "  " + ok.Render("cur")
		}
		line := fmt.Sprintf("%s %s  %s%s", userRoleIndicator(u.Role), compactLabel(u.Name, 12), selectorStat("r", compactRole(u.Role)), marker)
		lines = append(lines, renderSelectorLineWithGlyph(i == cursor && !m.userFormOpen, userRailGlyph(u), line))
	}
	return renderSelectionPanelWithMeta("USER DIRECTORY", "OPERATORS", 34, lines)
}

func (m model) currentUserIndex() int {
	if len(m.users) == 0 {
		return 0
	}
	cursor := m.cursor
	if cursor < 0 {
		return 0
	}
	if cursor >= len(m.users) {
		return len(m.users) - 1
	}
	return cursor
}
