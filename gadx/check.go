package gadx

import (
	"strings"

	gad "github.com/gad-lang/gad"
)

// The output checks of a gadx file (issue #22): `<<< <div>hello</div>` after
// markup checks what was rendered into the current node (`$el`) since the
// previous check — gad.RenderedChecker.

var (
	_ gad.RenderedChecker = (*Tag)(nil)
	_ gad.RenderedChecker = (*Elements)(nil)
)

// CheckRendered is the rendering of the children added since the previous
// call.
func (t *Tag) CheckRendered(vm *gad.VM) (string, error) {
	s, err := renderElements(vm, t.Children[min(t.checked, len(t.Children)):])
	t.checked = len(t.Children)
	return s, err
}

// CheckRendered is the rendering of the items added since the previous call.
func (e *Elements) CheckRendered(vm *gad.VM) (string, error) {
	s, err := renderElements(vm, e.Items[min(e.checked, len(e.Items)):])
	e.checked = len(e.Items)
	return s, err
}

func renderElements(vm *gad.VM, els []Element) (string, error) {
	var b strings.Builder
	for _, el := range els {
		if _, err := el.WriteTo(vm, &b); err != nil {
			return "", err
		}
	}
	return b.String(), nil
}
