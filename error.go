package annot

import (
	"fmt"
)

// OverlapError occurs if Annot.Col of two annotations overlap.
type OverlapError struct {
	colEnd, firstAnnotPos, secondAnnotCol int
}

func newOverlapError(colEnd, firstAnnotPos, secondAnnotCol int) *OverlapError {
	return &OverlapError{colEnd, firstAnnotPos, secondAnnotCol}
}

func (e *OverlapError) Error() string {
	return fmt.Sprintf("annot: ColEnd %d of %d. annotation overlaps with Col %d of %d. annotation",
		e.colEnd, e.firstAnnotPos, e.secondAnnotCol, e.firstAnnotPos+1)
}

// ColExceedsColEndError occurs if the Annot.Col is equal or higher than Annot.ColEnd.
type ColExceedsColEndError struct {
	annotPos, col, colEnd int
}

func newColExceedsColEndError(annotPos, col, colEnd int) *ColExceedsColEndError {
	return &ColExceedsColEndError{annotPos, col, colEnd}
}

func (e *ColExceedsColEndError) Error() string {
	return fmt.Sprintf("annot: in %d. annotation Col %d needs to be lower than ColEnd %d",
		e.annotPos, e.col, e.colEnd)
}

// ColExceedsWidthError occurs if the Annot.Col or Annot.ColEnd is equal or
// higher than Renderer.Width.
type ColExceedsWidthError struct {
	annotPos, col, width int
}

func newColExceedsWidthError(annotPos, col, width int) *ColExceedsWidthError {
	return &ColExceedsWidthError{annotPos, col, width}
}

func (e *ColExceedsWidthError) Error() string {
	return fmt.Sprintf("annot: in %d. annotation column %d needs to be lower than width %d",
		e.annotPos, e.col, e.width)
}
