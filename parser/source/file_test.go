package source

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestFile_LineData(t *testing.T) {
	tests := NewTestFileSet().
		AddCases("").
		AddCases("\n").
		AddCases("\n\n\n").
		AddCases("a b\nc  d\ne f g   \nh").
		AddCases("i j\nk\nl").
		AddCases("i j\nk\nl\n").
		AddCases("\ni j\nk\nl\n\n").
		AddCases("var x;\n\nvar y;\nparam a,b\nvar z\nz2\nz3\nz4").
		AddCases(`strings, json := [import("strings"), import("json")]

		const x = func() {
			{
				b()
			}
		}
		return x
		`).
		cases

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotD, gotErr := tt.f.LineData(tt.line)
			if !reflect.DeepEqual(string(gotD), tt.wantD) {
				t.Errorf("File(%q).LineData() gotD = %q, want %q", tt.f.Name, string(gotD), tt.wantD)
			}
			if gotErr != tt.wantErr {
				t.Errorf("File(%q).LineData() gotValid = %v, want %v", tt.f.Name, gotErr, tt.wantErr)
			}
		})
	}
}

type FileLineDataTestCase struct {
	name    string
	f       *File
	line    int
	wantD   string
	wantErr error
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
			f.AddLine(i + 1)
		}
	}

	return
}

func (set *TestFileSet) addFakeLines(f *File, lineStart, count int) {
	for i := 0; i < count; i++ {
		line := lineStart + i + 1
		set.cases = append(set.cases, &FileLineDataTestCase{
			name:    fmt.Sprintf("%s:%d(fake)", f.Name, line),
			f:       f,
			line:    line,
			wantErr: ErrIllegalLineNumber,
		})
	}
}

func (set *TestFileSet) AddCases(data string) *TestFileSet {
	f := set.Add(data)

	lines := strings.Split(data, "\n")

	if len(lines) == 0 {
		lines = []string{""}
	}

	for i, line := range lines {
		set.cases = append(set.cases, &FileLineDataTestCase{
			name:  fmt.Sprintf("%s:%d", f.Name, i+1),
			f:     f,
			line:  i + 1,
			wantD: line,
		})
	}

	set.addFakeLines(f, len(lines), 2)
	return set
}
