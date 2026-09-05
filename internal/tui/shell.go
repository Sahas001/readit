package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sahas/readit/internal/sanitize"
)

// shellDimensions calculates the width, height, and content area dimensions
// for the centered application canvas based on terminal size.
func (m *Model) shellDimensions() (shellWidth, shellHeight, contentWidth, contentHeight int) {
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	// Calculate shell width: bounded between 108 columns max and responsive minimums
	switch {
	case w >= 120:
		shellWidth = min(108, w-8)
	case w >= 84:
		shellWidth = w - 4
	default:
		shellWidth = max(24, w-2)
	}

	// Calculate shell height: bounded between 42 lines max and responsive minimums
	switch {
	case h >= 32:
		shellHeight = min(42, h-2)
	case h >= 22:
		shellHeight = h - 1
	default:
		shellHeight = max(12, h)
	}

	// Content area inside the shell
	contentWidth = shellWidth
	// Header takes 2 lines (header + rule), Footer takes 2 lines (rule + status bar)
	headerLines := 2
	footerLines := 2
	contentHeight = max(4, shellHeight-headerLines-footerLines)

	return shellWidth, shellHeight, contentWidth, contentHeight
}

// renderAppShell wraps the content with a cohesive header and status bar,
// and centers the resulting application canvas on the terminal.
func (m *Model) renderAppShell(contextTitle string, content string, shortcuts [][2]string) string {
	shellWidth, _, _, contentHeight := m.shellDimensions()

	// 1. Build Header
	header := m.renderHeader(contextTitle, shellWidth)
	rule := styleRule.Render(strings.Repeat("─", shellWidth))
	topHeader := header + "\n" + rule

	// 2. Build Footer / Status Bar
	statusBar := m.renderFooter(contextTitle, shortcuts, shellWidth)
	bottomFooter := rule + "\n" + statusBar

	// 3. Measure & Vertically Pad Content to maintain stable layout
	content = strings.TrimRight(content, "\n")
	actualContentH := 0
	if content != "" {
		actualContentH = lipgloss.Height(content)
	}

	if actualContentH < contentHeight {
		padding := strings.Repeat("\n", contentHeight-actualContentH)
		content = content + padding
	}

	shellBox := topHeader + "\n" + content + "\n" + bottomFooter

	// 4. Center on Terminal Canvas
	if m.width <= 0 || m.height <= 0 {
		return shellBox
	}

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		shellBox,
	)
}

// renderHeader renders the top application title bar.
func (m *Model) renderHeader(contextTitle string, targetWidth int) string {
	logo := styleLogoBadge.Render("ReadIT")

	var titlePart string
	if contextTitle != "" && contextTitle != "ReadIT" {
		titlePart = styleTitle.Render(contextTitle)
	} else {
		titlePart = styleTagline.Render("The Terminal Forum")
	}

	left := logo + "  " + titlePart

	// Right side: user handle + online indicator
	var userHandle string
	if m.user != nil {
		userHandle = sanitize.SingleLine(m.user.Handle)
	} else {
		userHandle = "guest"
	}
	dot := styleStatusDot.Render("●")
	right := styleMeta.Render("@"+userHandle) + " " + dot

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)

	if targetWidth < leftW+rightW+4 {
		// Truncate or simplify on narrow widths
		right = styleMeta.Render("@" + userHandle)
		rightW = lipgloss.Width(right)
		if targetWidth < leftW+rightW+2 {
			return left
		}
	}

	spaces := max(1, targetWidth-leftW-rightW)
	return left + strings.Repeat(" ", spaces) + right
}

// renderFooter renders the bottom status bar with centered navigation pills and notices.
func (m *Model) renderFooter(_ string, shortcuts [][2]string, targetWidth int) string {
	if targetWidth <= 0 {
		return ""
	}

	// Center: key shortcuts or flash notice
	var centerStr string
	if m.flashMsg != "" {
		centerStr = styleStatusFlash.Render(m.flashMsg)
		if lipgloss.Width(centerStr) > targetWidth {
			centerStr = lipgloss.NewStyle().MaxWidth(targetWidth).Render(centerStr)
		}
	} else {
		centerStr = formatAdaptiveKeyPills(shortcuts, targetWidth)
	}

	return lipgloss.PlaceHorizontal(targetWidth, lipgloss.Center, centerStr)
}

// formatAdaptiveKeyPills renders key shortcut pills that fit within maxAllowedWidth
// without ever clipping or truncating a pill mid-text (preventing issues like "[q] qu").
func formatAdaptiveKeyPills(shortcuts [][2]string, maxAllowedWidth int) string {
	if len(shortcuts) == 0 || maxAllowedWidth <= 0 {
		return ""
	}

	// 1. Try rendering all shortcuts
	full := formatKeyPills(shortcuts)
	if lipgloss.Width(full) <= maxAllowedWidth {
		return full
	}

	// 2. Progressive reduction strategy:
	// Stage A: Omit secondary jump keys ("g/G")
	filteredA := make([][2]string, 0, len(shortcuts))
	for _, s := range shortcuts {
		if s[0] != "g/G" {
			filteredA = append(filteredA, s)
		}
	}
	fStrA := formatKeyPills(filteredA)
	if lipgloss.Width(fStrA) <= maxAllowedWidth {
		return fStrA
	}

	// Stage B: Omit vote keys ("u/d")
	filteredB := make([][2]string, 0, len(filteredA))
	for _, s := range filteredA {
		if s[0] != "u/d" {
			filteredB = append(filteredB, s)
		}
	}
	fStrB := formatKeyPills(filteredB)
	if lipgloss.Width(fStrB) <= maxAllowedWidth {
		return fStrB
	}

	// Stage C: Iteratively drop middle items, preserving the final action (e.g. [q] quit)
	curr := filteredB
	for len(curr) > 2 {
		// Drop the item right before the last one (keep first and last)
		dropIdx := len(curr) - 2
		curr = append(curr[:dropIdx], curr[dropIdx+1:]...)
		cStr := formatKeyPills(curr)
		if lipgloss.Width(cStr) <= maxAllowedWidth {
			return cStr
		}
	}

	// Stage D: Only the final critical shortcut (e.g. [q] quit)
	if len(shortcuts) > 0 {
		last := formatKeyPills([][2]string{shortcuts[len(shortcuts)-1]})
		if lipgloss.Width(last) <= maxAllowedWidth {
			return last
		}
	}

	return ""
}
