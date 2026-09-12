package app

import (
	"fmt"
	"strconv"
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

type InspectorField struct {
	Name  string
	Value string
}

var paletteWidgets = []struct {
	Type   string
	Name   string
	Width  int
	Height int
}{
	{Type: "text", Name: "Text", Width: 20, Height: 1},
	{Type: "button", Name: "Button", Width: 20, Height: 3},
	{Type: "input", Name: "Input", Width: 24, Height: 3},
	{Type: "panel", Name: "Panel", Width: 30, Height: 8},
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

	editorStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 1)
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

	inspectorCursor int
	editing         bool
	editValue       string
	editField       string
}

func New() Model {
	document := model.NewDocument()

	return Model{
		document: document,
		panel:    PanelPalette,
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
		if m.editing {
			return m.updateEditor(msg)
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "q":
			return m, tea.Quit

		case "tab":
			m.nextPanel()

		case "shift+tab":
			m.previousPanel()

		case "left":
			if m.panel == PanelCanvas {
				m.moveSelected(-1, 0)
			} else {
				m.previousPanel()
			}

		case "right":
			if m.panel == PanelCanvas {
				m.moveSelected(1, 0)
			} else {
				m.nextPanel()
			}

		case "up":
			if m.panel == PanelCanvas {
				m.moveSelected(0, -1)
			} else {
				m.moveCursor(-1)
			}

		case "down":
			if m.panel == PanelCanvas {
				m.moveSelected(0, 1)
			} else {
				m.moveCursor(1)
			}

		case "shift+left":
			if m.panel == PanelCanvas {
				m.resizeSelected(-1, 0)
			}

		case "shift+right":
			if m.panel == PanelCanvas {
				m.resizeSelected(1, 0)
			}

		case "shift+up":
			if m.panel == PanelCanvas {
				m.resizeSelected(0, -1)
			}

		case "shift+down":
			if m.panel == PanelCanvas {
				m.resizeSelected(0, 1)
			}

		case "enter":
			m.activate()

		case "a":
			m.addSelectedWidget()

		case "d", "delete", "backspace":
			m.deleteSelectedWidget()

		case "esc":
			m.panel = PanelCanvas
			m.status = "Canvas focused"

		case "ctrl+s":
			m.status = "Project saved"
		}
	}

	return m, nil
}

func (m *Model) updateEditor(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.editing = false
		m.editValue = ""
		m.editField = ""
		m.status = "Edit cancelled"
		return m, nil

	case "enter":
		m.applyInspectorEdit()
		return m, nil

	case "backspace":
		if len(m.editValue) > 0 {
			m.editValue = m.editValue[:len(m.editValue)-1]
		}

	default:
		text := msg.String()

		if len(text) == 1 {
			m.editValue += text
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

	case PanelInspector:
		fields := m.inspectorFields()

		if len(fields) == 0 {
			return
		}

		m.inspectorCursor += delta

		if m.inspectorCursor < 0 {
			m.inspectorCursor = 0
		}

		if m.inspectorCursor >= len(fields) {
			m.inspectorCursor = len(fields) - 1
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
		m.inspectorCursor = 0

		m.status = "Selected " + m.selected.ID

	case PanelCanvas:
		if m.selected != nil {
			m.status = "Selected " + m.selected.ID
		}

	case PanelInspector:
		m.startInspectorEdit()
	}
}

func (m *Model) inspectorFields() []InspectorField {
	if m.selected == nil {
		return nil
	}

	fields := []InspectorField{
		{
			Name:  "X",
			Value: strconv.Itoa(m.selected.X),
		},
		{
			Name:  "Y",
			Value: strconv.Itoa(m.selected.Y),
		},
		{
			Name:  "Width",
			Value: strconv.Itoa(m.selected.Width),
		},
		{
			Name:  "Height",
			Value: strconv.Itoa(m.selected.Height),
		},
	}

	if m.selected.Properties != nil {
		for key, value := range m.selected.Properties {
			fields = append(fields, InspectorField{
				Name:  key,
				Value: value,
			})
		}
	}

	return fields
}

func (m *Model) startInspectorEdit() {
	fields := m.inspectorFields()

	if len(fields) == 0 ||
		m.inspectorCursor >= len(fields) {
		return
	}

	field := fields[m.inspectorCursor]

	m.editing = true
	m.editField = field.Name
	m.editValue = field.Value

	m.status = "Editing " + field.Name
}

func (m *Model) applyInspectorEdit() {
	if m.selected == nil {
		m.editing = false
		return
	}

	value := m.editValue

	switch m.editField {
	case "X":
		if n, err := strconv.Atoi(value); err == nil && n >= 0 {
			m.selected.X = n
		}

	case "Y":
		if n, err := strconv.Atoi(value); err == nil && n >= 0 {
			m.selected.Y = n
		}

	case "Width":
		if n, err := strconv.Atoi(value); err == nil && n >= 1 {
			m.selected.Width = n
		}

	case "Height":
		if n, err := strconv.Atoi(value); err == nil && n >= 1 {
			m.selected.Height = n
		}

	default:
		if m.selected.Properties == nil {
			m.selected.Properties = make(map[string]string)
		}

		m.selected.Properties[m.editField] = value
	}

	m.editing = false
	m.status = "Updated " + m.editField
	m.editField = ""
	m.editValue = ""
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
		Type:   widget.Type,
		X:      2,
		Y:      2 + len(m.document.Root.Children)*2,
		Width:  widget.Width,
		Height: widget.Height,
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
	m.inspectorCursor = 0

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
			m.inspectorCursor = 0
			m.status = "Deleted " + deletedID
			return
		}

		if i >= len(m.document.Root.Children) {
			i = len(m.document.Root.Children) - 1
		}

		m.hierarchyCursor = i
		m.selected = m.document.Root.Children[i]
		m.inspectorCursor = 0

		m.status = "Deleted " + deletedID

		return
	}
}

func (m *Model) moveSelected(dx, dy int) {
	if m.selected == nil {
		return
	}

	m.selected.X += dx
	m.selected.Y += dy

	if m.selected.X < 0 {
		m.selected.X = 0
	}

	if m.selected.Y < 0 {
		m.selected.Y = 0
	}

	m.status = fmt.Sprintf(
		"%s → (%d, %d)",
		m.selected.ID,
		m.selected.X,
		m.selected.Y,
	)
}

func (m *Model) resizeSelected(dw, dh int) {
	if m.selected == nil {
		return
	}

	m.selected.Width += dw
	m.selected.Height += dh

	if m.selected.Width < 1 {
		m.selected.Width = 1
	}

	if m.selected.Height < 1 {
		m.selected.Height = 1
	}

	m.status = fmt.Sprintf(
		"%s → %d × %d",
		m.selected.ID,
		m.selected.Width,
		m.selected.Height,
	)
}

func (m Model) View() tea.View {
	view := tea.NewView(m.render())
	view.AltScreen = true

	return view
}

func (m Model) render() string {
	if m.width < 60 || m.height < 15 {
		return fmt.Sprintf(
			"TUI Designer\n\n"+
				"Terminal too small.\n\n"+
				"Current: %d × %d\n"+
				"Minimum: 60 × 15",
			m.width,
			m.height,
		)
	}

	const (
		headerHeight    = 1
		statusHeight    = 1
		hierarchyHeight = 6
		footerHeight    = 1
	)

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
	canvasWidth := m.width - paletteWidth - inspectorWidth

	header := m.renderHeader(m.width, headerHeight)

	status := m.renderStatus(m.width, statusHeight)

	editor := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderPalette(paletteWidth, editorHeight),
		m.renderCanvas(canvasWidth, editorHeight),
		m.renderInspector(inspectorWidth, editorHeight),
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
	left := titleStyle.Render(" TUI Designer ")

	right := mutedStyle.Render("Visual TUI editor")

	gap := width -
		lipgloss.Width(left) -
		lipgloss.Width(right)

	if gap < 1 {
		gap = 1
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Render(
			left +
				strings.Repeat(" ", gap) +
				right,
		)
}

func (m Model) renderStatus(width, height int) string {
	selected := "Nothing selected"

	if m.selected != nil {
		selected = fmt.Sprintf(
			"%s • %d,%d • %d×%d",
			m.selected.ID,
			m.selected.X,
			m.selected.Y,
			m.selected.Width,
			m.selected.Height,
		)
	}

	if m.editing {
		selected = "Editing " + m.editField
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

	canvasWidth := width - 4
	canvasHeight := height - 4

	if canvasWidth < 1 {
		canvasWidth = 1
	}

	if canvasHeight < 1 {
		canvasHeight = 1
	}

	canvas := make([][]rune, canvasHeight)

	for y := range canvas {
		canvas[y] = make([]rune, canvasWidth)

		for x := range canvas[y] {
			canvas[y][x] = ' '
		}
	}

	for _, node := range m.document.Root.Children {
		m.drawNode(
			canvas,
			node,
			canvasWidth,
			canvasHeight,
		)
	}

	for _, row := range canvas {
		lines = append(
			lines,
			string(row),
		)
	}

	return m.panelStyle(
		width,
		height,
		PanelCanvas,
	).Render(strings.Join(lines, "\n"))
}

func (m Model) drawNode(
	canvas [][]rune,
	node *model.Node,
	canvasWidth int,
	canvasHeight int,
) {
	x := node.X
	y := node.Y

	if x >= canvasWidth || y >= canvasHeight {
		return
	}

	width := node.Width
	height := node.Height

	if width < 1 {
		width = 1
	}

	if height < 1 {
		height = 1
	}

	selected := node == m.selected

	switch node.Type {
	case "text":
		text := node.Properties["text"]

		if selected {
			text = "▶ " + text
		}

		m.drawText(
			canvas,
			x,
			y,
			text,
			canvasWidth,
			canvasHeight,
		)

	case "button":
		m.drawBox(
			canvas,
			x,
			y,
			width,
			height,
			canvasWidth,
			canvasHeight,
			selected,
		)

		text := "[ " + node.Properties["text"] + " ]"

		tx := x + (width-len([]rune(text)))/2

		if tx < x {
			tx = x
		}

		ty := y + height/2

		m.drawText(
			canvas,
			tx,
			ty,
			text,
			canvasWidth,
			canvasHeight,
		)

	case "input":
		m.drawBox(
			canvas,
			x,
			y,
			width,
			height,
			canvasWidth,
			canvasHeight,
			selected,
		)

		m.drawText(
			canvas,
			x+2,
			y+height/2,
			node.Properties["text"],
			canvasWidth,
			canvasHeight,
		)

	case "panel":
		m.drawBox(
			canvas,
			x,
			y,
			width,
			height,
			canvasWidth,
			canvasHeight,
			selected,
		)

	default:
		m.drawBox(
			canvas,
			x,
			y,
			width,
			height,
			canvasWidth,
			canvasHeight,
			selected,
		)
	}
}

func (m Model) drawText(
	canvas [][]rune,
	x int,
	y int,
	text string,
	canvasWidth int,
	canvasHeight int,
) {
	if y < 0 || y >= canvasHeight {
		return
	}

	for i, r := range []rune(text) {
		px := x + i

		if px < 0 || px >= canvasWidth {
			continue
		}

		canvas[y][px] = r
	}
}

func (m Model) drawBox(
	canvas [][]rune,
	x int,
	y int,
	width int,
	height int,
	canvasWidth int,
	canvasHeight int,
	selected bool,
) {
	if width < 2 {
		width = 2
	}

	if height < 2 {
		height = 2
	}

	for px := x; px < x+width; px++ {
		if px < 0 || px >= canvasWidth {
			continue
		}

		if y >= 0 && y < canvasHeight {
			canvas[y][px] = '─'
		}

		bottom := y + height - 1

		if bottom >= 0 && bottom < canvasHeight {
			canvas[bottom][px] = '─'
		}
	}

	for py := y; py < y+height; py++ {
		if py < 0 || py >= canvasHeight {
			continue
		}

		if x >= 0 && x < canvasWidth {
			canvas[py][x] = '│'
		}

		right := x + width - 1

		if right >= 0 && right < canvasWidth {
			canvas[py][right] = '│'
		}
	}

	right := x + width - 1
	bottom := y + height - 1

	if x >= 0 && x < canvasWidth &&
		y >= 0 && y < canvasHeight {
		canvas[y][x] = '┌'
	}

	if right >= 0 && right < canvasWidth &&
		y >= 0 && y < canvasHeight {
		canvas[y][right] = '┐'
	}

	if x >= 0 && x < canvasWidth &&
		bottom >= 0 && bottom < canvasHeight {
		canvas[bottom][x] = '└'
	}

	if right >= 0 && right < canvasWidth &&
		bottom >= 0 && bottom < canvasHeight {
		canvas[bottom][right] = '┘'
	}

	if selected {
		markX := x + width/2

		if markX >= 0 &&
			markX < canvasWidth &&
			y >= 0 &&
			y < canvasHeight {
			canvas[y][markX] = '◆'
		}
	}
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
			"",
			mutedStyle.Render("Select a widget"),
			mutedStyle.Render("from the hierarchy."),
		)

		return m.panelStyle(
			width,
			height,
			PanelInspector,
		).Render(strings.Join(lines, "\n"))
	}

	fields := m.inspectorFields()

	for i, field := range fields {
		cursor := "  "
		style := lipgloss.NewStyle().
			Foreground(colorText)

		if i == m.inspectorCursor {
			cursor = "› "

			if m.panel == PanelInspector {
				style = selectedStyle
			}
		}

		value := field.Value

		if m.editing && i == m.inspectorCursor {
			value = m.editValue + "█"

			lines = append(
				lines,
				style.Render(cursor+field.Name),
				editorStyle.Render(value),
			)

			continue
		}

		lines = append(
			lines,
			style.Render(cursor+field.Name),
			mutedStyle.Render("    "+value),
		)
	}

	lines = append(
		lines,
		"",
		mutedStyle.Render("Enter  Edit"),
	)

	return m.panelStyle(
		width,
		height,
		PanelInspector,
	).Render(strings.Join(lines, "\n"))
}

func (m Model) renderHierarchy(width, height int) string {
	lines := []string{
		titleStyle.Render("HIERARCHY"),
		"",
		mutedStyle.Render("▼ root"),
	}

	for i, node := range m.document.Root.Children {
		cursor := "  "
		style := lipgloss.NewStyle().
			Foreground(colorText)

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
	var shortcuts []string

	if m.editing {
		shortcuts = []string{
			keyStyle.Render("Type") + " edit",
			keyStyle.Render("Enter") + " apply",
			keyStyle.Render("Esc") + " cancel",
		}
	} else {
		switch m.panel {
		case PanelPalette:
			shortcuts = []string{
				keyStyle.Render("↑↓") + " navigate",
				keyStyle.Render("Enter/A") + " add",
				keyStyle.Render("Tab") + " panels",
				keyStyle.Render("Q") + " quit",
			}

		case PanelCanvas:
			shortcuts = []string{
				keyStyle.Render("←↑↓→") + " move",
				keyStyle.Render("Shift+Arrows") + " resize",
				keyStyle.Render("D") + " delete",
				keyStyle.Render("Tab") + " panels",
			}

		case PanelHierarchy:
			shortcuts = []string{
				keyStyle.Render("↑↓") + " navigate",
				keyStyle.Render("Enter") + " select",
				keyStyle.Render("D") + " delete",
				keyStyle.Render("Tab") + " panels",
			}

		case PanelInspector:
			shortcuts = []string{
				keyStyle.Render("↑↓") + " properties",
				keyStyle.Render("Enter") + " edit",
				keyStyle.Render("Tab") + " panels",
				keyStyle.Render("Q") + " quit",
			}
		}
	}

	// Available content width after the left/right padding.
	available := width - 3

	if available < 1 {
		available = 1
	}

	// Build lines without allowing them to exceed the terminal.
	var rows []string
	var current string

	for _, shortcut := range shortcuts {
		// Lip Gloss escape sequences make len() inaccurate, so use
		// lipgloss.Width() for terminal display width.
		candidate := shortcut

		if current != "" {
			candidate = current + "  " + shortcut
		}

		if current != "" &&
			lipgloss.Width(candidate) > available {
			rows = append(rows, current)
			current = shortcut
			continue
		}

		current = candidate
	}

	if current != "" {
		rows = append(rows, current)
	}

	content := strings.Join(rows, "\n")

	footerHeight := len(rows)

	if footerHeight < 1 {
		footerHeight = 1
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(footerHeight).
		BorderTop(true).
		BorderForeground(colorBorder).
		PaddingLeft(1).
		Foreground(colorMuted).
		Render(content)
}
