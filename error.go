package annot

import (
	"fmt"
)

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
