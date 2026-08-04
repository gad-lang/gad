# CMS Example

The CMS example lives in `examples/cms`. It demonstrates a complete Go web app
using Gadx templates for the public site and a React admin dashboard.

## Run

```sh
cd examples/cms
go run .
```

Open:

```text
http://localhost:8080/
http://localhost:8080/admin
```

## What It Includes

- `net/http` server
- GORM models for pages, posts, tags, and menu items
- SQLite database at `cms.db`
- Seed data in `seed.yaml`
- Public Gadx templates in `public/*.gadx`
- Shared components in `public/components.gadx`
- Transpiled Gad output in `public/.transpiled/*.gad`
- Static seed images in `seed-data/images`
- React admin dashboard in `admin/`

## Seeding Behavior

The app checks whether `cms.db` exists before opening SQLite.

- If `cms.db` does not exist, the app migrates tables and seeds data from `seed.yaml`.
- If `cms.db` exists, the app migrates tables but does not re-seed.
- Delete `cms.db` to reset to the seed dataset.

## Template Flow

Routes load templates by name:

```text
/                 -> public/index.gadx
/pages/{slug}     -> public/page.gadx
/posts/{slug}     -> public/post.gadx
/tags/{slug}      -> public/tag.gadx
```

Each page imports components:

```gadx
@import "components.gadx"
```

The app resolves imports before compilation and writes a `.gad` file for
inspection.

## Public Components

`components.gadx` includes:

- `topbar()`
- `breadcrumbs()`
- `hero(title, summary; cover="")`
- `post_card(post)`
- `gallery(images)`
- `pager(pager)`
- `page_footer()`
- `page_wrapper(title)`

## Adding A Page Template

Create `public/landing.gadx`:

```gadx
@import "components.gadx"

@main
    +page_wrapper("Landing")
        +hero("Landing", "A focused marketing page." ; cover="/seed-data/images/about-hero.jpg")
        div.container
            section.page-body
                h2 Build faster
                p Compose layouts with Gadx components.
```

Add a handler that calls:

```go
a.render(w, "landing.gadx", a.model("Landing", []crumb{{"Home", "/"}}, gad.Dict{}))
```

## Adding Seed Data

Edit `seed.yaml`:

```yaml
pages:
  - title: "Landing"
    slug: landing
    summary: "A focused marketing page."
    body: "<p>Compose layouts with Gadx components.</p>"
    coverImage: "/seed-data/images/about-hero.jpg"
    images: []
    published: true
```

Delete `cms.db` and restart to apply seed changes.

## Image Paths

Use absolute paths for browser-safe URLs:

```yaml
coverImage: "/seed-data/images/about-hero.jpg"
```

Relative paths such as `seed-data/images/about-hero.jpg` break on nested routes
because browsers resolve them relative to the current URL.
