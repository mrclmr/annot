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
			var colExceedsColEndError *ColExceedsColEndError
			if !errors.As(err, &colExceedsColEndError) {
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

	var overlapError *OverlapError
	if !errors.As(err, &overlapError) {
		t.Fatalf("Got error = %v, want OverlapError", err)
	}
}
