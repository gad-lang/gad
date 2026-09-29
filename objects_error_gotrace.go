package gad

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/gad-lang/gad/parser/source"
)

// GoFrames are Go calls in a RuntimeError's trace. A gad function run from Go
// — a callback, through an Invoker or VM.Call — returns its error to the Go
// code that called it, which returns it to the gad code that called that: the
// Go calls in between are in the trace too, where they were.
type GoFrames struct {
	// At is how many positions the trace had when the error left gad for Go:
	// Trace[:At] are the frames of the gad function Go called, Trace[At:] the
	// ones of the gad code that called Go.
	At int
	// Frames are the Go calls, the innermost first: from the one that ran the
	// gad function to the one the outer gad code called.
	Frames []runtime.Frame
}

// TraceFrame is one frame of a RuntimeError's trace: a position in gad code,
// or a Go call (Go not nil).
type TraceFrame struct {
	Pos source.FilePos
	Go  *runtime.Frame
}

const (
	gadPkg = "github.com/gad-lang/gad."
	gadVM  = gadPkg + "(*VM)."
	// maxGoFrames bounds the Go calls kept for one crossing: past the outer
	// VM — a gad function run from Go with no gad code around — the stack is
	// the program's, not the error's.
	maxGoFrames = 32
)

// recordGoFrames records, on the RuntimeError err is, the Go calls that led
// to the gad function that failed: from the caller of the gad function that
// ran it (Invoker.Invoke, VM.Call) up to the VM of the gad code that called
// Go, gad's own calls left out. err not a RuntimeError: nothing.
func recordGoFrames(err error) {
	var re *RuntimeError
	if !errors.As(err, &re) {
		return
	}
	pcs := make([]uintptr, 64)
	n := runtime.Callers(2, pcs)
	it := runtime.CallersFrames(pcs[:n])
	var (
		frames []runtime.Frame
		outer  bool
	)
	for {
		f, more := it.Next()
		switch {
		case strings.HasPrefix(f.Function, gadVM) && outer:
			// back in the VM of the gad code that called Go
			more = false
		case strings.HasPrefix(f.Function, gadPkg), strings.HasPrefix(f.Function, "runtime."):
			// gad's own calls, and the runtime's
		default:
			outer = true
			frames = append(frames, f)
			more = more && len(frames) < maxGoFrames
		}
		if !more {
			break
		}
	}
	if len(frames) > 0 {
		re.GoTrace = append(re.GoTrace, GoFrames{At: len(re.Trace), Frames: frames})
	}
}

// Frames is the trace, the outermost frame first: the positions of the gad
// code, with the Go calls between them where the error crossed Go (GoTrace).
// An error of a gad function Go called comes back to the calling gad code
// wrapped (DoCall: "CallError: …") in an error of its own: the trace goes on
// into it, the Go calls between them.
func (o *RuntimeError) Frames() []TraceFrame {
	frames := o.ownFrames()
	if in := o.inner(); in != nil {
		frames = append(frames, in.Frames()...)
	}
	return frames
}

// ownFrames are the frames of this error alone, the outermost first.
func (o *RuntimeError) ownFrames() []TraceFrame {
	pos := func(p source.Pos) source.FilePos {
		if o.fileSet == nil {
			return source.FilePos{Offset: int(p)}
		}
		return o.fileSet.Position(p)
	}
	// the innermost first, as Trace and each GoFrames are
	var inner []TraceFrame
	goAt := func(i int) {
		for _, g := range o.GoTrace {
			if g.At == i {
				for j := range g.Frames {
					inner = append(inner, TraceFrame{Go: &g.Frames[j]})
				}
			}
		}
	}
	for i, p := range o.Trace {
		goAt(i)
		inner = append(inner, TraceFrame{Pos: pos(p)})
	}
	goAt(len(o.Trace))
	for i, j := 0, len(inner)-1; i < j; i, j = i+1, j-1 {
		inner[i], inner[j] = inner[j], inner[i]
	}
	return inner
}

// inner is the RuntimeError this one wraps, if any: the error of the gad
// function a Go function called.
func (o *RuntimeError) inner() *RuntimeError {
	for e := o.Unwrap(); e != nil; e = errors.Unwrap(e) {
		if re, ok := e.(*RuntimeError); ok && re != o {
			return re
		}
	}
	return nil
}

// crossedGo reports whether the error, or one it wraps, crossed Go calls.
func (o *RuntimeError) crossedGo() bool {
	for e := o; e != nil; e = e.inner() {
		if len(e.GoTrace) > 0 {
			return true
		}
	}
	return false
}

// String is the frame on one line: the gad position, or "go FUNC (FILE:LINE)".
func (f TraceFrame) String() string {
	if f.Go != nil {
		return fmt.Sprintf("go %s (%s:%d)", f.Go.Function, f.Go.File, f.Go.Line)
	}
	return fmt.Sprintf("%+v", f.Pos)
}

// formatFrames writes the trace with its Go calls, as source.FilePosStackTrace
// writes the positions alone.
func (o *RuntimeError) formatFrames(w io.Writer) {
	for i, f := range o.Frames() {
		if i > 0 {
			io.WriteString(w, "\n\t   ")
		} else {
			io.WriteString(w, "\n\tat ")
		}
		io.WriteString(w, f.String())
	}
}
