package main

import (
	"strings"
	"testing"
)

// TestGiomDoc verifies that `gad doc` documents a .giom template: the module
// heading, file-level prose, components/functions with their doc text, and a
// data-source-pos anchor pointing at each declaration.
func TestGiomDoc(t *testing.T) {
	src := []byte("/** Reusable UI widgets. */\n\n" +
		"/** Renders a greeting. */\n" +
		"@comp greeting(name; greeting=\"Hello\")\n" +
		"    p {= greeting }\n\n" +
		"/** Formats a label. */\n" +
		"@func label(text)\n" +
		"    span {= text }\n")

	md, err := (&DocGenerator{NoTest: true}).FromContent("widgets.giom", src)
	if err != nil {
		t.Fatalf("FromContent: %v", err)
	}

	for _, want := range []string{
		"# widgets",
		"Reusable UI widgets.", // file-level prose
		"## Components",
		"+greeting</span>(name; greeting=\"Hello\")",
		"Renders a greeting.", // comp doc
		"## Functions",
		"label</span>(text)",
		"Formats a label.", // func doc
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("doc missing %q:\n%s", want, md)
		}
	}

	// The greeting @comp is on line 4, column 1 → its heading carries that
	// data-source-pos for source navigation.
	if !strings.Contains(md, `data-source-pos="4,1">+greeting`) {
		t.Fatalf("missing data-source-pos for greeting:\n%s", md)
	}
	// The label @func is on line 8.
	if !strings.Contains(md, `data-source-pos="8,1">label`) {
		t.Fatalf("missing data-source-pos for label:\n%s", md)
	}
}

// TestGiomDocDeclarations verifies @param/@const/@var/@enum are documented too.
func TestGiomDocDeclarations(t *testing.T) {
	src := []byte("/** Config. */\n@param (title; theme=\"light\")\n\n" +
		"/** Version string. */\n@const Version = \"1.0\"\n\n" +
		"/** Counter. */\n@var (count = 0)\n\n" +
		"/** Permissions. */\n@enum Perm (Read, Write)\n")

	md, err := (&DocGenerator{NoTest: true}).FromContent("cfg.giom", src)
	if err != nil {
		t.Fatalf("FromContent: %v", err)
	}
	for _, want := range []string{
		"## Parameters", "Config.", `(title; theme="light")`,
		"## Constants", "Version string.", "Version",
		"## Variables", "Counter.", "count",
		"## Enums", "Permissions.", "Perm",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("doc missing %q:\n%s", want, md)
		}
	}
}
