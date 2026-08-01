package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		`    KIND: "${.fmt.style:-plain}"`,   // config self-reference
		`    PATHS: ["${HOME}/a", "b", "c"]`, // array → OS-joined
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
		"PATHS":    "/home/u/a" + sep + "b" + sep + "c",
		"HOME":     "/home/u", // process env is inherited
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
