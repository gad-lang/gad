// Copyright (c) 2022-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package json

import (
	"bytes"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/stdlib"
)

// Module represents json module.
var Module = map[string]gad.Object{
	// gad:doc
	// # json Module
	//
	// ## Functions
	// Marshal(v any) -> bytes
	// Returns the JSON encoding v or error.
	"Marshal": &gad.Function{
		Name:  "Marshal",
		Value: stdlib.FuncPpVM_ORO(marshalFunc),
	},
	// gad:doc
	// MarshalIndent(v any, prefix string, indent string) -> bytes
	// MarshalIndent is like Marshal but applies IndentCount to format the output.
	"MarshalIndent": &gad.Function{
		Name:  "MarshalIndent",
		Value: stdlib.FuncPpVM_OssRO(marshalIndentFunc),
	},
	// gad:doc
	// IndentCount(src bytes, prefix string, indent string) -> bytes
	// Returns indented form of the JSON-encoded src or error.
	"IndentCount": &gad.Function{
		Name:  "IndentCount",
		Value: stdlib.FuncPb2ssRO(indentFunc),
	},
	// gad:doc
	// RawMessage(v bytes) -> rawMessage
	// Returns a wrapped bytes to provide raw encoded JSON value to Marshal
	// functions.
	"RawMessage": &gad.Function{
		Name:  "RawMessage",
		Value: stdlib.FuncPb2RO(rawMessageFunc),
	},
	// gad:doc
	// Compact(data bytes, escape bool) -> bytes
	// Returns elided insignificant space characters from data or error.
	"Compact": &gad.Function{
		Name:  "Compact",
		Value: stdlib.FuncPb2bRO(compactFunc),
	},
	// gad:doc
	// Quote(v any) -> encoderOptions
	// Returns a wrapped object to provide Marshal functions to quote v.
	"Quote": &gad.Function{
		Name:  "Quote",
		Value: stdlib.FuncPORO(quoteFunc),
	},
	// gad:doc
	// NoQuote(v any) -> encoderOptions
	// Returns a wrapped object to provide Marshal functions not to quote while
	// encoding.
	// This can be used not to quote all array or map items.
	"NoQuote": &gad.Function{
		Name:  "NoQuote",
		Value: stdlib.FuncPORO(noQuoteFunc),
	},
	// gad:doc
	// NoEscape(v any) -> encoderOptions
	// Returns a wrapped object to provide Marshal functions not to escape html
	// while encoding.
	"NoEscape": &gad.Function{
		Name:  "NoEscape",
		Value: stdlib.FuncPORO(noEscapeFunc),
	},
	// gad:doc
	// Unmarshal(p bytes) -> any
	// Unmarshal parses the JSON-encoded p and returns the result or error.
	"Unmarshal": &gad.Function{
		Name:  "Unmarshal",
		Value: stdlib.FuncPb2RO(unmarshalFunc),
	},
	// gad:doc
	// Valid(p bytes) -> bool
	// Reports whether p is a valid JSON encoding.
	"Valid": &gad.Function{
		Name:  "Valid",
		Value: stdlib.FuncPb2RO(validFunc),
	},
}

func marshalFunc(vm *gad.VM, o gad.Object) gad.Object {
	b, err := Marshal(vm, o)
	if err != nil {
		return &gad.Error{Message: err.Error(), Cause: err}
	}
	return gad.Bytes(b)
}

func marshalIndentFunc(vm *gad.VM, o gad.Object, prefix, indent string) gad.Object {
	b, err := MarshalIndent(vm, o, prefix, indent)
	if err != nil {
		return &gad.Error{Message: err.Error(), Cause: err}
	}
	return gad.Bytes(b)
}

func indentFunc(src []byte, prefix, indent string) gad.Object {
	var buf bytes.Buffer
	err := indentBuffer(&buf, src, prefix, indent)
	if err != nil {
		return &gad.Error{Message: err.Error(), Cause: err}
	}
	return gad.Bytes(buf.Bytes())
}

func rawMessageFunc(b []byte) gad.Object { return &RawMessage{Value: b} }

func compactFunc(data []byte, escape bool) gad.Object {
	var buf bytes.Buffer
	err := compact(&buf, data, escape)
	if err != nil {
		return &gad.Error{Message: err.Error(), Cause: err}
	}
	return gad.Bytes(buf.Bytes())
}

func quoteFunc(o gad.Object) gad.Object {
	if v, ok := o.(*EncoderOptions); ok {
		v.Quote = true
		return v
	}
	return &EncoderOptions{Value: o, Quote: true, EscapeHTML: true}
}

func noQuoteFunc(o gad.Object) gad.Object {
	if v, ok := o.(*EncoderOptions); ok {
		v.Quote = false
		return v
	}
	return &EncoderOptions{Value: o, Quote: false, EscapeHTML: true}
}

func noEscapeFunc(o gad.Object) gad.Object {
	if v, ok := o.(*EncoderOptions); ok {
		v.EscapeHTML = false
		return v
	}
	return &EncoderOptions{Value: o}
}

func unmarshalFunc(b []byte) gad.Object {
	v, err := Unmarshal(b)
	if err != nil {
		return &gad.Error{Message: err.Error(), Cause: err}
	}
	return v
}

func validFunc(b []byte) gad.Object { return gad.Bool(valid(b)) }
