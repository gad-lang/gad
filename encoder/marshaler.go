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
func (o CompiledFunction) MarshalBinary() ([]byte, error) {
	var tmpBuf bytes.Buffer
	var vi varintConv

	f := *o.CompiledFunction

	// ModuleConstantIndex field #0
	if mod := f.GetModule(); mod != nil {
		// the field
		tmpBuf.WriteByte(0)
		b := vi.toBytes(int64(f.GetModule().ConstantIndex))
		tmpBuf.Write(b)
	}

	// Name field #1
	if f.FuncName != "" {
		tmpBuf.WriteByte(1)
		b, _ := String(f.FuncName).MarshalBinary()
		tmpBuf.Write(b)
	}

	// AllowMethods field #2
	if f.AllowMethods {
		tmpBuf.WriteByte(2)
	}

	if !f.Params.Empty() {
		// NumParams field #3
		tmpBuf.WriteByte(3)

		b := vi.toBytes(int64(f.Params.Len()))
		tmpBuf.Write(b)

		for _, p := range f.Params.Items {
			b, _ := String(p.Name).MarshalBinary()
			tmpBuf.Write(b)

			b, _ = (*SymbolInfo)(p.Symbol).MarshalBinary()
			tmpBuf.Write(b)

			if p.Var {
				tmpBuf.WriteByte(1)
			} else {
				tmpBuf.WriteByte(0)
			}

			symbols := make(gad.Array, len(p.TypesSymbols))
			for i, info := range p.TypesSymbols {
				symbols[i] = info
			}
			b, _ = Array(symbols).MarshalBinary()
			tmpBuf.Write(b)
		}
	}

	if f.NumLocals > 0 {
		// NumLocals field #4
		tmpBuf.WriteByte(4)
		b := vi.toBytes(int64(f.NumLocals))
		tmpBuf.Write(b)
	}

	if f.Instructions != nil {
		// Instructions field #5
		tmpBuf.WriteByte(5)
		data, err := Bytes(f.Instructions).MarshalBinary()
		if err != nil {
			return nil, err
		}
		tmpBuf.Write(data)
	}

	if l := f.NamedParams.Len(); l > 0 {
		// named params field #7
		tmpBuf.WriteByte(7)
		tmpBuf.Write(vi.toBytes(int64(l)))

		for _, n := range f.NamedParams.Items {
			b, _ := (*SymbolInfo)(n.Symbol).MarshalBinary()
			tmpBuf.Write(b)

			b, _ = String(n.Value).MarshalBinary()
			tmpBuf.Write(b)

			if n.Var {
				tmpBuf.WriteByte(1)
			} else {
				tmpBuf.WriteByte(0)
			}

			symbols := make(gad.Array, len(n.TypesSymbols))
			for i, info := range n.TypesSymbols {
				symbols[i] = info
			}
			b, _ = Array(symbols).MarshalBinary()
			tmpBuf.Write(b)
		}
	}

	// Ignore Free variables, doesn't make sense

	if f.SourceMap != nil {
		// SourceMap field #8
		tmpBuf.WriteByte(8)
		b := vi.toBytes(int64(len(f.SourceMap) * 2))
		tmpBuf.Write(b)
		for key, value := range f.SourceMap {
			b = vi.toBytes(int64(key))
			tmpBuf.Write(b)
			b = vi.toBytes(int64(value))
			tmpBuf.Write(b)
		}
	}

	var buf bytes.Buffer
	buf.WriteByte(binCompiledFunctionV1)
	buf.Write(vi.toBytes(int64(tmpBuf.Len())))
	buf.Write(tmpBuf.Bytes())
	return buf.Bytes(), nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (o *BuiltinFunction) MarshalBinary() ([]byte, error) {
	// Note: use string name instead of index of builtin
	s, err := String(o.FuncName).MarshalBinary()
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
func (s *SymbolInfo) MarshalBinary() ([]byte, error) {
	var tmpBuf bytes.Buffer
	var vi varintConv
	d, _ := String(s.Name).MarshalBinary()
	tmpBuf.Write(d)
	tmpBuf.Write(vi.toBytes(int64(s.Index)))
	tmpBuf.Write(vi.toBytes(int64(s.Scope)))

	var buf bytes.Buffer
	buf.WriteByte(binSymbolV1)
	size := int64(tmpBuf.Len())
	buf.Write(vi.toBytes(size))
	buf.Write(tmpBuf.Bytes())
	return buf.Bytes(), nil
}

// MarshalBinary implements encoding.BinaryMarshaler
func (s Module) MarshalBinary() ([]byte, error) {
	var (
		buf    bytes.Buffer
		tmpBuf bytes.Buffer
		vi     varintConv
		m      = gad.Module(s)
	)

	tmpBuf.Write(vi.toBytes(int64(m.ConstantIndex)))

	d, _ := String(m.Info.Name).MarshalBinary()
	tmpBuf.Write(d)

	d, _ = String(m.Info.File).MarshalBinary()
	tmpBuf.Write(d)

	d, _ = Array(m.Params.Positional).MarshalBinary()
	tmpBuf.Write(d)

	if cf, _ := m.Init.(*gad.CompiledFunction); cf != nil {
		tmpBuf.WriteByte(1)
		d, _ = CompiledFunction{CompiledFunction: cf}.MarshalBinary()
		tmpBuf.Write(d)
	} else {
		tmpBuf.WriteByte(0)
	}

	d, _ = Array(m.Params.Named.ToArray()).MarshalBinary()
	tmpBuf.Write(d)

	buf.WriteByte(binModuleV1)
	buf.Write(vi.toBytes(int64(tmpBuf.Len())))
	buf.Write(tmpBuf.Bytes())
	return buf.Bytes(), nil
}
