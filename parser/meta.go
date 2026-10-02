package parser

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// Metadata: a `[k=v, …]` block declared between a doc comment and a
// doc-commentable element (a declaration or a member). It is parsed as the
// named entries of a KeyValueArray (the `[ … ]` bracket, no leading `;`) and
// attached to the element's Meta field; at run time it is reachable as
// `Element.@meta`.

// ParseMetadata parses the entries of a metadata block — `k=v, …` — without
// the brackets around them: what a block holds when it is written elsewhere,
// as in a Go struct tag (`i18n:"type=html, label='Tab'"`). Single quotes
// delimit strings as double quotes do, so the text needs no escaping there.
//
// The positions of the nodes are those of the text from base: the position of
// its first byte in whatever holds it, so an error or a node points into that
// (base < 1 is 1: the text on its own, its first byte at offset 0).
func ParseMetadata(src string, base source.Pos) (kva *node.KeyValueArrayLit, err error) {
	if base < 1 {
		base = 1
	}
	fileSet := source.NewFileSet()
	fileSet.Base = int(base)
	file := fileSet.AddFileData("metadata", int(base), []byte(src))
	p := NewParserWithOptions(file, &ParserOptions{Mode: ParseCharAsString},
		&ScannerOptions{Mode: ScanCharAsString})
	defer func() {
		if e := recover(); e != nil {
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}
		p.Errors.Sort()
		if err = p.Errors.Err(); err != nil {
			kva = nil
		}
	}()
	if p.Errors.Len() > 0 {
		return nil, p.Errors.Err()
	}
	p.SkipSpace()
	kva = p.ParseKeyValueArrayLitAt(base, token.EOF)
	p.Expect(token.EOF)
	return kva, nil
}

// parseMetaBlock parses `[ k=v, … ]` (current token is `[`) into a
// KeyValueArrayLit, reusing the key-value-array entry grammar.
func (p *Parser) parseMetaBlock() *node.KeyValueArrayLit {
	lb := p.Expect(token.LBrack)
	kva := p.ParseKeyValueArrayLitAt(lb, token.RBrack)
	p.Expect(token.RBrack)
	return kva
}

// takeMeta returns and clears the pending metadata block (set by
// parseStmtMeta / parseMemberMeta before the element is parsed).
func (p *Parser) takeMeta() *node.KeyValueArrayLit {
	m := p.pendingMeta
	p.pendingMeta = nil
	return m
}

// parseMemberMeta parses an optional leading `[ … ]` metadata block inside a
// declaration BODY (interface/class/enum), where `[` can only be metadata (never
// an array). The element's doc comment is preserved across the block so the
// member still picks it up.
func (p *Parser) parseMemberMeta() {
	if p.Token.Token != token.LBrack {
		return
	}
	doc := p.leadComment
	p.pendingMeta = p.parseMetaBlock()
	// Skip the newline/`;` (and space) between the block and the member.
	for p.Token.Is(token.Semicolon) || p.Token.IsSpace() {
		p.Next()
	}
	p.leadComment = doc
}

// parseStmtMeta parses an optional leading `[ … ]` metadata block at STATEMENT
// position, where `[` is otherwise an array literal: it is metadata only when the
// matching `]` is followed by a doc-commentable declaration (see looksLikeMeta).
func (p *Parser) parseStmtMeta() {
	if p.Token.Token != token.LBrack || !p.looksLikeMeta() {
		return
	}
	doc := p.leadComment
	p.pendingMeta = p.parseMetaBlock()
	// Skip the newline/`;` (and space) between the block and the declaration.
	for p.Token.Is(token.Semicolon) || p.Token.IsSpace() {
		p.Next()
	}
	p.leadComment = doc
}

// looksLikeMeta reports whether the `[ … ]` starting at the current token is a
// metadata block: its matching `]` is followed by a token that begins a
// doc-commentable declaration.
func (p *Parser) looksLikeMeta() bool {
	if p.Token.Token != token.LBrack {
		return false
	}
	depth := 1
	result := false
	var rbrackPos source.Pos
	p.PeekCb(func(t PToken) bool {
		if depth > 0 {
			switch t.Token {
			case token.LBrack:
				depth++
			case token.RBrack:
				depth--
				if depth == 0 {
					rbrackPos = t.Pos
				}
			case token.EOF:
				return false
			}
			return true
		}
		// depth == 0: the declaration must be ADJACENT to the block — on the same
		// line as the `]`, or the immediately following line (the peek stream skips
		// comments, so a `[…]` result-expression statement followed some lines later
		// by a declaration is correctly rejected as an ordinary array, not
		// metadata). Only spaces and the newline's `;` may come between.
		if t.IsSpace() || t.Token == token.Semicolon {
			return true
		}
		if metaTargetStart(t) {
			declLine := source.MustFileLine(p.File, t.Pos)
			rbrackLine := source.MustFileLine(p.File, rbrackPos)
			result = declLine <= rbrackLine+1
		}
		return false
	})
	return result
}

// metaTargetStart reports whether t begins a doc-commentable declaration that
// metadata may precede: `func`, `enum`, `prop`, a `get`/`set`/`prop` accessor,
// or a contextual declaration keyword (`interface`/`class`/`mixin`/`type`),
// optionally after `export`.
func metaTargetStart(t PToken) bool {
	switch t.Token {
	case token.Func, token.Enum, token.Prop, token.Export:
		return true
	case token.Ident:
		switch t.Literal {
		case "interface", "class", "mixin", "type":
			return true
		}
	}
	return false
}
