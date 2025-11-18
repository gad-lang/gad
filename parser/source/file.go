package source

import (
	"fmt"
)

// FilePos represents a position information in the file.
type FilePos struct {
	File   *File // filename, if any
	Offset int   // offset, starting at 0
	Line   int   // line number, starting at 1
	Column int   // column number, starting at 1 (byte count)
}

func (p FilePos) Pos() Pos {
	return Pos(p.File.Base + p.Offset)
}

// IsValid returns true if the position is valid.
func (p FilePos) IsValid() bool {
	return p.Line > 0
}

func (p FilePos) FileName() string {
	if p.File != nil {
		return p.File.Name
	}
	return ""
}

// String returns a string in one of several forms:
//
//	[empty]             no valid pos
//	file:line:column    valid position with file name
//	file:line           valid position with file name but no column (column == 0)
//	line:column         valid position without file name
//	line                valid position without file name and no column (column == 0)
//	file                invalid position with file name
//	-                   invalid position without file name
func (p FilePos) String() string {
	if !p.IsValid() {
		return ""
	}

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

func (p FilePos) PositionString() (s string) {
	if p.IsValid() {
		s = fmt.Sprintf("%d", p.Line)
		if p.Column != 0 {
			s += fmt.Sprintf(":%d", p.Column)
		}
	}
	return
}

// SliceFile represets a slice source File
type SliceFile struct {
	*File
	source    *File
	startLine int
	numLines  int
}

// CastPos cast Pos from source File to sliced File Pos
func (f *SliceFile) CastPos(srcPos FilePos) (pos FilePos, err error) {
	var srcLineStartIndex int
	if srcLineStartIndex, err = f.source.Data.LineOffset(srcPos.Line); err != nil {
		return
	}

	var slicedStartIndex int
	if slicedStartIndex, err = f.source.Data.LineOffset(f.startLine); err != nil {
		return
	}

	f.Data.check()

	offset := srcLineStartIndex - slicedStartIndex
	offset += srcPos.Column - 1
	pos.Line, pos.Column = f.Data.Unpack(offset)
	pos.Offset = offset
	pos.File = f.File
	return
}

// File represents a source file.
type File struct {
	// File set for the file
	set *FileSet
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
	Data *Data
}

// Set returns FileSet.
func (f *File) Set() *FileSet {
	return f.set
}

// LineCount returns the current number of lines.
func (f *File) LineCount() int {
	return len(f.Lines)
}

// AddLine adds a new line by first data offset (after LF char).
func (f *File) AddLine(offset int) {
	if offset > f.Size {
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

// FileSetPos returns the position in the file set.
func (f *File) FileSetPos(offset int) (Pos, error) {
	if offset > f.Size {
		return NoPos, ErrIllegalFileOffset
	}
	return Pos(f.Base + offset), nil
}

// Offset translates the file set position into the file offset.
func (f *File) Offset(p Pos) (int, error) {
	if int(p) < f.Base || int(p) > f.Base+f.Size {
		return -1, ErrIllegalPosition
	}
	return int(p) - f.Base, nil
}

// Position translates the file set position into the file position.
func (f *File) Position(p Pos) (pos FilePos, err error) {
	if p != NoPos {
		if int(p) < f.Base || int(p) > f.Base+f.Size {
			err = ErrIllegalPosition
			return
		}
		pos = f.SafePosition(p)
	}
	return
}

// SafePosition return a position without validations
func (f *File) SafePosition(p Pos) (pos FilePos) {
	offset := int(p) - f.Base
	pos.Offset = offset
	pos.File = f
	_, pos.Line, pos.Column = f.Unpack(offset)
	return
}

// Unpack casts offset to filename, line and column.
func (f *File) Unpack(offset int) (filename string, line, column int) {
	filename = f.Name
	if i := searchInts(f.Lines, offset); i >= 0 {
		line, column = i+1, offset-f.Lines[i]+1
	}
	return
}

func (f *File) Slice(slicedSet *FileSet, name string, startLine, numLines int) (_ *SliceFile, err error) {
	var data []byte
	if data, err = f.Data.Slice(startLine, numLines); err != nil {
		return
	}

	return &SliceFile{
		File:      slicedSet.AppendFileData(name, data),
		source:    f,
		startLine: startLine,
		numLines:  numLines,
	}, nil
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
