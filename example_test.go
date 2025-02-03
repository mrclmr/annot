package annot_test

import (
	"fmt"

	"github.com/mrclmr/annot"
)

func Example() {
	fmt.Println("The greatest enemy of knowledge is not ignorance, it is the illusion of knowledge.")
	fmt.Println(annot.String(
		&annot.Annot{Col: 1, Lines: []string{"article"}},
		&annot.Annot{Col: 4, ColEnd: 11, Lines: []string{"adjective"}},
		&annot.Annot{Col: 22, ColEnd: 30, Lines: []string{
			"facts, information, and skills acquired",
			"through experience or education;",
			"the theoretical or practical understanding",
			"of a subject.",
		}},
		&annot.Annot{Col: 48, Lines: []string{"comma"}},
	))
	// Output:
	// The greatest enemy of knowledge is not ignorance, it is the illusion of knowledge.
	//  ↑  └──┬───┘          └───┬───┘                 ↑
	//  │     └─ adjective       │                     └─ comma
	//  │                        │
	//  └─ article               └─ facts, information, and skills acquired
	//                              through experience or education;
	//                              the theoretical or practical understanding
	//                              of a subject.
}
