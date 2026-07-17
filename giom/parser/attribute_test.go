package parser

import (
	"testing"

	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	giomnode "github.com/gad-lang/gad/giom/node"
)

// firstTag returns the first TagStmt found by depth-first walk.
func firstTag(file *giomnode.File) *giomnode.TagStmt {
	var walk func(stmts node.Stmts) *giomnode.TagStmt
	walk = func(stmts node.Stmts) *giomnode.TagStmt {
		for _, s := range stmts {
			switch n := s.(type) {
			case *giomnode.TagStmt:
				return n
			case *giomnode.CompDecl:
				if t := walk(n.Body); t != nil {
					return t
				}
			}
		}
		return nil
	}
	if t := walk(file.Stmts); t != nil {
		return t
	}
	for _, comp := range file.Comps {
		if t := walk(comp.Body); t != nil {
			return t
		}
	}
	return nil
}

// attrNames returns the ordered attribute names of a tag.
func attrNames(tag *giomnode.TagStmt) []string {
	names := make([]string, len(tag.Attributes))
	for i, a := range tag.Attributes {
		names[i] = a.Name
	}
	return names
}

func parseTagFile(t *testing.T, src string) (*source.FileSet, *giomnode.TagStmt) {
	t.Helper()
	fs, _, file := parseFileWith(t, src)
	tag := firstTag(file)
	if tag == nil {
		t.Fatalf("no tag found in:\n%s", src)
	}
	return fs, tag
}

func TestAttributeSingle(t *testing.T) {
	_, tag := parseTagFile(t, "@main\n    div[class=\"a\"] hi\n")
	if got := attrNames(tag); len(got) != 1 || got[0] != "class" {
		t.Fatalf("got %v, want [class]", got)
	}
	if tag.Attributes[0].IsFlag {
		t.Fatal("class should not be a flag")
	}
}

func TestAttributeSeparateGroups(t *testing.T) {
	_, tag := parseTagFile(t, "@main\n    div[class=\"a\"][title=\"b\"] hi\n")
	if got := attrNames(tag); len(got) != 2 || got[0] != "class" || got[1] != "title" {
		t.Fatalf("got %v, want [class title]", got)
	}
}

func TestAttributeMultiValueComma(t *testing.T) {
	_, tag := parseTagFile(t, "@main\n    div[class=\"a\", title=\"hello\"] hi\n")
	if got := attrNames(tag); len(got) != 2 || got[0] != "class" || got[1] != "title" {
		t.Fatalf("got %v, want [class title]", got)
	}
}

func TestAttributeMultiLineGroup(t *testing.T) {
	src := "@main\n" +
		"    div[\n" +
		"        class=\"a\"\n" +
		"        class=\"b\"\n" +
		"        title=\"hello\"\n" +
		"    ] hi\n"
	_, tag := parseTagFile(t, src)
	if got := attrNames(tag); len(got) != 3 {
		t.Fatalf("got %v, want 3 attributes", got)
	}
}

func TestAttributeMixedSeparators(t *testing.T) {
	// first attr on the opening line, remaining on the next, comma-separated
	src := "@main\n" +
		"    div[class=\"a\"\n" +
		"        class=\"b\", title=\"hello\"] hi\n"
	_, tag := parseTagFile(t, src)
	if got := attrNames(tag); len(got) != 3 || got[0] != "class" || got[1] != "class" || got[2] != "title" {
		t.Fatalf("got %v, want [class class title]", got)
	}
}

func TestAttributeFlagAndExpression(t *testing.T) {
	_, tag := parseTagFile(t, "@main\n    input[type=\"text\", disabled, value=1+2]\n")
	if got := attrNames(tag); len(got) != 3 {
		t.Fatalf("got %v, want 3", got)
	}
	if !tag.Attributes[1].IsFlag {
		t.Fatalf("disabled should be a flag: %+v", tag.Attributes[1])
	}
	if tag.Attributes[2].IsRaw {
		t.Fatal("value=1+2 should be an expression, not raw")
	}
}

func TestAttributeCommaInsideValueNotSplit(t *testing.T) {
	// A comma inside a call/string must not split the group.
	_, tag := parseTagFile(t, "@main\n    div[title=join([\"a\",\"b\"], \",\")] hi\n")
	if got := attrNames(tag); len(got) != 1 || got[0] != "title" {
		t.Fatalf("got %v, want [title]", got)
	}
}

func TestAttributeValuePositionMapsToSourceLine(t *testing.T) {
	// The expression value `compute()` on line 5 must resolve to line 5, col 15.
	src := "@main\n" +
		"    div[\n" +
		"        class=\"a\"\n" +
		"        class=\"b\"\n" +
		"        title=compute()\n" +
		"    ] hi\n"
	fs, tag := parseTagFile(t, src)
	var titleVal node.Expr
	for _, a := range tag.Attributes {
		if a.Name == "title" {
			titleVal = a.Value
		}
	}
	if titleVal == nil {
		t.Fatal("title attribute value not found")
	}
	p := fs.Position(titleVal.Pos())
	if p.Line != 5 || p.Column != 15 {
		t.Fatalf("title value resolved to %d:%d, want 5:15", p.Line, p.Column)
	}
}
