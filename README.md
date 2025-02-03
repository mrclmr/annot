[![build](https://github.com/mrclmr/annot/actions/workflows/build.yml/badge.svg)](https://github.com/mrclmr/annot/actions/workflows/build.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/mrclmr/annot)](https://goreportcard.com/report/github.com/mrclmr/annot)

# Annot

Annotate a string line leading to a description.

```
The greatest enemy of knowledge is not ignorance, it is the illusion of knowledge.
 ↑  └──┬───┘          └───┬───┘                 ↑
 │     └─ adjective       │                     └─ comma
 │                        │
 └─ article               └─ facts, information, and skills acquired
                             through experience or education;
                             the theoretical or practical understanding
                             of a subject.
```

## Installation

```
go get github.com/mrclmr/annot
```

## Example

```go
package main

import (
	"fmt"

	"github.com/mrclmr/annot"
)

func main() {
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
}
```

Output:

```
The greatest enemy of knowledge is not ignorance, it is the illusion of knowledge.
 ↑  └──┬───┘          └───┬───┘                 ↑
 │     └─ adjective       │                     └─ comma
 │                        │
 └─ article               └─ facts, information, and skills acquired
                             through experience or education;
                             the theoretical or practical understanding
                             of a subject.
```