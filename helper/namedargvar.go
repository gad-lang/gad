package helper

import "github.com/gad-lang/gad"

func NamedArgOfWriter(name string) *gad.NamedArgVar {
	return &gad.NamedArgVar{
		Name: name,
		TypeAssertion: &gad.TypeAssertion{
			Handlers: map[string]gad.TypeAssertionHandler{
				"reader": func(v gad.Object) (ok bool) {
					_, ok = v.(gad.Reader)
					return
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
				"writer": func(v gad.Object) (ok bool) {
					_, ok = v.(gad.Writer)
					return
				},
			},
		},
	}
}
