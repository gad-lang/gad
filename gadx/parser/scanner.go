package parser

import (
	"bufio"
	"container/list"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	gadparser "github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"

	gadxtoken "github.com/gad-lang/gad/gadx/token"
)

// =============================================================================
// scanner — implements gad's ScannerInterface for gadx template syntax
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

	// forceStack tracks the nested `@text` / `@p` / `@md` blocks currently in
	// scope. While a frame is active every deeper-indented line scans as a literal
	// Text token (no tag parsing) so the body is treated as verbatim text. A frame
	// is popped when the body dedents back to the directive's own depth. `@md`
	// frames additionally let `@`-prefixed lines fall through to normal scanning so
	// nested directives (which push their own frames) work inside Markdown.
	forceStack []forceFrame
}

// forceFrame is one entry of scanner.forceStack: the indent depth of a
// force-text directive (`@text` / `@p` / `@md`) and whether it is a Markdown
// block (`@md`), where `@`-prefixed lines are nested directives, not text.
type forceFrame struct {
	depth int
	md    bool
}

// newScanner creates a scanner that reads from r using file for position tracking.
func newScanner(file *source.File, r io.Reader) *scanner {
	s := &scanner{
		file:        file,
		reader:      bufio.NewReader(r),
		indentStack: list.New(),
		stash:       list.New(),
		state:       gadxtoken.ScnNewLine,
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
// each line. The gadx scanner reads the source through its own bufio.Reader and
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
	case gadxtoken.ScnEOF:
		if outdent := s.indentStack.Back(); outdent != nil {
			s.indentStack.Remove(outdent)
			return s.newToken(gadxtoken.Outdent, "", "")
		}
		return s.newToken(gadxtoken.EOF, "", "")

	case gadxtoken.ScnNewLine:
		s.state = gadxtoken.ScnLine
		// A blank line inside a `@text`/`@p`/`@md` body is kept as an empty text
		// line so the original paragraph breaks survive; it does not touch the
		// indent stack, so the block only ends on a non-blank line dedented back to
		// its own depth.
		if len(s.forceStack) > 0 && len(s.buffer) == 0 {
			return s.scanForcedTextLine()
		}
		if tok := s.scanIndent(); tok.Valid() {
			return tok
		}
		return s.Scan()

	case gadxtoken.ScnLine:
		// Inside a `@text`/`@p`/`@md` block every deeper-indented line is literal
		// text (no tag/directive parsing); frames are popped as the body dedents
		// back to each directive's own depth. In a `@md` frame an `@`-prefixed line
		// is a nested directive, so it falls through to normal scanning.
		for len(s.forceStack) > 0 {
			top := s.forceStack[len(s.forceStack)-1]
			if s.indentStack.Len() <= top.depth {
				s.forceStack = s.forceStack[:len(s.forceStack)-1]
				continue
			}
			if top.md && strings.HasPrefix(s.buffer, "@") {
				break
			}
			return s.scanForcedTextLine()
		}
		if tok := s.scanExport(); tok.Valid() {
			return tok
		}
		if tok := s.scanTextBlock(); tok.Valid() {
			return tok
		}
		if tok := s.scanParaBlock(); tok.Valid() {
			return tok
		}
		if tok := s.scanMdBlock(); tok.Valid() {
			return tok
		}
		if tok := s.scanGlobal(); tok.Valid() {
			return tok
		}
		if tok := s.scanParam(); tok.Valid() {
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
		if tok := s.scanHTML(); tok.Valid() {
			return tok
		}
		if tok := s.scanTag(); tok.Valid() {
			return tok
		}
		if tok := s.scanID(); tok.Valid() {
			return tok
		}
		if tok := s.scanClassName(); tok.Valid() {
			return tok
		}
		if tok := s.scanAttribute(); tok.Valid() {
			return tok
		}
		if tok := s.scanBlockComment(); tok.Valid() {
			return tok
		}
		if tok := s.scanComment(); tok.Valid() {
			return tok
		}
		if tok := s.scanPipeBlock(); tok.Valid() {
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

// =============================================================================
// Indentation scanning
// =============================================================================

var rgxIndent = regexp.MustCompile(`^(\s+)`)

func (s *scanner) scanIndent() gadparser.PToken {
	if len(s.buffer) == 0 {
		s.consume(0)
		return s.newToken(gadxtoken.Blank, "", "")
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
		// Inside an already-established force-text frame (`@text`/`@p`/`@md`), a
		// line indented deeper than the body must NOT open a new block: its extra
		// leading whitespace is literal content (a Markdown code block, a nested
		// list, aligned text). Leave it in the buffer so the force-text scan keeps
		// it, and emit no Indent. The frame's first body line still pushes its one
		// body indent (there indentStack.Len() == frame.depth, so this is skipped).
		if n := len(s.forceStack); n > 0 && s.indentStack.Len() > s.forceStack[n-1].depth {
			return gadparser.PToken{}
		}
		s.indentStack.PushBack(regexp.MustCompile(regexp.QuoteMeta(newIndent)))
		s.consume(len(newIndent))
		return s.newToken(gadxtoken.Indent, newIndent, newIndent)
	}

	if len(newIndent) == 0 && head != nil {
		for head != nil {
			next := head.Next()
			s.indentStack.Remove(head)
			if next == nil {
				return s.newToken(gadxtoken.Outdent, "", "")
			} else {
				t := s.newToken(gadxtoken.Outdent, "", "")
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
		return s.newToken(gadxtoken.Doctype, sm[0], val)
	}
	return gadparser.PToken{}
}

var rgxIf = regexp.MustCompile(`^@if\s+(.+)$`)
var rgxElse = regexp.MustCompile(`^@else(\s*|\s+if\s+(.+))$`)

func (s *scanner) scanCondition() gadparser.PToken {
	if sm := rgxIf.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return s.newToken(gadxtoken.If, sm[0], sm[1])
	}
	if sm := rgxElse.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		if strings.Contains(strings.TrimSpace(sm[0][4:]), "if") {
			return s.newToken(gadxtoken.ElseIf, sm[0], sm[2])
		}
		return s.newToken(gadxtoken.Else, sm[0], "")
	}
	return gadparser.PToken{}
}

var rgxFor = regexp.MustCompile(`^@for\s+(.+)$`)

func (s *scanner) scanFor() gadparser.PToken {
	if sm := rgxFor.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return s.newToken(gadxtoken.For, sm[0], strings.TrimSpace(sm[1]))
	}
	return gadparser.PToken{}
}

var rgxAssignment = regexp.MustCompile(`^(\$[\w0-9\-_]*)?\s*([+-/*:]?)=\s*(.+)$`)

func (s *scanner) scanAssignment() gadparser.PToken {
	if sm := rgxAssignment.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		pt := s.newToken(gadxtoken.Assignment, sm[0], sm[3])
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
		pt := s.newToken(gadxtoken.Code, sm[0], "")
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
		pt := s.newToken(gadxtoken.Code, "", "")
		pt.Set("values", code)
		pt.Set("valuePos", positions)
		return pt
	}
	return gadparser.PToken{}
}

// scanBlockComment scans a `/* … */` block comment at the start of a line. The
// comment is silent (renders nothing) and may span multiple lines: additional
// source lines are pulled into the buffer until the closing `*/`. A `/** … */`
// form is a doc comment (marked "doc"), which the parser attaches to an
// immediately following `@comp`/`@func`. Only recognized at line start; a `/*`
// mid-line stays literal text.
func (s *scanner) scanBlockComment() gadparser.PToken {
	if !strings.HasPrefix(s.buffer, "/*") {
		return gadparser.PToken{}
	}
	// Pull lines until the closing `*/` is in the buffer (search past the opening
	// `/*` so `/*/` / `/**/` are handled correctly).
	for !strings.Contains(s.buffer[2:], "*/") {
		buf, err := s.reader.ReadString('\n')
		if len(buf) == 0 {
			break
		}
		s.offset += len(buf)
		if buf[len(buf)-1] == '\n' {
			buf = buf[:len(buf)-1]
		}
		s.buffer += "\n" + buf
		if err != nil {
			break
		}
	}
	close := strings.Index(s.buffer[2:], "*/")
	if close < 0 {
		// Unterminated block comment: leave it for the text scanner.
		return gadparser.PToken{}
	}
	closeIdx := close + 2
	end := closeIdx + 2

	// A doc comment uses the gad convention `/** … **/`: opened with `/**` and
	// closed with `**/`. It is distinguished by the `/**` opening; the extra `*`
	// of the `**/` close is trimmed from the text below.
	doc := strings.HasPrefix(s.buffer, "/**") && closeIdx >= 3
	openLen := 2
	if doc {
		openLen = 3
	}
	inner := strings.TrimSpace(s.buffer[openLen:closeIdx])
	if doc {
		inner = strings.TrimSpace(strings.TrimRight(inner, "*"))
	}
	lit := s.buffer[:end]

	// Consume the comment; also consume trailing whitespace so the closing line
	// is fully processed when nothing follows `*/`.
	consumeLen := end
	if strings.TrimSpace(s.buffer[end:]) == "" {
		consumeLen = len(s.buffer)
	}
	s.consume(consumeLen)

	pt := s.newToken(gadxtoken.Comment, lit, strings.TrimSpace(inner))
	pt.Set("mode", "silent")
	pt.Set("block", "true")
	if doc {
		pt.Set("doc", "true")
	}
	return pt
}

var rgxTextBlock = regexp.MustCompile(`^@text\s*$`)

// scanTextBlock matches the `@text` directive and switches the scanner into
// force-text mode for its indented body: every deeper line becomes a literal
// Text token (see forceTextDepth / scanForcedTextLine).
func (s *scanner) scanTextBlock() gadparser.PToken {
	if rgxTextBlock.MatchString(s.buffer) {
		lit := s.buffer
		s.consume(len(s.buffer))
		s.pushForce(false)
		return s.newToken(gadxtoken.TextBlock, lit, "")
	}
	return gadparser.PToken{}
}

var rgxPipeBlock = regexp.MustCompile(`^\|(>)?\s*$`)

// scanPipeBlock matches a bare `|` (or `|>`) on its own line, opening a
// YAML-style text block: every deeper-indented line is literal text, so text
// does not need a `| ` prefix on each line. `|` is the literal style (line
// breaks preserved, like YAML `|`); `|>` is the folded style (line breaks become
// spaces, like YAML `>`). It reuses the `@text` force-text machinery; the token
// is marked "pipe" (and "fold" for `|>`) so the parser emits the right block.
func (s *scanner) scanPipeBlock() gadparser.PToken {
	if sm := rgxPipeBlock.FindStringSubmatch(s.buffer); sm != nil {
		lit := s.buffer
		s.consume(len(s.buffer))
		s.pushForce(false)
		pt := s.newToken(gadxtoken.TextBlock, lit, "")
		pt.Set("pipe", true)
		if sm[1] == ">" {
			pt.Set("fold", true)
		}
		return pt
	}
	return gadparser.PToken{}
}

var rgxParaBlock = regexp.MustCompile(`^@p\s*$`)

// scanParaBlock matches the `@p` directive and, like `@text`, switches the
// scanner into force-text mode for its indented body. The parser groups the
// resulting text lines into <p> paragraphs separated by blank lines.
func (s *scanner) scanParaBlock() gadparser.PToken {
	if rgxParaBlock.MatchString(s.buffer) {
		lit := s.buffer
		s.consume(len(s.buffer))
		s.pushForce(false)
		return s.newToken(gadxtoken.Para, lit, "")
	}
	return gadparser.PToken{}
}

var rgxMdBlock = regexp.MustCompile(`^@md\s*$`)

// scanMdBlock matches the `@md` directive and switches the scanner into a
// Markdown force-text frame: body lines are literal Markdown text (with `{ … }`
// interpolation), but `@`-prefixed lines fall through to normal scanning so
// nested directives work. The lowering renders the collected Markdown to HTML.
func (s *scanner) scanMdBlock() gadparser.PToken {
	if rgxMdBlock.MatchString(s.buffer) {
		lit := s.buffer
		s.consume(len(s.buffer))
		s.pushForce(true)
		return s.newToken(gadxtoken.Md, lit, "")
	}
	return gadparser.PToken{}
}

// pushForce opens a force-text frame at the current indent depth. md marks a
// `@md` frame, where `@`-prefixed lines are nested directives instead of text.
func (s *scanner) pushForce(md bool) {
	s.forceStack = append(s.forceStack, forceFrame{depth: s.indentStack.Len(), md: md})
}

// scanForcedTextLine emits the whole current line as a literal Text token (used
// inside a `@text` body). The valuePos points at the line start so embedded
// `{ … }` interpolations keep their source positions.
func (s *scanner) scanForcedTextLine() gadparser.PToken {
	lit := s.buffer
	s.consume(len(s.buffer))
	pt := s.newToken(gadxtoken.Text, lit, lit)
	pt.Set("mode", "raw")
	pt.Set("valuePos", []source.Pos{pt.Pos})
	return pt
}

var (
	rgxDocComment = regexp.MustCompile(`^\/\/\/\s?(.*)$`)
	rgxComment    = regexp.MustCompile(`^\/\/(-)?\s*(.*)$`)
)

func (s *scanner) scanComment() gadparser.PToken {
	// `/// …` is a single-line doc comment (the gad convention): silent, and
	// attached to the next documentable declaration like a `/** … **/` block doc.
	// It must be checked before the general `//` comment.
	if sm := rgxDocComment.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		pt := s.newToken(gadxtoken.Comment, sm[0], strings.TrimSpace(sm[1]))
		pt.Set("mode", "silent")
		pt.Set("doc", "true")
		return pt
	}
	if sm := rgxComment.FindStringSubmatch(s.buffer); len(sm) != 0 {
		mode := "embed"
		if len(sm[1]) != 0 {
			mode = "silent"
		}
		s.consume(len(sm[0]))
		pt := s.newToken(gadxtoken.Comment, sm[0], sm[2])
		pt.Set("mode", mode)
		return pt
	}
	return gadparser.PToken{}
}

var rgxID = regexp.MustCompile(`^#([\w-]+)(?:\s*\?\s*(.*)$)?`)

func (s *scanner) scanID() gadparser.PToken {
	if sm := rgxID.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		pt := s.newToken(gadxtoken.ID, sm[0], sm[1])
		pt.Set("condition", sm[2])
		return pt
	}
	return gadparser.PToken{}
}

var rgxClassName = regexp.MustCompile(`^\.([\w-]+)(?:\s*\?\s*(.*)$)?`)

func (s *scanner) scanClassName() gadparser.PToken {
	if sm := rgxClassName.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		pt := s.newToken(gadxtoken.ClassName, sm[0], sm[1])
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
	pt := s.newToken(gadxtoken.Attribute, lit, "")
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
			pt := s.newToken(gadxtoken.ImportModule, sm[0], sm[2])
			pt.Set("destructure", strings.TrimSpace(sm[1]))
			return pt
		}
		if sm := rgxImportModule.FindStringSubmatch(s.buffer); len(sm) != 0 {
			s.consume(len(sm[0]))
			pt := s.newToken(gadxtoken.ImportModule, sm[0], sm[1])
			pt.Set("ident", sm[3])
			return pt
		}
	}
	return gadparser.PToken{}
}

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

// readQuotedName reads a `"…"` interpolated name string beginning at start
// (which must be the opening quote). Interpolation braces `{…}` are tracked so a
// quote inside an interpolation does not close the name, and `\` escapes the next
// character. It returns the content between the quotes, the index just past the
// closing quote, and whether a closed string was found.
func (s *scanner) readQuotedName(start int) (string, int, bool) {
	if start >= len(s.buffer) || s.buffer[start] != '"' {
		return "", start, false
	}
	depth := 0
	escaped := false
	for i := start + 1; i < len(s.buffer); i++ {
		c := s.buffer[i]
		if escaped {
			escaped = false
			continue
		}
		switch c {
		case '\\':
			escaped = true
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		case '"':
			if depth == 0 {
				return s.buffer[start+1 : i], i + 1, true
			}
		}
	}
	return "", start, false
}

func (s *scanner) scanSlot() gadparser.PToken {
	if strings.TrimSpace(s.buffer) == "@wrap" {
		line := s.buffer
		s.consume(len(line))
		return s.newToken(gadxtoken.Wrap, line, "")
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
	if i < len(line) && line[i] == '"' {
		// Quoted, interpolated name: `@slot "line[{index}]"`. The content is a Gad
		// template string; store it verbatim with its absolute position.
		content, end, ok := s.readQuotedName(i)
		if !ok {
			return gadparser.PToken{}
		}
		name = content
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
	pt := s.newToken(gadxtoken.Slot, lit, name)
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
	if strings.HasPrefix(s.buffer, prefix) && len(s.buffer) > len(prefix) && s.buffer[len(prefix)] == '"' {
		// Quoted, interpolated name: `@slot #"line[{index}]"(args)`. The content is
		// a Gad template string; store it verbatim with its absolute position,
		// followed by an optional `(args)` group.
		line := s.buffer
		base0 := source.Pos(s.file.Base + s.offset - len(s.buffer) - 1)
		i := len(prefix)
		content, end, ok := s.readQuotedName(i)
		if !ok {
			return gadparser.PToken{}
		}
		name := content
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
		pt := s.newToken(gadxtoken.SlotPass, line, name)
		pt.Set("name", name)
		pt.Set("args", args)
		pt.Set("nameExpr", true)
		pt.Set("namePos", namePos)
		return pt
	}
	if sm := rgxSlotPass.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		pt := s.newToken(gadxtoken.SlotPass, sm[0], sm[1])
		pt.Set("header", sm[1])
		return pt
	}
	return gadparser.PToken{}
}

// scanHTML scans a self-contained raw HTML region beginning with `<` — an
// opening tag `<name …>` or a `<>` fragment. The region runs to its matching
// close tag (spanning multiple lines if needed) and is stored verbatim as the
// token value with its absolute base position; the parser turns it into write
// calls, collapsing whitespace and evaluating `{ … }` interpolations.
func (s *scanner) scanHTML() gadparser.PToken {
	if len(s.buffer) < 2 || s.buffer[0] != '<' {
		return gadparser.PToken{}
	}
	c := s.buffer[1]
	isLetter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
	if !(c == '>' || isLetter) {
		return gadparser.PToken{}
	}
	base0 := source.Pos(s.file.Base + s.offset - len(s.buffer) - 1)
	s.ensureHTMLComplete(0)
	end, ok := htmlRegionEnd(s.buffer, 0)
	if !ok {
		return gadparser.PToken{}
	}
	raw := s.buffer[:end]
	s.consume(end)
	pt := s.newToken(gadxtoken.HTML, raw, raw)
	pt.Set("htmlPos", base0)
	return pt
}

// ensureHTMLComplete pulls additional source lines into the buffer until the
// HTML region opened at s.buffer[start] closes, or input ends. The separating
// newline is preserved so buffer offsets stay aligned with file offsets.
func (s *scanner) ensureHTMLComplete(start int) {
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
		return s.newToken(gadxtoken.Tag, sm[0], sm[1])
	}
	return gadparser.PToken{}
}

var rgxExport = regexp.MustCompile(`^@export\s+([a-zA-Z_]\w*)(\s*=\s*(.+))?$`)

func (s *scanner) scanExport() gadparser.PToken {
	if sm := rgxExport.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		pt := s.newToken(gadxtoken.Export, sm[0], sm[1])
		pt.Set("name", sm[1])
		pt.Set("value", sm[3])
		return pt
	}
	return gadparser.PToken{}
}

func (s *scanner) scanGlobal() gadparser.PToken {
	return s.scanDeclDirective("@global", gadxtoken.Global)
}

func (s *scanner) scanParam() gadparser.PToken {
	return s.scanDeclDirective("@param", gadxtoken.Param)
}

func (s *scanner) scanVar() gadparser.PToken {
	return s.scanDeclDirective("@var", gadxtoken.Var)
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
	pt := s.newToken(gadxtoken.Enum, lit, strings.TrimSpace(inner))
	pt.Set("name", name)
	pt.Set("innerPos", base0+source.Pos(innerStart+lead))
	return pt
}

func (s *scanner) scanConst() gadparser.PToken {
	return s.scanDeclDirective("@const", gadxtoken.Const)
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
		paren      bool
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
		paren = true
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
	// Record whether the declaration was written in parenthesized form. `@global`
	// applies its legacy space-separated-names rewrite only to the bare form, so
	// the parenthesized form stays verbatim Gad (and can carry typed names).
	pt.Set("paren", paren)
	return pt
}

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
	// The signature is the name plus the rest of the line: optional
	// `[typeparams]`, `(params)` and `<return>`. It is validated and turned into a
	// FuncType by the parser (via the Gad parser), so parameter types, type
	// parameters and return types are all supported. sigoff records where the
	// signature starts so its source positions are preserved.
	sig := strings.TrimRight(line[i:], " \t")
	consumed := len(line)
	lit := line[:consumed]
	s.consume(consumed)
	pt := s.newToken(gadxtoken.Func, lit, name)
	pt.Set("sig", sig)
	pt.Set("sigoff", strconv.Itoa(i))
	pt.Set("exported", fmt.Sprint(exported))
	return pt
}

func (s *scanner) scanComp() gadparser.PToken {
	line := s.buffer
	if strings.HasPrefix(line, "@main") {
		// `@main` is anonymous: its signature (optional `[typeparams]`, `(params)`
		// and `<return>`) starts right after it. Only a boundary — end of line,
		// space, `(` or `[` — separates it from an unrelated `@mainxyz` tag.
		after := line[len("@main"):]
		if after == "" || after[0] == ' ' || after[0] == '(' || after[0] == '[' {
			i := len("@main") + (len(after) - len(strings.TrimLeft(after, " \t")))
			sig := strings.TrimRight(line[i:], " \t")
			consumed := len(line)
			lit := line[:consumed]
			s.consume(consumed)
			pt := s.newToken(gadxtoken.Comp, lit, "main")
			pt.Set("sig", sig)
			pt.Set("sigoff", strconv.Itoa(i))
			pt.Set("exported", "true")
			pt.Set("main", "true")
			return pt
		}
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
	// The signature is the name plus the rest of the line (optional
	// `[typeparams]`, `(params)`, `<return>`), parsed by the parser (see scanFunc).
	sig := strings.TrimRight(line[i:], " \t")
	consumed := len(line)
	lit := line[:consumed]
	s.consume(consumed)
	pt := s.newToken(gadxtoken.Comp, lit, name)
	pt.Set("sig", sig)
	pt.Set("sigoff", strconv.Itoa(i))
	pt.Set("exported", fmt.Sprint(exported))
	return pt
}

var rgxMatch = regexp.MustCompile(`^@match\s+(\S+)\s*$`)

func (s *scanner) scanMatch() gadparser.PToken {
	if sm := rgxMatch.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return s.newToken(gadxtoken.Match, sm[0], sm[1])
	}
	return gadparser.PToken{}
}

var rgxCase = regexp.MustCompile(`^@case\s+(.+)\s*$`)

func (s *scanner) scanCase() gadparser.PToken {
	if sm := rgxCase.FindStringSubmatch(s.buffer); len(sm) != 0 {
		s.consume(len(sm[0]))
		return s.newToken(gadxtoken.Case, sm[0], sm[1])
	}
	return gadparser.PToken{}
}

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
	pt := s.newToken(gadxtoken.CompCall, lit, name)
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
		pt := s.newToken(gadxtoken.Text, sm[0], sm[2])
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
		case gadxtoken.ScnEOF:
			pt := s.newToken(gadxtoken.Text, result, result)
			pt.Set("mode", "raw")
			return pt

		case gadxtoken.ScnNewLine:
			s.state = gadxtoken.ScnLine

			if tok := s.scanIndent(); tok.Valid() {
				if tok.Token == gadxtoken.Indent {
					level++
				} else if tok.Token == gadxtoken.Outdent {
					level--
				} else {
					result += "\n"
					continue
				}

				if level < 0 {
					s.stash.PushBack(s.newToken(gadxtoken.Outdent, "", ""))
					if len(result) > 0 && result[len(result)-1] == '\n' {
						result = result[:len(result)-1]
					}
					pt := s.newToken(gadxtoken.Text, result, result)
					pt.Set("mode", "raw")
					return pt
				}
			}

		case gadxtoken.ScnLine:
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
		case gadxtoken.ScnEOF:
			return
		case gadxtoken.ScnNewLine:
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
		s.state = gadxtoken.ScnEOF
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

		s.state = gadxtoken.ScnNewLine
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
