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
		// Tag is a tag element type; Text wraps a value as a text node.
		"Tag":    TagType,
		"Text":   TextType,
		"Md":     MdType,
		"escape": BuiltinEscape,
		"attr":   BuiltinAttr,
		"attrs":  BuiltinAttrs,
		"write":  BuiltinTextWrite,
		"render": BuiltinRender,
	}
}
