# CMS Example

The CMS example lives in `examples/cms`. It demonstrates a complete Go web app
using Giom templates for the public site and a React admin dashboard.

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
- Public Giom templates in `public/*.giom`
- Shared components in `public/components.giom`
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
/                 -> public/index.giom
/pages/{slug}     -> public/page.giom
/posts/{slug}     -> public/post.giom
/tags/{slug}      -> public/tag.giom
```

Each page imports components:

```giom
@import "components.giom"
```

The app resolves imports before compilation and writes a `.gad` file for
inspection.

## Public Components

`components.giom` includes:

- `topbar()`
- `breadcrumbs()`
- `hero(title, summary; cover="")`
- `post_card(post)`
- `gallery(images)`
- `pager(pager)`
- `page_footer()`
- `page_wrapper(title)`

## Adding A Page Template

Create `public/landing.giom`:

```giom
@import "components.giom"

@main
    +page_wrapper("Landing")
        +hero("Landing", "A focused marketing page." ; cover="/seed-data/images/about-hero.jpg")
        div.container
            section.page-body
                h2 Build faster
                p Compose layouts with Giom components.
```

Add a handler that calls:

```go
a.render(w, "landing.giom", a.model("Landing", []crumb{{"Home", "/"}}, gad.Dict{}))
```

## Adding Seed Data

Edit `seed.yaml`:

```yaml
pages:
  - title: "Landing"
    slug: landing
    summary: "A focused marketing page."
    body: "<p>Compose layouts with Giom components.</p>"
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
