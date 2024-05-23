package helper

import "github.com/gad-lang/gad"

func NamedArgOfWriter(name string) *gad.NamedArgVar {
	return &gad.NamedArgVar{
		Name: name,
		TypeAssertion: &gad.TypeAssertion{
			Handlers: map[string]gad.TypeAssertionHandler{
				"writer": func(v gad.Object) (ok bool) {
					return gad.WriterFrom(v) != nil
				},
			},
		},
	}
}

func NamedArgOfReader(name string) *gad.NamedArgVar {
	return &gad.NamedArgVar{
		Name: name,
		TypeAssertion: &gad.TypeAssertion{
			Handlers: map[string]gad.TypeAssertionHandler{
				"reader": func(v gad.Object) (ok bool) {
					return gad.ReaderFrom(v) != nil
				},
			},
		},
	}
}
