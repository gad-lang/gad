package encoder

import (
	"math"
	"strconv"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/parser/source"
	"github.com/shopspring/decimal"
)

func init() {
	ModuleSpecV1.Decode = func(r Reader, ctx *Context) (_ any, err error) {
		s := new(gad.ModuleSpec)
		if s.Name, err = readString(r); err != nil {
			return
		}
		if s.URL, err = readString(r); err != nil {
			return
		}
		if s.Main, err = readBool(r); err != nil {
			return
		}

		var hasPath byte
		if hasPath, err = r.ReadByte(); err != nil {
			return
		}

		if hasPath == 1 {
			if err = DecodeIterator(r, func(l int) {
				s.Path = make([]int, l)
			},
				func(i int) (err error) {
					s.Path[i], err = readInt(r)
					return
				}); err != nil {
				return
			}
		}

		var hasCompiledFunction bool
		if hasCompiledFunction, err = readBool(r); err != nil {
			return
		} else if hasCompiledFunction {
			s.InitCompiledFunc, err = DecodeT[*gad.CompiledFunction](r, ctx)
		}

		return s, err
	}

	BytecodeV1.Decode = func(r Reader, ctx *Context) (_ any, err error) {
		var sig uint32
		if sig, err = readUint32(r); err != nil {
			return
		}

		if sig != BytecodeSignature {
			err = &gad.Error{
				Name:    "encoder.Bytecode.Decode",
				Message: "signature mismatch",
			}
			return
		}

		var version uint16

		if version, err = readUint16(r); err != nil {
			return
		}

		if version != BytecodeVersion {
			err = &gad.Error{
				Name:    "encoder.Bytecode.Decode",
				Message: "unsupported version:" + strconv.Itoa(int(version)),
			}
			return
		}

		bc := new(gad.Bytecode)
		err = DecodeFields(r, func(field uint8) (err error) {
			switch field {
			case 0:
				bc.FileSet, err = DecodeT[*source.FileSet](r, ctx)
			case 1:
				if bc.Modules, err = DecodeArray[*gad.ModuleSpec](r, ctx); err != nil {
					return
				}

				ctx.Modules = bc.Modules
				bc.NumModules = len(bc.Modules)

				for i, module := range bc.Modules {
					module.Index = i
					if goMod, ok := ctx.GoModules[module.Name]; ok {
						module.InitGoFunc = goMod.Caller(module)
					}
				}

				err = DecodeIterator(r, nil, func(i int) (err error) {
					var modIndex int
					if modIndex, err = readInt(r); err != nil {
						return
					}
					bc.Modules[modIndex].InitCompiledFunc, err = DecodeT[*gad.CompiledFunction](r, ctx)
					return
				})
			case 2:
				bc.Constants, err = DecodeArray[gad.Object](r, ctx)
			case 3:
				bc.Main, err = DecodeT[*gad.CompiledFunction](r, ctx)
			}
			return
		})
		return bc, err
	}

	SourceFileV1.Decode = func(r Reader, ctx *Context) (_ any, err error) {
		sf := new(source.File)

		if sf.Name, err = readString(r); err != nil {
			return
		}

		if sf.Base, err = readInt(r); err != nil {
			return
		}

		if sf.Size, err = readInt(r); err != nil {
			return
		}

		err = DecodeIterator(r,
			func(l int) {
				sf.Lines = make([]int, l)
			},
			func(i int) (err error) {
				sf.Lines[i], err = readInt(r)
				return
			},
		)
		return sf, err
	}

	SourceFileSetV1.Decode = func(r Reader, ctx *Context) (_ any, err error) {
		sfs := new(source.FileSet)

		if sfs.Base, err = readInt(r); err != nil {
			return
		}

		if sfs.Files, err = DecodeArray[*source.File](r, ctx); err != nil {
			return
		}

		var lastFile int
		if lastFile, err = readInt(r); err != nil {
			return
		}

		sfs.LastFile = sfs.Files[lastFile]
		return sfs, nil
	}

	SymbolInfoV1.Decode = func(r Reader, _ *Context) (_ any, err error) {
		s := new(gad.SymbolInfo)
		if s.Name, err = readString(r); err != nil {
			return
		}

		if s.Index, err = readInt(r); err != nil {
			return
		}

		var i int
		if i, err = readInt(r); err != nil {
			return
		}

		s.Scope = gad.SymbolScope(i)
		return s, nil
	}

	NilV1.Decode = func(r Reader, _ *Context) (any, error) {
		return gad.Nil, nil
	}

	BoolV1.Decode = func(r Reader, _ *Context) (_ any, err error) {
		var b byte
		if b, err = r.ReadByte(); err == nil {
			if b == 1 {
				return gad.True, nil
			}
			return gad.False, nil
		}
		return
	}

	FlagV1.Decode = func(r Reader, _ *Context) (_ any, err error) {
		var b byte
		if b, err = r.ReadByte(); err == nil {
			if b == 1 {
				return gad.Yes, nil
			}
			return gad.No, nil
		}
		return
	}

	IntV1.Decode = func(r Reader, _ *Context) (_ any, err error) {
		var i int64
		i, err = readInt64(r)
		return gad.Int(i), err
	}

	UintV1.Decode = func(r Reader, _ *Context) (_ any, err error) {
		var i uint64
		i, err = readUint64(r)
		return gad.Uint(i), err
	}

	CharV1.Decode = func(r Reader, _ *Context) (_ any, err error) {
		var i uint64
		i, err = readUint64(r)
		return gad.Char(i), err
	}

	FloatV1.Decode = func(r Reader, _ *Context) (_ any, err error) {
		var i uint64
		if i, err = readUint64(r); err != nil {
			return
		}
		return gad.Float(math.Float64frombits(i)), err
	}

	DecimalV1.Decode = func(r Reader, _ *Context) (_ any, err error) {
		var buf []byte

		if buf, err = readChunk(r); err != nil {
			return
		}

		var dec decimal.Decimal
		if err = dec.UnmarshalBinary(buf); err != nil {
			return
		}

		return gad.Decimal(dec), nil
	}

	StrV1.Decode = func(r Reader, _ *Context) (_ any, err error) {
		var s string
		if s, err = readString(r); err != nil {
			return
		}
		return gad.Str(s), nil
	}

	RawStrV1.Decode = func(r Reader, _ *Context) (_ any, err error) {
		var s string
		if s, err = readString(r); err != nil {
			return
		}
		return gad.RawStr(s), nil
	}

	BytesV1.Decode = func(r Reader, _ *Context) (_ any, err error) {
		var s []byte
		if s, err = readChunk(r); err != nil {
			return
		}
		return gad.Bytes(s), nil
	}

	ArrayV1.Decode = func(r Reader, ctx *Context) (_ any, err error) {
		var arr []gad.Object
		if arr, err = DecodeArray[gad.Object](r, ctx); err != nil {
			return
		}
		return gad.Array(arr), nil
	}

	DictV1.Decode = func(r Reader, ctx *Context) (any, error) {
		return DecodeDict(r, ctx)
	}

	SyncDictV1.Decode = func(r Reader, ctx *Context) (_ any, err error) {
		var d gad.Dict
		if d, err = DecodeDict(r, ctx); err != nil {
			return
		}
		return &gad.SyncDict{Value: d}, nil
	}

	CompiledFunctionV1.Decode = func(r Reader, ctx *Context) (_ any, err error) {
		o := new(gad.CompiledFunction)

		err = DecodeFields(r, func(field uint8) (err error) {
			switch field {
			case 0:
				var v int
				if v, err = readInt(r); err != nil {
					return
				}
				o.SetModule(ctx.Modules[v])
			case 1:
				o.FuncName, err = readString(r)
			case 2:
				o.AllowMethods = true
			case 3:
				var params []*gad.Param

				err = DecodeIterator(r,
					func(l int) {
						params = make([]*gad.Param, l)
					},
					func(i int) (err error) {
						var p gad.Param
						if p.Name, err = readString(r); err != nil {
							return
						}

						if p.Var, err = readBool(r); err != nil {
							return
						}

						if p.Symbol, err = DecodeT[*gad.SymbolInfo](r, ctx); err != nil {
							return
						}

						if p.TypesSymbols, err = DecodeArray[*gad.SymbolInfo](r, ctx); err != nil {
							return
						}

						params[i] = &p
						return
					},
				)

				if err != nil {
					return
				}

				o.Params = *gad.NewParams(params...)
			case 4:
				o.NumLocals, err = readInt(r)
			case 5:
				o.Instructions, err = readChunk(r)
			case 6:
				var namedParams []*gad.NamedParam
				err = DecodeIterator(r,
					func(l int) {
						namedParams = make([]*gad.NamedParam, l)
					}, func(i int) (err error) {
						var s *gad.SymbolInfo
						if s, err = DecodeT[*gad.SymbolInfo](r, ctx); err != nil {
							return
						}

						var v string
						if v, err = readString(r); err != nil {
							return
						}

						p := gad.NewNamedParam(s.Name, v)
						namedParams[i] = p

						p.Symbol = s

						if p.Var, err = readBool(r); err != nil {
							return
						}

						p.TypesSymbols, err = DecodeArray[*gad.SymbolInfo](r, ctx)

						return
					})
				o.NamedParams = *gad.NewNamedParams(namedParams...)
			case 7:
				var l int
				if l, err = readInt(r); err != nil {
					return
				}

				o.SourceMap = make(map[int]int, l)

				for i := 0; i < l; i++ {
					var k, v int
					if k, err = readInt(r); err != nil {
						return
					}
					if v, err = readInt(r); err != nil {
						return
					}
					o.SourceMap[k] = v
				}
			}
			return
		})

		return o, err
	}

	ErrorV1.Decode = func(r Reader, ctx *Context) (_ any, err error) {
		e := new(gad.Error)
		if e.Name, err = readString(r); err != nil {
			return
		}
		if e.Message, err = readString(r); err != nil {
			return
		}
		return e, nil
	}
}
