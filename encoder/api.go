package encoder

import (
	"context"
	"io"
	"reflect"

	"github.com/gad-lang/gad"
)

// BytecodeV1 signature and version are written to the header of encoded BytecodeV1.
// BytecodeV1 is encoded with current BytecodeVersion and its format.
const (
	BytecodeSignature uint32 = 0x75474F
	BytecodeVersion   uint16 = 1
)

type ReadContextOption func(ctx *ReadContext)

func ReadContextWithModules(modules ModulesSpec) ReadContextOption {
	return func(ctx *ReadContext) {
		ctx.Modules = modules
	}
}

func ReadContextWithModuleMap(mm *gad.ModuleMap) ReadContextOption {
	return func(ctx *ReadContext) {
		ctx.GoModules = GoModulesFromModulesMap(mm)
	}
}

func ReadContextWithGoModules(modules GoModules) ReadContextOption {
	return func(ctx *ReadContext) {
		ctx.GoModules = modules
	}
}

func ReadContextWithEmbeddedReader(r io.ReaderAt) ReadContextOption {
	return func(ctx *ReadContext) {
		ctx.EmbeddedReader = r
	}
}

type Context struct {
	context.Context
	Modules   []*gad.ModuleSpec
	GoModules GoModules
}

func NewReadContext(r Reader, opt ...ReadContextOption) *ReadContext {
	ctx := &ReadContext{
		Reader:         r,
		EmbeddedReader: r,
	}

	for _, option := range opt {
		option(ctx)
	}

	if ctx.Context == nil {
		ctx.Context = context.Background()
	}

	return ctx
}

type WriteContextOption func(ctx *WriteContext)

func WriteContextWithEmbededWriter(w EmbeddedWriter) WriteContextOption {
	return func(ctx *WriteContext) {
		ctx.EmbeddedWriter = w
	}
}

type WriteContext struct {
	Context        context.Context
	EmbeddedWriter EmbeddedWriter
	Writer
}

func NewWriteContext(ctx context.Context, writer Writer, opt ...WriteContextOption) *WriteContext {
	w := &WriteContext{Context: ctx, Writer: writer}
	for _, option := range opt {
		option(w)
	}
	return w
}

func (ctx *WriteContext) WithValue(key string, value any) *WriteContext {
	ctx.Context = context.WithValue(ctx.Context, key, value)
	return ctx
}

func (ctx *WriteContext) Value(key string) any {
	return ctx.Context.Value(key)
}

type ReadContext struct {
	Reader
	Context        context.Context
	Modules        []*gad.ModuleSpec
	GoModules      GoModules
	EmbeddedReader io.ReaderAt
}

type EncodeFunc func(ctx *WriteContext, o any) error

type DecodeFunc func(ctx *ReadContext) (any, error)

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
