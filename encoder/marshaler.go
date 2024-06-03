package encoder

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"math"

	"github.com/gad-lang/gad"
	"github.com/shopspring/decimal"
)

// MarshalBinary implements encoding.BinaryMarshaler
func (o *NilType) MarshalBinary() ([]byte, error) {
	return []byte{binNilV1}, nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o Bool) MarshalBinary() ([]byte, error) {
	if o {
		return []byte{binTrueV1}, nil
	}
	return []byte{binFalseV1}, nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o Flag) MarshalBinary() ([]byte, error) {
	if o {
		return []byte{binOnV1}, nil
	}
	return []byte{binOffV1}, nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o Int) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2+binary.MaxVarintLen64)
	buf[0] = binIntV1

	if o == 0 {
		buf[1] = 0
		return buf[:2], nil
	}

	n := binary.PutVarint(buf[2:], int64(o))
	buf[1] = byte(n)
	return buf[:2+n], nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o Uint) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2+binary.MaxVarintLen64)
	buf[0] = binUintV1
	if o == 0 {
		buf[1] = 0
		return buf[:2], nil
	}

	n := binary.PutUvarint(buf[2:], uint64(o))
	buf[1] = byte(n)
	return buf[:2+n], nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o Char) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2+binary.MaxVarintLen32)
	buf[0] = binCharV1
	if o == 0 {
		buf[1] = 0
		return buf[:2], nil
	}

	n := binary.PutVarint(buf[2:], int64(o))
	buf[1] = byte(n)
	return buf[:2+n], nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o Float) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2+binary.MaxVarintLen64)
	buf[0] = binFloatV1
	if o == 0 {
		buf[1] = 0
		return buf[:2], nil
	}

	n := binary.PutUvarint(buf[2:], math.Float64bits(float64(o)))
	buf[1] = byte(n)
	return buf[:2+n], nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o Decimal) MarshalBinary() ([]byte, error) {
	dec := decimal.Decimal(o)
	if dec.IsZero() {
		return []byte{binDecimalV1, 0, 0}, nil
	}
	b, err := dec.MarshalBinary()
	if err != nil {
		return nil, err
	}

	l := len(b)
	buf := make([]byte, 3+len(b))
	buf[0] = binDecimalV1
	buf[1] = byte(l >> 8)
	buf[2] = byte(l)
	copy(buf[3:], b)
	return buf, nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o String) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(binStringV1)
	size := int64(len(o))

	if size == 0 {
		buf.WriteByte(0)
		return buf.Bytes(), nil
	}

	var vi varintConv
	b := vi.toBytes(size)
	buf.Write(b)
	buf.WriteString(string(o))
	return buf.Bytes(), nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o Bytes) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(binBytesV1)
	size := int64(len(o))

	if size == 0 {
		buf.WriteByte(0)
		return buf.Bytes(), nil
	}

	var vi varintConv
	b := vi.toBytes(size)
	buf.Write(b)
	buf.Write(o)
	return buf.Bytes(), nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o Array) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(binArrayV1)
	if len(o) == 0 {
		buf.WriteByte(0)
		return buf.Bytes(), nil
	}

	var tmpBuf bytes.Buffer
	var vi varintConv
	b := vi.toBytes(int64(len(o)))
	tmpBuf.Write(b)

	for _, v := range o {
		if m := marshaler(v); m != nil {
			d, err := m.MarshalBinary()
			if err != nil {
				return nil, err
			}
			tmpBuf.Write(d)
		} else {
			tmpBuf.WriteByte(binUnkownType)
			if err := gob.NewEncoder(&tmpBuf).Encode(&v); err != nil {
				return nil, err
			}
		}
	}

	b = vi.toBytes(int64(tmpBuf.Len()))
	buf.Write(b)
	buf.Write(tmpBuf.Bytes())
	return buf.Bytes(), nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o Map) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(binMapV1)

	var tmpBuf bytes.Buffer
	var vi varintConv

	for k, v := range o {
		b := vi.toBytes(int64(len(k)))
		tmpBuf.Write(b)
		tmpBuf.WriteString(k)

		if m := marshaler(v); m != nil {
			d, err := m.MarshalBinary()
			if err != nil {
				return nil, err
			}
			tmpBuf.Write(d)
		} else {
			tmpBuf.WriteByte(binUnkownType)
			if err := gob.NewEncoder(&tmpBuf).Encode(&v); err != nil {
				return nil, err
			}
		}
	}

	b := vi.toBytes(int64(tmpBuf.Len()))
	buf.Write(b)
	buf.Write(tmpBuf.Bytes())
	return buf.Bytes(), nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o *SyncMap) MarshalBinary() ([]byte, error) {
	(*gad.SyncDict)(o).RLock()
	defer (*gad.SyncDict)(o).RUnlock()

	var buf bytes.Buffer
	if o.Value == nil {
		buf.WriteByte(binSyncMapV1)
		buf.WriteByte(0)
		return buf.Bytes(), nil
	}

	b, err := Map(o.Value).MarshalBinary()
	if err != nil {
		return nil, err
	}

	if len(b) > 0 {
		b[0] = binSyncMapV1
	}
	return b, nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o *CompiledFunction) MarshalBinary() ([]byte, error) {
	var tmpBuf bytes.Buffer
	var vi varintConv

	o2 := gad.CompiledFunction(*o)
	o3 := &o2
	o3 = o3.ClearSourceFileInfo()
	*o = CompiledFunction(*o3)

	// Name field #0
	if o.Name != "" {
		tmpBuf.WriteByte(0)
		b, _ := String(o.Name).MarshalBinary()
		tmpBuf.Write(b)
	}

	if o.AllowMethods {
		tmpBuf.WriteByte(1)
	}

	if !o.Params.Empty() {
		// NumParams field #1
		tmpBuf.WriteByte(3)

		b := vi.toBytes(int64(len(o.Params)))
		tmpBuf.Write(b)

		for _, p := range o.Params {
			b, _ := String(p.Name).MarshalBinary()
			tmpBuf.Write(b)
			symbols := make(gad.Array, len(p.Type))
			for i, info := range p.Type {
				symbols[i] = info
			}
			b, _ = Array(symbols).MarshalBinary()
			tmpBuf.Write(b)
		}
	}

	if o.NumLocals > 0 {
		// NumLocals field #4
		tmpBuf.WriteByte(4)
		b := vi.toBytes(int64(o.NumLocals))
		tmpBuf.Write(b)
	}

	if o.Instructions != nil {
		// Instructions field #5
		tmpBuf.WriteByte(5)
		data, err := Bytes(o.Instructions).MarshalBinary()
		if err != nil {
			return nil, err
		}
		tmpBuf.Write(data)
	}

	// Variadic field #6
	if o.Params.Var() {
		tmpBuf.WriteByte(6)
	}

	if l := o.NamedParams.Len(); l > 0 {
		// Variadic field #7
		tmpBuf.WriteByte(7)
		tmpBuf.Write(vi.toBytes(int64(l)))
		for _, n := range o.NamedParams.Params {
			b, _ := String(n.Name).MarshalBinary()
			tmpBuf.Write(b)
			b, _ = String(n.Value).MarshalBinary()
			tmpBuf.Write(b)
			symbols := make(gad.Array, len(n.Type))
			for i, info := range n.Type {
				symbols[i] = info
			}
			b, _ = Array(symbols).MarshalBinary()
			tmpBuf.Write(b)
			n.Type = nil
		}
	}

	// Ignore Free variables, doesn't make sense

	if o.SourceMap != nil {
		// SourceMap field #8
		tmpBuf.WriteByte(8)
		b := vi.toBytes(int64(len(o.SourceMap) * 2))
		tmpBuf.Write(b)
		for key, value := range o.SourceMap {
			b = vi.toBytes(int64(key))
			tmpBuf.Write(b)
			b = vi.toBytes(int64(value))
			tmpBuf.Write(b)
		}
	}

	var buf bytes.Buffer
	size := vi.toBytes(int64(tmpBuf.Len()))
	buf.WriteByte(binCompiledFunctionV1)
	buf.Write(size)
	buf.Write(tmpBuf.Bytes())
	return buf.Bytes(), nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o *BuiltinFunction) MarshalBinary() ([]byte, error) {
	// Note: use string name instead of index of builtin
	s, err := String(o.Name).MarshalBinary()
	if err != nil {
		return nil, err
	}

	var vi varintConv
	b := vi.toBytes(int64(len(s)))
	data := make([]byte, 0, 1+len(b)+len(s))
	data = append(data, binBuiltinFunctionV1)
	data = append(data, b...)
	data = append(data, s...)
	return data, nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o *BuiltinObjType) MarshalBinary() ([]byte, error) {
	// Note: use string name instead of index of builtin
	s, err := String(o.NameValue).MarshalBinary()
	if err != nil {
		return nil, err
	}

	var vi varintConv
	b := vi.toBytes(int64(len(s)))
	data := make([]byte, 0, 1+len(b)+len(s))
	data = append(data, binBuiltinObjTypeV1)
	data = append(data, b...)
	data = append(data, s...)
	return data, nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o *Function) MarshalBinary() ([]byte, error) {
	s, err := String(o.Name).MarshalBinary()
	if err != nil {
		return nil, err
	}

	var vi varintConv
	b := vi.toBytes(int64(len(s)))
	data := make([]byte, 0, 1+len(b)+len(s))
	data = append(data, binFunctionV1)
	data = append(data, b...)
	data = append(data, s...)
	return data, nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (sf *SourceFile) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	d, err := String(sf.Name).MarshalBinary()
	if err != nil {
		return nil, err
	}

	buf.Write(d)
	var vi varintConv
	b := vi.toBytes(int64(sf.Base))
	buf.Write(b)

	b = vi.toBytes(int64(sf.Size))
	buf.Write(b)

	b = vi.toBytes(int64(len(sf.Lines)))
	buf.Write(b)

	for _, v := range sf.Lines {
		b = vi.toBytes(int64(v))
		buf.Write(b)
	}
	return buf.Bytes(), nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (sfs *SourceFileSet) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	var vi varintConv
	b := vi.toBytes(int64(sfs.Base))
	buf.Write(b)

	b = vi.toBytes(int64(len(sfs.Files)))
	buf.Write(b)

	for _, v := range sfs.Files {
		if v == nil {
			continue
		}
		d, err := (*SourceFile)(v).MarshalBinary()
		if err != nil {
			return nil, err
		}
		b := vi.toBytes(int64(len(d)))
		buf.Write(b)
		buf.Write(d)
	}

	return buf.Bytes(), nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (s Symbol) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(binSymbolV1)
	var vi varintConv
	d, _ := String(s.Name).MarshalBinary()
	buf.Write(d)
	buf.Write(vi.toBytes(int64(s.Index)))
	buf.Write(vi.toBytes(int64(s.Scope)))
	return buf.Bytes(), nil
}
