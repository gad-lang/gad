package gad_test

import (
	"bytes"
	"testing"

	gad "github.com/gad-lang/gad"
)

// runEnv compiles and runs src with the given env seeded into the VM.
func runEnv(t *testing.T, src string, env *gad.Env) gad.Object {
	t.Helper()
	b := gad.NewBuiltins()
	st := gad.NewSymbolTable(b.NameSet)
	_, bc, err := gad.Compile(st, []byte(src), gad.CompileOptions{})
	if err != nil {
		t.Fatalf("compile %q: %v", src, err)
	}
	var out bytes.Buffer
	ret, err := gad.NewVM(b.Build(), bc).SetRecover(true).RunOpts(&gad.RunOpts{StdOut: &out, Env: env})
	if err != nil {
		t.Fatalf("run %q: %v", src, err)
	}
	return ret
}

// TestEnvKeyword covers the `env` contextual keyword: read (env.X / env[e]),
// the parent/fork pseudo-keys, write (env.X = v / env[e] = v) and iteration.
func TestEnvKeyword(t *testing.T) {
	env := gad.NewEnvFromMap(map[string]string{"PATH": "/bin", "HOME": "/home"})

	// Read via selector and index; absent keys are nil; a root env has no parent.
	got := runEnv(t, `return [str(env.PATH), str(env["HOME"]), isNil(env.NOPE), env.parent == nil]`, env)
	want := `["/bin", "/home", true, true]`
	if got.ToString() != want {
		t.Fatalf("read = %s, want %s", got.ToString(), want)
	}

	// Write via selector and index.
	if v := runEnv(t, `env.X = "1"; env["Y"] = "2"; return [env.X, env.Y]`, env); v.ToString() != `["1", "2"]` {
		t.Fatalf("write = %s, want [\"1\", \"2\"]", v.ToString())
	}

	// `env` is a standalone value; `.fork` returns a child whose parent is env.
	if v := runEnv(t, `e := env; f := e.fork; return f.parent == e`, env); v.ToString() != "true" {
		t.Fatalf("fork.parent = %s, want true", v.ToString())
	}

	// `x.env` is an ordinary field, not the keyword.
	if v := runEnv(t, `x := {env: 42}; return x.env`, env); v.ToString() != "42" {
		t.Fatalf("x.env = %s, want 42", v.ToString())
	}
}

// TestEnvWithFork covers `with env { … }`: mutations inside the block go to a
// fork and are discarded on exit.
func TestEnvWithFork(t *testing.T) {
	env := gad.NewEnvFromMap(map[string]string{"K": "orig"})
	got := runEnv(t, `
		before := env.K
		func() { with env { env.K = "inner"; delete env.K } }()
		return [before, str(env.K)]
	`, env)
	if got.ToString() != `["orig", "orig"]` {
		t.Fatalf("with-fork = %s, want [\"orig\", \"orig\"]", got.ToString())
	}
}

// TestDeleteStmt covers the `delete` statement: selector form (single key),
// array form (multiple evaluated keys, with spread), on env and dicts.
func TestDeleteStmt(t *testing.T) {
	env := gad.NewEnvFromMap(map[string]string{"A": "1", "B": "2", "C": "3", "D": "4"})

	// Selector form: delete a single key.
	if v := runEnv(t, `delete env.A; return isNil(env.A)`, env); v.ToString() != "true" {
		t.Fatalf("delete env.A: %s", v.ToString())
	}
	// Array form with a spread: delete several evaluated keys.
	if v := runEnv(t, `ks := ["C"]; delete env ["B", *ks]; return [isNil(env.B), isNil(env.C), str(env.D)]`, env); v.ToString() != `[true, true, "4"]` {
		t.Fatalf("delete env [..]: %s", v.ToString())
	}

	// On a dict (both forms), and the `obj.delete(...)` method still works.
	if v := runEnv(t, `d := {x: 1, y: 2, z: 3}; delete d.x; delete d ["y"]; return d`, nil); v.ToString() != "{z: 3}" {
		t.Fatalf("delete on dict: %s", v.ToString())
	}
	// Deleting from a non-deletable target is an error.
	b := gad.NewBuiltins()
	st := gad.NewSymbolTable(b.NameSet)
	_, bc, _ := gad.Compile(st, []byte(`a := [1]; delete a ["x"]`), gad.CompileOptions{})
	if _, err := gad.NewVM(b.Build(), bc).SetRecover(true).RunOpts(&gad.RunOpts{}); err == nil {
		t.Fatalf("delete on array: expected error, got nil")
	}
}
