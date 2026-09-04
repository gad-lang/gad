package importers

import (
	"errors"
	"path/filepath"
	"testing"
)

// resolve runs Get+Name the way the compiler does.
func resolve(t *testing.T, im *FileImporter, name string) (string, error) {
	t.Helper()
	im.Get(name)
	return im.Name()
}

// A plain name resolves against the root, whichever module imports it, so the
// same name always means the same file.
func TestNamePlainResolvesFromRoot(t *testing.T) {
	root := t.TempDir()
	im := &FileImporter{Root: root, WorkDir: filepath.Join(root, "layouts")}

	got, err := resolve(t, im, "comps.gadx")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "comps.gadx"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// A name with its own directory is still read from the root, rather than
	// being appended to the importing module's directory.
	got, err = resolve(t, im, "layouts/default.gadx")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "layouts", "default.gadx"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// "./" and "../" resolve against the importing module's directory.
func TestNameRelativeResolvesFromWorkDir(t *testing.T) {
	root := t.TempDir()
	im := &FileImporter{Root: root, WorkDir: filepath.Join(root, "layouts")}

	for name, want := range map[string]string{
		"./default.gadx": filepath.Join(root, "layouts", "default.gadx"),
		"../comps.gadx":  filepath.Join(root, "comps.gadx"),
	} {
		got, err := resolve(t, im, name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
}

// An import may not climb above the root.
func TestNameRefusesEscapingRoot(t *testing.T) {
	root := t.TempDir()
	im := &FileImporter{Root: root, WorkDir: filepath.Join(root, "layouts")}

	for _, name := range []string{"../../secret.gadx", "../.."} {
		if _, err := resolve(t, im, name); !errors.Is(err, ErrImportOutsideRoot) {
			t.Errorf("%s: err = %v, want ErrImportOutsideRoot", name, err)
		}
	}
}

// Fork carries the root down, so nesting never moves what a plain name means.
func TestForkKeepsRoot(t *testing.T) {
	root := t.TempDir()
	im := &FileImporter{Root: root, WorkDir: root}

	deep := im.Fork(filepath.Join(root, "a", "b", "c.gadx")).(*FileImporter)
	if deep.Root != root {
		t.Fatalf("Root = %q, want %q", deep.Root, root)
	}

	got, err := resolve(t, deep, "comps.gadx")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "comps.gadx"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// An importer created without a root takes one from its WorkDir on the first
// fork, so a caller that never set Root still gets a stable base.
func TestForkAdoptsWorkDirAsRoot(t *testing.T) {
	root := t.TempDir()
	im := &FileImporter{WorkDir: root}

	forked := im.Fork(filepath.Join(root, "layouts", "index.gadx")).(*FileImporter)
	if forked.Root != root {
		t.Errorf("Root = %q, want %q", forked.Root, root)
	}
}

// An absolute name is taken as it is, root or no root.
func TestNameAbsolute(t *testing.T) {
	root := t.TempDir()
	im := &FileImporter{Root: root, WorkDir: root}

	abs := filepath.Join(root, "x.gadx")
	got, err := resolve(t, im, abs)
	if err != nil {
		t.Fatal(err)
	}
	if got != abs {
		t.Errorf("got %q, want %q", got, abs)
	}
}
