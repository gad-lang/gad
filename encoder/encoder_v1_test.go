package encoder_test

import (
	"fmt"
	"math"
	"regexp"
	"testing"
	"time"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/token"
	"github.com/stretchr/testify/require"
)

func TestEncDecRegexp(t *testing.T) {
	for _, pat := range []string{`ab+`, `[0-9]+/[a-z]*`, `^\d{3}-\d{4}$`} {
		re := (*gad.Regexp)(regexp.MustCompile(pat))
		data, err := encode(re)
		require.NoError(t, err, pat)
		require.Greater(t, len(data), 0, pat)

		got, err := decode[*gad.Regexp](data)
		require.NoError(t, err, pat)
		require.Equal(t, pat, got.Go().String(), pat)
	}
}

func TestEmbeddedV1BigTree(t *testing.T) {
	const (
		depth = 4
		width = 5
	)

	root := &gad.Embedded{
		Name:    "root",
		Entries: make(map[string]*gad.Embedded),
	}

	var totalFiles int
	expectedContent := make(map[string]string)

	var addNodes func(parent *gad.Embedded, parentPath string, depth int)
	addNodes = func(parent *gad.Embedded, parentPath string, depth int) {
		for i := 0; i < width; i++ {
			fileName := fmt.Sprintf("f%d.txt", i)
			content := fmt.Sprintf("content-%s-%s", parentPath, fileName)
			file := &gad.Embedded{
				Name:          fileName,
				Mode:          0644,
				ModTime:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				ReaderFactory: gad.EmbeddedBytesReaderFactory(content),
				Parent:        parent,
			}
			parent.Entries[fileName] = file
			expectedContent[parentPath+"/"+fileName] = content
			totalFiles++

			if depth > 1 {
				dirName := fmt.Sprintf("d%d", i)
				dir := &gad.Embedded{
					Name:    dirName,
					Entries: make(map[string]*gad.Embedded),
					Parent:  parent,
				}
				parent.Entries[dirName] = dir
				addNodes(dir, parentPath+"/"+dirName, depth-1)
			}
		}
	}

	addNodes(root, "root", depth)

	data, edata, err := eencode(root)
	require.NoError(t, err)
	t.Logf("encoded main size: %d bytes, embedded data size: %d bytes", len(data), len(edata))

	obj, err := edecode[*gad.Embedded](data, edata)
	require.NoError(t, err)
	require.Equal(t, root.Name, obj.Name)
	require.True(t, obj.IsDir())

	// Walk decoded tree and verify all files
	var decodedCount int
	obj.Walk(func(path []string, n *gad.Embedded) error {
		fullPath := ""
		for _, p := range path {
			fullPath += "/" + p
		}
		fullPath += "/" + n.Name
		fullPath = fullPath[1:] // strip leading "/"

		expected, ok := expectedContent[fullPath]
		require.True(t, ok, "unexpected file: %s", fullPath)
		data, err := n.Read()
		require.NoError(t, err)
		require.Equal(t, expected, string(data), "content mismatch for %s", fullPath)
		decodedCount++
		return nil
	})

	require.Equal(t, totalFiles, decodedCount, "file count mismatch")
}

func TestEncDecTimeTypes(t *testing.T) {
	t.Run("duration", func(t *testing.T) {
		o := gad.Duration(90 * time.Minute)
		b, eb, err := eencode(o)
		require.NoError(t, err)
		got, err := edecode[gad.Duration](b, eb)
		require.NoError(t, err)
		require.Equal(t, o, got)
	})

	t.Run("date", func(t *testing.T) {
		o := gad.CalendarDate(20260131)
		b, eb, err := eencode(o)
		require.NoError(t, err)
		got, err := edecode[gad.CalendarDate](b, eb)
		require.NoError(t, err)
		require.Equal(t, o, got)
	})

	t.Run("time", func(t *testing.T) {
		o := &gad.Time{Value: time.Date(2026, 1, 31, 9, 0, 0, 0, time.UTC)}
		b, eb, err := eencode(o)
		require.NoError(t, err)
		got, err := edecode[*gad.Time](b, eb)
		require.NoError(t, err)
		require.True(t, o.Value.Equal(got.Value))
	})
}

func TestEncDecObjects(t *testing.T) {

	t.Run("encodded embedded single file", func(t *testing.T) {
		o := &gad.Embedded{
			Name:          "test.txt",
			Mode:          0644,
			ModTime:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			ReaderFactory: gad.EmbeddedBytesReaderFactory(`hello world`),
		}

		data, edata, err := eencode(o)
		require.NoError(t, err)
		obj, err := edecode[*gad.Embedded](data, edata)
		require.NoError(t, err)
		require.Equal(t, o.Name, obj.Name)
		require.Equal(t, o.Mode, obj.Mode)
		require.Equal(t, o.ModTime.UnixNano(), obj.ModTime.UnixNano())
		eData, err := o.Read()
		require.NoError(t, err)
		gData, err := obj.Read()
		require.NoError(t, err)
		require.Equal(t, eData, gData)
	})

	t.Run("encodded embedded dir tree", func(t *testing.T) {
		root := &gad.Embedded{
			Name:    "root",
			Entries: make(map[string]*gad.Embedded),
		}

		f1 := &gad.Embedded{
			Name:          "f1.txt",
			Mode:          0644,
			ModTime:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			ReaderFactory: gad.EmbeddedBytesReaderFactory(`content1`),
			Parent:        root,
		}
		root.Entries["f1.txt"] = f1

		sub := &gad.Embedded{
			Name:    "sub",
			Entries: make(map[string]*gad.Embedded),
			Parent:  root,
		}
		root.Entries["sub"] = sub

		f2 := &gad.Embedded{
			Name:          "f2.txt",
			Mode:          0644,
			ModTime:       time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC),
			ReaderFactory: gad.EmbeddedBytesReaderFactory(`content2`),
			Parent:        sub,
		}
		sub.Entries["f2.txt"] = f2

		data, edata, err := eencode(root)
		require.NoError(t, err)
		obj, err := edecode[*gad.Embedded](data, edata)
		require.NoError(t, err)
		require.Equal(t, root.Name, obj.Name)
		require.True(t, obj.IsDir())

		f1Obj := obj.Entries["f1.txt"]
		require.NotNil(t, f1Obj)
		require.Equal(t, f1.Name, f1Obj.Name)
		require.Equal(t, f1.Mode, f1Obj.Mode)
		f1Data, err := f1Obj.Read()
		require.NoError(t, err)
		require.Equal(t, "content1", string(f1Data))

		subObj := obj.Entries["sub"]
		require.NotNil(t, subObj)
		require.True(t, subObj.IsDir())
		require.Equal(t, sub.Name, subObj.Name)

		f2Obj := subObj.Entries["f2.txt"]
		require.NotNil(t, f2Obj)
		require.Equal(t, f2.Name, f2Obj.Name)
		f2Data, err := f2Obj.Read()
		require.NoError(t, err)
		require.Equal(t, "content2", string(f2Data))
	})

	t.Run("encodded embedded empty dir", func(t *testing.T) {
		o := &gad.Embedded{
			Name:    "empty",
			Entries: make(map[string]*gad.Embedded),
		}

		data, err := encode(o)
		require.NoError(t, err)
		obj, err := decode[*gad.Embedded](data)
		require.NoError(t, err)
		require.Equal(t, o.Name, obj.Name)
		require.True(t, obj.IsDir())
		require.Empty(t, obj.Entries)
	})

	data, err := encode(gad.Nil)
	require.NoError(t, err)
	if obj, err := decode[*gad.NilType](data); err != nil {
		t.Fatal(err)
	} else {
		require.Equal(t, gad.Nil, obj)
	}

	boolObjects := []gad.Bool{gad.True, gad.False, gad.Bool(true), gad.Bool(false)}

	for _, tC := range boolObjects {
		msg := fmt.Sprintf("Bool(%v)", tC)
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v gad.Bool
		v, err = decode[gad.Bool](data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)
	}

	flagObjects := []gad.Flag{gad.Yes, gad.No, gad.Flag(true), gad.Flag(false)}
	for _, tC := range flagObjects {
		msg := fmt.Sprintf("Flag(%v)", tC)
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v gad.Flag
		v, err = decode[gad.Flag](data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)
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
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v gad.Int
		v, err = decode[gad.Int](data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)
	}

	uintObjects := []gad.Uint{gad.Uint(0), gad.Uint(1), ^gad.Uint(0)}
	for i := 0; i < 1000; i++ {
		v := seededRand.Uint64()
		uintObjects = append(uintObjects, gad.Uint(v))
	}

	for _, tC := range uintObjects {
		msg := fmt.Sprintf("Uint(%v)", tC)
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v gad.Uint
		v, err = decode[gad.Uint](data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)
	}

	charObjects := []gad.Char{gad.Char(0)}
	for i := 0; i < 1000; i++ {
		v := seededRand.Int31()
		charObjects = append(charObjects, gad.Char(v))
	}

	for i, tC := range charObjects {
		msg := fmt.Sprintf("Char[%d](%v)", i, tC)
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v gad.Char
		v, err = decode[gad.Char](data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)
	}

	floatObjects := []gad.Float{gad.Float(0), gad.Float(-1)}
	for i := 0; i < 1000; i++ {
		v := seededRand.Float64()
		floatObjects = append(floatObjects, gad.Float(v))
	}
	floatObjects = append(floatObjects, gad.NaN)
	for _, tC := range floatObjects {
		msg := fmt.Sprintf("Float(%v)", tC)
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v gad.Float
		v, err = decode[gad.Float](data)
		require.NoError(t, err, msg)
		if !math.IsNaN(float64(tC)) || !math.IsNaN(float64(v)) {
			require.Equal(t, float64(tC), float64(v), msg)
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
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v gad.Str
		v, err = decode[gad.Str](data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)
	}

	rawStringObjects := []gad.RawStr{gad.RawStr(""), gad.RawStr("çığöşü")}
	for i := 0; i < 1000; i++ {
		rawStringObjects = append(rawStringObjects, gad.RawStr(randString(i)))
	}
	for _, tC := range rawStringObjects {
		msg := fmt.Sprintf("RawStr(%v)", tC)
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v gad.RawStr
		v, err = decode[gad.RawStr](data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)
	}

	bytesObjects := []gad.Bytes{{}, gad.Bytes("çığöşü")}
	for i := 0; i < 1000; i++ {
		bytesObjects = append(bytesObjects, gad.Bytes(randString(i)))
	}
	for _, tC := range bytesObjects {
		msg := fmt.Sprintf("Bytes(%v)", tC)
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v gad.Bytes
		v, err = decode[gad.Bytes](data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)
	}

	decimalObjects := []gad.Decimal{gad.DecimalFromFloat(gad.Float(0)), gad.DecimalFromFloat(gad.Float(-1))}
	for i := 0; i < 1000; i++ {
		v := seededRand.Float64()
		decimalObjects = append(decimalObjects, gad.DecimalFromFloat(gad.Float(v)))
	}
	for _, tC := range decimalObjects {
		msg := fmt.Sprintf("Decimal(%v)", tC)
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v gad.Decimal
		v, err = decode[gad.Decimal](data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)
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
	arrays = append(arrays, gad.Array{&gad.SymbolInfo{Scope: gad.ScopeBuiltin, Index: 10, Name: "test"}})

	for _, tC := range arrays {
		msg := fmt.Sprintf("Array(%v)", tC)
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v gad.Array
		v, err = decode[gad.Array](data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)
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
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v gad.Dict
		v, err = decode[gad.Dict](data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)
	}

	syncMaps := []*gad.SyncDict{}
	for _, m := range maps {
		syncMaps = append(syncMaps, &gad.SyncDict{Value: m})
	}

	for _, tC := range syncMaps {
		msg := fmt.Sprintf("SyncDict(%v)", tC)
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v *gad.SyncDict
		v, err = decode[*gad.SyncDict](data)
		require.NoError(t, err, msg)
		require.Equal(t, tC, v, msg)
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
			withParams("*_"),
		),
		compFunc(nil,
			withParams("*_"),
		),
		compFunc(nil,
			withSourceMap(map[int]int{0: 1, 3: 1, 5: 1}),
		),
		compFunc(concatInsts(
			makeInst(gad.OpConstant, 0),
			makeInst(gad.OpConstant, 1),
			makeInst(gad.OpBinary, int(token.Add)),
		),
			withParams("*a"),
			withLocals(2),
			withSourceMap(map[int]int{0: 1, 3: 1, 5: 1}),
		),
	}

	for i, tC := range compFuncs {
		msg := fmt.Sprintf("CompiledFunction #%d", i)
		data, err := encode(tC)
		require.NoError(t, err, msg)
		require.Greater(t, len(data), 0, msg)
		var v *gad.CompiledFunction
		v, err = decode[*gad.CompiledFunction](data)
		require.NoError(t, err, msg)

		if len(v.Instructions) == 0 {
			v.Instructions = nil
		}

		require.Equal(t, tC, v, msg)

	}
}
