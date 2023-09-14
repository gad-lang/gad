package encoder

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/parser"
	"github.com/shopspring/decimal"
)

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *NilType) UnmarshalBinary(data []byte) error {
	if len(data) < 1 || data[0] != binNilV1 {
		return errors.New("invalid gad.Nil data")
	}
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *Bool) UnmarshalBinary(data []byte) error {
	if len(data) < 1 {
		return errors.New("invalid gad.Bool data")
	}

	if data[0] == binTrueV1 {
		*o = true
		return nil
	}

	if data[0] == binFalseV1 {
		*o = false
		return nil
	}
	return errors.New("invalid gad.Bool data")
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *Int) UnmarshalBinary(data []byte) error {
	if len(data) < 2 || data[0] != binIntV1 {
		return errors.New("invalid gad.Int data")
	}

	size := int(data[1])
	if size <= 0 {
		return nil
	}

	if len(data) < 2+size {
		return errors.New("invalid gad.Int data size")
	}

	v, n := binary.Varint(data[2:])
	if n < 1 {
		if n == 0 {
			return errors.New("gad.Int data buffer too small")
		}
		return errors.New("gad.Int value larger than 64 bits")
	}

	*o = Int(v)
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *Uint) UnmarshalBinary(data []byte) error {
	if len(data) < 2 || data[0] != binUintV1 {
		return errors.New("invalid gad.Uint data")
	}

	size := int(data[1])
	if size <= 0 {
		return nil
	}

	if len(data) < 2+size {
		return errors.New("invalid gad.Uint data size")
	}

	v, n := binary.Uvarint(data[2:])
	if n < 1 {
		if n == 0 {
			return errors.New("gad.Uint data buffer too small")
		}
		return errors.New("gad.Uint value larger than 64 bits")
	}

	*o = Uint(v)
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *Char) UnmarshalBinary(data []byte) error {
	if len(data) < 2 || data[0] != binCharV1 {
		return errors.New("invalid gad.Char data")
	}

	size := int(data[1])
	if size <= 0 {
		return nil
	}

	if len(data) < 2+size {
		return errors.New("invalid gad.Char data size")
	}

	v, n := binary.Varint(data[2:])
	if n < 1 {
		if n == 0 {
			return errors.New("gad.Char data buffer too small")
		}
		return errors.New("gad.Char value larger than 64 bits")
	}

	if int64(rune(v)) != v {
		return errors.New("gad.Char value larger than 32 bits")
	}

	*o = Char(v)
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *Float) UnmarshalBinary(data []byte) error {
	if len(data) < 2 || data[0] != binFloatV1 {
		return errors.New("invalid gad.Float data")
	}

	size := int(data[1])
	if size <= 0 {
		return nil
	}

	if len(data) < 2+size {
		return errors.New("invalid gad.Float data size")
	}

	v, n := binary.Uvarint(data[2:])
	if n < 1 {
		if n == 0 {
			return errors.New("gad.Float data buffer too small")
		}
		return errors.New("gad.Float value larger than 64 bits")
	}

	*o = Float(math.Float64frombits(v))
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *Decimal) UnmarshalBinary(data []byte) error {
	if len(data) < 3 || data[0] != binDecimalV1 {
		return errors.New("invalid gad.Decimal data")
	}
	size := int(uint16(data[2]) | uint16(data[1])<<8)
	if size <= 0 {
		*o = Decimal(gad.DecimalZero)
		return nil
	}

	if len(data) < 3+size {
		return errors.New("invalid gad.Decimal data size")
	}

	var dec decimal.Decimal
	if err := dec.UnmarshalBinary(data[3:]); err != nil {
		return err
	}

	*o = Decimal(dec)
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *String) UnmarshalBinary(data []byte) error {
	if len(data) < 2 || data[0] != binStringV1 {
		return errors.New("invalid gad.String data")
	}

	size, offset, err := toVarint(data[1:])
	if err != nil {
		return err
	}

	if size <= 0 {
		return nil
	}

	ub := 1 + offset + int(size)
	if len(data) < ub {
		return errors.New("invalid gad.String data size")
	}

	*o = String(data[1+offset : ub])
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *Bytes) UnmarshalBinary(data []byte) error {
	if len(data) < 2 || data[0] != binBytesV1 {
		return errors.New("invalid gad.Bytes data")
	}

	size, offset, err := toVarint(data[1:])
	if err != nil {
		return err
	}

	if size <= 0 {
		return nil
	}

	ub := 1 + offset + int(size)
	if len(data) < ub {
		return errors.New("invalid gad.Bytes data size")
	}

	*o = []byte(string(data[1+offset : ub]))
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *Array) UnmarshalBinary(data []byte) error {
	if len(data) < 2 || data[0] != binArrayV1 {
		return errors.New("invalid gad.Array data")
	}

	size, offset, err := toVarint(data[1:])
	if err != nil {
		return err
	}

	if size <= 0 {
		return nil
	}
	ub := 1 + offset + int(size)
	if len(data) < ub {
		return errors.New("invalid gad.Array data size")
	}

	rd := bytes.NewReader(data[1+offset : ub])
	var vi varintConv
	vi.reader = rd

	length, err := vi.read()
	if err != nil {
		return err
	}

	arr := make([]gad.Object, 0, int(length))
	for rd.Len() > 0 {
		o, err := DecodeObject(rd)
		if err != nil {
			return err
		}
		arr = append(arr, o)
	}

	*o = arr
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *Map) UnmarshalBinary(data []byte) error {
	if len(data) < 2 || data[0] != binMapV1 {
		return errors.New("invalid gad.Map data")
	}

	size, offset, err := toVarint(data[1:])
	if err != nil {
		return err
	}

	if size <= 0 {
		return nil
	}

	if len(data) < 1+offset+int(size) {
		return errors.New("invalid gad.Map data size")
	}

	rd := bytes.NewReader(data[1+offset : 1+offset+int(size)])
	strBuf := bytes.NewBuffer(nil)
	var vi varintConv
	vi.reader = rd
	m := *o

	for rd.Len() > 0 {
		value, err := vi.read()
		if err != nil {
			return err
		}

		var k string
		if value > 0 {
			strBuf.Reset()
			if _, err = io.CopyN(strBuf, rd, value); err != nil {
				return err
			}
			k = strBuf.String()
		}

		o, err := DecodeObject(rd)
		if err != nil {
			return err
		}
		m[k] = o
	}
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *SyncMap) UnmarshalBinary(data []byte) error {
	if len(data) < 2 || data[0] != binSyncMapV1 {
		return errors.New("invalid gad.SyncMap data")
	}

	if data[1] == 0 {
		return nil
	}

	data[0] = binMapV1
	m := Map{}
	if err := m.UnmarshalBinary(data); err != nil {
		data[0] = binSyncMapV1
		return err
	}

	data[0] = binSyncMapV1
	o.Value = (gad.Map)(m)
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *CompiledFunction) UnmarshalBinary(data []byte) error {
	if len(data) < 2 || data[0] != binCompiledFunctionV1 {
		return errors.New("invalid gad.CompiledFunction data")
	}

	size, offset, err := toVarint(data[1:])
	if err != nil {
		return err
	}

	if size <= 0 {
		return nil
	}

	rd := bytes.NewReader(data[1+offset : 1+offset+int(size)])
	var vi varintConv
	vi.reader = rd

	for rd.Len() > 0 {
		field, err := rd.ReadByte()
		if err != nil {
			return err
		}
		switch field {
		case 0:
			v, err := vi.read()
			if err != nil {
				return err
			}
			o.Params.Len = int(v)
		case 1:
			v, err := vi.read()
			if err != nil {
				return err
			}
			o.NumLocals = int(v)
		case 2:
			obj, err := DecodeObject(rd)
			if err != nil {
				return err
			}
			o.Instructions = obj.(gad.Bytes)
		case 3:
			o.Params.Var = true
		case 4:
			v, err := vi.read()
			if err != nil {
				return err
			}
			namedParams := make([]*gad.NamedParam, int(v))
			for i := range namedParams {
				if name, err := DecodeObject(rd); err != nil {
					return err
				} else if value, err := DecodeObject(rd); err != nil {
					return err
				} else {
					namedParams[i] = &gad.NamedParam{Name: string(name.(gad.String)), Value: string(value.(gad.String))}
				}
			}
			o.NamedParams = *gad.NewNamedParams(namedParams...)
		case 5:
			return errors.New("unexpected field #6")
		case 6:
			length, err := vi.read()
			if err != nil {
				return err
			}

			sz := int(length / 2)
			// always put size to the map to decode faster
			o.SourceMap = make(map[int]int, sz)
			for i := 0; i < sz; i++ {
				key, err := vi.read()
				if err != nil {
					return err
				}
				value, err := vi.read()
				if err != nil {
					return err
				}
				o.SourceMap[int(key)] = int(value)
			}
		default:
			return errors.New("unknown field:" + strconv.Itoa(int(field)))
		}
	}
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *BuiltinFunction) UnmarshalBinary(data []byte) error {
	if len(data) < 2 || data[0] != binBuiltinFunctionV1 {
		return errors.New("invalid gad.BuiltinFunction data")
	}

	size, offset, err := toVarint(data[1:])
	if err != nil {
		return err
	}

	if size <= 0 {
		return errors.New("invalid gad.BuiltinFunction data size")
	}

	var s String
	if err := s.UnmarshalBinary(data[1+offset:]); err != nil {
		return err
	}

	index, ok := gad.BuiltinsMap[string(s)]
	if !ok {
		return fmt.Errorf("builtin '%s' not found", s)
	}

	obj := gad.BuiltinObjects[index]
	f, ok := obj.(*BuiltinFunction)
	if ok {
		*o = *f
		return nil
	}
	return fmt.Errorf("builtin '%s' not a gad.BuiltinFunction type", s)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (o *Function) UnmarshalBinary(data []byte) error {
	if len(data) < 2 || data[0] != binFunctionV1 {
		return errors.New("invalid gad.Function data")
	}

	size, offset, err := toVarint(data[1:])
	if err != nil {
		return err
	}

	if size <= 0 {
		return errors.New("invalid gad.Function data size")
	}

	var s String
	if err := s.UnmarshalBinary(data[1+offset:]); err != nil {
		return err
	}
	o.Name = string(s)
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (sf *SourceFile) UnmarshalBinary(data []byte) error {
	rd := bytes.NewReader(data)

	obj, err := DecodeObject(rd)
	if err != nil {
		return err
	}

	sf.Name = obj.String()
	var vi varintConv
	vi.reader = rd
	v, err := vi.read()
	if err != nil {
		return err
	}

	sf.Base = int(v)

	v, err = vi.read()
	if err != nil {
		return err
	}

	sf.Size = int(v)

	v, err = vi.read()
	if err != nil {
		return err
	}

	length := int(v)

	lines := make([]int, length)
	for i := 0; i < length; i++ {
		v, err = vi.read()
		if err != nil {
			return err
		}
		lines[i] = int(v)
	}

	if rd.Len() > 0 {
		return errors.New("unread bytes")
	}

	sf.Lines = lines
	return nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (sfs *SourceFileSet) UnmarshalBinary(data []byte) error {
	rd := bytes.NewReader(data)
	var vi varintConv
	vi.reader = rd
	v, err := vi.read()
	if err != nil {
		return err
	}

	sfs.Base = int(v)

	v, err = vi.read()
	if err != nil {
		return err
	}

	length := int(v)
	files := make([]*parser.SourceFile, length)

	for i := 0; i < length; i++ {
		v, err = vi.read()
		if err != nil {
			return err
		}
		data := make([]byte, v)
		if _, err = io.ReadFull(rd, data); err != nil {
			return err
		}
		var file SourceFile
		if err = file.UnmarshalBinary(data); err != nil {
			return err
		}
		files[i] = (*parser.SourceFile)(&file)
	}

	if rd.Len() > 0 {
		return errors.New("unread bytes")
	}

	sfs.Files = files
	return nil
}

func readByteFrom(r io.Reader) (byte, error) {
	if br, ok := r.(io.ByteReader); ok {
		return br.ReadByte()
	}

	var one = []byte{0}
	n, err := r.Read(one)
	if err != nil {
		if err == io.EOF {
			if n == 1 {
				return one[0], nil
			}
		}
		return 0, err
	}

	if n == 1 {
		return one[0], nil
	}
	return 0, errors.New("byte read error")
}

type varintConv struct {
	buf    [1 + binary.MaxVarintLen64]byte
	reader *bytes.Reader
}

func (vi *varintConv) toBytes(v int64) []byte {
	n := binary.PutVarint(vi.buf[1:], v)
	vi.buf[0] = byte(n)
	return vi.buf[:n+1]
}

func (vi *varintConv) read() (value int64, err error) {
	var n byte
	n, err = vi.reader.ReadByte()
	if err != nil {
		return
	}

	if int(n) > len(vi.buf) {
		return 0, errVarintOverflow
	}

	data := vi.buf[:n]
	if n == 0 {
		return
	}

	if _, err = io.ReadFull(vi.reader, data); err != nil {
		return
	}

	var offset int
	value, offset = binary.Varint(data)
	if offset < 1 {
		if offset == 0 {
			err = errVarintTooSmall
			return
		}
		err = errVarintOverflow
		return
	}
	return
}

func (vi *varintConv) readBytes(r io.Reader) (value int64, readBytes []byte, err error) {
	var n byte
	n, err = readByteFrom(r)
	if err != nil {
		return
	}

	if 1+int(n) > len(vi.buf) {
		return 0, nil, errVarintOverflow
	}

	readBytes = vi.buf[:1+n]
	readBytes[0] = n
	if n == 0 {
		return
	}

	if _, err = io.ReadFull(r, readBytes[1:]); err != nil {
		return
	}

	var offset int
	value, offset = binary.Varint(readBytes[1:])
	if offset < 1 {
		if offset == 0 {
			err = errVarintTooSmall
			return
		}
		err = errVarintOverflow
		return
	}
	return
}

// toVarint converts a byte slice to int64. If length of slice is 0, it panics.
func toVarint(data []byte) (value int64, offset int, err error) {
	size := int(data[0])
	if size == 0 {
		offset = 1
		return
	}

	if len(data) < 1+size {
		err = errVarintTooSmall
		return
	}

	value, offset = binary.Varint(data[1:])
	if offset < 1 {
		if offset == 0 {
			err = errVarintTooSmall
			return
		}
		err = errVarintOverflow
		return
	}

	offset++
	return
}
