# Serving over HTTP

`gadx.Render` is a ready-to-use template engine: it compiles a `.gadx` file (or
retrieves cached bytecode), runs it, and writes the rendered HTML to an
`io.Writer`. Because an `http.ResponseWriter` **is** an `io.Writer`, a Gadx
template renders straight into an HTTP response — making Gadx a natural frontend
layer for a Go web app.

```go
func (r *Render) Render(out io.Writer, filePath string, globals gad.Dict) error
```

## Minimal server (`HandlerFunc`)

`Render.HandlerFunc` returns an `http.HandlerFunc` you hand straight to
`http.HandleFunc` / `http.Handle`. Create one `*gadx.Render` for the whole
process (it is safe for concurrent use and caches compiled bytecode across
requests) and map each route to a template:

```go
package main

import (
    "log"
    "net/http"

    "github.com/gad-lang/gad"
    "github.com/gad-lang/gad/gadx"
)

func main() {
    // One engine, reused across requests. The workdir is where .gadx files and
    // their @import siblings live.
    r := gadx.NewRender("./templates")

    // The model callback maps the request to the template globals; its Dict keys
    // become global variables inside the template. Pass nil when a template needs
    // no request data.
    http.HandleFunc("/", r.HandlerFunc("index.gadx", func(req *http.Request) (gad.Dict, error) {
        return gad.Dict{
            "Path":  gad.Str(req.URL.Path),
            "Query": gad.Str(req.URL.RawQuery),
        }, nil
    }))

    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

`templates/index.gadx`:

```gadx
@main
    html
        head
            title Home
        body
            h1 Hello from Gadx
            p You requested {= Path }
```

`HandlerFunc` renders into a buffer first, so a model or render error becomes a
clean `500` with no partial body, and a successful response is written with a
`text/html; charset=utf-8` Content-Type.

```go
func (r *Render) HandlerFunc(filePath string, model func(*http.Request) (gad.Dict, error)) http.HandlerFunc
```

## Custom handlers (`Render`)

For full control over status codes, streaming, headers or content type, call
`Render` from your own handler. Streaming straight to the `ResponseWriter` is
cheapest, but a template that fails **after** the first bytes are written cannot
change the status — the client already began receiving `200 OK`. Buffer first
when you need a correct error status (this is exactly what `HandlerFunc` does):

```go
func handle(r *gadx.Render, file string) http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        var buf bytes.Buffer
        globals := gad.Dict{"Path": gad.Str(req.URL.Path)}
        if err := r.Render(&buf, file, globals); err != nil {
            http.Error(w, "template error", http.StatusInternalServerError)
            log.Printf("render %s: %v", file, err)
            return
        }
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        _, _ = buf.WriteTo(w)
    }
}
```

## Passing request data to the template

Everything the template needs arrives through `globals`. Marshal path/query
params, the authenticated user, a page model, etc. into a `gad.Dict`:

```go
globals := gad.Dict{
    "User":  gad.Str(currentUser(req)),
    "Page":  gad.Dict{"Title": gad.Str("Dashboard")},
    "Items": items, // any gad.Object (e.g. a gad.Array of gad.Dict)
}
```

Inside the template these are ordinary globals:

```gadx
@main
    h1 {= Page.Title }
    @if User
        p Signed in as {= User }
    ul
        @for it in Items
            li {= it.name }
```

## Performance across requests

- **Bytecode cache** — the first request for a template compiles it; later
  requests reuse the cached bytecode. A source change triggers a deferred
  recompile (`Render.TemplateDelay`, default 15s), so edits show up in dev
  without restarting.
- **Interface-satisfaction cache** — each compiled template carries a
  `gad.InterfaceSatCache` that `Render` reuses across requests, so an
  `obj :: SomeInterface` check (or an interface parameter type) is validated
  once per value type instead of every render. See
  [API › `(*Render) Render`](api.md) for details.

For a full application wiring routes, a database and Gadx components together,
see the [CMS example](cms-example.md).
