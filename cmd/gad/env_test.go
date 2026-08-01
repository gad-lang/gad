package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fromSlashJoin converts each entry's '/' to the OS separator and joins them
// with sep, mirroring how the array path-list form is materialized. On Unix this
// is a plain join; on Windows the slashes become backslashes.
func fromSlashJoin(sep string, entries ...string) string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = filepath.FromSlash(e)
	}
	return strings.Join(out, sep)
}

// writeGadYAML writes a .gad.yaml with the given body into dir.
func writeGadYAML(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, defaultCfgFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadWorkspaceEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", "/home/u")
	t.Setenv("WS_USER", "alice")
	os.Unsetenv("WS_MISSING")

	writeGadYAML(t, dir, strings.Join([]string{
		"env:",
		`    APP_HOME: "${HOME}/app"`,
		`    GREETING: "hi ${WS_USER:-nobody}"`,
		`    FALLBACK: "${WS_MISSING:-default}"`,
		`    KIND: "${.fmt.style:-plain}"`,              // config self-reference
		`    PATHS: ["${HOME}/a", "b", "c"]`,            // array → OS-joined
		`    PACKED: ["x", "u/v:w/z"]`,                  // an element may pack ':'-separated entries
		`    OPCOLON: ["${WS_MISSING:-def}", "y"]` + "", // ':' in ${:-} is an operator, not a split
		"fmt:",
		"    style: fancy",
	}, "\n"))

	env := loadWorkspaceEnv(dir)

	sep := string(os.PathListSeparator)
	cases := map[string]string{
		"APP_HOME": "/home/u/app",
		"GREETING": "hi alice",
		"FALLBACK": "default",
		"KIND":     "fancy",
		"PATHS":    fromSlashJoin(sep, "/home/u/a", "b", "c"),
		"PACKED":   fromSlashJoin(sep, "x", "u/v", "w/z"), // element split on ':'
		"OPCOLON":  fromSlashJoin(sep, "def", "y"),        // ${:-def} kept whole
		"HOME":     "/home/u",                             // process env is inherited
	}
	for k, want := range cases {
		got, _ := env.Get(k)
		if got != want {
			t.Errorf("env[%q] = %q, want %q", k, got, want)
		}
	}
}

func TestLoadWorkspaceEnvNoConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", "/h")
	env := loadWorkspaceEnv(dir) // no .gad.yaml
	if v, _ := env.Get("HOME"); v != "/h" {
		t.Fatalf("HOME = %q, want /h (process env)", v)
	}
}

func TestGadPathFromEnv(t *testing.T) {
	dir := t.TempDir()
	sep := string(os.PathListSeparator)
	writeGadYAML(t, dir, "env:\n    GADPATH: [\"/one\", \"/two\"]\n")
	env := loadWorkspaceEnv(dir)
	got := gadPathFromEnv(env)
	want := []string{"/one", "/two"}
	if len(got) != len(want) {
		t.Fatalf("GADPATH dirs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("GADPATH[%d] = %q, want %q (joined %q)", i, got[i], want[i], strings.Join(want, sep))
		}
	}
}
