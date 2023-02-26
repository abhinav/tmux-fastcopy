package ui

import "fmt"

// Pos is a position in the terminal UI.
type Pos struct{ X, Y int }

// Get returns the coordinates as a pair.
//
//	x, y = pos.Get()
func (p Pos) Get() (x, y int) {
	return p.X, p.Y
}

func (p Pos) String() string {
	return fmt.Sprintf("(%d, %d)", p.X, p.Y)
}
