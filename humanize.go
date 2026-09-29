package gad

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

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

	// The error may come wrapped — gadx's "render main.gad: …", an
	// application's own —: the gad error inside is found, and what wraps it
	// is said first, so neither where it was nor the position is lost.
	var (
		rt *RuntimeError
		ce *CompilerError
		el parser.ErrorList
	)
	switch {
	case errors.As(err, &rt):
		h.context(out, err, rt)
		fmt.Fprintf(out, "%+v\n\n", rt)
		// each frame, the outermost first: a gad position with the lines
		// around it (more of them at the innermost), a Go call on its line
		frames := rt.Frames()
		last := -1
		for i, f := range frames {
			if f.Go == nil {
				last = i
			}
		}
		for i, f := range frames {
			switch {
			case f.Go != nil:
				fmt.Fprintf(out, "%s\n\n", f)
			case rt.FileSet() != nil && f.Pos.File != nil:
				fmt.Fprint(out, f.Pos.String()+":\n")
				if i == last {
					f.Pos.File.Data.TraceLines(out, f.Pos.Line, f.Pos.Column, up, down)
				} else {
					f.Pos.File.Data.TraceLines(out, f.Pos.Line, f.Pos.Column, h.Other.Up, h.Other.Down)
					out.Write([]byte("\n"))
				}
			}
		}
	case errors.As(err, &ce):
		h.context(out, err, ce)
		fmt.Fprintf(out, "%+"+strconv.Itoa(up)+"."+strconv.Itoa(down)+"v\n", ce)
	case errors.As(err, &el):
		h.context(out, err, el)
		fmt.Fprintf(out, "%+"+strconv.Itoa(up)+"."+strconv.Itoa(down)+"v\n", el)
	default:
		fmt.Fprintf(out, "ERROR: %v\n", err)
	}
}

// context writes what wraps the gad error inner in err — err's message up to
// inner's, "render main.gad" —; nothing when err is inner itself.
func (h *ErrorHumanizing) context(out io.Writer, err, inner error) {
	if err == inner {
		return
	}
	prefix := strings.TrimSuffix(err.Error(), inner.Error())
	if prefix = strings.TrimRight(prefix, ": "); prefix != "" {
		fmt.Fprintf(out, "%s:\n", prefix)
	}
}
