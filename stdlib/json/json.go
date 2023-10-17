// Copyright (c) 2022-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package json

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/registry"
)

func init() {
	registry.RegisterObjectConverter(reflect.TypeOf(json.RawMessage(nil)),
		func(in any) (any, bool) {
			rm := in.(json.RawMessage)
			if rm == nil {
				return &RawMessage{Value: gad.Bytes{}}, true
			}
			return &RawMessage{Value: rm}, true
		},
	)

	registry.RegisterAnyConverter(reflect.TypeOf((*RawMessage)(nil)),
		func(in any) (any, bool) {
			rm := in.(*RawMessage)
			return json.RawMessage(rm.Value), true
		},
	)
}

// gad:doc
// ## Important Note
// All numeric types is unmarshaled to `gad.Decimal` type.

// gad:doc
// ## Types
// ### encoderOptions
//
// ToInterface Type
//
// ```go
// // EncoderOptions represents the encoding options (quote, html escape) to
// // Marshal any Object.
// type EncoderOptions struct {
// 	gad.ObjectImpl
// 	Value      gad.Object
// 	Quote      bool
// 	EscapeHTML bool
// }
// ```

var EncoderOptionsType = &gad.BuiltinObjType{
	NameValue: "encoderOptions",
}

// EncoderOptions represents the encoding options (quote, html escape) to
// Marshal any Object.
type EncoderOptions struct {
	gad.ObjectImpl
	Value      gad.Object
	Quote      bool
	EscapeHTML bool
}

func (eo *EncoderOptions) Type() gad.ObjectType {
	return EncoderOptionsType
}

// ToString implements gad.Object interface.
func (eo *EncoderOptions) ToString() string {
	return fmt.Sprintf("encoderOptions{Quote:%t EscapeHTML:%t Value:%s}",
		eo.Quote, eo.EscapeHTML, eo.Value)
}

// gad:doc
// #### encoderOptions Getters
//
//
// | Selector  | Return Type |
// |:----------|:------------|
// |.Value     | any         |
// |.Quote     | bool        |
// |.EscapeHTML| bool        |

// IndexGet implements gad.Object interface.
func (eo *EncoderOptions) IndexGet(_ *gad.VM, index gad.Object) (ret gad.Object, err error) {
	switch index.ToString() {
	case "Value":
		ret = eo.Value
	case "Quote":
		ret = gad.Bool(eo.Quote)
	case "EscapeHTML":
		ret = gad.Bool(eo.EscapeHTML)
	default:
		ret = gad.Nil
	}
	return
}

// gad:doc
// #### encoderOptions Setters
//
//
// | Selector  | Value Type  |
// |:----------|:------------|
// |.Value     | any         |
// |.Quote     | bool        |
// |.EscapeHTML| bool        |

// IndexSet implements gad.Object interface.
func (eo *EncoderOptions) IndexSet(_ *gad.VM, index, value gad.Object) error {
	switch index.ToString() {
	case "Value":
		eo.Value = value
	case "Quote":
		eo.Quote = !value.IsFalsy()
	case "EscapeHTML":
		eo.EscapeHTML = !value.IsFalsy()
	default:
		return gad.ErrInvalidIndex
	}
	return nil
}

// gad:doc
// ## Types
// ### rawMessage
//
// ToInterface Type
//
// ```go
// // RawMessage represents raw encoded json message to directly use value of
// // MarshalJSON without encoding.
// type RawMessage struct {
// 	gad.ObjectImpl
// 	Value []byte
// }
// ```

var RawMessageType = &gad.BuiltinObjType{
	NameValue: "rawMessage",
}

// RawMessage represents raw encoded json message to directly use value of
// MarshalJSON without encoding.
type RawMessage struct {
	gad.ObjectImpl
	Value []byte
}

var _ Marshaler = (*RawMessage)(nil)

func (rm *RawMessage) Type() gad.ObjectType {
	return RawMessageType
}

// ToString implements gad.Object interface.
func (rm *RawMessage) ToString() string {
	return string(rm.Value)
}

// MarshalJSON implements Marshaler interface and returns rm as the JSON
// encoding of rm.Value.
func (rm *RawMessage) MarshalJSON() ([]byte, error) {
	if rm == nil || rm.Value == nil {
		return []byte("null"), nil
	}
	return rm.Value, nil
}

// gad:doc
// #### rawMessage Getters
//
//
// | Selector  | Return Type |
// |:----------|:------------|
// |.Value     | bytes       |

// IndexGet implements gad.Object interface.
func (rm *RawMessage) IndexGet(_ *gad.VM, index gad.Object) (ret gad.Object, err error) {
	switch index.ToString() {
	case "Value":
		ret = gad.Bytes(rm.Value)
	default:
		ret = gad.Nil
	}
	return
}

// gad:doc
// #### rawMessage Setters
//
//
// | Selector  | Value Type  |
// |:----------|:------------|
// |.Value     | bytes       |

// IndexSet implements gad.Object interface.
func (rm *RawMessage) IndexSet(_ *gad.VM, index, value gad.Object) error {
	switch index.ToString() {
	case "Value":
		if v, ok := gad.ToBytes(value); ok {
			rm.Value = v
		} else {
			return gad.ErrType
		}
	default:
		return gad.ErrInvalidIndex
	}
	return nil
}
