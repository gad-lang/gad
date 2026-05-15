package encoder

import (
	"context"
	"reflect"

	"github.com/gad-lang/gad"
)

// BytecodeV1 signature and version are written to the header of encoded BytecodeV1.
// BytecodeV1 is encoded with current BytecodeVersion and its format.
const (
	BytecodeSignature uint32 = 0x75474F
	BytecodeVersion   uint16 = 1
)

type ContextOption func(ctx *Context)

func ContextWithParent(parent context.Context) ContextOption {
	return func(ctx *Context) {
		ctx.Context = parent
	}
}

func ContextWithModules(modules ModulesSpec) ContextOption {
	return func(ctx *Context) {
		ctx.Modules = modules
	}
}

func ContextWithModuleMap(mm *gad.ModuleMap) ContextOption {
	return func(ctx *Context) {
		ctx.GoModules = GoModulesFromModulesMap(mm)
	}
}

func ContextWithGoModules(modules GoModules) ContextOption {
	return func(ctx *Context) {
		ctx.GoModules = modules
	}
}

type Context struct {
	context.Context
	Modules   []*gad.ModuleSpec
	GoModules GoModules
}

func NewContext(opt ...ContextOption) *Context {
	ctx := &Context{}

	for _, option := range opt {
		option(ctx)
	}

	if ctx.Context == nil {
		ctx.Context = context.Background()
	}

	return ctx
}

type EncodeFunc func(w Writer, o any) error

type DecodeFunc func(r Reader, ctx *Context) (any, error)

type EncDec struct {
	Encode EncodeFunc
	Decode DecodeFunc
}

type TypeEncoder struct {
	Encoders    map[byte]*EncDec
	LastVersion byte
}

type encDecRegistrator struct {
	byVersion map[byte]*EncDec
	byType    map[reflect.Type]*TypeEncoder
}

var Encoders = encDecRegistrator{
	byVersion: make(map[byte]*EncDec),
	byType:    make(map[reflect.Type]*TypeEncoder),
}

func Register[T any](version byte, encDec *EncDec) {
	Encoders.byVersion[version] = encDec
	rt := reflect.TypeFor[T]()

	for rt.Kind() == reflect.Interface {
		rt = rt.Elem()
	}

	te := Encoders.byType[rt]

	if te == nil {
		te = &TypeEncoder{
			Encoders:    make(map[byte]*EncDec),
			LastVersion: version,
		}
		Encoders.byType[rt] = te
	}
	te.Encoders[version] = encDec
	if te.LastVersion < version {
		te.LastVersion = version
	}
}
