# Examples Cookbook

These examples are intentionally small and composable. Copy a pattern, then
extend it for your application.

## Static Page

```giom
@main
    !!! 5
    html
        head
            title Static page
        body
            h1 Static page
            p Rendered by Giom.
```

## Page With Model

```giom
~~
global (Model)
~~

@main
    h1 {= Model.Title}
    p {= Model.Summary}
```

Model shape:

```go
gad.Dict{
    "Title": gad.Str("Fast pages"),
    "Summary": gad.Str("A compact introduction."),
}
```

## Blog Index

```giom
@main
    section.hero
        h1 {= Model.Title}
    div.grid
        @for post in Model.Posts
            article.card
                h2
                    a[href=post.URL] {= post.Title}
                p {= post.Summary}
```

## Blog Post

```giom
@main
    article.post
        h1 {= Post.Title}
        p.meta {= Post.PublishedAt}
        div.body {= Post.Body}
```

## Navigation Menu

```giom
@export comp menu(items)
    nav.menu
        @for item in items
            a[href=item.URL] {= item.Label}

@main
    +menu(Model.Menu)
```

## Breadcrumbs

```giom
@export comp breadcrumbs(items)
    nav.breadcrumbs[aria-label="Breadcrumb"]
        @for item in items
            a[href=item.URL] {= item.Label}
            span / 
```

## Empty State

```giom
@if Model.Posts
    div.grid
        @for post in Model.Posts
            article.card {= post.Title}
@else
    section.empty
        h2 No posts yet
        p Create the first post in the admin dashboard.
```

## Reusable Button

```giom
@export comp button(label; href="#", variant="primary")
    a.button[href=href][class="button--" + variant] {= label}
```

## Form Field

```giom
@export comp field(label, name; type="text", value="", required=false)
    label.field
        span {= label}
        input[type=type][name=name][value=value][required=required]
```

## Attribute Groups

Group related attributes in one `[ … ]` and let long tags wrap across lines.

```giom
@export comp card(post)
    article[
        class="card"
        class=post.Featured ? "card--featured" : ""
        data-id=post.ID
        aria-label=post.Title
    ]
        h3 {= post.Title }
        a[href=post.URL, rel="bookmark"] Read more
```

Usage:

```giom
form[action="/signup"][method="post"]
    +field("Email", "email" ; type="email", required=true)
    button[type="submit"] Join
```

## Dashboard Shell

```giom
@export comp dashboard(title)
    div.dashboard
        aside.sidebar
            @slot sidebar
                a[href="/"] Home
        main.panel
            h1 {= title}
            @slot main

@main
    +dashboard("Reports")
        @slot #sidebar
            a[href="/reports"] Reports
            a[href="/settings"] Settings
        p Report content.
```

## Table

```giom
table.table
    thead
        tr
            th Title
            th Status
    tbody
        @for row in Rows
            tr
                td {= row.Title}
                td {= row.Status}
```

## Gallery

```giom
@export comp gallery(images)
    @if images
        div.gallery
            @for image in images
                img[src=image.URL][alt=image.Alt]
```

## Pagination

```giom
@export comp pager(pager)
    @if pager.HasPrev || pager.HasNext
        nav.pager
            @if pager.HasPrev
                a[href=pager.PrevURL] Previous
            span Page {= pager.Page} of {= pager.TotalPages}
            @if pager.HasNext
                a[href=pager.NextURL] Next
```

## Layout With CSS

```giom
~~
const css = `body{font-family:system-ui;margin:0}.container{max-width:960px;margin:auto}`
~~

@export comp page(title)
    !!! 5
    html
        head
            title {= title}
            style {= css}
        body
            main.container
                @slot main
```

## Extending Examples

- Move repeated markup into `@export comp` components.
- Use a layout component with a `main` slot for pages.
- Add named slots for sidebars, toolbars, and metadata areas.
- Keep database querying and pagination calculation in Go.
- Pass pre-shaped model data to templates.
