package parser

import (
	"fmt"

	gadxnode "github.com/gad-lang/gad/gadx/node"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// Install the HTML-to-gadx-nodes hook used by gadx/node to lower an `@md` block
// through the HTML front-end (see node.HTMLToNodes). gadx/node cannot import
// gadx/parser directly — the parser imports the node package — so the parser
// registers buildHTMLNodes here in an init instead.
func init() {
	gadxnode.HTMLToNodes = func(html string, base source.Pos) (gnode.Stmts, error) {
		nodes, errs := buildHTMLNodes(html, base)
		if len(errs) > 0 {
			return nil, fmt.Errorf("parse markdown-generated HTML: %s", errs[0].msg)
		}
		return nodes, nil
	}
}
