package ui

import (
	tcell "github.com/gdamore/tcell/v3"
	"github.com/rivo/uniseg"
)

type testCell struct {
	str   string
	style tcell.Style
	width int
}

type renderScreen struct {
	w     int
	h     int
	cells []testCell
}

func newRenderScreen(w, h int) *renderScreen {
	return &renderScreen{
		w:     w,
		h:     h,
		cells: make([]testCell, w*h),
	}
}

func (s *renderScreen) Size() (int, int) {
	return s.w, s.h
}

func (s *renderScreen) Put(x, y int, str string, style tcell.Style) (string, int) {
	if x < 0 || x >= s.w || y < 0 || y >= s.h {
		return str, 0
	}

	cluster, rest, width, _ := uniseg.FirstGraphemeCluster([]byte(str), -1)
	cellStr := string(cluster)
	s.cells[y*s.w+x] = testCell{str: cellStr, style: style, width: width}
	return string(rest), width
}

func (s *renderScreen) Get(x, y int) (string, tcell.Style, int) {
	if x < 0 || x >= s.w || y < 0 || y >= s.h {
		return "", tcell.StyleDefault, 0
	}

	cell := s.cells[y*s.w+x]
	return cell.str, cell.style, cell.width
}

func (s *renderScreen) Clear() {
	clear(s.cells)
}

func (*renderScreen) Show() {}
