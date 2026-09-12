package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/maniprojs/tui-designer/internal/app"
)

func main() {
	p := tea.NewProgram(app.New())

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
