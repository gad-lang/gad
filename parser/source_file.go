// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// Copyright (c) 2019 Daniel Kang.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE.tengo file.

// Copyright 2009 The ToInterface Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.golang file.

package parser

import (
	"fmt"
	"sort"

	"github.com/gad-lang/gad/parser/source"
)

// SourceFilePos represents a position information in the file.
type SourceFilePos struct {
	Filename string // filename, if any
	Offset   int    // offset, starting at 0
	Line     int    // line number, starting at 1
	Column   int    // column number, starting at 1 (byte count)
}

// IsValid returns true if the position is valid.
func (p SourceFilePos) IsValid() bool {
	return p.Line > 0
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
	s := p.Filename
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

// SourceFileSet represents a set of source files.
type SourceFileSet struct {
	Base     int           // base offset for the next file
	Files    []*SourceFile // list of files in the order added to the set
	LastFile *SourceFile   // cache of last file looked up
}

// NewFileSet creates a new file set.
func NewFileSet() *SourceFileSet {
	return &SourceFileSet{
		Base: 1, // 0 == NoPos
	}
}

// AddFile adds a new file in the file set.
func (s *SourceFileSet) AddFile(filename string, base, size int) *SourceFile {
	if base < 0 {
		base = s.Base
	}
	if base < s.Base || size < 0 {
		panic("illegal base or size")
	}

	f := &SourceFile{
		set:   s,
		Name:  filename,
		Base:  base,
		Size:  size,
		Lines: []int{0},
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
func (s *SourceFileSet) File(p source.Pos) (f *SourceFile) {
	if p != source.NoPos {
		f = s.file(p)
	}
	return
}

// Position converts a SourcePos p in the fileset into a SourceFilePos value.
func (s *SourceFileSet) Position(p source.Pos) (pos SourceFilePos) {
	if p != source.NoPos {
		if f := s.file(p); f != nil {
			return f.SafePosition(p)
		}
	}
	return
}

func (s *SourceFileSet) file(p source.Pos) *SourceFile {
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

func searchFiles(a []*SourceFile, x int) int {
	return sort.Search(len(a), func(i int) bool { return a[i].Base > x }) - 1
}

// SourceFile represents a source file.
type SourceFile struct {
	// SourceFile set for the file
	set *SourceFileSet
	// SourceFile name as provided to AddFile
	Name string
	// SourcePos value range for this file is [base...base+size]
	Base int
	// SourceFile size as provided to AddFile
	Size int
	// Lines contains the offset of the first character for each line
	// (the first entry is always 0)
	Lines []int
}

// Set returns SourceFileSet.
func (f *SourceFile) Set() *SourceFileSet {
	return f.set
}

// LineCount returns the current number of lines.
func (f *SourceFile) LineCount() int {
	return len(f.Lines)
}

// AddLine adds a new line.
func (f *SourceFile) AddLine(offset int) {
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

// LineStart returns the position of the first character in the line.
func (f *SourceFile) LineStart(line int) source.Pos {
	if line < 1 {
		panic("illegal line number (line numbering starts at 1)")
	}
	if line > len(f.Lines) {
		panic("illegal line number")
	}
	return source.Pos(f.Base + f.Lines[line-1])
}

// FileSetPos returns the position in the file set.
func (f *SourceFile) FileSetPos(offset int) source.Pos {
	if offset > f.Size {
		panic("illegal file offset")
	}
	return source.Pos(f.Base + offset)
}

// Offset translates the file set position into the file offset.
func (f *SourceFile) Offset(p source.Pos) int {
	if int(p) < f.Base || int(p) > f.Base+f.Size {
		panic("illegal SourcePos value")
	}
	return int(p) - f.Base
}

// Line returns the line of given position.
func (f *SourceFile) Line(p source.Pos) int {
	return f.Position(p).Line
}

// Position translates the file set position into the file position.
func (f *SourceFile) Position(p source.Pos) (pos SourceFilePos) {
	if p != source.NoPos {
		if int(p) < f.Base || int(p) > f.Base+f.Size {
			panic("illegal SourcePos value")
		}
		pos = f.SafePosition(p)
	}
	return
}

func (f *SourceFile) SafePosition(p source.Pos) (pos SourceFilePos) {
	offset := int(p) - f.Base
	pos.Offset = offset
	pos.Filename, pos.Line, pos.Column = f.Unpack(offset)
	return
}

func (f *SourceFile) Unpack(offset int) (filename string, line, column int) {
	filename = f.Name
	if i := searchInts(f.Lines, offset); i >= 0 {
		line, column = i+1, offset-f.Lines[i]+1
	}
	return
}

func (f *SourceFile) LineIndexOf(p source.Pos) int {
	l := len(f.Lines)
	for i := l; i > 0; i-- {
		p2 := f.Lines[i-1]
		if p2 <= int(p) {
			return i - 1
		}
	}
	return 0
}

func (f *SourceFile) LinePos(p source.Pos) source.Pos {
	l := len(f.Lines)
	for i := l; i > 0; i-- {
		p2 := f.Lines[i-1]
		if p2 <= int(p) {
			if i == l {
				// last line, first column
				return source.Pos(f.Lines[i-1] + 1)
			}
			return source.Pos(f.Lines[i-1])
		}
	}
	return p
}

func (f *SourceFile) NextLinePos(p source.Pos) source.Pos {
	l := len(f.Lines)
	for i := l; i > 0; i-- {
		p2 := f.Lines[i-1]
		if p2 <= int(p) {
			if i == l {
				// last line, first column
				c1 := source.Pos(f.Lines[i-1] + 1)
				if p <= c1 {
					c1--
				}
				return c1
			}
			return source.Pos(f.Lines[i-1])
		}
	}
	return p
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
