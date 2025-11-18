package source

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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
			gotD, gotErr := tt.f.Data.LineData(tt.line)
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
	*FileSet
	cases []*FileLineDataTestCase
}

func NewTestFileSet() *TestFileSet {
	return &TestFileSet{FileSet: NewFileSet()}
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

func TestFile_Slice(t *testing.T) {
	const numLines = 15
	type testCase struct {
		name      string
		lines     []string
		data      []byte
		startLine int
		numLines  int
		posByte   []byte
		srcPos    []FilePos
		wantPos   []FilePos
	}

	var (
		set       = NewFileSet()
		slicesSet = NewFileSet()
		dataList  = make([]string, 15)
		tests     []testCase
		lineIndex int
	)

	for i := range dataList {
		var (
			lines     = make([]string, numLines)
			startLine = (i * numLines) + 1
			tc        = testCase{
				name:      fmt.Sprintf("test-%d", i+1),
				startLine: startLine,
				numLines:  numLines,
			}
		)

		for j := range lines {
			line := fmt.Sprintf("%d.sliced-file-%d,line-%d", lineIndex+1, i+1, j+1)
			lineIndex++

			lines[j] = line
			for r, b := range []byte(line) {
				tc.posByte = append(tc.posByte, b)
				tc.srcPos = append(tc.srcPos, FilePos{
					Line:   startLine + j,
					Column: r + 1,
				})
				tc.wantPos = append(tc.wantPos, FilePos{
					Line:   j + 1,
					Column: r + 1,
				})
			}
		}

		data := strings.Join(lines, "\n")

		tc.data = []byte(data)
		tc.lines = lines

		tests = append(tests, tc)
		dataList[i] = data
	}

	joinedFile := set.AppendFileData("joined", []byte(strings.Join(dataList, "\n")))
	t.Run("joined", func(t *testing.T) {
		expectedLines := bytes.Split(joinedFile.Data.data, []byte("\n"))
		gotLines := joinedFile.Data.SplitLines()
		if !reflect.DeepEqual(expectedLines, gotLines) {
			t.Errorf("joinedFile.Data.SplitLines() gotLines = %q, want %q", gotLines, expectedLines)
		}
	})

	for _, tt := range tests[9:] {
		t.Run(tt.name, func(t *testing.T) {
			sf, err := joinedFile.Slice(slicesSet, tt.name, tt.startLine, tt.numLines)
			require.NoError(t, err)
			require.Equal(t, string(tt.data), string(sf.Data.Bytes()))

			for i := range tt.srcPos[7:13] {
				srcPos := tt.srcPos[i]
				wantPos := tt.wantPos[i]
				name := fmt.Sprintf("%s#%d(%s-%s)", tt.name, i, srcPos.PositionString(), wantPos.PositionString())
				t.Run(name, func(t *testing.T) {
					gotPos, err := sf.CastPos(srcPos)
					require.NoError(t, err)
					require.Equal(t, wantPos.PositionString(), gotPos.PositionString())
					gotByte := sf.Data.Bytes()[gotPos.Offset]
					joinedPositionOffset, err := joinedFile.Data.Pack(srcPos.Line, srcPos.Column)
					require.NoError(t, err)
					joinedByte := joinedFile.Data.Bytes()[joinedPositionOffset]
					require.Equal(t, string(joinedByte), string(gotByte))
					require.Equal(t, string(tt.posByte[i]), string(gotByte))
				})
			}
		})
	}
}
