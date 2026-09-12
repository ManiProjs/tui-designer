package model

type Document struct {
	Version int   `yaml:"version"`
	Root    *Node `yaml:"root"`
}

func NewDocument() *Document {
	return &Document{
		Version: 1,
		Root: &Node{
			ID:   "root",
			Type: "column",
			Children: []*Node{
				{
					ID:   "title",
					Type: "text",
					Properties: map[string]string{
						"text": "Hello, TUI!",
					},
				},
				{
					ID:   "button",
					Type: "button",
					Properties: map[string]string{
						"text": "Click me",
					},
				},
			},
		},
	}
}