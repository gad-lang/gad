package helper

import "github.com/gad-lang/gad"

var (
	WriterAssertion = &gad.TypeAssertion{
		Handlers: map[string]gad.TypeAssertionHandler{
			"writer": func(v gad.Object) (ok bool) {
				return gad.WriterFrom(v) != nil
			},
		},
	}

	ReaderAssertion = &gad.TypeAssertion{
		Handlers: map[string]gad.TypeAssertionHandler{
			"reader": func(v gad.Object) (ok bool) {
				return gad.ReaderFrom(v) != nil
			},
		},
	}
)

func NamedArgOfWriter(name string) *gad.NamedArgVar {
	return &gad.NamedArgVar{
		Name:          name,
		TypeAssertion: WriterAssertion,
	}
}

func NamedArgOfReader(name string) *gad.NamedArgVar {
	return &gad.NamedArgVar{
		Name:          name,
		TypeAssertion: ReaderAssertion,
	}
}

func ArgOfWriter(name string) *gad.Arg {
	return &gad.Arg{
		Name:          name,
		TypeAssertion: WriterAssertion,
	}
}

func ArgOfReader(name string) *gad.Arg {
	return &gad.Arg{
		Name:          name,
		TypeAssertion: ReaderAssertion,
	}
}
