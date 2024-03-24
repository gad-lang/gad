package encoder_test

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"testing"
	gotime "time"

	"github.com/gad-lang/gad"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/gad-lang/gad/stdlib/json"
	"github.com/gad-lang/gad/stdlib/time"
	"github.com/gad-lang/gad/token"

	. "github.com/gad-lang/gad/encoder"
)

func TestGobEncoder(t *testing.T) {
	objects := []gad.Object{
		gad.Nil,
		gad.Bool(true),
		gad.Flag(true),
		gad.Int(0),
		gad.Uint(0),
		gad.Char(0),
		gad.Float(0),
		gad.DecimalZero,
		gad.Str("abc"),
		gad.Bytes{},
		gad.Array{gad.Bool(true), gad.Flag(true), gad.Str("")},
		gad.Dict{"b": gad.Bool(true), "f": gad.Flag(true), "s": gad.Str("")},
		&gad.SyncDict{Value: gad.Dict{"i": gad.Int(0), "u": gad.Uint(0), "d": gad.MustDecimalFromString("123.456")}},
		&gad.ObjectPtr{},
		&time.Time{Value: gotime.Now()},
		&json.EncoderOptions{Value: gad.Float(0)},
		&json.RawMessage{},
	}
	for _, obj := range objects {
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(obj)
		require.NoError(t, err)
	}
}

func TestEncDecObjects(t *testing.T) {
	data, err := (*NilType)(gad.Nil.(*gad.NilType)).MarshalBinary()
	require.NoError(t, err)
	if obj, err := DecodeObject(bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	} else {
		require.Equal(t, gad.Nil, obj)
	}

	boolObjects := []gad.Bool{gad.True, gad.False, gad.Bool(true), gad.Bool(false)}
	for _, tC := range boolObjects {
		msg := fmt.Sprintf("Bool(%v)", tC)
		data, err := Bool(tC).MarshalBinary()
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v Bool
		err = v.UnmarshalBinary(data)
		require.NoError(t, err, msg)
		require.Equal(t, Bool(tC), v, msg)

		obj, err := DecodeObject(bytes.NewReader(data))
		require.NoError(t, err, msg)
		require.Equal(t, tC, obj, msg)
	}

	flagObjects := []gad.Flag{gad.Yes, gad.No, gad.Flag(true), gad.Flag(false)}
	for _, tC := range flagObjects {
		msg := fmt.Sprintf("Flag(%v)", tC)
		data, err := Flag(tC).MarshalBinary()
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v Flag
		err = v.UnmarshalBinary(data)
		require.NoError(t, err, msg)
		require.Equal(t, Flag(tC), v, msg)

		obj, err := DecodeObject(bytes.NewReader(data))
		require.NoError(t, err, msg)
		require.Equal(t, tC, obj, msg)
	}

	intObjects := []gad.Int{
		gad.Int(-1), gad.Int(0), gad.Int(1), gad.Int(1<<63 - 1),
	}
	for i := 0; i < 1000; i++ {
		v := seededRand.Int63()
		if i%2 == 0 {
			intObjects = append(intObjects, gad.Int(-v))
		} else {
			intObjects = append(intObjects, gad.Int(v))
		}
	}
	for _, tC := range intObjects {
		msg := fmt.Sprintf("Int(%v)", tC)
		data, err := Int(tC).MarshalBinary()
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v Int
		err = v.UnmarshalBinary(data)
		require.NoError(t, err, msg)
		require.Equal(t, Int(tC), v, msg)

		obj, err := DecodeObject(bytes.NewReader(data))
		require.NoError(t, err, msg)
		require.Equal(t, tC, obj, msg)
	}

	uintObjects := []gad.Uint{gad.Uint(0), gad.Uint(1), ^gad.Uint(0)}
	for i := 0; i < 1000; i++ {
		v := seededRand.Uint64()
		uintObjects = append(uintObjects, gad.Uint(v))
	}
	for _, tC := range uintObjects {
		msg := fmt.Sprintf("Uint(%v)", tC)
		data, err := Uint(tC).MarshalBinary()
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v Uint
		err = v.UnmarshalBinary(data)
		require.NoError(t, err, msg)
		require.Equal(t, Uint(tC), v, msg)

		obj, err := DecodeObject(bytes.NewReader(data))
		require.NoError(t, err, msg)
		require.Equal(t, tC, obj, msg)
	}

	charObjects := []gad.Char{gad.Char(0)}
	for i := 0; i < 1000; i++ {
		v := seededRand.Int31()
		charObjects = append(charObjects, gad.Char(v))
	}
	for _, tC := range charObjects {
		msg := fmt.Sprintf("Char(%v)", tC)
		data, err := Char(tC).MarshalBinary()
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v Char
		err = v.UnmarshalBinary(data)
		require.NoError(t, err, msg)
		require.Equal(t, Char(tC), v, msg)

		obj, err := DecodeObject(bytes.NewReader(data))
		require.NoError(t, err, msg)
		require.Equal(t, tC, obj, msg)
	}

	floatObjects := []gad.Float{gad.Float(0), gad.Float(-1)}
	for i := 0; i < 1000; i++ {
		v := seededRand.Float64()
		floatObjects = append(floatObjects, gad.Float(v))
	}
	floatObjects = append(floatObjects, gad.Float(math.NaN()))
	for _, tC := range floatObjects {
		msg := fmt.Sprintf("Float(%v)", tC)
		data, err := Float(tC).MarshalBinary()
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v Float
		err = v.UnmarshalBinary(data)
		require.NoError(t, err, msg)
		if math.IsNaN(float64(tC)) {
			require.True(t, math.IsNaN(float64(v)))
		} else {
			require.Equal(t, Float(tC), v, msg)
		}

		obj, err := DecodeObject(bytes.NewReader(data))
		require.NoError(t, err, msg)
		if math.IsNaN(float64(tC)) {
			require.True(t, math.IsNaN(float64(obj.(gad.Float))))
		} else {
			require.Equal(t, tC, obj, msg)
		}
	}
	// remove NaN from Floats slice, array tests below requires NaN check otherwise fails.
	floatObjects = floatObjects[:len(floatObjects)-1]

	stringObjects := []gad.Str{gad.Str(""), gad.Str("çığöşü")}
	for i := 0; i < 1000; i++ {
		stringObjects = append(stringObjects, gad.Str(randString(i)))
	}
	for _, tC := range stringObjects {
		msg := fmt.Sprintf("Str(%v)", tC)
		data, err := String(tC).MarshalBinary()
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v String
		err = v.UnmarshalBinary(data)
		require.NoError(t, err, msg)
		require.Equal(t, String(tC), v, msg)

		obj, err := DecodeObject(bytes.NewReader(data))
		require.NoError(t, err, msg)
		require.Equal(t, tC, obj, msg)
	}

	bytesObjects := []gad.Bytes{{}, gad.Bytes("çığöşü")}
	for i := 0; i < 1000; i++ {
		bytesObjects = append(bytesObjects, gad.Bytes(randString(i)))
	}
	for _, tC := range bytesObjects {
		msg := fmt.Sprintf("Bytes(%v)", tC)
		data, err := Bytes(tC).MarshalBinary()
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v = Bytes{}
		err = v.UnmarshalBinary(data)
		require.NoError(t, err, msg)
		require.Equal(t, Bytes(tC), v, msg)

		obj, err := DecodeObject(bytes.NewReader(data))
		require.NoError(t, err, msg)
		require.Equal(t, tC, obj, msg)
	}

	decimalObjects := []gad.Decimal{gad.DecimalFromFloat(gad.Float(0)), gad.DecimalFromFloat(gad.Float(-1))}
	for i := 0; i < 1000; i++ {
		v := seededRand.Float64()
		decimalObjects = append(decimalObjects, gad.DecimalFromFloat(gad.Float(v)))
	}
	for _, tC := range decimalObjects {
		msg := fmt.Sprintf("Decimal(%v)", tC)
		data, err := Decimal(tC).MarshalBinary()
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v Decimal
		err = v.UnmarshalBinary(data)
		require.NoError(t, err, msg)
		require.Equal(t, tC.ToString(), decimal.Decimal(v).String(), msg)

		obj, err := DecodeObject(bytes.NewReader(data))
		require.NoError(t, err, msg)
		require.Equal(t, tC.ToString(), obj.ToString(), msg)
	}
	// remove NaN from Decimal slice, array tests below requires NaN check otherwise fails.
	decimalObjects = decimalObjects[:len(decimalObjects)-1]

	arrays := []gad.Array{}
	temp1 := gad.Array{}
	for i := range bytesObjects[:100] {
		temp1 = append(temp1, bytesObjects[i])
	}
	arrays = append(arrays, temp1)
	temp2 := gad.Array{}
	for i := range stringObjects[:100] {
		temp2 = append(temp2, stringObjects[i])
	}
	arrays = append(arrays, temp2)
	temp3 := gad.Array{}
	for i := range floatObjects[:100] {
		temp3 = append(temp3, floatObjects[i])
	}
	arrays = append(arrays, temp3)
	temp4 := gad.Array{}
	for i := range charObjects[:100] {
		temp4 = append(temp4, charObjects[i])
	}
	arrays = append(arrays, temp4)
	temp5 := gad.Array{}
	for i := range uintObjects[:100] {
		temp5 = append(temp5, uintObjects[i])
	}
	arrays = append(arrays, temp5)
	temp6 := gad.Array{}
	for i := range intObjects[:100] {
		temp6 = append(temp6, intObjects[i])
	}
	arrays = append(arrays, temp6)
	temp7 := gad.Array{}
	for i := range boolObjects {
		temp7 = append(temp7, boolObjects[i])
	}
	arrays = append(arrays, temp7)
	temp8 := gad.Array{}
	for i := range decimalObjects[:100] {
		temp8 = append(temp8, gad.Str(decimalObjects[i].ToString()))
	}
	arrays = append(arrays, temp8)
	arrays = append(arrays, gad.Array{gad.Nil})

	for _, tC := range arrays {
		msg := fmt.Sprintf("Array(%v)", tC)
		data, err := Array(tC).MarshalBinary()
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v = gad.Array{}
		err = (*Array)(&v).UnmarshalBinary(data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)

		obj, err := DecodeObject(bytes.NewReader(data))
		require.NoError(t, err, msg)
		require.Equal(t, tC, obj, msg)
	}

	maps := []gad.Dict{}
	for _, array := range arrays {
		m := gad.Dict{}
		s := randString(10)
		r := seededRand.Intn(len(array))
		m[s] = array[r]
		maps = append(maps, m)
	}

	for _, tC := range maps {
		msg := fmt.Sprintf("Dict(%v)", tC)
		data, err := Map(tC).MarshalBinary()
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v = gad.Dict{}
		err = (*Map)(&v).UnmarshalBinary(data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)

		obj, err := DecodeObject(bytes.NewReader(data))
		require.NoError(t, err, msg)
		require.Equal(t, tC, obj, msg)
	}

	syncMaps := []*gad.SyncDict{}
	for _, m := range maps {
		syncMaps = append(syncMaps, &gad.SyncDict{Value: m})
	}
	for _, tC := range syncMaps {
		msg := fmt.Sprintf("SyncDict(%v)", tC)
		data, err := (*SyncMap)(tC).MarshalBinary()
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v = &gad.SyncDict{}
		err = (*SyncMap)(v).UnmarshalBinary(data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)

		obj, err := DecodeObject(bytes.NewReader(data))
		require.NoError(t, err, msg)
		require.Equal(t, tC, obj, msg)
	}

	compFuncs := []*gad.CompiledFunction{
		compFunc(nil),
		compFunc(nil,
			withLocals(10),
		),
		compFunc(nil,
			withParams("a", "b"),
		),
		compFunc(nil,
			withVarParams(),
		),
		compFunc(nil,
			withSourceMap(map[int]int{0: 1, 3: 1, 5: 1}),
		),
		compFunc(concatInsts(
			makeInst(gad.OpConstant, 0),
			makeInst(gad.OpConstant, 1),
			makeInst(gad.OpBinaryOp, int(token.Add)),
		),
			withParams("a"),
			withVarParams(),
			withLocals(2),
			withSourceMap(map[int]int{0: 1, 3: 1, 5: 1}),
		),
	}
	for i, tC := range compFuncs {
		msg := fmt.Sprintf("CompiledFunction #%d", i)
		data, err := (*CompiledFunction)(tC).MarshalBinary()
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v = &gad.CompiledFunction{}
		err = (*CompiledFunction)(v).UnmarshalBinary(data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)

		obj, err := DecodeObject(bytes.NewReader(data))
		require.NoError(t, err, msg)
		require.Equal(t, tC, obj, msg)
	}

	builtinFuncs := []*BuiltinFunction{}
	for _, o := range gad.BuiltinObjects {
		if f, ok := o.(*BuiltinFunction); ok {
			builtinFuncs = append(builtinFuncs, f)
		}
	}
	for _, tC := range builtinFuncs {
		msg := fmt.Sprintf("BuiltinFunction %s", tC.Name)
		data, err := tC.MarshalBinary()
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v = &gad.BuiltinFunction{}
		err = (*BuiltinFunction)(v).UnmarshalBinary(data)
		require.NoError(t, err, msg)
		require.Equal(t, tC.Name, v.Name)
		require.NotNil(t, v.Value)

		obj, err := DecodeObject(bytes.NewReader(data))
		require.NoError(t, err, msg)
		require.Equal(t, tC.Name, obj.(*BuiltinFunction).Name, msg)
		require.NotNil(t, obj.(*BuiltinFunction).Value, msg)
	}

}

func TestEncDecBytecode(t *testing.T) {
	testEncDecBytecode(t, `
	param (arg0, arg1, *varg; na0=100, na1=200, **na)
	return [arg0, arg1, varg, na0, na1, na.dict]`, &testopts{
		args:      Array{gad.Int(1), gad.Int(2), gad.Int(3)},
		namedArgs: gad.Dict{"na0": gad.Int(4), "na2": gad.Int(5)},
	}, gad.Array{gad.Int(1), gad.Int(2), gad.Array{gad.Int(3)}, gad.Int(4), gad.Int(200), gad.Dict{"na2": gad.Int(5)}})

	testEncDecBytecode(t, `
	param (arg0, arg1, *varg; na0=100, na1=200, **na)
	return [arg0, arg1, varg, na0, na1, na.dict]`, &testopts{
		args:      Array{gad.Int(1), gad.Int(2), gad.Int(3)},
		namedArgs: gad.Dict{"na2": gad.Int(5)},
	}, gad.Array{gad.Int(1), gad.Int(2), gad.Array{gad.Int(3)}, gad.Int(100), gad.Int(200), gad.Dict{"na2": gad.Int(5)}})

	testEncDecBytecode(t, `
	f := func(arg0, arg1, *varg; na0=100, **na) {
		return [arg0, arg1, varg, na0, na.dict]
	}
	return f(1,2,3,na0=4,na1=5)`, nil, gad.Array{gad.Int(1), gad.Int(2), gad.Array{gad.Int(3)}, gad.Int(4), gad.Dict{"na1": gad.Int(5)}})

	testEncDecBytecode(t, `
	f := func() {
		return [nil, true, false, "", -1, 0, 1, 2u, 3.0, 123.456d, 'a', bytes(0, 1, 2)]
	}
	f()
	m := {a: 1, b: ["abc"], c: {x: bytes()}, builtins: [append, len]}`, nil, gad.Nil)

	testEncDecBytecode(t, `
	f := func(arg0, arg1, *varg; na0=3, **na) {
		return [arg0, arg1, varg, na0, na.dict, nil, true, false, "", -1, 0, 1, 2u, 3.0, 123.456d, 'a', bytes(0, 1, 2)]
	}
	f(1,2,na0=4,na1=5)
	m := {a: 1, b: ["abc"], c: {x: bytes()}, builtins: [append, len]}`, nil, gad.Nil)
}

func TestEncDecBytecode_modules(t *testing.T) {
	testEncDecBytecode(t, `
	mod1 := import("mod1")
	mod2 := import("mod2")
	return mod1.run() + mod2.run()
	`, newOpts().Module("mod1", gad.Dict{
		"run": &gad.Function{
			Name: "run",
			Value: func(gad.Call) (gad.Object, error) {
				return gad.Str("mod1"), nil
			},
		},
	}).Module("mod2", `return {run: func(){ return "mod2" }}`), gad.Str("mod1mod2"))
}

func testEncDecBytecode(t *testing.T, script string, opts *testopts, expected gad.Object) {
	t.Helper()
	if opts == nil {
		opts = newOpts()
	}

	if cfn, ok := expected.(*gad.CompiledFunction); ok {
		expected = cfn.ClearSourceFileInfo()
	}

	var initialModuleMap *gad.ModuleMap
	if opts.moduleMap != nil {
		initialModuleMap = opts.moduleMap.Copy()
	}
	bc, err := gad.Compile([]byte(script),
		gad.CompileOptions{CompilerOptions: gad.CompilerOptions{
			ModuleMap: opts.moduleMap,
		}},
	)
	require.NoError(t, err)
	vm := gad.NewVM(bc)
	items, _ := opts.namedArgs.Items(vm)
	ret, err := vm.RunOpts(&gad.RunOpts{
		Globals:   opts.globals,
		Args:      gad.Args{opts.args},
		NamedArgs: gad.NewNamedArgs(items),
	})
	require.NoError(t, err)
	require.Equal(t, expected, ret)

	var buf bytes.Buffer
	err = gob.NewEncoder(&buf).Encode((*Bytecode)(bc))
	require.NoError(t, err)
	t.Logf("GobSize:%d", len(buf.Bytes()))
	bcData, err := (*Bytecode)(bc).MarshalBinary()
	require.NoError(t, err)
	t.Logf("BinSize:%d", len(bcData))

	if opts.moduleMap == nil {
		var bc2 gad.Bytecode
		err = gob.NewDecoder(&buf).Decode((*Bytecode)(&bc2))
		require.NoError(t, err)
		testDecodedBytecodeEqual(t, bc, &bc2)

		items, _ = opts.namedArgs.Items(vm)
		ret, err := gad.NewVM(&bc2).RunOpts(&gad.RunOpts{
			Globals:   opts.globals,
			Args:      gad.Args{opts.args},
			NamedArgs: gad.NewNamedArgs(items),
		})

		require.NoError(t, err)
		require.Equal(t, expected, ret)

		var bc3 gad.Bytecode
		err = (*Bytecode)(&bc3).UnmarshalBinary(bcData)
		require.NoError(t, err)
		testDecodedBytecodeEqual(t, bc, &bc3)
		items, _ = opts.namedArgs.Items(vm)
		ret, err = gad.NewVM(&bc3).RunOpts(&gad.RunOpts{
			Globals:   opts.globals,
			Args:      gad.Args{opts.args},
			NamedArgs: gad.NewNamedArgs(items),
		})
		require.NoError(t, err)
		require.Equal(t, expected, ret)
	}

	bc4, err := DecodeBytecodeFrom(bytes.NewReader(bcData), opts.moduleMap)
	require.NoError(t, err)
	testDecodedBytecodeEqual(t, bc, bc4)
	items, _ = opts.namedArgs.Items(vm)
	ret, err = gad.NewVM(bc4).RunOpts(&gad.RunOpts{
		Globals:   opts.globals,
		Args:      gad.Args{opts.args},
		NamedArgs: gad.NewNamedArgs(items),
	})
	require.NoError(t, err)
	require.Equal(t, expected, ret)
	// ensure moduleMap is not updated during compilation and decoding
	require.Equal(t, initialModuleMap, opts.moduleMap)
}

func testDecodedBytecodeEqual(t *testing.T, actual, decoded *gad.Bytecode) {
	t.Helper()
	msg := fmt.Sprintf("actual:%s\ndecoded:%s\n", actual, decoded)

	testBytecodeConstants(t, gad.NewVM(actual).Init(), actual.Constants, decoded.Constants)
	require.Equal(t, actual.Main, decoded.Main, msg)
	require.Equal(t, actual.NumModules, decoded.NumModules, msg)
	if actual.FileSet == nil {
		require.Nil(t, decoded.FileSet, msg)
	} else {
		require.Equal(t, actual.FileSet.Base, decoded.FileSet.Base, msg)
		require.Equal(t, len(actual.FileSet.Files), len(decoded.FileSet.Files), msg)
		for i, f := range actual.FileSet.Files {
			f2 := decoded.FileSet.Files[i]
			require.Equal(t, f.Base, f2.Base, msg)
			require.Equal(t, f.Lines, f2.Lines, msg)
			require.Equal(t, f.Name, f2.Name, msg)
			require.Equal(t, f.Size, f2.Size, msg)
		}
		require.NotNil(t, actual.FileSet.LastFile, msg)
		require.Nil(t, decoded.FileSet.LastFile, msg)
	}
}

func getModuleName(obj gad.Object) (string, bool) {
	if m, ok := obj.(gad.Dict); ok {
		if n, ok := m[gad.AttrModuleName]; ok {
			return string(n.(gad.Str)), true
		}
	}
	return "", false
}

func testBytecodeConstants(t *testing.T, vm *gad.VM, expected, decoded []gad.Object) {
	t.Helper()
	if len(decoded) != len(expected) {
		t.Fatalf("constants length not equal want %d, got %d", len(decoded), len(expected))
	}
	Len := func(v gad.Object) gad.Object {
		ret, err := gad.MustCall(gad.BuiltinObjects[gad.BuiltinLen], v)
		if err != nil {
			t.Fatalf("%v: length error for '%v'", err, v)
		}
		return ret
	}

	next := func(ok bool, err error) bool {
		require.NoError(t, err)
		return ok
	}

	for i := range decoded {
		modName, ok1 := getModuleName(expected[i])
		decModName, ok2 := getModuleName(decoded[i])
		if ok1 {
			require.True(t, ok2)
			require.Equal(t, modName, decModName)
			require.Equal(t, reflect.TypeOf(expected[i]), reflect.TypeOf(decoded[i]))
			require.Equal(t, Len(expected[i]), Len(decoded[i]))
			if !gad.Iterable(vm, expected[i]) {
				require.False(t, gad.Iterable(vm, decoded[i]))
				continue
			}

			_, it, err := gad.ToStateIterator(vm, expected[i], gad.NewNamedArgs())
			require.NoError(t, err)

			_, decIt, err := gad.ToStateIterator(vm, decoded[i], gad.NewNamedArgs())
			require.NoError(t, err)

			for next(decIt.Read()) {
				require.True(t, next(it.Read()))
				key := decIt.Key()
				v1, err := gad.Val(expected[i].(gad.IndexGetter).IndexGet(vm, key))
				require.NoError(t, err)
				v2 := decIt.Value()
				require.NoError(t, err)
				if (v1 != nil && v2 == nil) || (v1 == nil && v2 != nil) {
					t.Fatalf("decoded constant index %d not equal", i)
				}
				f1, ok := v1.(*gad.Function)
				if ok {
					f2 := v2.(*gad.Function)
					require.Equal(t, f1.Name, f2.Name)
					require.NotNil(t, f2.Value)
					// Note that this is not a guaranteed way to compare func pointers
					require.Equal(t, reflect.ValueOf(f1.Value).Pointer(),
						reflect.ValueOf(f2.Value).Pointer())
				} else {
					require.Equal(t, v1, v2)
				}
			}
			require.False(t, next(it.Read()))
			continue
		}
		require.Equalf(t, expected[i], decoded[i],
			"constant index %d not equal want %v, got %v", i, expected[i], decoded[i])
		require.NotNil(t, decoded[i])
	}
}

type funcOpt func(*gad.CompiledFunction)

func withParams(names ...string) funcOpt {
	return func(cf *gad.CompiledFunction) {
		cf.Params.Len = len(names)
		cf.Params.Names = names
	}
}

func withLocals(numLocals int) funcOpt {
	return func(cf *gad.CompiledFunction) {
		cf.NumLocals = numLocals
	}
}

func withVarParams() funcOpt {
	return func(cf *gad.CompiledFunction) {
		cf.Params.Var = true
	}
}

func withSourceMap(m map[int]int) funcOpt {
	return func(cf *gad.CompiledFunction) {
		cf.SourceMap = m
	}
}

func compFunc(insts []byte, opts ...funcOpt) *gad.CompiledFunction {
	cf := &gad.CompiledFunction{
		Instructions: insts,
	}
	for _, f := range opts {
		f(cf)
	}
	return cf
}

func makeInst(op gad.Opcode, args ...int) []byte {
	b, err := gad.MakeInstruction(make([]byte, 8), op, args...)
	if err != nil {
		panic(err)
	}
	return b
}

func concatInsts(insts ...[]byte) []byte {
	var out []byte
	for i := range insts {
		out = append(out, insts[i]...)
	}
	return out
}

type testopts struct {
	globals       gad.IndexGetSetter
	args          []gad.Object
	namedArgs     gad.Dict
	moduleMap     *gad.ModuleMap
	skip2pass     bool
	isCompilerErr bool
	noPanic       bool
}

func newOpts() *testopts {
	return &testopts{}
}

func (t *testopts) Globals(globals gad.IndexGetSetter) *testopts {
	t.globals = globals
	return t
}

func (t *testopts) Args(args ...gad.Object) *testopts {
	t.args = args
	return t
}

func (t *testopts) Skip2Pass() *testopts {
	t.skip2pass = true
	return t
}

func (t *testopts) CompilerError() *testopts {
	t.isCompilerErr = true
	return t
}

func (t *testopts) NoPanic() *testopts {
	t.noPanic = true
	return t
}

func (t *testopts) Module(name string, module any) *testopts {
	if t.moduleMap == nil {
		t.moduleMap = gad.NewModuleMap()
	}
	switch v := module.(type) {
	case []byte:
		t.moduleMap.AddSourceModule(name, v)
	case string:
		t.moduleMap.AddSourceModule(name, []byte(v))
	case map[string]gad.Object:
		t.moduleMap.AddBuiltinModule(name, v)
	case gad.Dict:
		t.moduleMap.AddBuiltinModule(name, v)
	case gad.Importable:
		t.moduleMap.Add(name, v)
	default:
		panic(fmt.Errorf("invalid module type: %T", module))
	}
	return t
}

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand = rand.New(
	rand.NewSource(gotime.Now().UnixNano()))

func randStringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randString(length int) string {
	return randStringWithCharset(length, charset)
}
