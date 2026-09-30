// Package annot annotates a string line leading to a description.
package annot

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode"

	"github.com/rivo/uniseg"
)

// Annot annotates information with a text (represented in Lines)
// at a position in a line (Col).
type Annot struct {
	// Col is the position of the arrowhead of the annotation.
	// E.g. 0 draws an arrow to the first character in a line.
	Col int

	// ColEnd needs to be higher than Col. If ColEnd is set a
	// range is annotated.
	ColEnd int

	// Lines is the text of the annotation represented in one or more lines.
	Lines []string
}

// AppendLines adds initial or appends additional lines to an annotation.
func (a *Annot) AppendLines(lines ...string) {
	a.Lines = append(a.Lines, lines...)
}

// Renderer renders annotations with options.
type Renderer struct {
	// Width is the maximum width of the rendered annotations,
	// e.g. the width of the terminal. Annotations that do not fit
	// are wrapped or point to the left. 0 means unlimited width.
	Width int
}

// String returns the rendered annotations as a string.
func String(annots ...*Annot) string {
	return Renderer{}.String(annots...)
}

// Write renders the annotations and writes them to a writer w.
func Write(w io.Writer, annots ...*Annot) error {
	return Renderer{}.Write(w, annots...)
}

// String returns the rendered annotations as a string.
func (r Renderer) String(annots ...*Annot) string {
	b := &strings.Builder{}
	_ = r.Write(b, annots...)
	return b.String()
}

// Write renders the annotations and writes them to a writer w.
func (r Renderer) Write(w io.Writer, annots ...*Annot) error {
	// Clone to not modify the slice of the caller.
	annots = slices.CompactFunc(slices.Clone(annots), func(a1 *Annot, a2 *Annot) bool {
		return a1.Col == a2.Col
	})

	if len(annots) == 0 {
		return nil
	}

	slices.SortFunc(annots, func(a *Annot, b *Annot) int {
		return a.Col - b.Col
	})

	pipeCols := make([]int, len(annots))
	for aIdx, a := range annots {
		if a.ColEnd != 0 {
			if a.Col >= a.ColEnd {
				return newColExceedsColEndError(aIdx+1, a.Col, a.ColEnd)
			}
			pipeCols[aIdx] = (a.Col + a.ColEnd) / 2
		} else {
			pipeCols[aIdx] = a.Col
		}
		if r.Width > 0 && max(a.Col, a.ColEnd) >= r.Width {
			return newColExceedsWidthError(aIdx+1, max(a.Col, a.ColEnd), r.Width)
		}
		if aIdx > 0 && annots[aIdx-1].ColEnd != 0 && annots[aIdx-1].ColEnd >= a.Col {
			return newOverlapError(annots[aIdx-1].ColEnd, aIdx, a.Col)
		}
	}

	return write(w, annots, pipeCols, r.arrange(annots, pipeCols))
}

// label is the rendering state of an annotation.
type label struct {
	pipeCol int

	// right is true if the label points to the right ("└─ text")
	// and false if it points to the left ("text ─┘").
	right bool

	// lines are the (wrapped) lines of the annotation.
	lines  []string
	widths []int

	// row is the row of the first line of the label.
	row int
}

// arrange splits the annotations into right pointing annotations
// on the left side and left pointing annotations on the right side.
// Every split is laid out and the one with the lowest cost is returned.
func (r Renderer) arrange(annots []*Annot, pipeCols []int) []*label {
	n := len(annots)
	if r.Width <= 0 {
		labels, _, _, _ := r.labels(annots, pipeCols, n)
		layout(labels, n)
		return labels
	}

	var best []*label
	var bestCost []int
	for k := n; k >= 0; k-- {
		labels, overflow, extraLines, ok := r.labels(annots, pipeCols, k)
		if !ok {
			continue
		}
		rows := layout(labels, k)
		// Prefer no overflow, then no wrapping, then right pointing
		// labels and last a low height.
		cost := []int{overflow, extraLines, n - k, rows}
		if best == nil || slices.Compare(cost, bestCost) < 0 {
			best, bestCost = labels, cost
		}
	}
	return best
}

// labels creates labels where the first k annotations point to the right and
// the remaining annotations point to the left. Lines are wrapped to the
// available space. A right pointing label must not reach over the pipe of a
// left pointing label. The labels are invalid (ok is false) if a line does not
// fit anyway. The only exception are right pointing labels if all labels point
// to the right. They overflow the width and the count of overflowing cells is
// returned.
func (r Renderer) labels(annots []*Annot, pipeCols []int, k int) (labels []*label, overflow, extraLines int, ok bool) {
	labels = make([]*label, len(annots))
	for i, a := range annots {
		l := &label{pipeCol: pipeCols[i], right: i < k}

		lines := a.Lines
		if len(lines) == 0 {
			lines = []string{""}
		}

		// room is the available width for the text of a line.
		room := -1
		switch {
		case r.Width <= 0:
			// Unlimited width.
		case !l.right:
			//      2 for " ─" before "┘"
			room = l.pipeCol - 2
		case k < len(annots):
			//                 2 for the space before the next pipe
			//                                       3 for "└─ "
			room = pipeCols[k] - 2 - l.pipeCol - 3
		default:
			room = r.Width - l.pipeCol - 3
		}

		for _, line := range lines {
			if r.Width > 0 && room >= 1 {
				l.lines = append(l.lines, wrap(line, room)...)
			} else {
				l.lines = append(l.lines, line)
			}
		}
		extraLines += len(l.lines) - len(lines)

		l.widths = make([]int, len(l.lines))
		for lineIdx, line := range l.lines {
			l.widths[lineIdx] = uniseg.StringWidth(line)
			if r.Width <= 0 || l.widths[lineIdx] <= room {
				continue
			}
			if !l.right || k < len(annots) {
				return nil, 0, 0, false
			}
			overflow += l.widths[lineIdx] - room
		}

		labels[i] = l
	}
	return labels, overflow, extraLines, true
}

// wrap breaks s into lines with a maximum width of room at line
// break opportunities. Too long segments are broken at grapheme clusters.
func wrap(s string, room int) []string {
	if uniseg.StringWidth(s) <= room {
		return []string{s}
	}

	var lines []string
	cur := ""
	flush := func() {
		lines = append(lines, strings.TrimRightFunc(cur, unicode.IsSpace))
		cur = ""
	}

	state := -1
	var segment string
	for s != "" {
		segment, s, _, state = uniseg.FirstLineSegmentInString(s, state)

		if uniseg.StringWidth(strings.TrimRightFunc(cur+segment, unicode.IsSpace)) <= room {
			cur += segment
			continue
		}
		if cur != "" {
			flush()
		}

		text := strings.TrimRightFunc(segment, unicode.IsSpace)
		space := segment[len(text):]
		if uniseg.StringWidth(text) <= room {
			cur = segment
			continue
		}

		// Break a too long segment at grapheme clusters.
		gState := -1
		var cluster string
		var width int
		for text != "" {
			cluster, text, width, gState = uniseg.FirstGraphemeClusterInString(text, gState)
			if cur != "" && uniseg.StringWidth(cur)+width > room {
				flush()
			}
			cur += cluster
		}
		cur += space
	}
	if cur != "" || len(lines) == 0 {
		flush()
	}
	return lines
}

// Bit flags of a cell in the grid.
const (
	// ink is a cell of a drawn character.
	ink uint8 = 1 << iota
	// keepOut is a cell that must not contain the ink of another label.
	keepOut
	// keepOutRight is a cell that must not contain the ink of a right pointing label.
	keepOutRight
	// keepOutLeft is a cell that must not contain the ink of a left pointing label.
	keepOutLeft
)

type cell struct {
	row, col int
	flag     uint8
	pipe     bool
}

// cells returns the cells of a label placed at row. Cells marked
// with keep out ensure the spacing between annotations. Those of a
// right pointing label (█ = keep out, ░ = keep out right):
//
//	row | column → p = pipeCol
//	----+-------------------------
//	0   |   ░░│
//	1   |   ░░│
//	r   |   ░░└─ line1██
//	r+1 |    ████line2██
//	r+2 |      ██line3██
//	r+3 |       ████████
//
// Cells of a left pointing label are mirrored at the pipe column.
func (l *label) cells(r int) []cell {
	p := l.pipeCol
	// dir mirrors the columns for left pointing labels.
	dir, keepOutPipe := 1, keepOutRight
	if !l.right {
		dir, keepOutPipe = -1, keepOutLeft
	}
	c := func(row, offset int, flag uint8) cell {
		return cell{row: row, col: p + dir*offset, flag: flag}
	}

	var cells []cell
	for row := range r + 1 {
		if row < r {
			cells = append(cells, cell{row: row, col: p, flag: ink, pipe: true})
		}
		cells = append(cells, c(row, -2, keepOutPipe), c(row, -1, keepOutPipe))
	}

	// Connector "└─ " or " ─┘".
	for offset := range 3 {
		cells = append(cells, c(r, offset, ink))
	}

	for i, w := range l.widths {
		row := r + i
		end := 2
		if w > 0 {
			for offset := 3; offset < 3+w; offset++ {
				cells = append(cells, c(row, offset, ink))
			}
			end = 2 + w
		}
		if w > 0 || i == 0 {
			cells = append(cells, c(row, end+1, keepOut), c(row, end+2, keepOut))
		}
		switch {
		case i == 1:
			for offset := -1; offset <= 2; offset++ {
				cells = append(cells, c(row, offset, keepOut))
			}
		case i > 1:
			cells = append(cells, c(row, 1, keepOut), c(row, 2, keepOut))
		}
	}
	// Keep the row below the text free. Otherwise a label pointing in
	// the other direction could look like a continuation of the text.
	for offset := 2; offset <= 2+slices.Max(l.widths); offset++ {
		cells = append(cells, c(r+len(l.widths), offset, keepOut))
	}
	return cells
}

type grid [][]uint8

func (g *grid) get(row, col int) uint8 {
	if row < 0 || row >= len(*g) || col < 0 || col >= len((*g)[row]) {
		return 0
	}
	return (*g)[row][col]
}

func (g *grid) set(row, col int, flag uint8) {
	if col < 0 {
		return
	}
	for row >= len(*g) {
		*g = append(*g, nil)
	}
	for col >= len((*g)[row]) {
		(*g)[row] = append((*g)[row], 0)
	}
	(*g)[row][col] |= flag
}

func (g *grid) fits(l *label, r int) bool {
	blocksInk := ink | keepOut | keepOutRight
	if !l.right {
		blocksInk = ink | keepOut | keepOutLeft
	}
	for _, c := range l.cells(r) {
		existing := g.get(c.row, c.col)
		switch {
		case c.pipe:
			// Pipes may cross keep out cells.
			if existing&ink != 0 {
				return false
			}
		case c.flag == ink:
			if existing&blocksInk != 0 {
				return false
			}
		case c.flag == keepOut:
			if existing&ink != 0 {
				return false
			}
		}
	}
	return true
}

// layout places the labels in the lowest possible row. The right pointing
// labels (first k labels) are placed from right to left and the left pointing
// labels from left to right. A pipe below a label is therefore always placed
// before that label. The row count is returned.
func layout(labels []*label, k int) int {
	g := &grid{}
	rowCount := 0
	order := slices.Concat(labels[:k], labels[k:])
	slices.Reverse(order[:k])
	for _, l := range order {
		r := 0
		for !g.fits(l, r) {
			r++
			// A row below all labels fits as long as no pipe crosses a label.
			if r > len(*g)+1 {
				panic("annot: no row fits the label")
			}
		}
		l.row = r
		for _, c := range l.cells(r) {
			g.set(c.row, c.col, c.flag)
		}
		rowCount = max(rowCount, r+len(l.lines))
	}
	return rowCount
}

func write(writer io.Writer, annots []*Annot, pipeCols []int, labels []*label) error {
	rowCount := 0
	for _, l := range labels {
		rowCount = max(rowCount, l.row+len(l.lines))
	}

	// canvas contains a grapheme cluster per cell. Unwritten cells are
	// empty and the continuation cells of wide characters are wideCont.
	const wideCont = "\x00"
	canvas := make([][]string, rowCount)
	draw := func(row, col int, s string) {
		state := -1
		var cluster string
		var width int
		for s != "" {
			cluster, s, width, state = uniseg.FirstGraphemeClusterInString(s, state)
			for col+max(width, 1) > len(canvas[row]) {
				canvas[row] = append(canvas[row], "")
			}
			canvas[row][col] = cluster
			for i := 1; i < width; i++ {
				canvas[row][col+i] = wideCont
			}
			col += max(width, 1)
		}
	}

	for _, l := range labels {
		p := l.pipeCol
		for row := range l.row {
			draw(row, p, "│")
		}
		if l.right {
			draw(l.row, p, "└─ ")
			for i, line := range l.lines {
				draw(l.row+i, p+3, line)
			}
			continue
		}
		draw(l.row, p-2, " ─┘")
		for i, line := range l.lines {
			draw(l.row+i, p-2-l.widths[i], line)
		}
	}

	b := &strings.Builder{}
	b.WriteString(arrowOrRangeString(annots, pipeCols))
	b.WriteString("\n")
	for _, cells := range canvas {
		for _, c := range cells {
			switch c {
			case "":
				b.WriteString(" ")
			case wideCont:
			default:
				b.WriteString(c)
			}
		}
		b.WriteString("\n")
	}
	_, err := fmt.Fprint(writer, b.String())
	return err
}

func arrowOrRangeString(annots []*Annot, pipeCols []int) string {
	widthWritten := 0

	b := &strings.Builder{}

	for aIdx, a := range annots {
		pipeColIdx := pipeCols[aIdx]
		if a.ColEnd == 0 {
			b.WriteString(strings.Repeat(" ", pipeColIdx-widthWritten))
			b.WriteString("↑")
			widthWritten = pipeColIdx + 1
			continue
		}

		b.WriteString(strings.Repeat(" ", a.Col-widthWritten))
		if a.Col == pipeColIdx {
			b.WriteString("├")
		} else {
			b.WriteString("└")
			b.WriteString(strings.Repeat("─", pipeColIdx-a.Col-1))
			b.WriteString("┬")
		}
		b.WriteString(strings.Repeat("─", a.ColEnd-pipeColIdx-1))
		b.WriteString("┘")
		widthWritten = a.ColEnd + 1
	}
	return b.String()
}
