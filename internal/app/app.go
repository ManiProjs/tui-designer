package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/maniprojs/tui-designer/internal/model"
)

type Panel int

const (
	PanelPalette Panel = iota
	PanelCanvas
	PanelHierarchy
	PanelInspector
)

var paletteWidgets = []struct {
	Type string
	Name string
}{
	{Type: "text", Name: "Text"},
	{Type: "button", Name: "Button"},
	{Type: "input", Name: "Input"},
	{Type: "panel", Name: "Panel"},
}

var (
	colorPrimary    = lipgloss.Color("#8B5CF6")
	colorAccent     = lipgloss.Color("#22D3EE")
	colorText       = lipgloss.Color("#E5E7EB")
	colorMuted      = lipgloss.Color("#6B7280")
	colorBorder     = lipgloss.Color("#374151")
	colorActive     = lipgloss.Color("#A78BFA")
	colorDanger     = lipgloss.Color("#F87171")
	colorBackground = lipgloss.Color("#111827")
)

var (
	appStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorBackground)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorActive)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	dangerStyle = lipgloss.NewStyle().
			Foreground(colorDanger)
)

type Model struct {
	document *model.Document

	panel Panel

	paletteCursor   int
	hierarchyCursor int

	selected *model.Node

	width  int
	height int

	status string
}

func New() Model {
	document := model.NewDocument()

	var selected *model.Node

	if document.Root != nil && len(document.Root.Children) > 0 {
		selected = document.Root.Children[0]
	}

	return Model{
		document: document,
		panel:    PanelPalette,
		selected: selected,
		status:   "Ready",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "q":
			return m, tea.Quit

		case "tab":
			m.nextPanel()

		case "shift+tab":
			m.previousPanel()

		case "esc":
			m.panel = PanelCanvas
			m.status = "Canvas focused"

		case "up", "k":
			m.moveCursor(-1)

		case "down", "j":
			m.moveCursor(1)

		case "enter":
			m.activate()

		case "a":
			m.addSelectedWidget()

		case "d", "delete":
			m.deleteSelectedWidget()

		case "ctrl+s":
			m.status = "Project saved"
		}
	}

	return m, nil
}

func (m *Model) nextPanel() {
	m.panel++

	if m.panel > PanelInspector {
		m.panel = PanelPalette
	}

	m.status = m.panelName() + " focused"
}

func (m *Model) previousPanel() {
	m.panel--

	if m.panel < PanelPalette {
		m.panel = PanelInspector
	}

	m.status = m.panelName() + " focused"
}

func (m Model) panelName() string {
	switch m.panel {
	case PanelPalette:
		return "Palette"
	case PanelCanvas:
		return "Canvas"
	case PanelHierarchy:
		return "Hierarchy"
	case PanelInspector:
		return "Inspector"
	default:
		return "Unknown"
	}
}

func (m *Model) moveCursor(delta int) {
	switch m.panel {
	case PanelPalette:
		m.paletteCursor += delta

		if m.paletteCursor < 0 {
			m.paletteCursor = 0
		}

		if m.paletteCursor >= len(paletteWidgets) {
			m.paletteCursor = len(paletteWidgets) - 1
		}

	case PanelHierarchy:
		count := len(m.document.Root.Children)

		if count == 0 {
			return
		}

		m.hierarchyCursor += delta

		if m.hierarchyCursor < 0 {
			m.hierarchyCursor = 0
		}

		if m.hierarchyCursor >= count {
			m.hierarchyCursor = count - 1
		}
	}
}

func (m *Model) activate() {
	switch m.panel {
	case PanelPalette:
		m.addSelectedWidget()

	case PanelHierarchy:
		if len(m.document.Root.Children) == 0 {
			return
		}

		m.selected = m.document.Root.Children[m.hierarchyCursor]
		m.status = "Selected " + m.selected.ID

	case PanelCanvas:
		m.status = "Canvas"

	case PanelInspector:
		m.status = "Inspector"
	}
}

func (m *Model) addSelectedWidget() {
	if m.panel != PanelPalette {
		return
	}

	widget := paletteWidgets[m.paletteCursor]

	node := &model.Node{
		ID: fmt.Sprintf(
			"%s_%d",
			widget.Type,
			len(m.document.Root.Children)+1,
		),
		Type: widget.Type,
		Properties: map[string]string{
			"text": widget.Name,
		},
	}

	m.document.Root.Children = append(
		m.document.Root.Children,
		node,
	)

	m.hierarchyCursor = len(m.document.Root.Children) - 1
	m.selected = node

	m.status = "Added " + node.ID
}

func (m *Model) deleteSelectedWidget() {
	if m.selected == nil {
		m.status = "Nothing selected"
		return
	}

	children := m.document.Root.Children

	for i, node := range children {
		if node != m.selected {
			continue
		}

		deletedID := node.ID

		m.document.Root.Children = append(
			children[:i],
			children[i+1:]...,
		)

		if len(m.document.Root.Children) == 0 {
			m.selected = nil
			m.hierarchyCursor = 0
			m.status = "Deleted " + deletedID
			return
		}

		if i >= len(m.document.Root.Children) {
			i = len(m.document.Root.Children) - 1
		}

		m.hierarchyCursor = i
		m.selected = m.document.Root.Children[i]

		m.status = "Deleted " + deletedID
		return
	}
}

func (m Model) View() tea.View {
	view := tea.NewView(m.render())
	view.AltScreen = true

	return view
}

func (m Model) render() string {
	if m.width < 60 || m.height < 15 {
		return fmt.Sprintf(
			"TUI Designer\n\nTerminal too small.\n\nCurrent: %d × %d\nMinimum: 60 × 15",
			m.width,
			m.height,
		)
	}

	/*
		The terminal is divided into:

		1 row     header
		1 row     status
		remaining editor
		1 row     hierarchy
		1 row     footer

		Everything is calculated from the actual terminal size.
	*/

	headerHeight := 1
	statusHeight := 1
	hierarchyHeight := 5
	footerHeight := 1

	editorHeight :=
		m.height -
			headerHeight -
			statusHeight -
			hierarchyHeight -
			footerHeight

	if editorHeight < 5 {
		editorHeight = 5
	}

	paletteWidth := 18
	inspectorWidth := 26

	canvasWidth :=
		m.width -
			paletteWidth -
			inspectorWidth

	if canvasWidth < 10 {
		canvasWidth = 10
	}

	header := m.renderHeader(
		m.width,
		headerHeight,
	)

	status := m.renderStatus(
		m.width,
		statusHeight,
	)

	editor := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderPalette(
			paletteWidth,
			editorHeight,
		),
		m.renderCanvas(
			canvasWidth,
			editorHeight,
		),
		m.renderInspector(
			inspectorWidth,
			editorHeight,
		),
	)

	hierarchy := m.renderHierarchy(
		m.width,
		hierarchyHeight,
	)

	footer := m.renderFooter(
		m.width,
		footerHeight,
	)

	return appStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			status,
			editor,
			hierarchy,
			footer,
		),
	)
}

func (m Model) renderHeader(width, height int) string {
	left := titleStyle.Render("TUI Designer")

	right := mutedStyle.Render(
		"Go • Visual TUI editor",
	)

	gap := width -
		lipgloss.Width(left) -
		lipgloss.Width(right)

	if gap < 1 {
		gap = 1
	}

	content :=
		left +
			strings.Repeat(" ", gap) +
			right

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		PaddingLeft(1).
		PaddingRight(1).
		Render(content)
}

func (m Model) renderStatus(width, height int) string {
	selected := "Nothing selected"

	if m.selected != nil {
		selected = "Selected: " + m.selected.ID
	}

	content := mutedStyle.Render(
		"  " + m.status + "  •  " + selected,
	)

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(content)
}

func (m Model) panelStyle(
	width int,
	height int,
	panel Panel,
) lipgloss.Style {
	border := colorBorder

	if m.panel == panel {
		border = colorPrimary
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1)
}

func (m Model) renderPalette(width, height int) string {
	lines := []string{
		titleStyle.Render("PALETTE"),
		"",
	}

	for i, widget := range paletteWidgets {
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(colorText)

		if i == m.paletteCursor {
			cursor = "› "

			if m.panel == PanelPalette {
				style = selectedStyle
			}
		}

		lines = append(
			lines,
			style.Render(cursor+widget.Name),
		)
	}

	lines = append(
		lines,
		"",
		mutedStyle.Render("Enter / A  Add"),
	)

	return m.panelStyle(
		width,
		height,
		PanelPalette,
	).Render(strings.Join(lines, "\n"))
}

func (m Model) renderCanvas(width, height int) string {
	lines := []string{
		titleStyle.Render("CANVAS"),
		"",
	}

	if len(m.document.Root.Children) == 0 {
		lines = append(
			lines,
			mutedStyle.Render("Nothing here yet."),
			"",
			mutedStyle.Render("Select a widget"),
			mutedStyle.Render("and press Enter."),
		)
	} else {
		for _, node := range m.document.Root.Children {
			text := node.Properties["text"]

			switch node.Type {
			case "button":
				text = "[ " + text + " ]"

			case "input":
				text = "[ " + text + " ]"

			case "panel":
				text = "┌─ " + text + " ─┐"
			}

			if node == m.selected {
				lines = append(
					lines,
					selectedStyle.Render("▶ "+text),
				)
			} else {
				lines = append(
					lines,
					"  "+text,
				)
			}
		}
	}

	return m.panelStyle(
		width,
		height,
		PanelCanvas,
	).Render(strings.Join(lines, "\n"))
}

func (m Model) renderInspector(width, height int) string {
	lines := []string{
		titleStyle.Render("INSPECTOR"),
		"",
	}

	if m.selected == nil {
		lines = append(
			lines,
			mutedStyle.Render("Nothing selected."),
		)

		return m.panelStyle(
			width,
			height,
			PanelInspector,
		).Render(strings.Join(lines, "\n"))
	}

	lines = append(
		lines,
		mutedStyle.Render("Type"),
		selectedStyle.Render(m.selected.Type),
		"",
		mutedStyle.Render("ID"),
		m.selected.ID,
	)

	for key, value := range m.selected.Properties {
		lines = append(
			lines,
			"",
			mutedStyle.Render(key),
			value,
		)
	}

	return m.panelStyle(
		width,
		height,
		PanelInspector,
	).Render(strings.Join(lines, "\n"))
}

func (m Model) renderHierarchy(width, height int) string {
	lines := []string{
		titleStyle.Render("HIERARCHY"),
	}

	root := mutedStyle.Render("▼ root")

	lines = append(
		lines,
		root,
	)

	for i, node := range m.document.Root.Children {
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(colorText)

		if i == m.hierarchyCursor {
			cursor = "› "

			if m.panel == PanelHierarchy {
				style = selectedStyle
			}
		}

		lines = append(
			lines,
			style.Render(
				cursor+"├─ "+node.ID,
			),
		)
	}

	return m.panelStyle(
		width,
		height,
		PanelHierarchy,
	).Render(strings.Join(lines, "\n"))
}

func (m Model) renderFooter(width, height int) string {
	shortcuts := []string{
		keyStyle.Render("Tab") + " panels",
		keyStyle.Render("↑↓") + " navigate",
		keyStyle.Render("Enter") + " select/add",
		keyStyle.Render("A") + " add",
		keyStyle.Render("D") + " delete",
		keyStyle.Render("Ctrl+S") + " save",
		keyStyle.Render("Q") + " quit",
	}

	content := strings.Join(shortcuts, "  ")

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		BorderTop(true).
		BorderForeground(colorBorder).
		PaddingLeft(1).
		Foreground(colorMuted).
		Render(content)
}
