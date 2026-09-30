package annot

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestWrite(t *testing.T) {
	tests := []struct {
		name   string
		annots []*Annot
		wantW  string
	}{
		{
			name: "utilise every space to the limit",
			annots: []*Annot{
				{Col: 0, Lines: []string{"000", "100", "2000", "300000", "400000", "5000000", "6000000000000"}},
				{Col: 8, Lines: []string{"line1", "line2", "line3", "line4"}},
				{Col: 9, Lines: []string{"line1"}},
			},
			wantW: `
↑       ↑↑
│       │└─ line1
└─ 000  │
   100  └─ line1
   2000    line2
   300000  line3
   400000  line4
   5000000
   6000000000000
`,
		},
		{
			name: "empty annotation",
			annots: []*Annot{
				{},
			},
			wantW: `
↑
└─ 
`,
		},
		{
			name: "two empty annotations",
			annots: []*Annot{
				{Col: 0},
				{Col: 1},
			},
			wantW: `
↑↑
│└─ 
└─ 
`,
		},
		{
			name: "annotation without a column",
			annots: []*Annot{
				{Lines: []string{"line1"}},
			},
			wantW: `
↑
└─ line1
`,
		},
		{
			name: "annotation without a line",
			annots: []*Annot{
				{Col: 0},
			},
			wantW: `
↑
└─ 
`,
		},
		{
			name: "annotation with one line",
			annots: []*Annot{
				{Col: 1, Lines: []string{"line1"}},
			},
			wantW: `
 ↑
 └─ line1
`,
		},
		{
			name: "annotation with five lines",
			annots: []*Annot{
				{Col: 0, Lines: []string{"line1", "line2", "line3", "line4", "line5"}},
			},
			wantW: `
↑
└─ line1
   line2
   line3
   line4
   line5
`,
		},
		{
			name: "utf8, japanese characters and emojis",
			annots: []*Annot{
				{Col: 0, Lines: []string{"⭐️漢", "æñ🥏Ǌ"}},
				{Col: 10, Lines: []string{"æñŶǼǊ", "字ñŶǼǊ"}},
			},
			wantW: `
↑         ↑
└─ ⭐️漢   └─ æñŶǼǊ
   æñ🥏Ǌ     字ñŶǼǊ
`,
		},
		{
			name: "next to each other with enough distance and second annotation has more lines",
			annots: []*Annot{
				{Col: 5, Lines: []string{"line1", "line2"}},
				{Col: 20, Lines: []string{"line1", "line2", "line3", "line4"}},
			},
			wantW: `
     ↑              ↑
     └─ line1       └─ line1
        line2          line2
                       line3
                       line4
`,
		},
		{
			name: "next to each other with enough distance and first annotation has more lines",
			annots: []*Annot{
				{Col: 5, Lines: []string{"line1", "line2", "line3", "line4"}},
				{Col: 20, Lines: []string{"line1", "line2"}},
			},
			wantW: `
     ↑              ↑
     └─ line1       └─ line1
        line2          line2
        line3
        line4
`,
		},
		{
			name: "annots are close",
			annots: []*Annot{
				{Col: 0, Lines: []string{"line1", "line2"}},
				{Col: 1, Lines: []string{"line1", "line2"}},
			},
			wantW: `
↑↑
│└─ line1
│   line2
│
└─ line1
   line2
`,
		},
		{
			name: "annots have correct vertical distance",
			annots: []*Annot{
				{Col: 5, Lines: []string{"line1", "line2"}},
				{Col: 10, Lines: []string{"line1", "line2"}},
				{Col: 15, Lines: []string{"line1", "line2"}},
			},
			wantW: `
     ↑    ↑    ↑
     │    │    └─ line1
     │    │       line2
     │    │
     │    └─ line1
     │       line2
     │
     └─ line1
        line2
`,
		},
		{
			name: "first annotation is at the height of the pipes of the second annotation",
			annots: []*Annot{
				{Col: 0, Lines: []string{"line1", "line2"}},
				{Col: 10, Lines: []string{"line1", "line2"}},
				{Col: 15, Lines: []string{"line1", "line2"}},
			},
			wantW: `
↑         ↑    ↑
└─ line1  │    └─ line1
   line2  │       line2
          │
          └─ line1
             line2
`,
		},
		{
			name: "correct vertical and horizontal distance",
			annots: []*Annot{
				{Col: 5, Lines: []string{"line1", "line2"}},
				{Col: 10, Lines: []string{"line1", "line2"}},
				{Col: 20, Lines: []string{"line1", "line2"}},
			},
			wantW: `
     ↑    ↑         ↑
     │    └─ line1  └─ line1
     │       line2     line2
     │
     └─ line1
        line2
`,
		},
		{
			name: "correct horizontal distance",
			annots: []*Annot{
				{Col: 0, Lines: []string{"line1", "line2"}},
				{Col: 10, Lines: []string{"line1", "line2"}},
			},
			wantW: `
↑         ↑
└─ line1  └─ line1
   line2     line2
`,
		},
		{
			name: "allow one space more at the edge",
			annots: []*Annot{
				{Col: 0, Lines: []string{"line1", "line2"}},
				{Col: 6, Lines: []string{"line1", "line2"}},
			},
			wantW: `
↑     ↑
│     └─ line1
│        line2
└─ line1
   line2
`,
		},
		{
			name: "allow one space more at lines after the second line",
			annots: []*Annot{
				{Col: 0, Lines: []string{"line1", "line2"}},
				{Col: 7, Lines: []string{"line1", "line2", "line3", "line4"}},
			},
			wantW: `
↑      ↑
│      └─ line1
│         line2
└─ line1  line3
   line2  line4
`,
		},
		{
			name: "long line of first annotation uses available space",
			annots: []*Annot{
				{Col: 2, Lines: []string{"line1", "line2", "line3long"}},
				{Col: 10, Lines: []string{"line1", "line2", "line3"}},
			},
			wantW: `
  ↑       ↑
  │       └─ line1
  │          line2
  └─ line1   line3
     line2
     line3long
`,
		},
		{
			name: "last annotation shares row with first annotation",
			annots: []*Annot{
				{Col: 0, Lines: []string{"line1", "line2"}},
				{Col: 10, Lines: []string{"line1"}},
				{Col: 20, Lines: []string{"line1", "line2"}},
			},
			wantW: `
↑         ↑         ↑
└─ line1  └─ line1  └─ line1
   line2               line2
`,
		},
		{
			name: "complex annotation arrangement",
			annots: []*Annot{
				{Col: 0, Lines: []string{"line1", "line2", "line3", "line4", "line5", "line6", "line7", "line8verylongverylongverylong"}},
				{Col: 10, Lines: []string{"line1"}},
				{Col: 20, Lines: []string{"line1"}},
				{Col: 25, Lines: []string{"line1", "line2"}},
				{Col: 35, Lines: []string{"line1long", "line2", "line3"}},
				{Col: 40, Lines: []string{"line1", "line2", "line3", "line4"}},
			},
			wantW: `
↑         ↑         ↑    ↑         ↑    ↑
└─ line1  └─ line1  │    └─ line1  │    └─ line1
   line2            │       line2  │       line2
   line3            │              │       line3
   line4            └─ line1       │       line4
   line5                           │
   line6                           └─ line1long
   line7                              line2
   line8verylongverylongverylong      line3
`,
		},
		{
			name: "last annotation shares row with first annotation and first annotation uses available space",
			annots: []*Annot{
				{Col: 0, Lines: []string{"line1", "line2", "line3verylongverylong"}},
				{Col: 10, Lines: []string{"line1"}},
				{Col: 20, Lines: []string{"line1", "line2", "line3"}},
			},
			wantW: `
↑         ↑         ↑
│         └─ line1  └─ line1
│                      line2
└─ line1               line3
   line2
   line3verylongverylong
`,
		},
		{
			name: "first annotation uses indentation space of second annotation",
			annots: []*Annot{
				{Col: 0, Lines: []string{"line1"}},
				{Col: 8, Lines: []string{"line1", "line2", "line3"}},
			},
			wantW: `
↑       ↑
│       └─ line1
│          line2
└─ line1   line3
`,
		},
		{
			name: "one ranged annotation",
			annots: []*Annot{
				{Col: 0, ColEnd: 2},
			},
			wantW: `
└┬┘
 └─ 
`,
		},
		{
			name: "one narrow ranged annotation",
			annots: []*Annot{
				{Col: 0, ColEnd: 1},
			},
			wantW: `
├┘
└─ 
`,
		},
		{
			name: "two ranged annotation",
			annots: []*Annot{
				{Col: 0, ColEnd: 2},
				{Col: 6, ColEnd: 10},
			},
			wantW: `
└┬┘   └─┬─┘
 └─     └─ 
`,
		},
		{
			name: "mix ranged and arrowed annotation",
			annots: []*Annot{
				{Col: 0, ColEnd: 2},
				{Col: 4},
				{Col: 5},
				{Col: 7, ColEnd: 11},
				{Col: 13},
				{Col: 14},
			},
			wantW: `
└┬┘ ↑↑ └─┬─┘ ↑↑
 │  ││   │   │└─ 
 │  ││   │   └─ 
 │  ││   └─ 
 │  │└─ 
 │  └─ 
 └─ 
`,
		},
		{
			name: "remove second annotation with same column position",
			annots: []*Annot{
				{Col: 0, Lines: []string{"line1"}},
				{Col: 0, Lines: []string{"same column position"}},
			},
			wantW: `
↑
└─ line1
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			err := Write(w, tt.annots...)
			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if gotW := "\n" + w.String(); gotW != tt.wantW {
				t.Errorf("Write() gotW = %v, want %v", gotW, tt.wantW)
			}

			// A width that is large enough must not change the output.
			w.Reset()
			err = Renderer{Width: 1000}.Write(w, tt.annots...)
			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if gotW := "\n" + w.String(); gotW != tt.wantW {
				t.Errorf("Renderer{Width: 1000}.Write() gotW = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func TestRendererWidth(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		annots []*Annot
		wantW  string
	}{
		{
			name:  "annotation at right edge points left",
			width: 20,
			annots: []*Annot{
				{Col: 6, Lines: []string{"space"}},
				{Col: 19, Lines: []string{"exclamation mark"}},
			},
			wantW: `
      ↑            ↑
      └─ space     │
                   │
 exclamation mark ─┘
`,
		},
		{
			name:  "single annotation at last column with lines right aligned",
			width: 10,
			annots: []*Annot{
				{Col: 9, Lines: []string{"abc", "de"}},
			},
			wantW: `
         ↑
    abc ─┘
     de
`,
		},
		{
			name:  "wrap right pointing annotation",
			width: 20,
			annots: []*Annot{
				{Col: 0, Lines: []string{"one two three four five six"}},
			},
			wantW: `
↑
└─ one two three
   four five six
`,
		},
		{
			name:  "wrap left pointing annotation",
			width: 12,
			annots: []*Annot{
				{Col: 11, Lines: []string{"one two three four"}},
			},
			wantW: `
           ↑
  one two ─┘
    three
     four
`,
		},
		{
			name:  "right pointing annotation is limited by pipe of left pointing annotation",
			width: 20,
			annots: []*Annot{
				{Col: 0, Lines: []string{"alpha beta gamma"}},
				{Col: 19, Lines: []string{"x"}},
			},
			wantW: `
↑                  ↑
└─ alpha beta   x ─┘
   gamma
`,
		},
		{
			name:  "left pointing annots are close",
			width: 10,
			annots: []*Annot{
				{Col: 8, Lines: []string{"a"}},
				{Col: 9, Lines: []string{"b"}},
			},
			wantW: `
        ↑↑
     a ─┘│
         │
      b ─┘
`,
		},
		{
			name:  "wide characters at right edge",
			width: 10,
			annots: []*Annot{
				{Col: 9, Lines: []string{"漢字"}},
			},
			wantW: `
         ↑
   漢字 ─┘
`,
		},
		{
			name:  "wrap wide characters",
			width: 8,
			annots: []*Annot{
				{Col: 7, Lines: []string{"漢字漢"}},
			},
			wantW: `
       ↑
 漢字 ─┘
   漢
`,
		},
		{
			name:  "complex annotation arrangement with right edge",
			width: 50,
			annots: []*Annot{
				{Col: 0, Lines: []string{"line1", "line2", "line3", "line4"}},
				{Col: 10, Lines: []string{"line1"}},
				{Col: 20, Lines: []string{"line1"}},
				{Col: 25, Lines: []string{"line1", "line2"}},
				{Col: 35, Lines: []string{"line1long", "line2", "line3"}},
				{Col: 40, Lines: []string{"line1", "line2", "line3", "line4"}},
				{Col: 49, Lines: []string{"line1", "line2"}},
			},
			wantW: `
↑         ↑         ↑    ↑         ↑    ↑        ↑
└─ line1  └─ line1  │    └─ line1  │    │        │
   line2            │       line2  │    │        │
   line3            │              │    │        │
   line4            └─ line1       │    │        │
                                   │    │        │
                        line1long ─┘    │        │
                            line2       │        │
                            line3       │        │
                                        │        │
                                 line1 ─┘        │
                                 line2    line1 ─┘
                                 line3    line2
                                 line4
`,
		},
		{
			name:  "left pointing annots have correct vertical distance",
			width: 30,
			annots: []*Annot{
				{Col: 5, Lines: []string{"line1", "line2"}},
				{Col: 20, Lines: []string{"line1", "line2"}},
				{Col: 25, Lines: []string{"line1", "line2"}},
				{Col: 29, Lines: []string{"line1", "line2"}},
			},
			wantW: `
     ↑              ↑    ↑   ↑
     └─ line1       │    │   │
        line2       │    │   │
             line1 ─┘    │   │
             line2       │   │
                         │   │
                  line1 ─┘   │
                  line2      │
                             │
                      line1 ─┘
                      line2
`,
		},
		{
			name:  "left pointing annots utilise every space to the limit",
			width: 20,
			annots: []*Annot{
				{Col: 12, Lines: []string{"000", "100", "2000", "300000", "400000", "5000000"}},
				{Col: 18, Lines: []string{"line1", "line2", "line3", "line4"}},
				{Col: 19, Lines: []string{"line1"}},
			},
			wantW: `
            ↑     ↑↑
       000 ─┘     ││
       100        ││
      2000        ││
    300000        ││
    400000        ││
   5000000        ││
           line1 ─┘│
           line2   │
           line3   │
           line4   │
                   │
            line1 ─┘
`,
		},
		{
			name:  "mix ranged, empty and wrapped annotations in both directions",
			width: 40,
			annots: []*Annot{
				{Col: 0, ColEnd: 4, Lines: []string{"a range wraps when it is too long"}},
				{Col: 8},
				{Col: 9},
				{Col: 20, ColEnd: 26, Lines: []string{"points left", "and wraps a long line"}},
				{Col: 38},
				{Col: 39, Lines: []string{"end"}},
			},
			wantW: `
└─┬─┘   ↑↑          └──┬──┘           ↑↑
  │     │└─            │             ─┘│
  │     └─             │               │
  │                    │          end ─┘
  └─ a range wraps     │
     when it is too    │
     long              │
                       │
          points left ─┘
and wraps a long line
`,
		},
		{
			name:  "wrapped left pointing annotation between other annotations",
			width: 24,
			annots: []*Annot{
				{Col: 2, Lines: []string{"left side"}},
				{Col: 11, Lines: []string{"middle"}},
				{Col: 17, Lines: []string{"a b c d e f g h i j k"}},
				{Col: 23, Lines: []string{"right edge"}},
			},
			wantW: `
  ↑        ↑     ↑     ↑
  └─ left  │     │     │
     side  │     │     │
           │     │     │
   middle ─┘     │     │
                 │     │
a b c d e f g h ─┘     │
          i j k        │
                       │
           right edge ─┘
`,
		},
		{
			name:  "ranged annotation at right edge",
			width: 10,
			annots: []*Annot{
				{Col: 6, ColEnd: 9, Lines: []string{"end"}},
			},
			wantW: `
      └┬─┘
  end ─┘
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			err := Renderer{Width: tt.width}.Write(w, tt.annots...)
			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if gotW := "\n" + w.String(); gotW != tt.wantW {
				t.Errorf("Write() gotW = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func TestRendererLargeWidthEqualsString(t *testing.T) {
	annots := func() []*Annot {
		return []*Annot{
			{Col: 1, Lines: []string{"article"}},
			{Col: 4, ColEnd: 11, Lines: []string{"adjective"}},
			{Col: 22, ColEnd: 30, Lines: []string{
				"facts, information, and skills acquired",
				"through experience or education;",
				"the theoretical or practical understanding",
				"of a subject.",
			}},
			{Col: 48, Lines: []string{"comma"}},
		}
	}
	want := String(annots()...)
	if got := (Renderer{Width: 82}).String(annots()...); got != want {
		t.Errorf("Renderer{Width: 82}.String() = %v, want %v", got, want)
	}
}

func TestWriteTwiceWithSameAnnots(t *testing.T) {
	annots := []*Annot{
		{Col: 0, Lines: []string{"line1", "line2"}},
		{Col: 1, Lines: []string{"line1", "line2"}},
	}
	first := String(annots...)
	if second := String(annots...); second != first {
		t.Errorf("second String() = %v, want %v", second, first)
	}
}

func TestColExceedsWidthError(t *testing.T) {
	tests := []struct {
		name   string
		annots []*Annot
	}{
		{
			name:   "col is equal to width",
			annots: []*Annot{{Col: 10}},
		},
		{
			name:   "col end is equal to width",
			annots: []*Annot{{Col: 5, ColEnd: 10}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Renderer{Width: 10}.Write(io.Discard, tt.annots...)
			if _, ok := errors.AsType[*ColExceedsWidthError](err); !ok {
				t.Fatalf("Got error = %v, want ColExceedsWidthError", err)
			}
		})
	}
}

func TestColExceedsColEndError(t *testing.T) {
	tests := []struct {
		name   string
		annots []*Annot
	}{
		{
			name: "col is equal to col end",
			annots: []*Annot{
				{Col: 1, ColEnd: 1},
			},
		},
		{
			name: "col is higher than col end",
			annots: []*Annot{
				{Col: 2, ColEnd: 1},
			},
		},
		{
			name: "col exceeds col end error occurs before overlapping error",
			annots: []*Annot{
				{Col: 1, ColEnd: 1},
				{Col: 1, ColEnd: 2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Write(io.Discard, tt.annots...)
			if _, ok := errors.AsType[*ColExceedsColEndError](err); !ok {
				t.Fatalf("Got error = %v, want ColExceedsColEndError", err)
			}
		})
	}
}

func TestOverlapError(t *testing.T) {
	err := Write(io.Discard, []*Annot{
		{Col: 0, ColEnd: 1},
		{Col: 1, ColEnd: 2},
	}...)

	if _, ok := errors.AsType[*OverlapError](err); !ok {
		t.Fatalf("Got error = %v, want OverlapError", err)
	}
}
