package encoder

import (
	"bytes"
	"errors"
	"io"

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
	return bc.FixObjects(modules)
}

func (bc *Bytecode) FixObjects(modules *gad.ModuleMap) error {
	for i := range bc.Constants {
		switch obj := bc.Constants[i].(type) {
		case *gad.Module:
			obj.ConstantIndex = i
			switch t := modules.Get(obj.Name()).(type) {
			case *gad.BuiltinInitModule:
				obj.Init = t.Init.Caller(obj)
			case *gad.BuiltinModule:
				obj.Init = t.InitFunc().Caller(obj)
			case *gad.SourceModule:
				obj.Init.(*gad.CompiledFunction).SetModule(obj)
			}
		case *CompiledFunction:
			f := obj.CompiledFunction
			if obj.moduleConstantIndex >= 0 {
				f.SetModule(bc.Constants[obj.moduleConstantIndex].(*gad.Module))
			}
			bc.Constants[i] = f
		}
	}
	return nil
}
