package encoder

import (
	"errors"
	"io"
	"math"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/parser/source"
)

type bcModuleCompiledFunc struct {
	moduleIndex int
	cf          *gad.CompiledFunction
}

func init() {
	ModuleSpecV1.Encode = func(ctx *WriteContext, o any) (err error) {
		s := o.(*gad.ModuleSpec)
		if err = writeString(ctx, s.Name); err != nil {
			return
		}
		if err = writeString(ctx, s.URL); err != nil {
			return
		}
		if err = writeBool(ctx, s.Main); err != nil {
			return
		}

		if s.Path != nil {
			if err = ctx.WriteByte(1); err != nil {
				return
			}
			if err = WriteArray(ctx, s.Path, writeInt); err != nil {
				return
			}
		} else if err = ctx.WriteByte(0); err != nil {
			return
		}

		if err = writeBool(ctx, s.InitCompiledFunc != nil); err != nil {
			return
		}

		if s.InitCompiledFunc != nil {
			err = Encode(ctx, binCompiledFunctionV1, s.InitCompiledFunc)
		}

		return
	}

	BytecodeV1.Encode = func(ctx *WriteContext, o any) (err error) {
		bc := o.(*gad.Bytecode)
		if err = writeUint32(ctx, BytecodeSignature); err != nil {
			return
		}
		if err = writeUint16(ctx, BytecodeVersion); err != nil {
			return
		}

		// FileSet, field  #0
		if bc.FileSet != nil {
			if err = ctx.WriteByte(0); err != nil {
				return
			}
			if err = EncodeObject(ctx, bc.FileSet); err != nil {
				return
			}
		}

		// Modules, field #1
		if len(bc.Modules) > 0 {
			if err = ctx.WriteByte(1); err != nil {
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

			if err = EncodeArray(ctx, bc.Modules); err != nil {
				return
			}

			if err = WriteArray(ctx, modCompiledFuncs, func(w Writer, v *bcModuleCompiledFunc) (err error) {
				if err = writeInt(w, v.moduleIndex); err != nil {
					return
				}

				return EncodeObject(ctx, v.cf)
			}); err != nil {
				return
			}
		}

		// Constants, field #2
		if len(bc.Constants) > 0 {
			if err = ctx.WriteByte(2); err != nil {
				return
			}
			if err = EncodeArray(ctx, bc.Constants); err != nil {
				return
			}
		}

		// Main, field #3
		if bc.Main != nil {
			if err = ctx.WriteByte(3); err != nil {
				return
			}
			if err = EncodeObject(ctx, bc.Main); err != nil {
				return
			}
		}

		err = ctx.WriteByte(FieldEOF)
		return
	}

	SourceFileV1.Encode = func(ctx *WriteContext, o any) (err error) {
		sf := o.(*source.File)

		if err = writeString(ctx, sf.Name); err != nil {
			return
		}

		if err = writeInt(ctx, sf.Base); err != nil {
			return
		}

		if err = writeInt(ctx, sf.Size); err != nil {
			return
		}

		return WriteArray(ctx, sf.Lines, writeInt)
	}

	SourceFileSetV1.Encode = func(ctx *WriteContext, o any) (err error) {
		sf := o.(*source.FileSet)
		if err = writeInt(ctx, sf.Base); err != nil {
			return
		}
		if err = EncodeArray(ctx, sf.Files); err != nil {
			return
		}

		for i, file := range sf.Files {
			if file == sf.LastFile {
				err = writeInt(ctx, i)
				return
			}
		}
		return errors.New("source.FileSet does not contain a LastFile")
	}

	SymbolInfoV1.Encode = func(ctx *WriteContext, o any) (err error) {
		s := o.(*gad.SymbolInfo)
		if err = writeString(ctx, s.Name); err != nil {
			return
		}
		if err = writeInt(ctx, s.Index); err != nil {
			return
		}
		return writeInt(ctx, int(s.Scope))
	}

	NilV1.Encode = func(ctx *WriteContext, o any) (err error) {
		return nil
	}

	BoolV1.Encode = func(ctx *WriteContext, o any) (err error) {
		if o.(gad.Bool) {
			return ctx.WriteByte(1)
		}
		return ctx.WriteByte(0)
	}

	FlagV1.Encode = func(ctx *WriteContext, o any) (err error) {
		if o.(gad.Flag) {
			return ctx.WriteByte(1)
		}
		return ctx.WriteByte(0)
	}

	IntV1.Encode = func(ctx *WriteContext, o any) (err error) {
		return writeInt64(ctx, int64(o.(gad.Int)))
	}

	UintV1.Encode = func(ctx *WriteContext, o any) (err error) {
		return writeUint64(ctx, uint64(o.(gad.Uint)))
	}

	CharV1.Encode = func(ctx *WriteContext, o any) (err error) {
		return writeUint64(ctx, uint64(o.(gad.Char)))
	}

	FloatV1.Encode = func(ctx *WriteContext, o any) (err error) {
		return writeUint64(ctx, math.Float64bits(float64(o.(gad.Float))))
	}

	DecimalV1.Encode = func(ctx *WriteContext, o any) (err error) {
		var data []byte
		if data, err = o.(gad.Decimal).GobEncode(); err != nil {
			return
		}
		return writeChunk(ctx, data)
	}

	StrV1.Encode = func(ctx *WriteContext, o any) (err error) {
		return writeString(ctx, string(o.(gad.Str)))
	}

	RawStrV1.Encode = func(ctx *WriteContext, o any) (err error) {
		return writeString(ctx, string(o.(gad.RawStr)))
	}

	BytesV1.Encode = func(ctx *WriteContext, o any) (err error) {
		return writeChunk(ctx, o.(gad.Bytes))
	}

	ArrayV1.Encode = func(ctx *WriteContext, o any) (err error) {
		return EncodeArray(ctx, o.(gad.Array))
	}

	DictV1.Encode = func(ctx *WriteContext, o any) (err error) {
		return EncodeDict(ctx, o.(gad.Dict))
	}

	SyncDictV1.Encode = func(ctx *WriteContext, o any) (err error) {
		return EncodeDict(ctx, o.(*gad.SyncDict).Value)
	}

	CompiledFunctionV1.Encode = func(ctx *WriteContext, o any) (err error) {
		cf := o.(*gad.CompiledFunction)

		if m := cf.GetModule(); m != nil {
			if err = ctx.WriteByte(0); err != nil {
				return
			}
			if err = writeInt(ctx, m.Index); err != nil {
				return
			}
		}

		if len(cf.FuncName) > 0 {
			if err = ctx.WriteByte(1); err != nil {
				return
			}
			if err = writeString(ctx, cf.FuncName); err != nil {
				return
			}
		}

		if cf.AllowMethods {
			if err = ctx.WriteByte(2); err != nil {
				return
			}
		}

		if l := cf.Params.Len(); l > 0 {
			if err = ctx.WriteByte(3); err != nil {
				return
			}

			if err = writeInt(ctx, l); err != nil {
				return
			}

			for i := 0; i < l; i++ {
				p := cf.Params.Items[i]
				if err = writeString(ctx, p.Name); err != nil {
					return
				}

				if err = writeBool(ctx, p.Var); err != nil {
					return
				}

				if err = EncodeObject(ctx, p.Symbol); err != nil {
					return
				}

				if err = EncodeArray(ctx, p.TypesSymbols); err != nil {
					return
				}
			}
		}

		if err = ctx.WriteByte(4); err != nil {
			return
		}

		if err = writeInt(ctx, cf.NumLocals); err != nil {
			return
		}

		if err = ctx.WriteByte(5); err != nil {
			return
		}

		if err = writeChunk(ctx, cf.Instructions); err != nil {
			return
		}

		if l := cf.NamedParams.Len(); l > 0 {
			if err = ctx.WriteByte(6); err != nil {
				return
			}

			if err = writeInt(ctx, l); err != nil {
				return
			}

			for i := 0; i < l; i++ {
				p := cf.NamedParams.Items[i]

				if err = EncodeObject(ctx, p.Symbol); err != nil {
					return
				}

				if err = writeString(ctx, p.Value); err != nil {
					return
				}

				if err = writeBool(ctx, p.Var); err != nil {
					return
				}

				if err = EncodeArray(ctx, p.TypesSymbols); err != nil {
					return
				}
			}
		}

		if l := len(cf.SourceMap); l > 0 {
			if err = ctx.WriteByte(7); err != nil {
				return
			}
			if err = writeInt(ctx, l); err != nil {
				return
			}

			for k, v := range cf.SourceMap {
				if err = writeInt(ctx, k); err != nil {
					return
				}
				if err = writeInt(ctx, v); err != nil {
					return
				}
			}
		}

		err = ctx.WriteByte(FieldEOF)
		return
	}

	ErrorV1.Encode = func(ctx *WriteContext, o any) (err error) {
		e := o.(*gad.Error)
		if err = writeString(ctx, e.Name); err != nil {
			return
		}
		return writeString(ctx, e.Message)
	}

	EmbeddedV1.Encode = func(ctx *WriteContext, o any) (err error) {
		const encodingTree = "encodingTree"
		var (
			e  = o.(*gad.Embedded)
			ew = ctx.EmbeddedWriter

			writeNode = func(e *gad.Embedded) (err error) {
				var isDirB byte
				if e.IsDir() {
					isDirB = 1
				}

				if err = ctx.WriteByte(isDirB); err != nil {
					return
				}
				if err = writeString(ctx, e.Name); err != nil {
					return
				}
				if err = writeString(ctx, e.AbsPath); err != nil {
					return
				}
				if err = writeUint32(ctx, uint32(e.Mode)); err != nil {
					return
				}
				if err = writeInt64(ctx, e.ModTime.UnixNano()); err != nil {
					return
				}

				if e.IsDir() {
					return
				}

				var (
					r    io.Reader
					size int64
				)

				if size, err = e.Size(); err != nil {
					return
				}

				if err = writeInt64(ctx, size); err != nil {
					return
				}

				if r, err = e.Reader(); err != nil {
					return
				}

				// the bytes written + start value size (uint32 = 4 bytes)
				if err = writeInt64(ctx, int64(ew.BytesWritten())); err != nil {
					return
				}
				if _, err = io.Copy(ew, r); err != nil {
					return
				}
				return
			}
		)

		if err = writeNode(e); err != nil {
			return
		}

		if e.IsDir() {
			var (
				indexMap = map[*gad.Embedded]int{
					e: 0,
				}

				nodes = []*embeddedNode{nil}
			)

			e.Walk(func(path []string, n *gad.Embedded) error {
				indexMap[n] = len(nodes)
				nodes = append(nodes, &embeddedNode{
					parent: indexMap[n.Parent],
					entry:  n,
				})
				return nil
			})

			if err = writeInt64(ctx, int64(len(nodes))-1); err != nil {
				return
			}

			for _, n := range nodes[1:] {
				if err = writeInt(ctx, n.parent); err != nil {
					return
				}
				if err = writeNode(n.entry); err != nil {
					return
				}
			}
		}
		return
	}
}

type embeddedNode struct {
	parent int
	entry  *gad.Embedded
}
