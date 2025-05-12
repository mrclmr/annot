// Package annot annotates a string line leading to a description.
package annot

import (
	"fmt"
	"io"
	"slices"
	"strings"

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

	pipeColIdx int

	row               int
	lines             []*line
	pipeLeadingSpaces []int
}

// line is an internal parallel to a string in Lines.
type line struct {
	length        int
	leadingSpaces int
}

type section int

// #   |                  column |0123456789...
// ----+-------------------------+-------------------------------------------------
// row | section                 |     ↓ column position of annotation
// -   | not relevant            |     ↑
// 0 · | above · · · · · · · · · | · ██│· · · · · · · · · · · · · · · · · · · · · ·
// 1   | above                   |   ██│
// 2 · | lineOne · · · · · · · · | · ██└─ line1 · · · · row position of annotation
// 3   | lineTwo                 |    ████line2
// 4 · | linesAfterSecond· · · · | ·  · ██line3 · · · · · · · · · · · · · · · · · ·
// 5   | linesAfterSecond        |      ██line4
// 6 · | trailingSpaceLines(1) · | ·  · ·█· · · · · · · · · · · · · · · · · · · · ·
// 7   | noAnnot                 |
// 8 · | noAnnot · · · · · · · · | ·  · · · · · · · · · · · · · · · · · · · · · · ·
// 9   | noAnnot                 |
// ----+-------------------------+-------------------------------------------------
// #   |                         |    █ = needed space.
const (
	above section = iota
	lineOne
	lineTwo
	linesAfterSecond
	trailingSpaceLines
	noAnnot
)

func (s *section) space() int {
	switch *s {
	case above, lineOne:
		return 2
	case lineTwo:
		return 4
	case linesAfterSecond:
		return 2
	case trailingSpaceLines:
		return 1
	default:
		return -1
	}
}

func (s *section) colPosShift() int {
	switch *s {
	case above, lineOne:
		return 0
	case lineTwo, linesAfterSecond, trailingSpaceLines:
		return 3
	default:
		return -1
	}
}

// AppendLines adds initial or appends additional lines to an annotation.
func (a *Annot) AppendLines(lines ...string) {
	a.Lines = append(a.Lines, lines...)
}

// String returns the rendered annotations as a string.
func String(annots ...*Annot) string {
	b := &strings.Builder{}
	_ = Write(b, annots...)
	return b.String()
}

// Write renders the annotations and writes them to a writer w.
func Write(w io.Writer, annots ...*Annot) error {
	annots = slices.CompactFunc(annots, func(a1 *Annot, a2 *Annot) bool {
		return a1.Col == a2.Col
	})

	if len(annots) == 0 {
		return nil
	}

	slices.SortFunc(annots, func(a *Annot, b *Annot) int {
		return a.Col - b.Col
	})

	for aIdx, a := range annots {
		if a.ColEnd != 0 {
			if a.Col >= a.ColEnd {
				return newColExceedsColEndError(aIdx+1, a.Col, a.ColEnd)
			}
			a.pipeColIdx = (a.Col + a.ColEnd) / 2
		} else {
			a.pipeColIdx = a.Col
		}
		if aIdx > 0 && annots[aIdx-1].ColEnd != 0 && annots[aIdx-1].ColEnd >= a.Col {
			return newOverlapError(annots[aIdx-1].ColEnd, aIdx, a.Col)
		}
		a.createLines()
	}

	// Start with second last annotation index and decrement.
	// The last annotation will always be on row=0 and needs
	// no adjustment.
	for aIdxDecr := len(annots) - 2; 0 <= aIdxDecr; aIdxDecr-- {
		setRow(annots[aIdxDecr], annots[aIdxDecr+1:])
	}

	return write(w, annots)
}

// createLines creates an array of lines parallel to Lines.
func (a *Annot) createLines() {
	if len(a.Lines) == 0 {
		a.lines = make([]*line, 1)
		a.lines[0] = &line{leadingSpaces: a.pipeColIdx}
		return
	}

	a.lines = make([]*line, len(a.Lines))
	for i := range a.Lines {
		leadingSpaces := a.pipeColIdx
		if i > 0 {
			leadingSpaces += 3
		}

		a.lines[i] = &line{
			length:        uniseg.StringWidth(a.Lines[i]),
			leadingSpaces: leadingSpaces,
		}
	}
}

func setRow(a *Annot, rightAnnots []*Annot) {
	row := 0

	for {
		rowBefore := row - 1
		if rowBefore != -1 {
			setSpace(rowBefore, a, rightAnnots)
		}

		annotFits := checkLinesAndSetSpaces(row, a, rightAnnots)
		if annotFits {
			return
		}
		row++
	}
}

func setSpace(rowBefore int, a *Annot, rightAnnots []*Annot) {
	closestA, s := closestAnnot(rowBefore, rightAnnots, 0)
	switch s {
	case above:
		closestA.pipeLeadingSpaces[rowBefore] = closestA.pipeColIdx + s.colPosShift() - a.pipeColIdx - 1
	case lineOne, lineTwo, linesAfterSecond:
		closestA.lines[rowBefore-closestA.row].leadingSpaces = closestA.pipeColIdx + s.colPosShift() - a.pipeColIdx - 1
	case noAnnot, trailingSpaceLines:
		// Do nothing
	}
}

func checkLinesAndSetSpaces(row int, a *Annot, rightAnnots []*Annot) bool {
	for aLineIdx := range a.lines {
		lineFits := checkLineAndSetSpace(row, aLineIdx, a, rightAnnots)
		if !lineFits {
			return false
		}
	}
	return true
}

func checkLineAndSetSpace(row, aLineIdx int, a *Annot, rightAnnots []*Annot) bool {
	rowPlusLineIdx := row + aLineIdx

	closestA, s := closestAnnot(rowPlusLineIdx, rightAnnots, 1)
	if s == noAnnot {
		return true
	}

	//            3 for "└─ " or "   " (indentation)
	lineLength := 3 + a.lines[aLineIdx].length

	remainingSpaces := closestA.pipeColIdx + s.colPosShift() - a.pipeColIdx - lineLength

	if remainingSpaces-s.space() < 0 {
		a.row++
		a.pipeLeadingSpaces = append(a.pipeLeadingSpaces, a.pipeColIdx)
		return false
	}

	switch s {
	case above:
		closestA.pipeLeadingSpaces[rowPlusLineIdx] = remainingSpaces
	case lineOne, lineTwo, linesAfterSecond:
		closestA.lines[rowPlusLineIdx-closestA.row].leadingSpaces = remainingSpaces
	case trailingSpaceLines:
		closestA2, s2 := closestAnnot(rowPlusLineIdx, rightAnnots, 0)
		if s2 == noAnnot {
			return true
		}
		leadingSpaces2 := closestA2.pipeColIdx + s2.colPosShift() - a.pipeColIdx - lineLength
		if s2 == above {
			closestA2.pipeLeadingSpaces[rowPlusLineIdx] = leadingSpaces2
			return true
		}
		closestA2.lines[rowPlusLineIdx-closestA2.row].leadingSpaces = leadingSpaces2
	default:
		panic("should never be reached")
	}
	return true
}

func closestAnnot(row int, rightAnnots []*Annot, trailingVerticalSpaceLinesCount int) (*Annot, section) {
	for _, a := range rightAnnots {
		if row < a.row {
			return a, above
		}
		if a.row == row {
			return a, lineOne
		}
		if a.row+1 == row && row < a.row+len(a.lines) {
			return a, lineTwo
		}
		if 2 <= row && row < a.row+len(a.lines) {
			return a, linesAfterSecond
		}
		if a.row+len(a.lines) <= row && row < a.row+len(a.lines)+trailingVerticalSpaceLinesCount {
			return a, trailingSpaceLines
		}

	}
	return nil, noAnnot
}

func write(writer io.Writer, annots []*Annot) error {
	rowCount := 0
	for _, a := range annots {
		rowCount = max(rowCount, a.row+len(a.lines))
	}

	b := &strings.Builder{}

	b.WriteString(arrowOrRangeString(annots))

	b.WriteString("\n")
	_, err := fmt.Fprint(writer, b.String())
	if err != nil {
		return err
	}
	b.Reset()

	for row := range rowCount {
		for _, a := range annots {
			switch {
			case row < a.row:
				b.WriteString(strings.Repeat(" ", a.pipeLeadingSpaces[row]))
				b.WriteString("│")
			case row == a.row:
				b.WriteString(strings.Repeat(" ", a.lines[row-a.row].leadingSpaces))
				b.WriteString("└─ ")
				if len(a.Lines) != 0 {
					b.WriteString(a.Lines[0])
				}
			case row < a.row+len(a.lines):
				b.WriteString(strings.Repeat(" ", a.lines[row-a.row].leadingSpaces))
				b.WriteString(a.Lines[row-a.row])
			}
		}
		b.WriteString("\n")
		_, err = fmt.Fprint(writer, b.String())
		if err != nil {
			return err
		}
		b.Reset()
	}

	return nil
}

func arrowOrRangeString(annots []*Annot) string {
	widthWritten := 0

	b := &strings.Builder{}

	for _, a := range annots {
		if a.ColEnd == 0 {
			b.WriteString(strings.Repeat(" ", a.pipeColIdx-widthWritten))
			b.WriteString("↑")
			widthWritten = a.pipeColIdx + 1
			continue
		}

		b.WriteString(strings.Repeat(" ", a.Col-widthWritten))
		if a.Col == a.pipeColIdx {
			b.WriteString("├")
		} else {
			b.WriteString("└")
			b.WriteString(strings.Repeat("─", a.pipeColIdx-a.Col-1))
			b.WriteString("┬")
		}
		b.WriteString(strings.Repeat("─", a.ColEnd-a.pipeColIdx-1))
		b.WriteString("┘")
		widthWritten = a.ColEnd + 1
	}
	return b.String()
}
