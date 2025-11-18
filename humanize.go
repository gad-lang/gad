package gad

import (
	"fmt"
	"io"
	"strconv"

	"github.com/gad-lang/gad/parser"
)

type UpDownLines struct {
	Up, Down int
}

type ErrorHumanizing struct {
	Current, Other UpDownLines
}

func (h *ErrorHumanizing) Humanize(out io.Writer, err error) {
	var (
		up, down = h.Current.Up, h.Current.Down
	)

	if up == 0 {
		up = 3
	}

	if down == 0 {
		down = 3
	}

	switch t := err.(type) {
	case *RuntimeError:
		fmt.Fprintf(out, "%+v\n\n", t)
		if st := t.StackTrace(); len(st) > 0 {
			for _, stPos := range st[:len(st)-1] {
				pos := t.FileSet().Position(stPos.Pos())
				fmt.Fprintf(out, pos.String()+":\n")
				pos.File.Data.TraceLines(out, pos.Line, pos.Column, h.Other.Up, h.Other.Down)
				out.Write([]byte("\n"))
			}

			pos := t.FileSet().Position(st[len(st)-1].Pos())
			fmt.Fprintf(out, pos.String()+":\n")
			pos.File.Data.TraceLines(out, pos.Line, pos.Column, up, down)
		}
	case parser.ErrorList, *CompilerError:
		fmt.Fprintf(out, "%+"+strconv.Itoa(up)+"."+strconv.Itoa(down)+"v\n", t)
	default:
		fmt.Fprintf(out, "ERROR: %v\n", err)
	}
}
