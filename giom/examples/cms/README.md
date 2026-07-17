# GION v2 CMS Example

This is a small CMS application using:

- Go `net/http` backend
- GORM with SQLite
- GION/v2 public templates from `public/*.giom`
- React admin dashboard at `/admin`

The renderer resolves `@import` directives at load time and writes the
transpiled Gad source to `public/.transpiled/*.gad` for inspection.

Run it:

```sh
go run .
```

Open:

- Public site: `http://localhost:8080/`
- Admin dashboard: `http://localhost:8080/admin`

The app creates `cms.db` on first run and seeds pages, tags, posts, and menu entries.
