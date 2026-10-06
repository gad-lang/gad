package gad_test

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad"
)

// The import statement — `@import "m"`, `@import "m" as name`,
// `@import { a, b: c, d = 1 } from "m"` —, as the templates (gadx) write it.
func TestImportStmt(t *testing.T) {
	mm := gad.NewModuleMap().AddSourceModule("v", []byte("export cep(v) => \"cep:\" + v\nexport cpf = 2"))
	run := func(src string) (gad.Object, error) {
		bi := gad.NewBuiltins()
		res, err := gad.Compile(gad.NewSymbolTable(bi.NameSet), []byte(src), gad.CompileOptions{
			CompilerOptions: gad.CompilerOptions{ModuleMap: mm},
		})
		if err != nil {
			return nil, err
		}
		return gad.NewVM(bi.Build(), res.Bytecode).Run()
	}
	for src, want := range map[string]string{
		"@import { cep, cpf: x } from \"v\"\nreturn [cep(\"a\"), x]":                   `["cep:a", 2]`,
		"@import {\n\tcep,\n\tmissing = 7\n} from \"v\"\nreturn [cep(\"b\"), missing]": `["cep:b", 7]`,
		"@import \"v\" as m\nreturn [m.cep(\"c\"), m.cpf]":                             `["cep:c", 2]`,
		"@import \"v\"\nreturn 1":                                                      `1`,
		// an ordinary name elsewhere
		"from := 1; as := 2; return from + as": `3`,
	} {
		got, err := run(src)
		if err != nil {
			t.Errorf("%s: %v", src, err)
			continue
		}
		if got.ToString() != want {
			t.Errorf("%s:\n got %s\nwant %s", src, got.ToString(), want)
		}
	}
	for src, want := range map[string]string{
		`@import { cep } "v"`:    "expected 'from'",
		`@import cep from "v"`:   "expected module name",
		`@import { cep } from x`: "expected module name",
		`@import "nothing" as m`: "module 'nothing' not found",
	} {
		if _, err := run(src); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (want %q)", src, err, want)
		}
	}
}
