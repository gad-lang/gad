package encoder

import (
	"errors"
	"math"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/parser/source"
)

type bcModuleCompiledFunc struct {
	moduleIndex int
	cf          *gad.CompiledFunction
}

func init() {
	ModuleSpecV1.Encode = func(w Writer, o any) (err error) {
		s := o.(*gad.ModuleSpec)
		if err = writeString(w, s.Name); err != nil {
			return
		}
		if err = writeString(w, s.URL); err != nil {
			return
		}
		if err = writeBool(w, s.Main); err != nil {
			return
		}

		if s.Path != nil {
			if err = w.WriteByte(1); err != nil {
				return
			}
			if err = WriteArray(w, s.Path, writeInt); err != nil {
				return
			}
		} else if err = w.WriteByte(0); err != nil {
			return
		}

		if err = writeBool(w, s.InitCompiledFunc != nil); err != nil {
			return
		}

		if s.InitCompiledFunc != nil {
			err = Encode(w, binCompiledFunctionV1, s.InitCompiledFunc)
		}

		return
	}

	BytecodeV1.Encode = func(w Writer, o any) (err error) {
		bc := o.(*gad.Bytecode)
		if err = writeUint32(w, BytecodeSignature); err != nil {
			return
		}
		if err = writeUint16(w, BytecodeVersion); err != nil {
			return
		}

		// FileSet, field  #0
		if bc.FileSet != nil {
			if err = w.WriteByte(0); err != nil {
				return
			}
			if err = EncodeObject(w, bc.FileSet); err != nil {
				return
			}
		}

		// Modules, field #1
		if len(bc.Modules) > 0 {
			if err = w.WriteByte(1); err != nil {
				return
			}

			var modCompiledFuncs []*bcModuleCompiledFunc

			for i, m := range bc.Modules {
				if m.InitCompiledFunc != nil {
					modCompiledFuncs = append(modCompiledFuncs, &bcModuleCompiledFunc{
						moduleIndex: i,
						cf:          m.InitCompiledFunc,
					})
					m.InitCompiledFunc = nil
				}
			}

			defer func() {
				for _, mcf := range modCompiledFuncs {
					bc.Modules[mcf.moduleIndex].InitCompiledFunc = mcf.cf
				}
			}()

			if err = EncodeArray(w, bc.Modules); err != nil {
				return
			}

			if err = WriteArray(w, modCompiledFuncs, func(w Writer, v *bcModuleCompiledFunc) (err error) {
				if err = writeInt(w, v.moduleIndex); err != nil {
					return
				}

				return EncodeObject(w, v.cf)
			}); err != nil {
				return
			}
		}

		// Constants, field #2
		if len(bc.Constants) > 0 {
			if err = w.WriteByte(2); err != nil {
				return
			}
			if err = EncodeArray(w, bc.Constants); err != nil {
				return
			}
		}

		// Main, field #3
		if bc.Main != nil {
			if err = w.WriteByte(3); err != nil {
				return
			}
			if err = EncodeObject(w, bc.Main); err != nil {
				return
			}
		}

		err = w.WriteByte(FieldEOF)
		return
	}

	SourceFileV1.Encode = func(w Writer, o any) (err error) {
		sf := o.(*source.File)

		if err = writeString(w, sf.Name); err != nil {
			return
		}

		if err = writeInt(w, sf.Base); err != nil {
			return
		}

		if err = writeInt(w, sf.Size); err != nil {
			return
		}

		return WriteArray(w, sf.Lines, writeInt)
	}

	SourceFileSetV1.Encode = func(w Writer, o any) (err error) {
		sf := o.(*source.FileSet)
		if err = writeInt(w, sf.Base); err != nil {
			return
		}
		if err = EncodeArray(w, sf.Files); err != nil {
			return
		}

		for i, file := range sf.Files {
			if file == sf.LastFile {
				err = writeInt(w, i)
				return
			}
		}
		return errors.New("source.FileSet does not contain a LastFile")
	}

	SymbolInfoV1.Encode = func(w Writer, o any) (err error) {
		s := o.(*gad.SymbolInfo)
		if err = writeString(w, s.Name); err != nil {
			return
		}
		if err = writeInt(w, s.Index); err != nil {
			return
		}
		return writeInt(w, int(s.Scope))
	}

	NilV1.Encode = func(w Writer, o any) (err error) {
		return nil
	}

	BoolV1.Encode = func(w Writer, o any) (err error) {
		if o.(gad.Bool) {
			return w.WriteByte(1)
		}
		return w.WriteByte(0)
	}

	FlagV1.Encode = func(w Writer, o any) (err error) {
		if o.(gad.Flag) {
			return w.WriteByte(1)
		}
		return w.WriteByte(0)
	}

	IntV1.Encode = func(w Writer, o any) (err error) {
		return writeInt64(w, int64(o.(gad.Int)))
	}

	UintV1.Encode = func(w Writer, o any) (err error) {
		return writeUint64(w, uint64(o.(gad.Uint)))
	}

	CharV1.Encode = func(w Writer, o any) (err error) {
		return writeUint64(w, uint64(o.(gad.Char)))
	}

	FloatV1.Encode = func(w Writer, o any) (err error) {
		return writeUint64(w, math.Float64bits(float64(o.(gad.Float))))
	}

	DecimalV1.Encode = func(w Writer, o any) (err error) {
		var data []byte
		if data, err = o.(gad.Decimal).GobEncode(); err != nil {
			return
		}
		return writeChunk(w, data)
	}

	StrV1.Encode = func(w Writer, o any) (err error) {
		return writeString(w, string(o.(gad.Str)))
	}

	RawStrV1.Encode = func(w Writer, o any) (err error) {
		return writeString(w, string(o.(gad.RawStr)))
	}

	BytesV1.Encode = func(w Writer, o any) (err error) {
		return writeChunk(w, o.(gad.Bytes))
	}

	ArrayV1.Encode = func(w Writer, o any) (err error) {
		return EncodeArray(w, o.(gad.Array))
	}

	DictV1.Encode = func(w Writer, o any) (err error) {
		return EncodeDict(w, o.(gad.Dict))
	}

	SyncDictV1.Encode = func(w Writer, o any) (err error) {
		return EncodeDict(w, o.(*gad.SyncDict).Value)
	}

	CompiledFunctionV1.Encode = func(w Writer, o any) (err error) {
		cf := o.(*gad.CompiledFunction)

		if m := cf.GetModule(); m != nil {
			if err = w.WriteByte(0); err != nil {
				return
			}
			if err = writeInt(w, m.Index); err != nil {
				return
			}
		}

		if len(cf.FuncName) > 0 {
			if err = w.WriteByte(1); err != nil {
				return
			}
			if err = writeString(w, cf.FuncName); err != nil {
				return
			}
		}

		if cf.AllowMethods {
			if err = w.WriteByte(2); err != nil {
				return
			}
		}

		if l := cf.Params.Len(); l > 0 {
			if err = w.WriteByte(3); err != nil {
				return
			}

			if err = writeInt(w, l); err != nil {
				return
			}

			for i := 0; i < l; i++ {
				p := cf.Params.Items[i]
				if err = writeString(w, p.Name); err != nil {
					return
				}

				if err = writeBool(w, p.Var); err != nil {
					return
				}

				if err = EncodeObject(w, p.Symbol); err != nil {
					return
				}

				if err = EncodeArray(w, p.TypesSymbols); err != nil {
					return
				}
			}
		}

		if err = w.WriteByte(4); err != nil {
			return
		}

		if err = writeInt(w, cf.NumLocals); err != nil {
			return
		}

		if err = w.WriteByte(5); err != nil {
			return
		}

		if err = writeChunk(w, cf.Instructions); err != nil {
			return
		}

		if l := cf.NamedParams.Len(); l > 0 {
			if err = w.WriteByte(6); err != nil {
				return
			}

			if err = writeInt(w, l); err != nil {
				return
			}

			for i := 0; i < l; i++ {
				p := cf.NamedParams.Items[i]

				if err = EncodeObject(w, p.Symbol); err != nil {
					return
				}

				if err = writeString(w, p.Value); err != nil {
					return
				}

				if err = writeBool(w, p.Var); err != nil {
					return
				}

				if err = EncodeArray(w, p.TypesSymbols); err != nil {
					return
				}
			}
		}

		if l := len(cf.SourceMap); l > 0 {
			if err = w.WriteByte(7); err != nil {
				return
			}
			if err = writeInt(w, l); err != nil {
				return
			}

			for k, v := range cf.SourceMap {
				if err = writeInt(w, k); err != nil {
					return
				}
				if err = writeInt(w, v); err != nil {
					return
				}
			}
		}

		err = w.WriteByte(FieldEOF)
		return
	}

	ErrorV1.Encode = func(w Writer, o any) (err error) {
		e := o.(*gad.Error)
		if err = writeString(w, e.Name); err != nil {
			return
		}
		return writeString(w, e.Message)
	}
}
