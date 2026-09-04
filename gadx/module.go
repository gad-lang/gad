package gadx

import "github.com/gad-lang/gad"

var ModuleSpec = gad.NewModuleSpecFromName("gadx")

// Module returns the `gadx` builtin namespace.
func Module() gad.Dict { return newModule() }

// newModule builds the `gadx` builtin namespace.
func newModule() gad.Dict {
	return gad.Dict{
		// gad:doc
		// # gadx module
		// ## Types
		// Tag is a tag element type; Text wraps a value as a text node;
		// Elements is a wrapper-less fragment (the value comps/@main return);
		// Empty is the type of EMPTY.
		"Tag":      TagType,
		"Text":     TextType,
		"Md":       MdType,
		"Elements": ElementsType,
		"Empty":    EmptyType,
		// gad:doc
		// ## Values
		// EMPTY is an attribute value that is present and empty: it renders as
		// `name=""`, which a plain `""` cannot, since a falsy value is dropped so
		// that a conditional attribute disappears. Written `@empty` in a
		// template, and written back that way by `gad fmt`.
		"EMPTY":  EMPTY,
		"escape": BuiltinEscape,
		"attr":   BuiltinAttr,
		"attrs":  BuiltinAttrs,
		"write":  BuiltinTextWrite,
		"render": BuiltinRender,
	}
}
