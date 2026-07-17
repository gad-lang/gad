package parser

import (
	"bufio"
	"container/list"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	gadparser "github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"

	giomtoken "github.com/gad-lang/gad/giom/token"
)

// =============================================================================
// scanner — implements gad's ScannerInterface for giom template syntax
// =============================================================================

type scanner struct {
	file        *source.File
	reader      *bufio.Reader
	indentStack *list.List
	stash       *list.List

	state  int32
	buffer string

	offset        int // current byte offset within file
	line          int // current line number (0-based)
	col           int // current column
	lastTokenPos  source.Pos
	lastTokenSize int

	readRaw        bool
	mode           gadparser.ScanMode
	mixedDelimiter gadparser.MixedDelimiter
	errorHandler   []source.ScannerErrorHandler
}

// newScanner creates a scanner that reads from r using file for position tracking.
func newScanner(file *source.File, r io.Reader) *scanner {
	s := &scanner{
		file:        file,
		reader:      bufio.NewReader(r),
		indentStack: list.New(),
		stash:       list.New(),
		state:       giomtoken.ScnNewLine,
		line:        -1,
		col:         0,
		mixedDelimiter: gadparser.MixedDelimiter{
			Start: []rune("{"),
			End:   []rune("}"),
		},
	}
	registerLines(file)
	return s
}

// registerLines populates file.Lines with the offset of the first character of
// each line. The giom scanner reads the source through its own bufio.Reader and
// never advances gad's source.Reader, which is what normally calls File.AddLine.
// Without this, File.Lines stays [0] and every token position resolves to line
// 1 (column = byte offset), corrupting error traces and node positions. This
// mirrors the newline scan in source.Data.check.
func registerLines(file *source.File) {
	if file == nil || file.Data == nil {
		return
	}
	for i, c := range file.Data.Bytes() {
		if c == '\n' {
			file.AddLine(i + 1)
		}
	}
}

// =============================================================================
// ScannerInterface implementation
// =============================================================================

func (s *scanner) Scan() (t gadparser.PToken) {
	if s.readRaw {
		s.readRaw = false
		return s.NextRaw()
	}

	s.ensureBuffer()

	if stashed := s.stash.Front(); stashed != nil {
		tok := stashed.Value.(gadparser.PToken)
		s.stash.Remove(stashed)
		return tok
	}

	switch s.state {
	case giomtoken.ScnEOF:
		if outdent := s.indentStack.Back(); outdent != nil {
			s.indentStack.Remove(outdent)
			return s.newToken(giomtoken.Outdent, "", "")
		}
		return s.newToken(giomtoken.EOF, "", "")

	case giomtoken.ScnNewLine:
		s.state = giomtoken.ScnLine
		if tok := s.scanIndent(); tok.Valid() {
			return tok
		}
		return s.Scan()

	case giomtoken.ScnLine:
		if tok := s.scanExport(); tok.Valid() {
			return tok
		}
		if tok := s.scanGlobal(); tok.Valid() {
			return tok
		}
		if tok := s.scanVar(); tok.Valid() {
			return tok
		}
		if tok := s.scanConst(); tok.Valid() {
			return tok
		}
		if tok := s.scanEnum(); tok.Valid() {
			return tok
		}
		if tok := s.scanFunc(); tok.Valid() {
			return tok
		}
		if tok := s.scanComp(); tok.Valid() {
			return tok
		}
		if tok := s.scanCompCall(); tok.Valid() {
			return tok
		}
		if tok := s.scanMatch(); tok.Valid() {
			return tok
		}
		if tok := s.scanCase(); tok.Valid() {
			return tok
		}
		if tok := s.scanDoctype(); tok.Valid() {
			return tok
		}
		if tok := s.scanCondition(); tok.Valid() {
			return tok
		}
		if tok := s.scanFor(); tok.Valid() {
			return tok
		}
		if tok := s.scanImportModule(); tok.Valid() {
			return tok
		}
		if tok := s.scanSlot(); tok.Valid() {
			return tok
		}
		if tok := s.scanSlotPass(); tok.Valid() {
			return tok
		}
		if tok := s.scanAssignment(); tok.Valid() {
			return tok
		}
		if tok := s.scanCode(); tok.Valid() {
			return tok
		}
		if tok := s.scanMCode(); tok.Valid() {
			return tok
		}
		if tok := s.scanHtml(); tok.Valid() {
			return tok
		}
		if tok := s.scanTag(); tok.Valid() {
			return tok
		}
		if tok := s.scanId(); tok.Valid() {
			return tok
		}
		if tok := s.scanClassName(); tok.Valid() {
			return tok
		}
		if tok := s.scanAttribute(); tok.Valid() {
			return tok
		}
		if tok := s.scanComment(); tok.Valid() {
			return tok
		}
		if tok := s.scanText(); tok.Valid() {
			return tok
		}
	}

	return s.newToken(token.Illegal, "", "")
}

func (s *scanner) Mode() gadparser.ScanMode     { return s.mode }
func (s *scanner) SetMode(m gadparser.ScanMode) { s.mode = m }
func (s *scanner) SourceFile() *source.File     { return s.file }
func (s *scanner) Source() []byte               { return s.file.Data.Bytes() }

func (s *scanner) ErrorHandler(h ...source.ScannerErrorHandler) {
	s.errorHandler = append(s.errorHandler, h...)
}

func (s *scanner) GetMixedDelimiter() *gadparser.MixedDelimiter {
	return &s.mixedDelimiter
}

// =============================================================================
// Token construction
// =============================================================================

func (s *scanner) newToken(kind token.Token, literal, value string) gadparser.PToken {
	pt := gadparser.PToken{
		TokenLit: node.TokenLit{
			Pos:     s.lastTokenPos,
			Token:   kind,
			Literal: literal,
		},
	}
	if value != "" {
		pt.Set("value", value)
	}
	return pt
}

func (s *scanner) newTokenWithData(kind token.Token, literal string, data map[string]string) gadparser.PToken {
	pt := gadparser.PToken{
		TokenLit: node.TokenLit{
			Pos:     s.lastTokenPos,
			Token:   kind,
			Literal: literal,
		},
	}
	for k, v := range data {
		pt.Set(k, v)
	}
	return pt
}

// =============================================================================
// Indentation scanning
// =============================================================================

var rgxIndent = regexp.MustCompile(`^(\s+)`)

func (s *scanner) scanIndent() gadparser.PToken {
	if len(s.buffer) == 0 {
		s.consume(0)
		return s.newToken(giomtoken.Blank, "", "")
	}

	var head *list.Element
	for head = s.indentStack.Front(); head != nil; head = head.Next() {
		value := head.Value.(*regexp.Regexp)
		if match := value.FindString(s.buffer); len(match) != 0 {
			s.consume(len(match))
		} else {
			break
		}
	}

	newIndent := rgxIndent.FindString(s.buffer)

	if len(newIndent) != 0 && head == nil {
		s.indentStack.PushBack(regexp.MustCompile(regexp.QuoteMeta(newIndent)))
		s.consume(len(newIndent))
		return s.newToken(giomtoken.Indent, newIndent, newIndent)
	}

	if len(newIndent) == 0 && head != nil {
		for head != nil {
			next := head.Next()
			s.indentStack.Remove(head)
			if next == nil {
				return s.newToken(giomtoken.Outdent, "", "")
			} else {
				t := s.newToken(giomtoken.Outdent, "", "")
				s.stash.PushBack(t)
			}
			head = next
		}
	}

	if len(newIndent) != 0 && head != nil {
		panic("Mismatching indentation. Please use a coherent indent schema.")
	}

	return gadparser.PToken{}
}

func (s *scanner) Indentation() string {
	var b strings.Builder
	for e := s.indentStack.Front(); e != nil; e = e.Next() {
		b.WriteString(e.Value.(*regexp.Regexp).String())
	}
	return b.String()
}

// =============================================================================
// Scan methods — regex-based line matching
// =============================================================================

var rgxDoctype = regexp.MustCompile(`^(!!!|@doctype)\s*(.*)`)

func (s *scanner) scanDoctype() gadparser.PToken {
	if sm := rgxDoctype.FindStringSubmatch(s.buffer); len(sm) != 0 {
		val := sm[2]
		if val == "" {
			val = "html"
		}
		s.consume(len(sm[0]))
		return s.newToken(giomtoken.Doctype, sm[0], val)
	}
	return gadparser.PToken{}
}

var rgxIf = regexp.MustCompile(`^@if\s+(.+)$`)
var rgxElse = regexp.MustCompile(`^@else(\s*|\s+if\s+(.+))$`)

func (s *scanner) scanCondition() gadparser.PToken {
	if sm := rgxIf.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return s.newToken(giomtoken.If, sm[0], sm[1])
	}
	if sm := rgxElse.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		if strings.Contains(strings.TrimSpace(sm[0][4:]), "if") {
			return s.newToken(giomtoken.ElseIf, sm[0], sm[2])
		}
		return s.newToken(giomtoken.Else, sm[0], "")
	}
	return gadparser.PToken{}
}

var rgxFor = regexp.MustCompile(`^@for\s+(.+)$`)

func (s *scanner) scanFor() gadparser.PToken {
	if sm := rgxFor.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return s.newToken(giomtoken.For, sm[0], strings.TrimSpace(sm[1]))
	}
	return gadparser.PToken{}
}

var rgxAssignment = regexp.MustCompile(`^(\$[\w0-9\-_]*)?\s*([+-/*:]?)=\s*(.+)$`)

func (s *scanner) scanAssignment() gadparser.PToken {
	if sm := rgxAssignment.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		pt := s.newToken(giomtoken.Assignment, sm[0], sm[3])
		pt.Set("x", sm[1])
		pt.Set("op", sm[2])
		return pt
	}
	return gadparser.PToken{}
}

var rgxCode = regexp.MustCompile(`^\s*~\s+(.+)$`)

func (s *scanner) scanCode() gadparser.PToken {
	if sm := rgxCode.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		pt := s.newToken(giomtoken.Code, sm[0], "")
		pt.Set("values", []string{sm[1]})
		// Absolute position of the code content (sm[1]) so parseCode can map
		// the parsed statement back onto the original source line/column.
		pt.Set("valuePos", []source.Pos{pt.Pos + source.Pos(len(sm[0])-len(sm[1]))})
		return pt
	}
	return gadparser.PToken{}
}

var rgxMCode = regexp.MustCompile(`^\s*~~\s*$`)

func (s *scanner) scanMCode() gadparser.PToken {
	if sm := rgxMCode.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		code, positions := s.NextRawCode("~~")
		pt := s.newToken(giomtoken.Code, "", "")
		pt.Set("values", code)
		pt.Set("valuePos", positions)
		return pt
	}
	return gadparser.PToken{}
}

var rgxComment = regexp.MustCompile(`^\/\/(-)?\s*(.*)$`)

func (s *scanner) scanComment() gadparser.PToken {
	if sm := rgxComment.FindStringSubmatch(s.buffer); len(sm) != 0 {
		mode := "embed"
		if len(sm[1]) != 0 {
			mode = "silent"
		}
		s.consume(len(sm[0]))
		pt := s.newToken(giomtoken.Comment, sm[0], sm[2])
		pt.Set("mode", mode)
		return pt
	}
	return gadparser.PToken{}
}

var rgxId = regexp.MustCompile(`^#([\w-]+)(?:\s*\?\s*(.*)$)?`)

func (s *scanner) scanId() gadparser.PToken {
	if sm := rgxId.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		pt := s.newToken(giomtoken.Id, sm[0], sm[1])
		pt.Set("condition", sm[2])
		return pt
	}
	return gadparser.PToken{}
}

var rgxClassName = regexp.MustCompile(`^\.([\w-]+)(?:\s*\?\s*(.*)$)?`)

func (s *scanner) scanClassName() gadparser.PToken {
	if sm := rgxClassName.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		pt := s.newToken(giomtoken.ClassName, sm[0], sm[1])
		pt.Set("condition", sm[2])
		return pt
	}
	return gadparser.PToken{}
}

// scanAttribute scans an attribute group `[ … ]`. A group may hold one or many
// attributes separated by commas or newlines, like a GAD KeyValueArray
// `(; … )`, and may span multiple physical lines up to the closing `]`:
//
//	div[class="a"]
//	div[class="a", title="hello"]
//	div[
//	    class="a"
//	    class="b"
//	    title="hello"
//	]
//
// The raw inner text (between the brackets) is preserved verbatim together with
// its absolute base position; the parser splits it into individual attributes.
func (s *scanner) scanAttribute() gadparser.PToken {
	if !strings.HasPrefix(s.buffer, "[") {
		return gadparser.PToken{}
	}
	// Pull continuation lines until the group is balanced-closed.
	s.ensureBracketClosed()
	group, end, ok := s.readBalanced(0, '[', ']')
	if !ok {
		return gadparser.PToken{}
	}
	inner := group[1 : len(group)-1]

	// Optional trailing `? condition` on the same line as the closing `]`.
	condition := ""
	rest := s.buffer[end:]
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		rest = rest[:nl]
	}
	consumed := end
	if strings.HasPrefix(rest, " ?") {
		condition = strings.TrimSpace(rest[2:])
		consumed = end + len(rest)
	}

	// Base position of inner (the byte right after the opening `[`).
	innerPos := source.Pos(s.file.Base+s.offset-len(s.buffer)-1) + 1
	lit := s.buffer[:consumed]
	s.consume(consumed)
	pt := s.newToken(giomtoken.Attribute, lit, "")
	pt.Set("inner", inner)
	pt.Set("innerPos", innerPos)
	pt.Set("condition", condition)
	return pt
}

// ensureBracketClosed appends subsequent physical lines to the buffer until the
// bracket group starting at s.buffer[0] is balanced-closed, or input ends.
func (s *scanner) ensureBracketClosed() { s.ensureBalanced(0, '[', ']') }

// ensureBalanced appends subsequent physical lines to the buffer until the group
// opened at s.buffer[start] is balanced-closed, or input ends. The separating
// newline is preserved so buffer offsets stay aligned with file offsets
// (verbatim), keeping value positions accurate across lines.
func (s *scanner) ensureBalanced(start int, open, close byte) {
	for {
		if _, _, ok := s.readBalanced(start, open, close); ok {
			return
		}
		buf, err := s.reader.ReadString('\n')
		if len(buf) == 0 {
			return
		}
		s.offset += len(buf)
		if buf[len(buf)-1] == '\n' {
			buf = buf[:len(buf)-1]
		}
		s.buffer += "\n" + buf
		if err != nil {
			return
		}
	}
}

var rgxImportModule = regexp.MustCompile(`^@import\s+("[0-9a-zA-Z_\-\. \/][0-9a-zA-Z_\-\. \/]*")(\s+as\s+([a-zA-Z$_]\w*))?$`)
var rgxImportDestructure = regexp.MustCompile(`^@import\s*\{([^}]*)\}\s+from\s+("[0-9a-zA-Z_\-\. \/][0-9a-zA-Z_\-\. \/]*")$`)

func (s *scanner) scanImportModule() gadparser.PToken {
	if strings.HasPrefix(s.buffer, "@import") {
		if sm := rgxImportDestructure.FindStringSubmatch(s.buffer); len(sm) != 0 {
			s.consume(len(sm[0]))
			pt := s.newToken(giomtoken.ImportModule, sm[0], sm[2])
			pt.Set("destructure", strings.TrimSpace(sm[1]))
			return pt
		}
		if sm := rgxImportModule.FindStringSubmatch(s.buffer); len(sm) != 0 {
			s.consume(len(sm[0]))
			pt := s.newToken(giomtoken.ImportModule, sm[0], sm[1])
			pt.Set("ident", sm[3])
			return pt
		}
	}
	return gadparser.PToken{}
}

var rgxSlot = regexp.MustCompile(`^@slot\s+([a-zA-Z_-]+\w*)(\((.*)\))?$`)

func (s *scanner) readBalanced(start int, open, close byte) (string, int, bool) {
	if start >= len(s.buffer) || s.buffer[start] != open {
		return "", start, false
	}
	depth := 0
	inString := byte(0)
	escaped := false
	for i := start; i < len(s.buffer); i++ {
		c := s.buffer[i]
		if inString != 0 {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == inString {
				inString = 0
			}
			continue
		}
		switch c {
		case '\'', '"', '`':
			inString = c
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return s.buffer[start : i+1], i + 1, true
			}
		}
	}
	return "", start, false
}

func (s *scanner) scanSlot() gadparser.PToken {
	if strings.TrimSpace(s.buffer) == "@wrap" {
		line := s.buffer
		s.consume(len(line))
		return s.newToken(giomtoken.Wrap, line, "")
	}
	if !strings.HasPrefix(s.buffer, "@slot ") || strings.HasPrefix(s.buffer, "@slot #") {
		return gadparser.PToken{}
	}
	line := s.buffer
	base0 := source.Pos(s.file.Base + s.offset - len(s.buffer) - 1)
	i := len("@slot ")

	var (
		name      string
		nameExpr  bool
		namePos   source.Pos
		afterName int
	)
	if i < len(line) && line[i] == '(' {
		// Parenthesized, interpolated name: `@slot (line[{index}])`. The content
		// is a Gad template string; store it verbatim with its absolute position.
		balanced, end, ok := s.readBalanced(i, '(', ')')
		if !ok {
			return gadparser.PToken{}
		}
		name = balanced[1 : len(balanced)-1]
		nameExpr = true
		namePos = base0 + source.Pos(i+1)
		afterName = end
	} else {
		j := i
		for j < len(line) {
			c := line[j]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
				j++
				continue
			}
			break
		}
		if j == i {
			return gadparser.PToken{}
		}
		name = line[i:j]
		afterName = j
	}

	rest := strings.TrimSpace(line[afterName:])
	args := ""
	consumed := afterName
	if rest != "" {
		if afterName >= len(line) || line[afterName] != '(' {
			return gadparser.PToken{}
		}
		balanced, end, ok := s.readBalanced(afterName, '(', ')')
		if !ok {
			return gadparser.PToken{}
		}
		args = balanced[1 : len(balanced)-1]
		if strings.TrimSpace(line[end:]) != "" {
			return gadparser.PToken{}
		}
		consumed = len(line)
	}
	lit := line[:consumed]
	s.consume(consumed)
	pt := s.newToken(giomtoken.Slot, lit, name)
	pt.Set("args", args)
	if nameExpr {
		pt.Set("nameExpr", true)
		pt.Set("namePos", namePos)
	}
	return pt
}

var rgxSlotPass = regexp.MustCompile(`^@slot\s+#(.+)$`)

func (s *scanner) scanSlotPass() gadparser.PToken {
	const prefix = "@slot #"
	if strings.HasPrefix(s.buffer, prefix) && len(s.buffer) > len(prefix) && s.buffer[len(prefix)] == '(' {
		// Parenthesized, interpolated name: `@slot #(line[{index}])(args)`. The
		// content is a Gad template string; store it verbatim with its absolute
		// position, followed by an optional `(args)` group.
		line := s.buffer
		base0 := source.Pos(s.file.Base + s.offset - len(s.buffer) - 1)
		i := len(prefix)
		balanced, end, ok := s.readBalanced(i, '(', ')')
		if !ok {
			return gadparser.PToken{}
		}
		name := balanced[1 : len(balanced)-1]
		namePos := base0 + source.Pos(i+1)

		args := ""
		if rest := strings.TrimSpace(line[end:]); rest != "" {
			if line[end] != '(' {
				return gadparser.PToken{}
			}
			b2, end2, ok := s.readBalanced(end, '(', ')')
			if !ok || strings.TrimSpace(line[end2:]) != "" {
				return gadparser.PToken{}
			}
			args = b2[1 : len(b2)-1]
		}
		s.consume(len(line))
		pt := s.newToken(giomtoken.SlotPass, line, name)
		pt.Set("name", name)
		pt.Set("args", args)
		pt.Set("nameExpr", true)
		pt.Set("namePos", namePos)
		return pt
	}
	if sm := rgxSlotPass.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		pt := s.newToken(giomtoken.SlotPass, sm[0], sm[1])
		pt.Set("header", sm[1])
		return pt
	}
	return gadparser.PToken{}
}

// scanHtml scans a self-contained raw HTML region beginning with `<` — an
// opening tag `<name …>` or a `<>` fragment. The region runs to its matching
// close tag (spanning multiple lines if needed) and is stored verbatim as the
// token value with its absolute base position; the parser turns it into write
// calls, collapsing whitespace and evaluating `{ … }` interpolations.
func (s *scanner) scanHtml() gadparser.PToken {
	if len(s.buffer) < 2 || s.buffer[0] != '<' {
		return gadparser.PToken{}
	}
	c := s.buffer[1]
	isLetter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
	if !(c == '>' || isLetter) {
		return gadparser.PToken{}
	}
	base0 := source.Pos(s.file.Base + s.offset - len(s.buffer) - 1)
	s.ensureHtmlComplete(0)
	end, ok := htmlRegionEnd(s.buffer, 0)
	if !ok {
		return gadparser.PToken{}
	}
	raw := s.buffer[:end]
	s.consume(end)
	pt := s.newToken(giomtoken.Html, raw, raw)
	pt.Set("htmlPos", base0)
	return pt
}

// ensureHtmlComplete pulls additional source lines into the buffer until the
// HTML region opened at s.buffer[start] closes, or input ends. The separating
// newline is preserved so buffer offsets stay aligned with file offsets.
func (s *scanner) ensureHtmlComplete(start int) {
	for {
		if _, ok := htmlRegionEnd(s.buffer, start); ok {
			return
		}
		buf, err := s.reader.ReadString('\n')
		if len(buf) == 0 {
			return
		}
		s.offset += len(buf)
		if buf[len(buf)-1] == '\n' {
			buf = buf[:len(buf)-1]
		}
		s.buffer += "\n" + buf
		if err != nil {
			return
		}
	}
}

var rgxTag = regexp.MustCompile(`^(\w[-:/\w]*)`)

func (s *scanner) scanTag() gadparser.PToken {
	if sm := rgxTag.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return s.newToken(giomtoken.Tag, sm[0], sm[1])
	}
	return gadparser.PToken{}
}

var rgxExport = regexp.MustCompile(`^@export\s+([a-zA-Z_]\w*)(\s*=\s*(.+))?$`)

func (s *scanner) scanExport() gadparser.PToken {
	if sm := rgxExport.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		pt := s.newToken(giomtoken.Export, sm[0], sm[1])
		pt.Set("name", sm[1])
		pt.Set("value", sm[3])
		return pt
	}
	return gadparser.PToken{}
}

func (s *scanner) scanGlobal() gadparser.PToken {
	return s.scanDeclDirective("@global", giomtoken.Global)
}

func (s *scanner) scanVar() gadparser.PToken {
	return s.scanDeclDirective("@var", giomtoken.Var)
}

var rgxEnumHead = regexp.MustCompile(`^@enum\s+([a-zA-Z_]\w*)\s*\(`)

// scanEnum scans `@enum IDENT ( … )`. The parenthesized body holds the enum
// fields, whose syntax mirrors a `@var` declaration (comma- or newline-separated
// `Name` / `Name = value`, and the Gad enum extras `bit`, `+`/`-`). The body may
// span multiple lines up to the balanced `)`. The field text is stored verbatim
// as the token value (with its absolute base position) alongside the enum name;
// the parser rewrites it into a Gad `enum IDENT { … }` statement.
func (s *scanner) scanEnum() gadparser.PToken {
	m := rgxEnumHead.FindStringSubmatch(s.buffer)
	if m == nil {
		return gadparser.PToken{}
	}
	name := m[1]
	start := len(m[0]) - 1 // index of the opening '('

	base0 := source.Pos(s.file.Base + s.offset - len(s.buffer) - 1)

	s.ensureBalanced(start, '(', ')')
	balanced, end, ok := s.readBalanced(start, '(', ')')
	if !ok || strings.TrimSpace(s.buffer[end:]) != "" {
		return gadparser.PToken{}
	}
	inner := balanced[1 : len(balanced)-1]
	innerStart := start + 1
	lead := len(inner) - len(strings.TrimLeft(inner, " \t\r\n"))

	lit := s.buffer[:end]
	s.consume(end)
	pt := s.newToken(giomtoken.Enum, lit, strings.TrimSpace(inner))
	pt.Set("name", name)
	pt.Set("innerPos", base0+source.Pos(innerStart+lead))
	return pt
}

func (s *scanner) scanConst() gadparser.PToken {
	return s.scanDeclDirective("@const", giomtoken.Const)
}

// scanDeclDirective scans `@var`/`@const` declarations in either form:
//
//	@var a                 // bare, single
//	@var a, b, c = 1       // bare, comma-separated (single line)
//	@var (a               // parenthesized, may span lines up to `)`
//	    b, c = 2)
//
// The declaration text (without the surrounding parentheses, if any) is stored
// verbatim as the token value together with its absolute base position; the
// parser wraps it in a Gad grouped declaration.
func (s *scanner) scanDeclDirective(prefix string, tk token.Token) gadparser.PToken {
	line := s.buffer
	if !strings.HasPrefix(line, prefix) || len(line) <= len(prefix) || line[len(prefix)] != ' ' {
		return gadparser.PToken{}
	}
	start := len(prefix)
	for start < len(s.buffer) && s.buffer[start] == ' ' {
		start++
	}
	if start >= len(s.buffer) {
		return gadparser.PToken{}
	}

	base0 := source.Pos(s.file.Base + s.offset - len(s.buffer) - 1)

	var (
		inner      string
		innerStart int
		consumed   int
	)
	if s.buffer[start] == '(' {
		s.ensureBalanced(start, '(', ')')
		balanced, end, ok := s.readBalanced(start, '(', ')')
		if !ok || strings.TrimSpace(s.buffer[end:]) != "" {
			return gadparser.PToken{}
		}
		inner = balanced[1 : len(balanced)-1]
		innerStart = start + 1
		consumed = end
	} else {
		inner = s.buffer[start:]
		innerStart = start
		consumed = len(s.buffer)
	}

	lead := len(inner) - len(strings.TrimLeft(inner, " \t\r\n"))
	lit := s.buffer[:consumed]
	s.consume(consumed)
	pt := s.newToken(tk, lit, strings.TrimSpace(inner))
	pt.Set("innerPos", base0+source.Pos(innerStart+lead))
	return pt
}

var rgxFunc = regexp.MustCompile(`^@(export\s+)?func ([a-zA-Z_-]+\w*)(\((.*)\))?$`)

func (s *scanner) scanFunc() gadparser.PToken {
	line := s.buffer
	exported := false
	prefix := "@func "
	if strings.HasPrefix(line, "@export func ") {
		exported = true
		prefix = "@export func "
	} else if !strings.HasPrefix(line, prefix) {
		return gadparser.PToken{}
	}

	i := len(prefix)
	j := i
	for j < len(line) {
		c := line[j]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			j++
			continue
		}
		break
	}
	if j == i {
		return gadparser.PToken{}
	}
	name := line[i:j]
	rest := strings.TrimSpace(line[j:])
	args := ""
	consumed := j
	if rest != "" {
		if j >= len(line) || line[j] != '(' {
			return gadparser.PToken{}
		}
		balanced, end, ok := s.readBalanced(j, '(', ')')
		if !ok {
			return gadparser.PToken{}
		}
		args = balanced[1 : len(balanced)-1]
		if strings.TrimSpace(line[end:]) != "" {
			return gadparser.PToken{}
		}
		consumed = len(line)
	}
	lit := line[:consumed]
	s.consume(consumed)
	pt := s.newToken(giomtoken.Func, lit, name)
	pt.Set("args", args)
	pt.Set("exported", fmt.Sprint(exported))
	return pt
}

var rgxComp = regexp.MustCompile(`^@(export\s+)?comp ([a-zA-Z_-]+\w*)(\((.*)\))?$`)
var rgxMainComp = regexp.MustCompile(`^@main\s*(\((.*)\))?$`)

func (s *scanner) scanComp() gadparser.PToken {
	line := s.buffer
	if strings.HasPrefix(line, "@main") {
		rest := strings.TrimSpace(line[len("@main"):])
		args := ""
		consumed := len("@main")
		if rest != "" {
			if consumed >= len(line) || line[consumed] != ' ' {
				return gadparser.PToken{}
			}
			start := consumed
			for start < len(line) && line[start] == ' ' {
				start++
			}
			if start >= len(line) || line[start] != '(' {
				return gadparser.PToken{}
			}
			balanced, end, ok := s.readBalanced(start, '(', ')')
			if !ok {
				return gadparser.PToken{}
			}
			args = balanced[1 : len(balanced)-1]
			if strings.TrimSpace(line[end:]) != "" {
				return gadparser.PToken{}
			}
			consumed = len(line)
		}
		lit := line[:consumed]
		s.consume(consumed)
		pt := s.newToken(giomtoken.Comp, lit, "main")
		pt.Set("args", args)
		pt.Set("exported", "true")
		pt.Set("main", "true")
		return pt
	}

	exported := false
	prefix := "@comp "
	if strings.HasPrefix(line, "@export comp ") {
		exported = true
		prefix = "@export comp "
	} else if !strings.HasPrefix(line, prefix) {
		return gadparser.PToken{}
	}

	i := len(prefix)
	j := i
	for j < len(line) {
		c := line[j]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			j++
			continue
		}
		break
	}
	if j == i {
		return gadparser.PToken{}
	}
	name := line[i:j]
	rest := strings.TrimSpace(line[j:])
	args := ""
	consumed := j
	if rest != "" {
		if j >= len(line) || line[j] != '(' {
			return gadparser.PToken{}
		}
		balanced, end, ok := s.readBalanced(j, '(', ')')
		if !ok {
			return gadparser.PToken{}
		}
		args = balanced[1 : len(balanced)-1]
		if strings.TrimSpace(line[end:]) != "" {
			return gadparser.PToken{}
		}
		consumed = len(line)
	}
	lit := line[:consumed]
	s.consume(consumed)
	pt := s.newToken(giomtoken.Comp, lit, name)
	pt.Set("args", args)
	pt.Set("exported", fmt.Sprint(exported))
	return pt
}

var rgxMatch = regexp.MustCompile(`^@match\s+(\S+)\s*$`)

func (s *scanner) scanMatch() gadparser.PToken {
	if sm := rgxMatch.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return s.newToken(giomtoken.Match, sm[0], sm[1])
	}
	return gadparser.PToken{}
}

var rgxCase = regexp.MustCompile(`^@case\s+(.+)\s*$`)

func (s *scanner) scanCase() gadparser.PToken {
	if sm := rgxCase.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return s.newToken(giomtoken.Case, sm[0], sm[1])
	}
	return gadparser.PToken{}
}

var rgxCompCall = regexp.MustCompile(`^\+([@\$A-Za-z_-]+[.\w]*)(\((.*)\)\s*(~?))?$`)

func (s *scanner) scanCompCall() gadparser.PToken {
	if !strings.HasPrefix(s.buffer, "+") {
		return gadparser.PToken{}
	}
	line := s.buffer
	i := 1
	j := i
	for j < len(line) {
		c := line[j]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '$' || c == '@' || c == '.' {
			j++
			continue
		}
		break
	}
	if j == i {
		return gadparser.PToken{}
	}
	name := line[i:j]
	rest := strings.TrimSpace(line[j:])
	args := ""
	withCode := false
	consumed := j
	if rest != "" {
		if j >= len(line) || line[j] != '(' {
			return gadparser.PToken{}
		}
		balanced, end, ok := s.readBalanced(j, '(', ')')
		if !ok {
			return gadparser.PToken{}
		}
		args = balanced[1 : len(balanced)-1]
		tail := strings.TrimSpace(line[end:])
		switch tail {
		case "":
		case "~":
			withCode = true
		default:
			return gadparser.PToken{}
		}
		consumed = len(line)
	}
	lit := line[:consumed]
	s.consume(consumed)
	pt := s.newToken(giomtoken.CompCall, lit, name)
	pt.Set("args", args)
	pt.Set("withCode", fmt.Sprint(withCode))
	return pt
}

var rgxText = regexp.MustCompile(`^(\|)? ?(.*)$`)

func (s *scanner) scanText() gadparser.PToken {
	if sm := rgxText.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		mode := "inline"
		if sm[1] == "|" {
			mode = "piped"
		}
		pt := s.newToken(giomtoken.Text, sm[0], sm[2])
		pt.Set("mode", mode)
		// Absolute position of the text content (sm[2], a suffix of sm[0]) so
		// embedded {= expr } interpolations map back to the original source.
		pt.Set("valuePos", []source.Pos{pt.Pos + source.Pos(len(sm[0])-len(sm[2]))})
		return pt
	}
	return gadparser.PToken{}
}

// =============================================================================
// Raw text mode (for <script>, <style> content)
// =============================================================================

func (s *scanner) NextRaw() gadparser.PToken {
	result := ""
	level := 0

	for {
		s.ensureBuffer()

		switch s.state {
		case giomtoken.ScnEOF:
			pt := s.newToken(giomtoken.Text, result, result)
			pt.Set("mode", "raw")
			return pt

		case giomtoken.ScnNewLine:
			s.state = giomtoken.ScnLine

			if tok := s.scanIndent(); tok.Valid() {
				if tok.Token == giomtoken.Indent {
					level++
				} else if tok.Token == giomtoken.Outdent {
					level--
				} else {
					result += "\n"
					continue
				}

				if level < 0 {
					s.stash.PushBack(s.newToken(giomtoken.Outdent, "", ""))
					if len(result) > 0 && result[len(result)-1] == '\n' {
						result = result[:len(result)-1]
					}
					pt := s.newToken(giomtoken.Text, result, result)
					pt.Set("mode", "raw")
					return pt
				}
			}

		case giomtoken.ScnLine:
			if len(result) > 0 {
				result += "\n"
			}
			for i := 0; i < level; i++ {
				result += "\t"
			}
			result += s.buffer
			s.consume(len(s.buffer))
		}
	}
}

// NextRawCode collects the raw lines of a multi-line code block up to the eof
// marker. Lines are returned verbatim (indentation preserved) alongside the
// absolute base position of each line, so the parser can map the parsed
// statements back onto the original source. Leading indentation is
// insignificant to gad, so preserving it keeps positions faithful without
// affecting compilation.
func (s *scanner) NextRawCode(eof string) (lines []string, positions []source.Pos) {
	for {
		s.ensureBuffer()

		switch s.state {
		case giomtoken.ScnEOF:
			return
		case giomtoken.ScnNewLine:
			if strings.TrimSpace(s.buffer) == eof {
				s.consume(len(s.buffer))
				return
			}
			line := s.buffer
			if strings.TrimSpace(line) == "" {
				line = ""
			}
			s.consume(len(s.buffer))
			lines = append(lines, line)
			positions = append(positions, s.lastTokenPos)
		}
	}
}

// =============================================================================
// Position tracking
// =============================================================================

func (s *scanner) consume(runes int) {
	if len(s.buffer) < runes {
		panic(fmt.Sprintf("Unable to consume %d runes from buffer.", runes))
	}
	s.lastTokenPos = source.Pos(s.file.Base + s.offset - len(s.buffer) - 1)
	s.lastTokenSize = runes
	s.buffer = s.buffer[runes:]
	s.col += runes
}

func (s *scanner) ensureBuffer() {
	if len(s.buffer) > 0 {
		return
	}

	buf, err := s.reader.ReadString('\n')
	s.offset += len(buf)

process:
	if err != nil && err != io.EOF {
		panic(err)
	} else if err != nil && len(buf) == 0 {
		s.state = giomtoken.ScnEOF
	} else {
		if len(buf) > 0 && buf[len(buf)-1] == '\n' {
			buf = buf[:len(buf)-1]
		}

		if lq := lineQuote(buf); lq >= 0 {
			var tmp string
			if tmp, err = s.reader.ReadString('\n'); err == nil || err == io.EOF {
				s.line++
				buf = buf[0:lq] + trimLeftSpace(tmp)
			}
			s.offset += len(buf)
			goto process
		}

		s.state = giomtoken.ScnNewLine
		s.buffer = buf
		s.line++
		s.col = 0
	}
}

func trimLeftSpace(s string) string {
	start := 0
	for ; start < len(s); start++ {
		c := s[start]
		if c >= utf8.RuneSelf {
			return strings.TrimFunc(s[start:], unicode.IsSpace)
		}
		if asciiSpace[c] == 0 {
			break
		}
	}
	return s[start:]
}

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

func lineQuote(s string) (start int) {
	l := len(s)
	if l == 0 {
		return -1
	}
	if s[l-1] == '\\' {
		return l - 1
	}
	return -1
}

// =============================================================================
// Compile-time interface check
// =============================================================================

var _ gadparser.ScannerInterface = (*scanner)(nil)
