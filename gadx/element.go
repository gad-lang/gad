package gadx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/gad-lang/gad"
	gadxnode "github.com/gad-lang/gad/gadx/node"
)

// ElementType classifies the kind of a rendered Element.
type ElementType uint8

const (
	// ElementText is a text (leaf) element wrapping a value.
	ElementText ElementType = 0
	// ElementTag is an HTML/XML tag element with attributes and children.
	ElementTag ElementType = 1
)

// Element is a node in the render tree produced by a gadx template. Every
// Element is a Gad object that can write itself (and its subtree) to an
// io.Writer during the final render walk.
type Element interface {
	gad.ToWriter
	// ElType reports whether the element is a tag or text node.
	ElType() ElementType
}

// TagType is the Gad object type of *Tag. Calling it constructs a tag:
//
//	gadx.Tag([parent,] name, *children; **attrs)
//
// Omitting the name (gadx.Tag() / gadx.Tag(parent)) yields an anonymous
// (fragment) tag: it has no name and on render writes only its children, without
// a surrounding element. Components/slots build into one and return it.
var TagType = gad.NewBuiltinObjType("Tag").WithNew(tagCtor)

// TextType is the Gad object type of *TextElement. Calling it wraps a value as
// a text node: gadx.Text(value).
var TextType = gad.NewBuiltinObjType("Text").WithNew(textCtor)

// MdType is the Gad object type of a `@md` Markdown container. Calling it builds
// an anonymous markdown tag whose children are rendered as Markdown source and
// converted to HTML on render: gadx.Md([parent]).
var MdType = gad.NewBuiltinObjType("Md").WithNew(mdCtor)

func init() {
	TagType.SetModule(ModuleSpec)
	TextType.SetModule(ModuleSpec)
	MdType.SetModule(ModuleSpec)
}

// mdCtor implements gadx.Md([parent]): an anonymous tag flagged markdown, so on
// render its collected children become Markdown source converted to HTML.
func mdCtor(c gad.Call) (gad.Object, error) {
	parent, _ := parentArg(c)
	t := &Tag{markdown: true}
	if parent != nil {
		parent.Children = append(parent.Children, t)
	}
	return t, nil
}

// =============================================================================
// Tag
// =============================================================================

// Tag is a tag element: an optional name, ordered attributes and child
// elements. A Tag with an empty Name is an anonymous fragment that renders only
// its children. Tag implements Element and the operator interfaces used by the
// compiled template to build the tree (`tag += child`, `tag ++= children`,
// `tag[attr] = value`, `tag.attrs += attrs`).
type Tag struct {
	Name      string
	Attrs     gad.Dict
	Children  []Element
	ClassList []string
	Styles    []string
	// attrOrder preserves the insertion order of Attrs keys, since gad.Dict (a
	// Go map) is unordered and attribute output order is significant.
	attrOrder []string
	// markdown marks a `@md` container (built by gadx.Md): on render its children
	// are written to a buffer as Markdown source and converted to HTML via
	// goldmark (see WriteTo). Such a tag is anonymous — it has no wrapper element.
	markdown bool
}

// NewTag returns a tag with the given name and children, classifying attrs into
// the tag's structured attribute state (regular attributes, class list, styles).
func NewTag(parent *Tag, name string, children []Element, attrs gad.KeyValueArray) *Tag {
	t := &Tag{Name: name, Children: children}
	if parent != nil {
		parent.Children = append(parent.Children, t)
	}
	t.mergeAttrs(attrs)
	return t
}

// tagCtor implements gadx.Tag in two forms:
//
//	gadx.Tag(parent, name, *children; **attrs)  // explicit parent (a *Tag or nil)
//	gadx.Tag(name, *children; **attrs)           // no parent
//
// The parent is detected by the first positional argument's type (see
// parentArg); the tag name is the next positional and any remaining positionals
// are static children. When no name is given (gadx.Tag() / gadx.Tag(parent)),
// the tag is anonymous (empty name) and renders only its children.
func tagCtor(c gad.Call) (gad.Object, error) {
	parent, i := parentArg(c)
	name := ""
	if c.Args.Length() > i {
		name = c.Args.Get(i).ToString()
		i++
	}
	var children []Element
	for ; i < c.Args.Length(); i++ {
		children = append(children, toElement(c.Args.Get(i)))
	}
	return NewTag(parent, name, children, c.NamedArgs.Join()), nil
}

// parentArg inspects the first positional argument of a constructor call to
// support both the parented form `gadx.X(parent, …)` and the parentless form
// `gadx.X(…)`. When the first argument is a *Tag or a nil value it is the parent
// (returned, with offset 1 so callers skip it); otherwise there is no parent
// argument (nil, offset 0) and the first positional is content.
func parentArg(c gad.Call) (parent *Tag, offset int) {
	if c.Args.Length() == 0 {
		return nil, 0
	}
	switch first := c.Args.GetOnly(0).(type) {
	case *Tag:
		return first, 1
	case *gad.NilType:
		return nil, 1
	default:
		return nil, 0
	}
}

func (t *Tag) ElType() ElementType  { return ElementTag }
func (t *Tag) Type() gad.ObjectType { return TagType }
func (t *Tag) ToString() string     { return "gadx.Tag(" + t.Name + ")" }
func (t *Tag) IsFalsy() bool        { return false }

func (t *Tag) Equal(right gad.Object) bool {
	o, ok := right.(*Tag)
	return ok && o == t
}

// Enter implements gad.ObjectEnter so a tag is a `with` resource: entering a
// `with gadx.Tag(…) as tag { … }` block is a no-op (the tag is already built).
func (t *Tag) Enter(*gad.VM) error { return nil }

// Exit implements gad.ObjectExit: leaving the `with` block yields the tag. The
// block error (nil on success) is not altered, matching the prior no-op resource
// behaviour.
func (t *Tag) Exit(_ *gad.VM, _ error) (gad.Object, error) { return t, nil }

// append adds a single child element, skipping nil (e.g. an optional slot that
// rendered nothing).
func (t *Tag) append(child gad.Object) {
	if child == nil || child == gad.Nil {
		return
	}
	t.Children = append(t.Children, toElement(child))
}

// appendMany adds each element of an iterable value as a child.
func (t *Tag) appendMany(vm *gad.VM, values gad.Object) error {
	if arr, ok := gad.ToArray(values); ok {
		for _, v := range arr {
			t.append(v)
		}
		return nil
	}
	vals, err := gad.ValuesOf(vm, values, &gad.NamedArgs{})
	if err != nil {
		return err
	}
	for _, v := range vals {
		t.append(v)
	}
	return nil
}

// BinOpAdd implements `tag + child` (ObjectWithAddBinOperator), appending the
// child and yielding the tag. It also backs `tag += child` via the self-assign
// fallback.
func (t *Tag) BinOpAdd(_ *gad.VM, right gad.Object) (gad.Object, error) {
	t.append(right)
	return t, nil
}

// SelfAssignOpAdd implements `tag += child`, appending one child.
func (t *Tag) SelfAssignOpAdd(_ *gad.VM, value gad.Object) (gad.Object, error) {
	t.append(value)
	return t, nil
}

// SelfAssignOpInc implements `tag ++= children`, appending each element of the
// iterable value as a child.
func (t *Tag) SelfAssignOpInc(vm *gad.VM, value gad.Object) (gad.Object, error) {
	if err := t.appendMany(vm, value); err != nil {
		return nil, err
	}
	return t, nil
}

// IndexGet implements `tag.attrs` (the attribute collection as a KeyValueArray),
// `tag.name`, `tag.children` and single attribute reads `tag[name]`.
func (t *Tag) IndexGet(_ *gad.VM, index gad.Object) (gad.Object, error) {
	switch index.ToString() {
	case "attrs":
		return t.attrsKeyValueArray(), nil
	case "name":
		return gad.Str(t.Name), nil
	case "children":
		arr := make(gad.Array, len(t.Children))
		for i, c := range t.Children {
			arr[i] = c
		}
		return arr, nil
	default:
		if v, ok := t.Attrs[index.ToString()]; ok {
			return v, nil
		}
		return gad.Nil, nil
	}
}

// IndexSet implements `tag.attrs = kva` / `tag.attrs += kva` (re-classify the
// whole attribute collection) and single attribute writes `tag[name] = value`
// (a "class"/"style" name feeds the class list / styles; any other name sets a
// regular attribute).
func (t *Tag) IndexSet(_ *gad.VM, index, value gad.Object) error {
	if index.ToString() == "attrs" {
		t.Attrs = nil
		t.attrOrder = nil
		t.ClassList = nil
		t.Styles = nil
		t.mergeAttrs(toKeyValueArray(value))
		return nil
	}
	t.classifyAttr(&gad.KeyValue{K: index, V: value})
	return nil
}

// WriteTo renders the tag and its subtree as HTML. An anonymous tag (empty
// Name) writes only its children; a named tag writes its open tag with rendered
// attributes, then either self-closes (for void elements) or writes its
// children and a close tag.
func (t *Tag) WriteTo(vm *gad.VM, w io.Writer) (n int64, err error) {
	if t.markdown {
		return t.writeMarkdown(vm, w)
	}
	if t.Name == "" {
		return t.writeChildren(vm, w)
	}

	var wc writeCounter
	wc.writeString(w, "<"+t.Name)
	if err = wc.err; err != nil {
		return wc.n, err
	}

	if err = t.writeAttrs(vm, w, &wc); err != nil {
		return wc.n, err
	}

	if gadxnode.IsSelfClosing(t.Name) {
		wc.writeString(w, " />")
		return wc.n, wc.err
	}

	wc.writeString(w, ">")
	if err = wc.err; err != nil {
		return wc.n, err
	}

	var cn int64
	if cn, err = t.writeChildren(vm, w); err != nil {
		return wc.n + cn, err
	}
	wc.n += cn

	wc.writeString(w, "</"+t.Name+">")
	return wc.n, wc.err
}

// writeMarkdown renders a `@md` container: its children are written to a buffer
// as Markdown source (text lines, interpolations and any nested `@` directives'
// HTML), then converted to an HTML fragment via the package Markdown renderer.
func (t *Tag) writeMarkdown(vm *gad.VM, w io.Writer) (n int64, err error) {
	var buf bytes.Buffer
	if _, err = t.writeChildren(vm, &buf); err != nil {
		return 0, err
	}
	out, err := renderMarkdown(Markdown, buf.Bytes())
	if err != nil {
		return 0, err
	}
	i, err := w.Write(out)
	return int64(i), err
}

func (t *Tag) writeChildren(vm *gad.VM, w io.Writer) (n int64, err error) {
	for _, c := range t.Children {
		var cn int64
		if cn, err = c.WriteTo(vm, w); err != nil {
			return n + cn, err
		}
		n += cn
	}
	return n, nil
}

// =============================================================================
// Text
// =============================================================================

// Text is a text (leaf) node: a sequence of values written in order. On render
// each value is written with the same semantics as gadx.write — a RawStr is
// written verbatim, any other value goes through the VM's object writer.
type Text []gad.Object

// textCtor implements gadx.Text in two forms:
//
//	gadx.Text(parent, v1, v2, …)  // explicit parent (a *Tag or nil)
//	gadx.Text(v1, v2, …)          // no parent
//
// The parent is detected by the first positional argument's type (see
// parentArg); the remaining positionals are the text values. When parent is a
// tag, the text links itself as a child.
func textCtor(c gad.Call) (gad.Object, error) {
	parent, i := parentArg(c)
	var t Text
	for ; i < c.Args.Length(); i++ {
		t = append(t, c.Args.Get(i))
	}
	if parent != nil {
		parent.Children = append(parent.Children, t)
	}
	return t, nil
}

func (t Text) ElType() ElementType  { return ElementText }
func (t Text) Type() gad.ObjectType { return TextType }
func (t Text) IsFalsy() bool        { return len(t) == 0 }

func (t Text) ToString() string {
	var b strings.Builder
	for _, v := range t {
		b.WriteString(v.ToString())
	}
	return b.String()
}

func (t Text) Equal(right gad.Object) bool {
	o, ok := right.(Text)
	if !ok || len(o) != len(t) {
		return false
	}
	for i := range t {
		if !t[i].Equal(o[i]) {
			return false
		}
	}
	return true
}

// WriteTo writes each value in order, mirroring gadx.write / gad's write builtin:
// a RawStr is written verbatim; any other value goes through the VM's object
// writer.
func (t Text) WriteTo(vm *gad.VM, w io.Writer) (n int64, err error) {
	for _, v := range t {
		var cn int64
		if rs, ok := v.(gad.RawStr); ok {
			var i int
			i, err = w.Write([]byte(rs))
			cn = int64(i)
		} else {
			otw := vm.ObjectToWriter
			if otw == nil {
				otw = gad.DefaultObjectToWrite
			}
			_, cn, err = otw.WriteTo(vm, w, v)
		}
		n += cn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// =============================================================================
// helpers
// =============================================================================

// toElement returns obj as an Element, wrapping non-Element values as text.
func toElement(obj gad.Object) Element {
	if el, ok := obj.(Element); ok {
		return el
	}
	return Text{obj}
}

// toKeyValueArray coerces value into a KeyValueArray for tag attributes.
func toKeyValueArray(value gad.Object) gad.KeyValueArray {
	switch v := value.(type) {
	case gad.KeyValueArray:
		return v
	case *gad.NamedArgs:
		return v.Join()
	case gad.Dict:
		arr := make(gad.KeyValueArray, 0, len(v))
		for k, val := range v {
			arr = append(arr, &gad.KeyValue{K: gad.Str(k), V: val})
		}
		return arr
	default:
		return nil
	}
}

// mergeAttrs classifies each attribute pair into the tag's structured state
// (regular Attrs in attrOrder, ClassList, Styles), mirroring the gadx.attrs
// builtin so tag rendering matches it without invoking the builtin.
func (t *Tag) mergeAttrs(attrs gad.KeyValueArray) {
	for _, na := range attrs {
		t.mergeAttr(na)
	}
}

// mergeAttr applies the outer gadx.attrs entry handling: a false Bool key skips
// the entry, a true Bool key unwraps to its value, and Array / KeyValue /
// KeyValueArray values are flattened into individual attribute pairs.
func (t *Tag) mergeAttr(na *gad.KeyValue) {
	v := gad.Object(na)
	if b, ok := na.K.(gad.Bool); ok {
		if !bool(b) {
			return
		}
		v = na.V
	}
	switch vv := v.(type) {
	case gad.Array:
		for _, e := range vv {
			if kv, ok := e.(*gad.KeyValue); ok {
				t.classifyAttr(kv)
			}
		}
	case *gad.KeyValue:
		t.classifyAttr(vv)
	case gad.KeyValueArray:
		for _, kv := range vv {
			t.classifyAttr(kv)
		}
	default:
		t.classifyAttr(na)
	}
}

// classifyAttr routes a single name/value pair to the class list, styles, or
// regular attributes, mirroring the cb in gadx.attrs.
func (t *Tag) classifyAttr(na *gad.KeyValue) {
	var k string
	switch kt := na.K.(type) {
	case gad.Bool:
		// empty name (matches gadx.attrs)
	case gad.Str:
		k = string(kt)
	case gad.RawStr:
		k = string(kt)
	default:
		k = na.K.ToString()
	}
	switch k {
	case "class":
		t.addClass(na.V)
	case "style":
		t.addStyle(na.V)
	default:
		v := na.V
		if kv, ok := v.(*gad.KeyValue); ok {
			if kv.K.IsFalsy() {
				return
			}
			v = kv.V
		}
		t.setAttr(k, v)
	}
}

// setAttr upserts a regular attribute, tracking first-seen order.
func (t *Tag) setAttr(name string, value gad.Object) {
	if t.Attrs == nil {
		t.Attrs = gad.Dict{}
	}
	if _, ok := t.Attrs[name]; !ok {
		t.attrOrder = append(t.attrOrder, name)
	}
	t.Attrs[name] = value
}

// addClass appends class token(s) from value, honoring the same shapes as
// gadx.attrs (string, raw string, keyed value, or array of any of these).
func (t *Tag) addClass(value gad.Object) {
	if value.IsFalsy() {
		return
	}
	switch v := value.(type) {
	case gad.Str:
		if len(v) > 0 {
			t.ClassList = append(t.ClassList, string(v))
		}
	case gad.RawStr:
		if len(v) > 0 {
			t.ClassList = append(t.ClassList, string(v))
		}
	case *gad.KeyValue:
		if !v.K.IsFalsy() {
			for _, o := range attrFilter(asArray(v.V)) {
				t.ClassList = append(t.ClassList, o.ToString())
			}
		}
	case gad.Array:
		for _, o := range attrFilter(v) {
			t.ClassList = append(t.ClassList, o.ToString())
		}
	}
}

// addStyle appends style declaration(s) from value, honoring the same shapes as
// gadx.attrs (string, raw string, keyed value, array, or dict of name:value).
func (t *Tag) addStyle(value gad.Object) {
	switch v := value.(type) {
	case gad.Str:
		if len(v) > 0 {
			t.Styles = append(t.Styles, string(v))
		}
	case gad.RawStr:
		if len(v) > 0 {
			t.Styles = append(t.Styles, string(v))
		}
	case *gad.KeyValue:
		if !v.K.IsFalsy() {
			for _, o := range attrFilter(asArray(v.V)) {
				t.Styles = append(t.Styles, o.ToString())
			}
		}
	case gad.Array:
		for _, o := range attrFilter(v) {
			t.Styles = append(t.Styles, o.ToString())
		}
	case gad.Dict:
		for key, o := range v {
			t.Styles = append(t.Styles, key+":"+o.ToString())
		}
	}
}

// attrsKeyValueArray serialises the structured attributes back into a
// KeyValueArray, so `tag.attrs` reads and `tag.attrs += kva` round-trip.
func (t *Tag) attrsKeyValueArray() gad.KeyValueArray {
	arr := make(gad.KeyValueArray, 0, len(t.attrOrder)+2)
	for _, name := range t.attrOrder {
		arr = append(arr, &gad.KeyValue{K: gad.Str(name), V: t.Attrs[name]})
	}
	if len(t.ClassList) > 0 {
		arr = append(arr, &gad.KeyValue{K: gad.Str("class"), V: gad.Str(strings.Join(t.ClassList, " "))})
	}
	if len(t.Styles) > 0 {
		arr = append(arr, &gad.KeyValue{K: gad.Str("style"), V: gad.Str(strings.Join(t.Styles, "; "))})
	}
	return arr
}

// writeAttrs renders the tag's attributes directly into w: each regular
// attribute via AttrFunc (the gadx.attr formatter), then the joined class list
// and styles. This mirrors gadx.attrs' output without invoking the builtin.
func (t *Tag) writeAttrs(vm *gad.VM, w io.Writer, wc *writeCounter) error {
	for _, name := range t.attrOrder {
		rs, err := AttrFunc(vm, gad.Str(name), t.Attrs[name])
		if err != nil {
			return err
		}
		if rs != "" {
			wc.writeString(w, " "+string(rs))
		}
	}
	if len(t.ClassList) > 0 {
		wc.writeString(w, " class="+strconv.Quote(strings.Join(t.ClassList, " ")))
	}
	if len(t.Styles) > 0 {
		wc.writeString(w, " style="+strconv.Quote(strings.Join(t.Styles, "; ")))
	}
	return wc.err
}

// ElementData returns a gad.Object tree describing el for structured encoding
// (e.g. JSON/YAML) instead of HTML rendering: a *Tag becomes a Dict
// {tag, attrs, children} (children is an Array of each child's data), and a Text
// node becomes the Str of its concatenated values. An anonymous tag keeps an
// empty "tag". Attribute order follows the tag's attrsKeyValueArray.
func ElementData(el Element) gad.Object {
	switch e := el.(type) {
	case *Tag:
		children := make(gad.Array, 0, len(e.Children))
		for _, c := range e.Children {
			children = append(children, ElementData(c))
		}
		attrs := gad.Dict{}
		for _, kv := range e.attrsKeyValueArray() {
			attrs[kv.K.ToString()] = kv.V
		}
		return gad.Dict{"tag": gad.Str(e.Name), "attrs": attrs, "children": children}
	case Text:
		var b strings.Builder
		for _, v := range e {
			b.WriteString(v.ToString())
		}
		return gad.Str(b.String())
	default:
		if o, ok := el.(gad.Object); ok {
			return gad.Str(o.ToString())
		}
		return gad.Nil
	}
}

// EncodeElement serializes el's structured data (see ElementData) to JSON or
// YAML (format is case-insensitive "json"/"yaml"/"yml"), for the tag-encode
// output mode that replaces HTML rendering. JSON is indented and newline-
// terminated.
func EncodeElement(el Element, format string) ([]byte, error) {
	data := gad.ToInterface(ElementData(el))
	switch strings.ToLower(format) {
	case "yaml", "yml":
		out, err := yaml.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("encode tag as YAML: %w", err)
		}
		return out, nil
	case "json":
		out, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode tag as JSON: %w", err)
		}
		return append(out, '\n'), nil
	default:
		return nil, fmt.Errorf("unknown tag encode format %q (want json or yaml)", format)
	}
}

// asArray returns o as an Array, wrapping a non-array value as a single element.
func asArray(o gad.Object) gad.Array {
	if arr, ok := o.(gad.Array); ok {
		return arr
	}
	return gad.Array{o}
}

// attrFilter flattens an Array for class/style collection: a KeyValue entry
// contributes its value only when its key is truthy, nested arrays recurse, and
// any other truthy value passes through. Mirrors the filter in gadx.attrs.
func attrFilter(arr gad.Array) (ret gad.Array) {
	for _, v := range arr {
		switch t := v.(type) {
		case *gad.KeyValue:
			if !t.K.IsFalsy() {
				ret = append(ret, t.V)
			}
		case gad.Array:
			ret = append(ret, attrFilter(t)...)
		default:
			if !t.IsFalsy() {
				ret = append(ret, t)
			}
		}
	}
	return
}

// writeCounter accumulates bytes written and the first error, so WriteTo can
// chain several writes without repeating error handling on each.
type writeCounter struct {
	n   int64
	err error
}

func (c *writeCounter) writeString(w io.Writer, s string) {
	if c.err != nil {
		return
	}
	i, err := io.WriteString(w, s)
	c.n += int64(i)
	c.err = err
}

var (
	_ Element         = (*Tag)(nil)
	_ Element         = Text(nil)
	_ gad.ToWriter    = (*Tag)(nil)
	_ gad.ToWriter    = Text(nil)
	_ gad.ObjectEnter = (*Tag)(nil)
	_ gad.ObjectExit  = (*Tag)(nil)
)
