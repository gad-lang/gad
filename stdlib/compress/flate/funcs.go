package flate

import (
	"compress/flate"
	"io"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/helper"
)

func Encode(c gad.Call) (_ gad.Object, err error) {
	var (
		writer = helper.ArgOfWriter("dst")
		reader = helper.ArgOfReader("src")
		level  = &gad.NamedArgVar{
			Name:          "level",
			Value:         gad.Int(9),
			TypeAssertion: gad.TypeAssertionFromTypes(gad.TInt),
		}

		zw *flate.Writer
	)

	if err = c.Args.Destructure(writer, reader); err != nil {
		return
	}

	if err = c.NamedArgs.Get(level); err != nil {
		return
	}

	zw, err = flate.NewWriter(writer.Value.(gad.Writer).GoWriter(), int(level.Value.(gad.Int)))
	if err != nil {
		return
	}

	defer zw.Close()

	_, err = io.Copy(zw, reader.Value.(gad.Reader).GoReader())
	return gad.Nil, err
}

func Decode(c gad.Call) (_ gad.Object, err error) {
	var (
		writer = helper.ArgOfWriter("dst")
		reader = helper.ArgOfReader("src")
		zr     io.ReadCloser
	)

	if err = c.Args.Destructure(writer, reader); err != nil {
		return
	}

	zr = flate.NewReader(reader.Value.(gad.Reader).GoReader())
	defer zr.Close()

	_, err = io.Copy(writer.Value.(gad.Writer).GoWriter(), zr)
	return gad.Nil, err
}
