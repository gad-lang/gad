package gomodule

// Request is what MakeSource is told, as JSON can say it (gad's wasm
// `gadGomodule`): Options but the dialect, given apart, and the module map.
type Request struct {
	Module   string            `json:"module"`
	Pkg      string            `json:"pkg,omitempty"`
	Imports  []string          `json:"imports,omitempty"`
	Files    map[string]string `json:"files,omitempty"`
	Generate string            `json:"generate,omitempty"`
}

// Response is what MakeSource makes, as JSON can say it: the package's text
// files by name (MODULE.go, MODULE_test.go, MODULE.gad), the bytecode
// (MODULE.gadb; base64 in JSON), or the error.
type Response struct {
	OK       bool              `json:"ok"`
	Files    map[string]string `json:"files,omitempty"`
	Bytecode []byte            `json:"bytecode,omitempty"`
	Imports  []string          `json:"imports,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
	Error    string            `json:"error,omitempty"`
}

// MakeSource is Make of src of the dialect sourceType ("gad", "gadTemplate",
// "gadx"), as the formatter takes it: gad's wasm `gadGomodule`.
func MakeSource(src, sourceType string, req Request) Response {
	files := make(map[string][]byte, len(req.Files))
	for n, s := range req.Files {
		files[n] = []byte(s)
	}
	r, err := Make([]byte(src), Options{
		Module:     req.Module,
		Pkg:        req.Pkg,
		SourceType: sourceType,
		Imports:    req.Imports,
		Files:      files,
		Generate:   req.Generate,
	})
	if err != nil {
		return Response{Error: err.Error()}
	}
	return Response{
		OK: true,
		Files: map[string]string{
			req.Module + ".go":      string(r.Go),
			req.Module + "_test.go": string(r.Test),
			req.Module + ".gad":     r.API,
		},
		Bytecode: r.Bytecode,
		Imports:  r.Imports,
		Warnings: r.Warnings,
	}
}
