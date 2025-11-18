package source

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unsafe"
)

// Data represents a File data
type Data struct {
	// Bytes is data of file
	data []byte
	// Lines contains the offset of the first character for each line
	// (the first entry is always 0)
	lines []int
}

func NewData(data []byte) *Data {
	last := len(data) - 1
	if pos := bytes.IndexByte(data, '\r'); pos >= 0 {
		// if line sep is only CR, replaces to EOL
		if pos < last && data[pos] != '\n' {
			for i, b := range data {
				if b == '\r' && i < last && data[i+1] != '\n' {
					data[i] = '\n'
				}
			}
		}
	}

	return &Data{data: data}
}

// SplitLines return a splited data into lines
func (d *Data) SplitLines() (r [][]byte) {
	d.check()

	r = make([][]byte, len(d.lines))
	lastLine := len(d.lines)

	for i := range r {
		line := i + 1
		startOffset, _ := d.LineOffset(line)
		if line == lastLine {
			r[i] = d.data[startOffset:]
		} else {
			nextOffset, _ := d.LineOffset(line + 1)
			endOffset := nextOffset - 1 // before LF
			r[i] = d.data[startOffset:endOffset]
		}
	}

	return
}

// Bytes return the data bytes
func (d *Data) Bytes() []byte {
	return d.data
}

// ToString return the data as string without copy
func (d *Data) ToString() string {
	return *(*string)(unsafe.Pointer(&d.data))
}

func (d *Data) check() {
	if d.lines == nil {
		d.lines = []int{0}
		for i, c := range d.data {
			if c == '\n' {
				d.lines = append(d.lines, i+1)
			}
		}
	}
}

// NumLines return a number of lines
func (d *Data) NumLines() int {
	d.check()
	return len(d.lines)
}

// Pack returns the offset of position by line and column.
func (d *Data) Pack(line, column int) (offset int, err error) {
	if offset, err = d.LineOffset(line); err != nil {
		return
	}
	offset += column - 1
	return
}

// Unpack converts offset to line and column pair.
func (d *Data) Unpack(offset int) (line, column int) {
	d.check()
	if i := searchInts(d.lines, offset); i >= 0 {
		line, column = i+1, offset-d.lines[i]+1
	}
	return
}

// LineOffset returns the offset of the first character in the line or error if line number isn't valid.
func (d *Data) LineOffset(line int) (offset int, _ error) {
	d.check()
	if line < 1 {
		return -1, ErrIllegalMinimalLineNumber
	}
	if line > len(d.lines) {
		return -1, ErrIllegalLineNumber
	}

	offset = d.lines[line-1]
	return
}

// LineData return line data, of error if isn't valid line.
func (d *Data) LineData(line int) (r []byte, _ error) {
	start, err := d.LineOffset(line)
	if err != nil {
		return nil, err
	}

	data := d.data[start:]
	end := bytes.IndexByte(data, '\n')

	if end >= 0 {
		r = data[:end]
	} else {
		r = data
	}
	return
}

// LinesData return slice data of lines
func (d *Data) LinesData(lineStart, count int) (s []*LineData) {
	for i := 0; i < count; i++ {
		if d, err := d.LineData(lineStart + i); err == nil && len(d) > 0 {
			s = append(s, &LineData{
				Line: lineStart + i,
				Data: d,
			})
		}
	}
	return
}

// LineSliceDataUpDown returns data of line and slices of up and down lines
func (d *Data) LineSliceDataUpDown(line, upCount, downCount int) (up, down []*LineData, s []byte) {
	var err error
	if s, err = d.LineData(line); err != nil {
		return
	}

	if line > 1 {
		firstLine := line - upCount
		if firstLine < 1 {
			firstLine = 1
		}

		up = d.LinesData(firstLine, line-firstLine)
	}

	if lastLine := len(d.lines); line < lastLine {
		endLine := line + downCount
		if endLine > lastLine {
			endLine = lastLine
		}
		down = d.LinesData(line+1, endLine-line)
	}

	return
}

func (d *Data) Slice(startLine, numLines int) (_ []byte, err error) {
	var startIndex, endIndex int
	if startIndex, err = d.LineOffset(startLine); err != nil {
		return
	}

	endLine := startLine + numLines

	if endLine == len(d.lines)+1 {
		return d.data[startIndex:], nil
	} else if endIndex, err = d.LineOffset(endLine); err != nil {
		return
	}

	return d.data[startIndex : endIndex-1], nil
}

func (d *Data) TraceLines(s io.Writer, line, column, up, down int) {
	upl, downl, l := d.LineSliceDataUpDown(line, up, down)
	var (
		linef = "     %5d| "
		lines []string
		add   = func(s ...*LineData) {
			for _, l := range s {
				lines = append(lines, fmt.Sprintf(linef+"%s", l.Line, string(l.Data)))
			}
		}
	)

	add(upl...)
	curLine := []rune(fmt.Sprintf(linef+"%s", line, string(l)))

loop:
	for i, b := range curLine {
		switch b {
		case ' ', '\t':
		default:
			i := i - 2
			if i < 0 {
				i = 0
			}
			curLine[i] = '🠆'
			break loop
		}
	}

	lines = append(lines, string(curLine))

	var (
		prefixCount = len(fmt.Sprintf(linef, line))
		lineOfChar  = make([]byte, column+prefixCount)
	)

	lineOfChar[0] = ' '
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

// LineData represents a Data of line
type LineData struct {
	Line int
	Data []byte
}
