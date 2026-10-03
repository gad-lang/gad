package gadx

import (
	"sync"

	"github.com/gad-lang/gad"
)

// AttrHandler is given each attribute of a tag as the tag is written — the
// tag's name, the attribute's, its value — and returns the value to write:
// the value itself to leave it, another to change it (a URI made versioned,
// absolute, …), nil to drop the attribute. It changes the HTML written, not
// the render tree.
type AttrHandler func(tag, name string, value gad.Object) gad.Object

// attrHandlers are the AttrHandler of each VM rendering (WithAttrHandler).
var attrHandlers sync.Map // *gad.VM → AttrHandler

// WithAttrHandler makes h the AttrHandler of the tags vm writes, until the
// returned function is called. Render does it for its AttrHandler; whoever
// runs a template on a VM of their own does it for theirs.
func WithAttrHandler(vm *gad.VM, h AttrHandler) (done func()) {
	if h == nil {
		return func() {}
	}
	attrHandlers.Store(vm, h)
	return func() { attrHandlers.Delete(vm) }
}

func attrHandlerOf(vm *gad.VM) AttrHandler {
	if vm == nil {
		return nil
	}
	if h, ok := attrHandlers.Load(vm); ok {
		return h.(AttrHandler)
	}
	return nil
}
