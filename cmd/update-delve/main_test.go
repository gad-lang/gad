package main

import (
	"bytes"
	"strings"
	"testing"
)

// minimalLoop is a stand-in for vm_loop.go: it contains the src marker and
// exactly one instruction-fetch anchor, which is all generate() requires.
const minimalLoop = "package gad\n\n" +
	"func (vm *VM) loop() {\n" +
	"\tfor {\n" +
	"\t\top = Opcode(vm.curInsts[vm.ip])\n" +
	"\t\t_ = op\n" +
	"\t}\n" +
	"}\n"

func TestGenerateInjectsHook(t *testing.T) {
	out, err := generate([]byte(minimalLoop))
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, dstMarker) {
		t.Errorf("output missing %q", dstMarker)
	}
	if strings.Contains(s, srcMarker) {
		t.Errorf("output still contains src marker %q", srcMarker)
	}
	if !strings.Contains(s, strings.TrimRight(hook, "\n")) {
		t.Errorf("output missing debugger hook %q", hook)
	}
}

// TestGenerateHandlesCRLF guards the Windows/autocrlf regression: a CRLF
// checkout must produce the same (LF) output as an LF checkout, not fail the
// anchor count. See cmd/update-delve/main.go generate().
func TestGenerateHandlesCRLF(t *testing.T) {
	lfOut, err := generate([]byte(minimalLoop))
	if err != nil {
		t.Fatalf("generate(LF): %v", err)
	}

	crlf := strings.ReplaceAll(minimalLoop, "\n", "\r\n")
	crlfOut, err := generate([]byte(crlf))
	if err != nil {
		t.Fatalf("generate(CRLF): %v", err)
	}

	if !bytes.Equal(lfOut, crlfOut) {
		t.Errorf("CRLF source produced different output than LF source\nLF:\n%s\nCRLF:\n%s", lfOut, crlfOut)
	}
}
