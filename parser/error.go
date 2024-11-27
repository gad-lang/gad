package parser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gad-lang/gad/parser/source"
)

// Error represents a parser error.
type Error struct {
	Pos source.SourceFilePos
	Msg string
}

func (e *Error) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('+') && e.Pos.File != nil {
			var (
				up, _   = f.Width()
				down, _ = f.Precision()
			)

			upl, downl, l := e.Pos.File.LineSliceDataUpDown(e.Pos.Line, up, down)

			fmt.Fprintln(f, e.Error())
			f.Write([]byte{'\n'})

			var (
				linef = "\t%-d| "
				lines []string
				add   = func(s ...*source.LineData) {
					for _, l := range s {
						lines = append(lines, fmt.Sprintf(linef+"%s", l.Line, string(l.Data)))
					}
				}
			)

			add(upl...)
			add(&source.LineData{Line: e.Pos.Line, Data: l})

			var (
				prefixCount = len(fmt.Sprintf(linef, e.Pos.Line))
				lineOfChar  = make([]byte, e.Pos.Column+prefixCount)
			)

			lineOfChar[0] = '\t'
			for i := 1; i < prefixCount; i++ {
				lineOfChar[i] = ' '
			}

			for i := 0; i < e.Pos.Column-1; i++ {
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
			f.Write([]byte(strings.Join(lines, "\n")))
		} else {
			f.Write([]byte(e.Error()))
		}
	}
}

func (e *Error) Error() string {
	if e.Pos.FileName() != "" || e.Pos.IsValid() {
		return fmt.Sprintf("Parse Error: %s\n\tat %s", e.Msg, e.Pos)
	}
	return fmt.Sprintf("Parse Error: %s", e.Msg)
}

// ErrorList is a collection of parser errors.
type ErrorList []*Error

// Add adds a new parser error to the collection.
func (p *ErrorList) Add(pos source.SourceFilePos, msg string) {
	*p = append(*p, &Error{pos, msg})
}

// Len returns the number of elements in the collection.
func (p ErrorList) Len() int {
	return len(p)
}

func (p ErrorList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p ErrorList) Less(i, j int) bool {
	e := &p[i].Pos
	f := &p[j].Pos

	if e.FileName() != f.FileName() {
		return e.FileName() < f.FileName()
	}
	if e.Line != f.Line {
		return e.Line < f.Line
	}
	if e.Column != f.Column {
		return e.Column < f.Column
	}
	return p[i].Msg < p[j].Msg
}

// Sort sorts the collection.
func (p ErrorList) Sort() {
	sort.Sort(p)
}

func (p ErrorList) Format(f fmt.State, verb rune) {
	l := len(p)
	switch l {
	case 0:
		f.Write([]byte("no errors"))
	case 1:
		p[0].Format(f, verb)
	default:
		p[0].Format(f, verb)
		fmt.Fprintf(f, " (and %d more errors)", l-1)
	}
}

func (p ErrorList) Error() string {
	switch len(p) {
	case 0:
		return "no errors"
	case 1:
		return p[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", p[0], len(p)-1)
}

// Err returns an error.
func (p ErrorList) Err() error {
	if len(p) == 0 {
		return nil
	}
	return p
}
