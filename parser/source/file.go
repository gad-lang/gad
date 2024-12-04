package source

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// SourceFilePos represents a position information in the file.
type SourceFilePos struct {
	File   *File // filename, if any
	Offset int   // offset, starting at 0
	Line   int   // line number, starting at 1
	Column int   // column number, starting at 1 (byte count)
}

// IsValid returns true if the position is valid.
func (p SourceFilePos) IsValid() bool {
	return p.Line > 0
}

func (p SourceFilePos) FileName() string {
	if p.File != nil {
		return p.File.Name
	}
	return ""
}

// String returns a string in one of several forms:
//
//	file:line:column    valid position with file name
//	file:line           valid position with file name but no column (column == 0)
//	line:column         valid position without file name
//	line                valid position without file name and no column (column == 0)
//	file                invalid position with file name
//	-                   invalid position without file name
func (p SourceFilePos) String() string {
	s := p.FileName()

	if p.IsValid() {
		if s != "" {
			s += ":"
		}
		s += fmt.Sprintf("%d", p.Line)
		if p.Column != 0 {
			s += fmt.Sprintf(":%d", p.Column)
		}
	}
	if s == "" {
		s = "-"
	}
	return s
}

func (p SourceFilePos) TraceLines(w io.Writer, up, down int) {
	if p.File != nil {
		p.File.TraceLines(w, p.Line, p.Column, up, down)
	}
}

// SourceFileSet represents a set of source files.
type SourceFileSet struct {
	Base     int     // base offset for the next file
	Files    []*File // list of files in the order added to the set
	LastFile *File   // cache of last file looked up
}

// NewFileSet creates a new file set.
func NewFileSet() *SourceFileSet {
	return &SourceFileSet{
		Base: 1, // 0 == NoPos
	}
}

// AddFileData adds a new file in the file set with data.
func (s *SourceFileSet) AddFileData(filename string, base int, data []byte) *File {
	f := s.AddFile(filename, base, len(data))
	f.Data = data
	return f
}

// AddFile adds a new file in the file set.
func (s *SourceFileSet) AddFile(filename string, base, size int) *File {
	if base < 0 {
		base = s.Base
	}
	if base < s.Base || size < 0 {
		panic("illegal base or size")
	}

	f := &File{
		set:   s,
		Name:  filename,
		Base:  base,
		Size:  size,
		Lines: []int{0},
		Index: len(s.Files),
	}
	base += size + 1 // +1 because EOF also has a position
	if base < 0 {
		panic("offset overflow (> 2G of source code in file set)")
	}

	// add the file to the file set
	s.Base = base
	s.Files = append(s.Files, f)
	s.LastFile = f
	return f
}

// File returns the file that contains the position p. If no such file is
// found (for instance for p == NoPos), the result is nil.
func (s *SourceFileSet) File(p Pos) (f *File) {
	if p != NoPos {
		f = s.file(p)
	}
	return
}

// Position converts a SourcePos p in the fileset into a SourceFilePos value.
func (s *SourceFileSet) Position(p Pos) (pos SourceFilePos) {
	if p != NoPos {
		if f := s.file(p); f != nil {
			return f.SafePosition(p)
		}
	}
	return
}

func (s *SourceFileSet) file(p Pos) *File {
	// common case: p is in last file
	f := s.LastFile
	if f != nil && f.Base <= int(p) && int(p) <= f.Base+f.Size {
		return f
	}

	// p is not in last file - search all files
	if i := searchFiles(s.Files, int(p)); i >= 0 {
		f := s.Files[i]

		// f.base <= int(p) by definition of searchFiles
		if int(p) <= f.Base+f.Size {
			s.LastFile = f // race is ok - s.last is only a cache
			return f
		}
	}
	return nil
}

func searchFiles(a []*File, x int) int {
	return sort.Search(len(a), func(i int) bool { return a[i].Base > x }) - 1
}

// File represents a source file.
type File struct {
	// File set for the file
	set *SourceFileSet
	// File name as provided to AddFile
	Name string
	// SourcePos value range for this file is [base...base+size]
	Base int
	// File size as provided to AddFile
	Size int
	// Lines contains the offset of the first character for each line
	// (the first entry is always 0)
	Lines []int
	// Index is a index of `set`
	Index int
	// Data is data
	Data []byte
}

// Set returns SourceFileSet.
func (f *File) Set() *SourceFileSet {
	return f.set
}

// LineCount returns the current number of lines.
func (f *File) LineCount() int {
	return len(f.Lines)
}

// AddLine adds a new line.
func (f *File) AddLine(offset int) {
	if offset >= f.Size {
		return
	}

	lc := len(f.Lines)
	if lc == 0 {
		f.Lines = append(f.Lines, offset)
	} else {
		for i := lc; i > 0; i-- {
			if off := f.Lines[i-1]; off == offset {
				return
			} else if off > offset {
				f.Lines = append(f.Lines, -1)
				copy(f.Lines[i:], f.Lines[i-1:])
				f.Lines[i-1] = offset
				return
			}
		}
		f.Lines = append(f.Lines, offset)
	}
}

// MustLineStart returns the position of the first character in the line if line is valid, other else -1.
func (f *File) MustLineStart(line int) Pos {
	if line < 1 || line > len(f.Lines) {
		return -1
	}
	return Pos(f.Base + f.Lines[line-1])
}

// LineStart returns the position of the first character in the line.
func (f *File) LineStart(line int) Pos {
	if line < 1 {
		panic("illegal line number (line numbering starts at 1)")
	}
	if line > len(f.Lines) {
		panic("illegal line number")
	}
	return Pos(f.Base + f.Lines[line-1])
}

// FileSetPos returns the position in the file set.
func (f *File) FileSetPos(offset int) Pos {
	if offset > f.Size {
		panic("illegal file offset")
	}
	return Pos(f.Base + offset)
}

// Offset translates the file set position into the file offset.
func (f *File) Offset(p Pos) int {
	if int(p) < f.Base || int(p) > f.Base+f.Size {
		panic("illegal SourcePos value")
	}
	return int(p) - f.Base
}

// Line returns the line of given position.
func (f *File) Line(p Pos) int {
	return f.Position(p).Line
}

// Position translates the file set position into the file position.
func (f *File) Position(p Pos) (pos SourceFilePos) {
	if p != NoPos {
		if int(p) < f.Base || int(p) > f.Base+f.Size {
			panic("illegal SourcePos value")
		}
		pos = f.SafePosition(p)
	}
	return
}

func (f *File) SafePosition(p Pos) (pos SourceFilePos) {
	offset := int(p) - f.Base
	pos.Offset = offset
	pos.File = f
	_, pos.Line, pos.Column = f.Unpack(offset)
	return
}

func (f *File) Unpack(offset int) (filename string, line, column int) {
	filename = f.Name
	if i := searchInts(f.Lines, offset); i >= 0 {
		line, column = i+1, offset-f.Lines[i]+1
	}
	return
}

func (f *File) LineIndexOf(p Pos) int {
	l := len(f.Lines)
	for i := l; i > 0; i-- {
		p2 := f.Lines[i-1]
		if p2 <= int(p) {
			return i - 1
		}
	}
	return 0
}

func (f *File) LinePos(p Pos) Pos {
	l := len(f.Lines)
	for i := l; i > 0; i-- {
		p2 := f.Lines[i-1]
		if p2 <= int(p) {
			if i == l {
				// last line, first column
				return Pos(f.Lines[i-1] + 1)
			}
			return Pos(f.Lines[i-1])
		}
	}
	return p
}

func (f *File) NextLinePos(p Pos) Pos {
	l := len(f.Lines)
	for i := l; i > 0; i-- {
		p2 := f.Lines[i-1]
		if p2 <= int(p) {
			if i == l {
				// last line, first column
				c1 := Pos(f.Lines[i-1] + 1)
				if p <= c1 {
					c1--
				}
				return c1
			}
			return Pos(f.Lines[i-1])
		}
	}
	return p
}

// LineData return line data
func (f *File) LineData(line int) (d []byte, valid bool) {
	start, end := f.MustLineStart(line), f.MustLineStart(line+1)
	if start < 1 {
		return nil, false
	}
	valid = true
	if end > 0 {
		end--
		d = f.Data[start-1 : end-1]
	} else {
		d = f.Data[start-1:]
	}
	return
}

// LineSliceData return slice data of lines
func (f *File) LineSliceData(lineStart, count int) (s []*LineData) {
	for i := 0; i < count; i++ {
		if d, ok := f.LineData(lineStart + i); ok {
			s = append(s, &LineData{
				Line: lineStart + i,
				Data: d,
			})
		}
	}
	return
}

// LineSliceDataUpDown returns data of line and slices of up and down lines
func (f *File) LineSliceDataUpDown(line, upCount, downCount int) (up, down []*LineData, s []byte) {
	var ok bool
	if s, ok = f.LineData(line); !ok {
		return
	}

	if line > 1 {
		firstLine := line - upCount
		if firstLine < 1 {
			firstLine = 1
		}

		up = f.LineSliceData(firstLine, line-firstLine)
	}

	if lastLine := len(f.Lines); line < lastLine {
		endLine := line + downCount
		if endLine > lastLine {
			endLine = lastLine
		}
		down = f.LineSliceData(line+1, endLine-line)
	}

	return
}

func (f *File) TraceLines(s io.Writer, line, column, up, down int) {
	upl, downl, l := f.LineSliceDataUpDown(line, up, down)
	s.Write([]byte{'\n'})

	var (
		linef = "\t%5d| "
		lines []string
		add   = func(s ...*LineData) {
			for _, l := range s {
				lines = append(lines, fmt.Sprintf(linef+"%s", l.Line, string(l.Data)))
			}
		}
	)

	add(upl...)
	add(&LineData{Line: line, Data: l})

	var (
		prefixCount = len(fmt.Sprintf(linef, line))
		lineOfChar  = make([]byte, column+1+prefixCount)
	)

	lineOfChar[0] = '\t'
	for i := 1; i < prefixCount; i++ {
		lineOfChar[i] = ' '
	}

	for i := 0; i < column; i++ {
		b := l[i]
		switch b {
		case '\t':
			b = '\t'
		default:
			b = ' '
		}
		lineOfChar[prefixCount+i] = b
	}
	lineOfChar[len(lineOfChar)-1] = '^'
	lines = append(lines, string(lineOfChar))
	add(downl...)
	s.Write([]byte(strings.Join(lines, "\n")))
}

type LineData struct {
	Line int
	Data []byte
}

func searchInts(a []int, x int) int {
	// This function body is a manually inlined version of:
	//   return sort.Search(len(a), func(i int) bool { return a[i] > x }) - 1
	i, j := 0, len(a)
	for i < j {
		h := i + (j-i)/2 // avoid overflow when computing h
		// i ≤ h < j
		if a[h] <= x {
			i = h + 1
		} else {
			j = h
		}
	}
	return i - 1
}
