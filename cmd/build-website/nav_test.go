package main

import (
	"html/template"
	"testing"
)

// TestBuildNavTree checks that pages sharing a subdirectory prefix in NavPath
// nest under an expandable submenu, recursively, while flat pages stay top level.
func TestBuildNavTree(t *testing.T) {
	body := template.HTML("<p>x</p>")
	pages := []*page{
		{Title: "Functions", OutFile: "lang-03_functions.html", NavPath: "03_functions", BodyHTML: body},
		{Title: "Classes", OutFile: "lang-class-classes.html", NavPath: "class/classes", BodyHTML: body},
		{Title: "The class keyword", OutFile: "lang-class-syntax.html", NavPath: "class/syntax", BodyHTML: body},
		{Title: "Deep", OutFile: "lang-class-a-b.html", NavPath: "class/a/b", BodyHTML: body},
		{Title: "Interfaces", OutFile: "lang-24_interfaces.html", NavPath: "24_interfaces", BodyHTML: body},
	}

	tree := buildNavTree(pages)

	// Top level: Functions (leaf), Class (submenu), Interfaces (leaf) — a submenu
	// takes the position of its first page.
	if len(tree) != 3 {
		t.Fatalf("want 3 top-level items, got %d: %+v", len(tree), tree)
	}
	if tree[0].Title != "Functions" || tree[0].Slug != "lang-03_functions" || len(tree[0].Children) != 0 {
		t.Errorf("item 0 should be the flat Functions leaf, got %+v", tree[0])
	}
	if tree[2].Title != "Interfaces" || len(tree[2].Children) != 0 {
		t.Errorf("item 2 should be the flat Interfaces leaf, got %+v", tree[2])
	}

	// The Class submenu: a header with no slug, holding classes, syntax and a
	// nested `a` submenu.
	class := tree[1]
	if class.Title != "Class" || class.Slug != "" || len(class.Children) != 3 {
		t.Fatalf("want a Class submenu with 3 children, got %+v", class)
	}
	if class.Children[0].Title != "Classes" || class.Children[0].Slug != "lang-class-classes" {
		t.Errorf("first Class child should be Classes, got %+v", class.Children[0])
	}

	// Nested one level deeper: class/a/b -> Class > A > Deep.
	deep := class.Children[2]
	if deep.Title != "A" || len(deep.Children) != 1 || deep.Children[0].Title != "Deep" {
		t.Errorf("want a nested A submenu holding Deep, got %+v", deep)
	}
}

// TestBuildNavTreeFlat checks that pages without a subdirectory NavPath (existing
// sections) produce a flat list with no submenus.
func TestBuildNavTreeFlat(t *testing.T) {
	body := template.HTML("<p>x</p>")
	pages := []*page{
		{Title: "Get Started", OutFile: "get-started.html", BodyHTML: body},
		{Title: "Formatting", OutFile: "formatting.html", BodyHTML: body},
	}
	tree := buildNavTree(pages)
	if len(tree) != 2 {
		t.Fatalf("want 2 flat items, got %d", len(tree))
	}
	for _, it := range tree {
		if len(it.Children) != 0 {
			t.Errorf("flat page should have no children: %+v", it)
		}
	}
}
