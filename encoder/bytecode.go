package encoder

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/gad-lang/gad"
)

// EncodeBytecodeTo encodes given bc to w io.Writer.
func EncodeBytecodeTo(bc *gad.Bytecode, w io.Writer) error {
	return (*Bytecode)(bc).Encode(w)
}

// DecodeBytecodeFrom decodes *gad.Bytecode from given r io.Reader.
func DecodeBytecodeFrom(r io.Reader, modules *gad.ModuleMap) (*gad.Bytecode, error) {
	var bc Bytecode
	err := bc.Decode(r, modules)
	return (*gad.Bytecode)(&bc), err
}

// Encode writes encoded data of Bytecode to writer.
func (bc *Bytecode) Encode(w io.Writer) error {
	data, err := bc.MarshalBinary()
	if err != nil {
		return err
	}

	n, err := w.Write(data)
	if err != nil {
		return err
	}

	if n != len(data) {
		return errors.New("short write")
	}
	return nil
}

// Decode decodes Bytecode data from the reader.
func (bc *Bytecode) Decode(r io.Reader, modules *gad.ModuleMap) error {
	dst := bytes.NewBuffer(nil)
	if _, err := io.Copy(dst, r); err != nil {
		return err
	}
	return bc.unmarshal(dst.Bytes(), modules)
}

// unmarshal unmarshals data and assigns receiver to the new Bytecode.
func (bc *Bytecode) unmarshal(data []byte, modules *gad.ModuleMap) error {
	err := bc.UnmarshalBinary(data)
	if err != nil {
		return err
	}

	if modules == nil {
		modules = gad.NewModuleMap()
	}
	return bc.fixObjects(modules)
}

func (bc *Bytecode) fixObjects(modules *gad.ModuleMap) error {
	for i := range bc.Constants {
		switch obj := bc.Constants[i].(type) {
		case gad.Dict:
			v, ok := obj[gad.AttrModuleName]
			if !ok {
				continue
			}

			name, ok := v.(gad.Str)
			if !ok {
				continue
			}

			bmod := modules.Get(string(name))
			if bmod == nil {
				return fmt.Errorf("module '%s' not found", name)
			}

			// copy items from given module to decoded object if key exists in obj
			for item := range obj {
				if item == gad.AttrModuleName {
					// module name may not present in given map, skip it.
					continue
				}
				o := bmod.(*gad.BuiltinModule).Attrs[item]
				// if item not exists in module, nil will not pass type check
				want := reflect.TypeOf(obj[item])
				got := reflect.TypeOf(o)
				if want != got {
					// this must not happen
					return fmt.Errorf("module '%s' item '%s' type mismatch:"+
						"want '%v', got '%v'", name, item, want, got)
				}
				obj[item] = o
			}
		case *Function:
			return fmt.Errorf("not decodable object of Function type:'%s'", obj.Name)
		}
	}
	return nil
}
