package http

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gad-lang/gad"
)

func Request(call gad.Call) (_ gad.Object, err error) {
	var (
		url = gad.Arg{
			Name:          "url",
			TypeAssertion: gad.TypeAssertionFromTypes(gad.TStr),
		}
		methodVar = gad.NamedArgVar{
			Name:          "method",
			TypeAssertion: gad.TypeAssertionFromTypes(gad.TStr),
		}
		bodyVar = gad.NamedArgVar{
			Name:          "body",
			TypeAssertion: gad.TypeAssertionFromTypes(gad.TReader),
		}

		method = "get"
		body   io.Reader
	)

	if err = call.Args.Destructure(&url); err != nil {
		return
	}

	if err = call.NamedArgs.Get(&methodVar, &bodyVar); err != nil {
		return
	}

	if methodVar.Value != nil && !methodVar.Value.IsFalsy() {
		method = methodVar.Value.ToString()
	}

	if bodyVar.Value != nil && !bodyVar.Value.IsFalsy() {
		body = bodyVar.Value.(gad.Reader)
	}

	var r *http.Request
	if r, err = http.NewRequest(method, url.Value.ToString(), body); err != nil {
		return
	}
	return gad.ToObject(r)
}

func Get(call gad.Call) (_ gad.Object, err error) {
	url := gad.Arg{
		Name: "url",
	}

	if err = call.Args.Destructure(&url); err != nil {
		return
	}

	var r *http.Response
	if r, err = http.Get(url.Value.ToString()); err != nil {
		return
	}
	if r.Body != nil {
		defer r.Body.Close()
	}
	if r.StatusCode < 300 {
		var b gad.Buffer
		if _, err = io.Copy(&b, r.Body); err != nil {
			return
		}

		return &b, nil
	}
	return nil, gad.ErrType.NewError(fmt.Sprintf("unseccessful response type %d %s", r.StatusCode, r.Status))
}

func URL(call gad.Call) (_ gad.Object, err error) {
	var (
		s = gad.Arg{
			Name:          "url",
			TypeAssertion: gad.TypeAssertionFromTypes(gad.TStr),
		}
	)

	if err = call.Args.Destructure(&s); err != nil {
		return
	}

	var URL *url.URL

	if URL, err = url.Parse(s.Value.ToString()); err != nil {
		return
	}

	q := URL.Query()

	call.NamedArgs.Walk(func(na *gad.KeyValue) error {
		q.Add(na.K.ToString(), na.V.ToString())
		return nil
	})

	URL.RawQuery = ""
	return gad.Str(URL.String()), nil
}

func Header(call gad.Call) (gad.Object, error) {
	h := make(http.Header)
	call.NamedArgs.Walk(func(na *gad.KeyValue) error {
		k := na.K.ToString()
		switch t := na.V.(type) {
		case gad.Array:
			arr := make([]string, len(t))
			for i, v := range t {
				arr[i] = v.ToString()
			}
			h[k] = arr
		default:
			h.Add(k, t.ToString())
		}
		return nil
	})
	return gad.MustToObject(h), nil
}
