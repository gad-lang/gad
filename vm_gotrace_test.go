package gad_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	. "github.com/gad-lang/gad"
)

// goInvoke and goCall are Go functions a script calls, that call back the gad
// function they are given: through an Invoker (a child VM) and through
// VM.Call (a sub-run of the same VM).
func goInvoke(c Call) (Object, error) { return NewInvoker(c.VM, c.Args.Get(0)).Invoke(Args{}, nil) }
func goCall(c Call) (Object, error)   { return c.VM.Call(c.Args.Get(0), Args{}, nil) }

// A gad function run from Go fails: its trace has the gad frames around the
// Go call, and the Go call itself between them — the call in the script, the
// Go function, the line in the callback, the outermost first.
func TestRuntimeErrorGoTrace(t *testing.T) {
	src := `global run
f := func() {
	a := nil
	return a.x
}
return run(f)`
	for name, fn := range map[string]CallableFunc{"invoker": goInvoke, "vm.Call": goCall} {
		t.Run(name, func(t *testing.T) {
			cr, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(src), DefaultCompileOptions)
			if err != nil {
				t.Fatal(err)
			}
			vm := NewVM(NewBuiltins().Build(), cr.BC())
			_, err = vm.RunOpts(&RunOpts{Globals: Dict{"run": &Function{FuncName: "run", Value: fn}}})
			var re *RuntimeError
			if !errors.As(err, &re) {
				t.Fatalf("not a RuntimeError: %T %v", err, err)
			}
			var lines []string
			for _, f := range re.Frames() {
				switch {
				case f.Go != nil:
					lines = append(lines, "go "+f.Go.Function[strings.LastIndex(f.Go.Function, ".")+1:])
				default:
					lines = append(lines, fmt.Sprintf("gad %d", f.Pos.Line))
				}
			}
			got := strings.Join(lines, ", ")
			want := "gad 6, go " + map[string]string{"invoker": "goInvoke", "vm.Call": "goCall"}[name] + ", gad 4"
			if got != want {
				t.Errorf("frames %s, want %s", got, want)
			}
			if s := fmt.Sprintf("%+v", re); !strings.Contains(s, "go github.com/gad-lang/gad_test.") {
				t.Errorf("%%+v has no Go call:\n%s", s)
			}
		})
	}
}

// With no Go call in between, the trace is the gad positions alone, as before.
func TestRuntimeErrorNoGoTrace(t *testing.T) {
	cr, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte("a := nil\nreturn a.x"), DefaultCompileOptions)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewVM(NewBuiltins().Build(), cr.BC()).RunOpts(&RunOpts{})
	var re *RuntimeError
	if !errors.As(err, &re) || len(re.GoTrace) != 0 {
		t.Fatalf("%T %v", err, err)
	}
	if s := fmt.Sprintf("%+v", re); strings.Contains(s, "\tat go ") || !strings.Contains(s, ":2:") {
		t.Errorf("%%+v:\n%s", s)
	}
}
