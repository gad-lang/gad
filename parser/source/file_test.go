package source

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestFile_LineData(t *testing.T) {
	tests := NewTestFileSet().
		AddCases("a b\nc  d\ne f g   \nh").
		AddCases("i j\nk\nl").
		AddCases(`strings, json := [import("strings"), import("json")]

const main = func($slots={}) {
	.{
		b()
	}
}
return {main: main}
`).
		cases

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotD, gotValid := tt.f.LineData(tt.line)
			if !reflect.DeepEqual(string(gotD), tt.wantD) {
				t.Errorf("File(%q).LineData() gotD = %q, want %q", tt.f.Name, string(gotD), tt.wantD)
			}
			if gotValid != tt.wantValid {
				t.Errorf("File(%q).LineData() gotValid = %v, want %v", tt.f.Name, gotValid, tt.wantValid)
			}
		})
	}
}

func TestFile_LineData2(t *testing.T) {
	tests := NewTestFileSet().AddCases(`strings, json := [import("strings"), import("json")]

const main = func($slots={}) {
	.{
		b()
	}
}
return {main: main}
`).cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotD, gotValid := tt.f.LineData(tt.line)
			if !reflect.DeepEqual(string(gotD), tt.wantD) {
				t.Errorf("LineData() gotD = %q, want %q", string(gotD), tt.wantD)
			}
			if gotValid != tt.wantValid {
				t.Errorf("LineData() gotValid = %v, want %v", gotValid, tt.wantValid)
			}
		})
	}
}

func TestFile_LineData3(t *testing.T) {
	tests := NewTestFileSet().AddCases("").cases

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotD, gotValid := tt.f.LineData(tt.line)
			if !reflect.DeepEqual(string(gotD), tt.wantD) {
				t.Errorf("LineData() gotD = %q, want %q", string(gotD), tt.wantD)
			}
			if gotValid != tt.wantValid {
				t.Errorf("LineData() gotValid = %v, want %v", gotValid, tt.wantValid)
			}
		})
	}
}

type FileLineDataTestCase struct {
	name      string
	f         *File
	line      int
	wantD     string
	wantValid bool
}

type TestFileSet struct {
	*SourceFileSet
	cases []*FileLineDataTestCase
}

func NewTestFileSet() *TestFileSet {
	return &TestFileSet{SourceFileSet: NewFileSet()}
}

func (set *TestFileSet) Add(data string) (f *File) {
	b := []byte(data)
	f = set.AppendFileData(fmt.Sprintf("mem-file-%d", len(set.Files)), b)

	for i, c := range b {
		if c == '\n' {
			f.AddLine(i - 1)
		}
	}

	return
}

func (set *TestFileSet) addFakeLines(f *File, lineStart, count int) {
	for i := 0; i < count; i++ {
		line := lineStart + i + 1
		set.cases = append(set.cases, &FileLineDataTestCase{
			name:      fmt.Sprintf("%s:%d(fake)", f.Name, line),
			f:         f,
			line:      line,
			wantValid: false,
		})
	}
}

func (set *TestFileSet) AddCases(data string) *TestFileSet {
	f := set.Add(data)

	if len(data) == 0 {
		set.addFakeLines(f, 0, 2)
	} else {
		lines := strings.Split(data, "\n")

		for i, line := range lines {
			set.cases = append(set.cases, &FileLineDataTestCase{
				name:      fmt.Sprintf("%s:%d", f.Name, i+1),
				f:         f,
				line:      i + 1,
				wantD:     line,
				wantValid: true,
			})
		}

		set.addFakeLines(f, len(lines), 2)
	}
	return set
}
