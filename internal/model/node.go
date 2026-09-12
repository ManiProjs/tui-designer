package model

type Node struct {
	ID         string            `yaml:"id,omitempty"`
	Type       string            `yaml:"type"`
	Properties map[string]string `yaml:"properties,omitempty"`
	Children   []*Node           `yaml:"children,omitempty"`
}