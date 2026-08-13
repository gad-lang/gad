//go:build !js
// +build !js

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-dap"
)

// TestDAPLaunchArgsEnv verifies a launch "run profile" — args and env from the
// launch configuration — reach the debugged program (DAP §5.2).
func TestDAPLaunchArgsEnv(t *testing.T) {
	dir := t.TempDir()
	prog := filepath.Join(dir, "p.gad")
	// print the first positional arg and an env var
	src := "param (*argv)\nprint(argv[0])\nprint(env[\"FOO\"])\n"
	if err := os.WriteFile(prog, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	c := startDAP(t)
	c.send(&dap.InitializeRequest{Request: c.header("initialize")})
	c.waitFor(isEvent("initialized"))
	c.send(&dap.ConfigurationDoneRequest{Request: c.header("configurationDone")})

	lr := &dap.LaunchRequest{Request: c.header("launch")}
	lr.Arguments = []byte(`{"program":"` + filepath.ToSlash(prog) +
		`","args":["hello"],"env":{"FOO":"bar"}}`)
	c.send(lr)

	// collect stdout output until terminated
	var out strings.Builder
	deadline := time.After(3 * time.Second)
	for {
		done := false
		select {
		case m, ok := <-c.msgs:
			if !ok {
				done = true
				break
			}
			switch ev := m.(type) {
			case *dap.OutputEvent:
				if ev.Body.Category == "stdout" {
					out.WriteString(ev.Body.Output)
				}
			case *dap.TerminatedEvent:
				done = true
			}
		case <-deadline:
			t.Fatalf("timed out; output so far=%q", out.String())
		}
		if done {
			break
		}
	}
	if got := out.String(); !strings.Contains(got, "hello") || !strings.Contains(got, "bar") {
		t.Fatalf("program output = %q, want it to contain the arg \"hello\" and env \"bar\"", got)
	}
}
