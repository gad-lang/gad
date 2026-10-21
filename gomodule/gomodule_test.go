package gomodule_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gad-lang/gad/gomodule"
)

const shapes = `/**
Shapes.
**/

/** A shape. **/
export class Shape {
    name str = "?"
    methods {
        /** what it is **/
        describe() => "shape " + this.name
    }
}

/** The area of w by h. **/
export area(w number, h number) <number> => w * h
`

const initGo = `package shapes

//gad:import strings

// gad:doc
// # shapes module
//
// ## Declarations
//
// ` + "```gad" + `
// export class GoShape { *Shape }
// ` + "```" + `
`

// Make, in memory: the package's files, the Go init's imports and docs
// (Files) in the API stub, the declarations before the functions.
func TestMake(t *testing.T) {
	r, err := gomodule.Make([]byte(shapes), gomodule.Options{
		Module: "shapes",
		Files:  map[string][]byte{"init.go": []byte(initGo)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Bytecode) == 0 {
		t.Fatal("no bytecode")
	}
	if strings.Join(r.Imports, ",") != "strings" {
		t.Fatalf("imports %v", r.Imports)
	}
	if !strings.Contains(string(r.Go), `var Imports = []string{"strings"}`) || strings.Contains(string(r.Go), "go:generate") {
		t.Fatalf("the Go source:\n%s", r.Go)
	}
	api := r.API
	for _, want := range []string{"export class Shape {", "/** what it is **/", "describe() => nil",
		"export area(w number, h number) <number> => nil", "export class GoShape { *Shape }"} {
		if !strings.Contains(api, want) {
			t.Errorf("the API stub has no %q:\n%s", want, api)
		}
	}
	if strings.Index(api, "class Shape") > strings.Index(api, "export area") {
		t.Errorf("the declarations go before the functions:\n%s", api)
	}
	if n := len(r.Files()); n != 4 {
		t.Errorf("%d files", n)
	}
}

// The dialect is the formatter's source type: a .gadt module.
func TestMakeTemplate(t *testing.T) {
	r, err := gomodule.Make([]byte(`{% export title = "Hi" %}`), gomodule.Options{Module: "page", SourceType: "gadTemplate"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.API, `export title = "Hi"`) {
		t.Fatalf("the API stub:\n%s", r.API)
	}
}

// MakeSource: Make as JSON says it (the wasm's gadGomodule); an error is said,
// not returned.
func TestMakeSource(t *testing.T) {
	res := gomodule.MakeSource(shapes, "gad", gomodule.Request{Module: "shapes", Generate: "go run x gomodule"})
	if !res.OK || res.Files["shapes.go"] == "" || res.Files["shapes.gad"] == "" || len(res.Bytecode) == 0 {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(res.Files["shapes.go"], "//go:generate go run x gomodule") {
		t.Fatal("no go:generate line")
	}
	if _, err := json.Marshal(res); err != nil {
		t.Fatal(err)
	}
	bad := gomodule.MakeSource(`x := import("./f.gad")`, "", gomodule.Request{Module: "m"})
	if bad.OK || !strings.Contains(bad.Error, "f.gad") {
		t.Fatalf("%+v", bad)
	}
	if res := gomodule.MakeSource(shapes, "", gomodule.Request{Module: "Bad"}); res.OK {
		t.Fatal("a module name not snake_case")
	}
}
