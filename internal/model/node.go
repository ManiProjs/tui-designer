package model

type Node struct {
	ID         string
	Type       string
	X          int
	Y          int
	Width      int
	Height     int
	Properties map[string]string
	Children   []*Node
}
