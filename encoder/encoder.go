// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package encoder

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"io"
	"strconv"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/stdlib/json"
	"github.com/gad-lang/gad/stdlib/time"
)

// Bytecode signature and version are written to the header of encoded Bytecode.
// Bytecode is encoded with current BytecodeVersion and its format.
const (
	BytecodeSignature uint32 = 0x75474F
	BytecodeVersion   uint16 = 1
)

// Types implementing encoding.BinaryMarshaler encoding.BinaryUnmarshaler.
type (
	Bytecode         gad.Bytecode
	CompiledFunction gad.CompiledFunction
	BuiltinFunction  gad.BuiltinFunction
	BuiltinObjType   gad.BuiltinObjType
	Function         gad.Function
	NilType          gad.NilType
	String           gad.Str
	Bytes            gad.Bytes
	Array            gad.Array
	Map              gad.Dict
	SyncMap          gad.SyncDict
	Int              gad.Int
	Uint             gad.Uint
	Char             gad.Char
	Float            gad.Float
	Decimal          gad.Decimal
	Bool             gad.Bool
	Flag             gad.Flag
	SourceFileSet    source.SourceFileSet
	SourceFile       source.File
	Symbol           gad.SymbolInfo
)

const (
	binNilV1 byte = iota
	binTrueV1
	binFalseV1
	binOnV1
	binOffV1
	binIntV1
	binUintV1
	binCharV1
	binFloatV1
	binDecimalV1
	binStringV1
	binBytesV1
	binArrayV1
	binMapV1
	binSyncMapV1
	binCompiledFunctionV1
	binFunctionV1
	binBuiltinFunctionV1
	binBuiltinObjTypeV1
	binSymbolV1

	binUnkownType byte = 255
)

var (
	errVarintTooSmall = errors.New("read varint error: buf too small")
	errVarintOverflow = errors.New("read varint error: value larger than 64 bits (overflow)")
	errBufTooSmall    = errors.New("read error: buf too small")
)

func init() {
	gob.Register(gad.Nil)
	gob.Register(gad.Bool(true))
	gob.Register(gad.Flag(true))
	gob.Register(gad.Int(0))
	gob.Register(gad.Uint(0))
	gob.Register(gad.Char(0))
	gob.Register(gad.Float(0))
	gob.Register(gad.DecimalZero)
	gob.Register(gad.Str(""))
	gob.Register(gad.Bytes(nil))
	gob.Register(gad.Array(nil))
	gob.Register(gad.Dict(nil))
	gob.Register((*gad.Error)(nil))
	gob.Register((*gad.RuntimeError)(nil))
	gob.Register((*gad.SyncDict)(nil))
	gob.Register((*gad.ObjectPtr)(nil))
	gob.Register((*time.Time)(nil))
	gob.Register((*json.EncoderOptions)(nil))
	gob.Register((*json.RawMessage)(nil))
	gob.Register((*gad.SymbolInfo)(nil))
	gob.Register(([]*gad.SymbolInfo)(nil))
}

// MarshalBinary implements encoding.BinaryMarshaler
func (bc *Bytecode) MarshalBinary() (data []byte, err error) {
	switch BytecodeVersion {
	case 1:
		var buf bytes.Buffer
		if err = bc.bytecodeV1Encoder(&buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	default:
		panic("invalid Bytecode version:" + strconv.Itoa(int(BytecodeVersion)))
	}
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
// Do not use this method if builtin modules are used, instead use Decode method.
func (bc *Bytecode) UnmarshalBinary(data []byte) error {
	if len(data) < 6 {
		return &gad.Error{
			Name:    "encoder.Bytecode.UnmarshalBinary",
			Message: "invalid data",
		}
	}

	sig := binary.BigEndian.Uint32(data[0:4])
	if sig != BytecodeSignature {
		return &gad.Error{
			Name:    "encoder.Bytecode.UnmarshalBinary",
			Message: "signature mismatch",
		}
	}

	version := binary.BigEndian.Uint16(data[4:6])
	switch version {
	case BytecodeVersion:
		buf := bytes.NewBuffer(data[6:])
		err := bc.bytecodeV1Decoder(buf)
		if err != nil {
			return err
		}
		return nil
	default:
		return &gad.Error{
			Name:    "encoder.Bytecode.UnmarshalBinary",
			Message: "unsupported version:" + strconv.Itoa(int(version)),
		}
	}
}

func putBytecodeHeader(w io.Writer) (err error) {
	sig := make([]byte, 4)
	binary.BigEndian.PutUint32(sig, BytecodeSignature)
	if _, err = io.Copy(w, bytes.NewReader(sig)); err != nil {
		return
	}

	bcVersion := make([]byte, 2)
	binary.BigEndian.PutUint16(bcVersion, BytecodeVersion)

	if _, err = io.Copy(w, bytes.NewReader(bcVersion)); err != nil {
		return
	}
	return nil
}

func (bc *Bytecode) bytecodeV1Encoder(w io.Writer) (err error) {
	if err = putBytecodeHeader(w); err != nil {
		return
	}

	// FileSet, field #0
	if bc.FileSet != nil {
		_ = writeByteTo(w, 0)
		var data []byte
		fs := (*SourceFileSet)(bc.FileSet)
		if data, err = fs.MarshalBinary(); err != nil {
			return
		}
		var sz []byte
		if sz, err = Int(len(data)).MarshalBinary(); err != nil {
			return
		}
		_, _ = w.Write(sz)
		_, _ = w.Write(data)
	}

	// Main, field #1
	if bc.Main != nil {
		_ = writeByteTo(w, 1)
		var data []byte
		if data, err = (*CompiledFunction)(bc.Main).MarshalBinary(); err != nil {
			return
		}
		if _, err = w.Write(data); err != nil {
			return
		}
	}

	// Constants, field #2
	if bc.Constants != nil {
		_ = writeByteTo(w, 2)
		var data []byte
		if data, err = Array(bc.Constants).MarshalBinary(); err != nil {
			return
		}
		if _, err = w.Write(data); err != nil {
			return
		}
	}

	// NumModules, field #3
	if bc.NumModules > 0 {
		_ = writeByteTo(w, 3)
		var data []byte
		data, err = Int(bc.NumModules).MarshalBinary()
		if err != nil {
			return
		}
		if _, err = w.Write(data); err != nil {
			return
		}
	}

	// NumEmbeds, field #4
	if bc.NumEmbeds > 0 {
		_ = writeByteTo(w, 4)
		var data []byte
		data, err = Int(bc.NumEmbeds).MarshalBinary()
		if err != nil {
			return
		}
		if _, err = w.Write(data); err != nil {
			return
		}
	}
	return nil
}

func (bc *Bytecode) bytecodeV1Decoder(r *bytes.Buffer) error {
	for {
		field, err := r.ReadByte()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch field {
		case 0:
			obj, err := DecodeObject(r)
			if err != nil {
				return err
			}

			sz := obj.(gad.Int)
			if sz <= 0 {
				continue
			}

			data := make([]byte, sz)
			if _, err = io.ReadFull(r, data); err != nil {
				return err
			}

			var fs SourceFileSet
			if err = fs.UnmarshalBinary(data); err != nil {
				return err
			}
			bc.FileSet = (*source.SourceFileSet)(&fs)
		case 1:
			f, err := DecodeObject(r)
			if err != nil {
				return err
			}

			bc.Main = f.(*gad.CompiledFunction)
		case 2:
			obj, err := DecodeObject(r)
			if err != nil {
				return err
			}

			bc.Constants = obj.(gad.Array)
		case 3:
			num, err := DecodeObject(r)
			if err != nil {
				return err
			}

			bc.NumModules = int(num.(gad.Int))
		case 4:
			num, err := DecodeObject(r)
			if err != nil {
				return err
			}

			bc.NumEmbeds = int(num.(gad.Int))
		default:
			return errors.New("unknown field:" + strconv.Itoa(int(field)))
		}
	}
}

// DecodeObject decodes and returns Object from a io.Reader which is encoded with MarshalBinary.
func DecodeObject(r io.Reader) (gad.Object, error) {
	btype, err := readByteFrom(r)
	if err != nil {
		return nil, err
	}

	switch btype {
	case binNilV1:
		return gad.Nil, nil
	case binTrueV1:
		return gad.True, nil
	case binFalseV1:
		return gad.False, nil
	case binOnV1:
		return gad.Yes, nil
	case binOffV1:
		return gad.No, nil
	case binIntV1,
		binUintV1,
		binFloatV1,
		binCharV1:

		size, err := readByteFrom(r)
		if err != nil {
			return nil, err
		}

		buf := make([]byte, 2+size)
		buf[0] = btype
		buf[1] = size
		if size > 0 {
			if _, err = io.ReadFull(r, buf[2:]); err != nil {
				return nil, err
			}
		}

		switch btype {
		case binIntV1:
			var v Int
			if err = v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			return gad.Int(v), nil
		case binUintV1:
			var v Uint
			if err = v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			return gad.Uint(v), nil
		case binFloatV1:
			var v Float
			if err = v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			return gad.Float(v), nil
		case binDecimalV1:
			var v Decimal
			if err = v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			return gad.Decimal(v), nil
		case binCharV1:
			var v Char
			if err = v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			return gad.Char(v), nil
		}
	case binDecimalV1:
		buf := make([]byte, 2)
		if n, err := r.Read(buf); err != nil {
			return nil, err
		} else if n != 2 {
			return nil, errBufTooSmall
		}

		size := int(uint16(buf[1]) | uint16(buf[0])<<8)

		if size == 0 {
			return gad.DecimalZero, nil
		}

		buf = make([]byte, 3+size)
		buf[0] = btype
		buf[1] = byte(size >> 8)
		buf[2] = byte(size)

		if _, err = io.ReadFull(r, buf[3:]); err != nil {
			return nil, err
		}
		var v Decimal
		if err = v.UnmarshalBinary(buf); err != nil {
			return nil, err
		}
		return gad.Decimal(v), nil
	case binCompiledFunctionV1,
		binArrayV1,
		binBytesV1,
		binStringV1,
		binMapV1,
		binSyncMapV1,
		binFunctionV1,
		binBuiltinFunctionV1,
		binBuiltinObjTypeV1,
		binSymbolV1:

		var vi varintConv
		value, readBytes, err := vi.readBytes(r)
		if err != nil {
			return nil, err
		}

		if value < 0 {
			return nil, errors.New("negative value")
		}

		n := 1 + len(readBytes)
		buf := make([]byte, n+int(value))
		buf[0] = btype
		copy(buf[1:], readBytes)

		if value > 0 {
			if _, err = io.ReadFull(r, buf[n:]); err != nil {
				return nil, err
			}
		}

		switch btype {
		case binCompiledFunctionV1:
			var v CompiledFunction
			if err := v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			return (*gad.CompiledFunction)(&v), nil
		case binArrayV1:
			var v = Array{}
			if err := v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			return gad.Array(v), nil
		case binBytesV1:
			var v = Bytes{}
			if err := v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			return gad.Bytes(v), nil
		case binStringV1:
			var v String
			if err := v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			return gad.Str(v), nil
		case binMapV1:
			var v = Map{}
			if err := v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			return gad.Dict(v), nil
		case binSyncMapV1:
			var v SyncMap
			if err := v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			return (*gad.SyncDict)(&v), nil
		case binFunctionV1:
			var v Function
			if err := v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			return (*gad.Function)(&v), nil
		case binBuiltinFunctionV1:
			var v BuiltinFunction
			if err := v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			return (*gad.BuiltinFunction)(&v), nil
		case binBuiltinObjTypeV1:
			var v BuiltinObjType
			if err := v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			return (*gad.BuiltinObjType)(&v), nil
		case binSymbolV1:
			var v Symbol
			if err := v.UnmarshalBinary(buf); err != nil {
				return nil, err
			}
			si := gad.SymbolInfo(v)
			return &si, nil
		}
	case binUnkownType:
		var v gad.Object
		if err := gob.NewDecoder(r).Decode(&v); err != nil {
			return nil, err
		}
		return v, nil
	}
	return nil, errors.New(
		"decode error: unknown encoding type:" + strconv.Itoa(int(btype)),
	)
}

func writeByteTo(w io.Writer, b byte) error {
	if bw, ok := w.(io.ByteWriter); ok {
		return bw.WriteByte(b)
	}

	n, err := w.Write([]byte{b})
	if err != nil {
		return err
	}

	if n != 1 {
		return errors.New("byte write error")
	}
	return nil
}

func marshaler(o gad.Object) encoding.BinaryMarshaler {
	switch v := o.(type) {
	case gad.Bool:
		return Bool(v)
	case gad.Int:
		return Int(v)
	case gad.Uint:
		return Uint(v)
	case gad.Char:
		return Char(v)
	case gad.Float:
		return Float(v)
	case gad.Decimal:
		return Decimal(v)
	case gad.Str:
		return String(v)
	case gad.Bytes:
		return Bytes(v)
	case gad.Array:
		return Array(v)
	case gad.Dict:
		return Map(v)
	case *gad.SyncDict:
		return (*SyncMap)(v)
	case *gad.CompiledFunction:
		return (*CompiledFunction)(v)
	case *gad.Function:
		return (*Function)(v)
	case *gad.BuiltinFunction:
		return (*BuiltinFunction)(v)
	case *gad.BuiltinObjType:
		return (*BuiltinObjType)(v)
	case *gad.NilType:
		return (*NilType)(v)
	case *gad.CallerObjectWithMethods:
		return marshaler(v.CallerObject)
	default:
		return nil
	}
}
