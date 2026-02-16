// YAPL Lexer - Pass 1 of the YAPL compiler
// Reads YAPL source from stdin, writes token stream to stdout

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Token categories
const (
	KEY   = "KEY"
	ID    = "ID"
	PUNCT = "PUNCT"
	LIT   = "LIT"
)

// Keywords (including type names which are keywords per design decision)
var keywords = map[string]bool{
	// Control flow
	"break": true, "continue": true, "else": true, "for": true,
	"goto": true, "if": true, "return": true, "while": true,
	// Declarations
	"const": true, "func": true, "struct": true, "var": true,
	// Types (made keywords per design decision)
	"byte": true, "uint8": true, "uint16": true, "int16": true,
	"void": true, "block32": true, "block64": true, "block128": true,
	// Other
	"sizeof": true,
}

// Multi-character operators (order matters - longer first)
var multiCharOps = []string{
	"&&", "||", "==", "!=", "<=", ">=", "<<", ">>", "->",
}

// Single-character punctuation/operators
var singleCharOps = map[byte]bool{
	'+': true, '-': true, '*': true, '/': true, '%': true,
	'&': true, '|': true, '^': true, '~': true, '!': true,
	'<': true, '>': true, '=': true, '@': true,
	'(': true, ')': true, '[': true, ']': true, '{': true, '}': true,
	';': true, ':': true, ',': true, '.': true,
}

// Lexer state
type Lexer struct {
	reader       *bufio.Reader
	line         int
	lastEmitLine int
	tokenNum     int
	filename     string
	constants    map[string]int64 // constant symbol table
	output       *bufio.Writer

	// For conditional compilation
	ifStack  []bool // stack of condition states
	skipping bool   // currently skipping code due to #if false
}

func NewLexer(reader io.Reader, filename string, output *bufio.Writer) *Lexer {
	return &Lexer{
		reader:       bufio.NewReader(reader),
		line:         1,
		lastEmitLine: 0,
		tokenNum:     0,
		filename:     filename,
		constants:    make(map[string]int64),
		output:       output,
		ifStack:      make([]bool, 0),
		skipping:     false,
	}
}

func (l *Lexer) peek() byte {
	buf, err := l.reader.Peek(1)
	if err != nil {
		return 0
	}
	return buf[0]
}

func (l *Lexer) peekN(n int) byte {
	buf, err := l.reader.Peek(n + 1)
	if err != nil || len(buf) <= n {
		return 0
	}
	return buf[n]
}

func (l *Lexer) advance() byte {
	ch, err := l.reader.ReadByte()
	if err != nil {
		return 0
	}
	if ch == '\n' {
		l.line++
	}
	return ch
}

func (l *Lexer) skipWhitespace() {
	for l.peek() != 0 {
		ch := l.peek()
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			l.advance()
		} else if ch == '/' && l.peekN(1) == '/' {
			// Line comment
			for l.peek() != '\n' && l.peek() != 0 {
				l.advance()
			}
		} else if ch == '/' && l.peekN(1) == '*' {
			// Block comment
			l.advance() // skip /
			l.advance() // skip *
			for !(l.peek() == '*' && l.peekN(1) == '/') && l.peek() != 0 {
				l.advance()
			}
			if l.peek() != 0 {
				l.advance() // skip *
				l.advance() // skip /
			}
		} else {
			break
		}
	}
}

func (l *Lexer) emitLineDirective() {
	if l.line != l.lastEmitLine {
		fmt.Fprintf(l.output, "#line %d\n", l.line)
		l.lastEmitLine = l.line
	}
}

func (l *Lexer) emitToken(category, value string) {
	if l.skipping {
		return
	}
	l.emitLineDirective()
	l.tokenNum++
	fmt.Fprintf(l.output, "%d, %s, %s\n", l.tokenNum, category, value)
}

func (l *Lexer) error(msg string) {
	fmt.Fprintf(os.Stderr, "%s:%d: error: %s\n", l.filename, l.line, msg)
	os.Exit(1)
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isHexDigit(ch byte) bool {
	return isDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isBinaryDigit(ch byte) bool {
	return ch == '0' || ch == '1'
}

func isOctalDigit(ch byte) bool {
	return ch >= '0' && ch <= '7'
}

func (l *Lexer) scanIdentifier() string {
	var b strings.Builder
	for isLetter(l.peek()) || isDigit(l.peek()) {
		b.WriteByte(l.advance())
	}
	return b.String()
}

func (l *Lexer) scanNumber() int64 {
	var value int64

	if l.peek() == '0' {
		l.advance()
		switch l.peek() {
		case 'x', 'X':
			// Hexadecimal
			l.advance()
			if l.peek() == '_' {
				l.advance()
			}
			for isHexDigit(l.peek()) || l.peek() == '_' {
				if l.peek() != '_' {
					digit := l.peek()
					if digit >= '0' && digit <= '9' {
						value = value*16 + int64(digit-'0')
					} else if digit >= 'a' && digit <= 'f' {
						value = value*16 + int64(digit-'a'+10)
					} else {
						value = value*16 + int64(digit-'A'+10)
					}
				}
				l.advance()
			}
		case 'b', 'B':
			// Binary
			l.advance()
			if l.peek() == '_' {
				l.advance()
			}
			for isBinaryDigit(l.peek()) || l.peek() == '_' {
				if l.peek() != '_' {
					value = value*2 + int64(l.peek()-'0')
				}
				l.advance()
			}
		case 'o', 'O':
			// Octal
			l.advance()
			if l.peek() == '_' {
				l.advance()
			}
			for isOctalDigit(l.peek()) || l.peek() == '_' {
				if l.peek() != '_' {
					value = value*8 + int64(l.peek()-'0')
				}
				l.advance()
			}
		default:
			// Just zero or decimal continuing with more digits
			for isDigit(l.peek()) || l.peek() == '_' {
				if l.peek() != '_' {
					value = value*10 + int64(l.peek()-'0')
				}
				l.advance()
			}
		}
	} else {
		// Decimal
		for isDigit(l.peek()) || l.peek() == '_' {
			if l.peek() != '_' {
				value = value*10 + int64(l.peek()-'0')
			}
			l.advance()
		}
	}

	return value
}

func (l *Lexer) scanCharLiteral() int64 {
	l.advance() // skip opening '
	var value int64

	if l.peek() == '\\' {
		value = int64(l.scanEscape())
	} else {
		value = int64(l.peek())
		l.advance()
	}

	if l.peek() != '\'' {
		l.error("unterminated character literal")
	}
	l.advance() // skip closing '
	return value
}

func (l *Lexer) scanEscape() byte {
	l.advance() // skip backslash
	ch := l.advance()
	switch ch {
	case '0':
		return 0
	case 'a':
		return '\a'
	case 'b':
		return '\b'
	case 'f':
		return '\f'
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case 'v':
		return '\v'
	case '\\':
		return '\\'
	case '\'':
		return '\''
	case '"':
		return '"'
	case 'x':
		// Hex escape \xNN
		if !isHexDigit(l.peek()) || !isHexDigit(l.peekN(1)) {
			l.error("invalid hex escape sequence")
		}
		h1 := l.advance()
		h2 := l.advance()
		return hexValue(h1)*16 + hexValue(h2)
	default:
		l.error(fmt.Sprintf("invalid escape sequence \\%c", ch))
		return 0
	}
}

func hexValue(ch byte) byte {
	if ch >= '0' && ch <= '9' {
		return ch - '0'
	}
	if ch >= 'a' && ch <= 'f' {
		return ch - 'a' + 10
	}
	return ch - 'A' + 10
}

func (l *Lexer) scanString() string {
	l.advance() // skip opening "
	var result strings.Builder
	result.WriteByte('"')

	for l.peek() != '"' && l.peek() != 0 && l.peek() != '\n' {
		if l.peek() == '\\' {
			// Keep escape sequences as-is in the output
			result.WriteByte(l.advance())
			if l.peek() != 0 {
				esc := l.advance()
				result.WriteByte(esc)
				if esc == 'x' {
					// Hex escape - copy two more chars
					if l.peek() != 0 {
						result.WriteByte(l.advance())
					}
					if l.peek() != 0 {
						result.WriteByte(l.advance())
					}
				}
			}
		} else {
			result.WriteByte(l.advance())
		}
	}

	if l.peek() != '"' {
		l.error("unterminated string literal")
	}
	l.advance() // skip closing "
	result.WriteByte('"')
	return result.String()
}

// scanToEndOfLine reads and returns the rest of the current line
// (up to but not including the newline or EOF). Trims leading/trailing whitespace.
func (l *Lexer) scanToEndOfLine() string {
	var b strings.Builder
	for l.peek() != '\n' && l.peek() != 0 {
		b.WriteByte(l.advance())
	}
	return strings.TrimSpace(b.String())
}

// scanRawString scans a string literal without processing escapes
// Used for #asm directive where escapes are not allowed
func (l *Lexer) scanRawString() string {
	l.advance() // skip opening "
	var result strings.Builder
	result.WriteByte('"')

	for l.peek() != '"' && l.peek() != 0 && l.peek() != '\n' {
		ch := l.peek()
		if ch == '\\' {
			l.error("escape sequences not allowed in #asm string")
			return ""
		}
		result.WriteByte(l.advance())
	}

	if l.peek() != '"' {
		l.error("unterminated #asm string literal")
	}
	l.advance() // skip closing "
	result.WriteByte('"')
	return result.String()
}

func (l *Lexer) handleDirective() {
	l.advance() // skip #

	// Read directive name
	name := l.scanIdentifier()
	l.skipWhitespace()

	switch name {
	case "if":
		// Parse constant expression for condition
		val := l.parseConstExpr()
		condition := val != 0
		l.ifStack = append(l.ifStack, condition)
		if !condition {
			l.skipping = true
		}

	case "else":
		if len(l.ifStack) == 0 {
			l.error("#else without matching #if")
		}
		// Flip the condition
		l.ifStack[len(l.ifStack)-1] = !l.ifStack[len(l.ifStack)-1]
		l.updateSkipping()

	case "endif":
		if len(l.ifStack) == 0 {
			l.error("#endif without matching #if")
		}
		l.ifStack = l.ifStack[:len(l.ifStack)-1]
		l.updateSkipping()

	case "file":
		// #file as r-value - emit the filename as a string literal
		if !l.skipping {
			l.emitToken(LIT, fmt.Sprintf("\"%s\"", l.filename))
		}

	case "line":
		// #line directive in source - update our line number
		val := l.scanNumber()
		l.line = int(val)

	case "asm":
		// #asm("assembly code") - inline assembly
		// Emit #asm as a keyword, then parse and emit the raw string
		if !l.skipping {
			l.emitToken(KEY, "#asm")
		}
		l.skipWhitespace()
		if l.peek() != '(' {
			l.error("expected '(' after #asm")
			return
		}
		l.advance() // consume '('
		l.skipWhitespace()
		if l.peek() != '"' {
			l.error("expected string literal in #asm")
			return
		}
		// Parse raw string (no escape processing)
		rawStr := l.scanRawString()
		if !l.skipping {
			l.emitToken(LIT, rawStr)
		}
		l.skipWhitespace()
		if l.peek() != ')' {
			l.error("expected ')' after #asm string")
			return
		}
		l.advance() // consume ')'

	case "pragma":
		// #pragma <name> [rest of line]
		// Handled entirely in the lexer; no tokens emitted.
		if !isLetter(l.peek()) {
			l.error("expected pragma name after #pragma")
			return
		}
		pragmaName := l.scanIdentifier()
		switch pragmaName {
		case "message":
			// #pragma message <text> - print rest of line to stderr
			// Skip whitespace before the message text (but not newlines)
			for l.peek() == ' ' || l.peek() == '\t' {
				l.advance()
			}
			msg := l.scanToEndOfLine()
			if !l.skipping {
				fmt.Fprintf(os.Stderr, "%s:%d: #pragma message: %s\n", l.filename, l.line, msg)
			}
		case "bootstrap":
			// #pragma bootstrap - emit #bootstrap meta-line for downstream passes
			if !l.skipping {
				fmt.Fprintln(l.output, "#bootstrap")
			}
		default:
			l.error(fmt.Sprintf("unknown pragma: %s", pragmaName))
		}

	default:
		l.error(fmt.Sprintf("unknown directive #%s", name))
	}
}

func (l *Lexer) updateSkipping() {
	l.skipping = false
	for _, cond := range l.ifStack {
		if !cond {
			l.skipping = true
			break
		}
	}
}

// Constant expression parser
// Returns the evaluated value

func (l *Lexer) parseConstExpr() int64 {
	return l.parseConstOr()
}

func (l *Lexer) parseConstOr() int64 {
	left := l.parseConstAnd()
	for {
		l.skipWhitespace()
		if l.peek() == '|' && l.peekN(1) == '|' {
			l.advance()
			l.advance()
			right := l.parseConstAnd()
			if left != 0 || right != 0 {
				left = 1
			} else {
				left = 0
			}
		} else {
			break
		}
	}
	return left
}

func (l *Lexer) parseConstAnd() int64 {
	left := l.parseConstCmp()
	for {
		l.skipWhitespace()
		if l.peek() == '&' && l.peekN(1) == '&' {
			l.advance()
			l.advance()
			right := l.parseConstCmp()
			if left != 0 && right != 0 {
				left = 1
			} else {
				left = 0
			}
		} else {
			break
		}
	}
	return left
}

func (l *Lexer) parseConstCmp() int64 {
	left := l.parseConstAdd()
	l.skipWhitespace()

	if l.peek() == '=' && l.peekN(1) == '=' {
		l.advance()
		l.advance()
		right := l.parseConstAdd()
		if left == right {
			return 1
		}
		return 0
	}
	if l.peek() == '!' && l.peekN(1) == '=' {
		l.advance()
		l.advance()
		right := l.parseConstAdd()
		if left != right {
			return 1
		}
		return 0
	}
	if l.peek() == '<' && l.peekN(1) == '=' {
		l.advance()
		l.advance()
		right := l.parseConstAdd()
		if left <= right {
			return 1
		}
		return 0
	}
	if l.peek() == '>' && l.peekN(1) == '=' {
		l.advance()
		l.advance()
		right := l.parseConstAdd()
		if left >= right {
			return 1
		}
		return 0
	}
	if l.peek() == '<' && l.peekN(1) != '<' {
		l.advance()
		right := l.parseConstAdd()
		if left < right {
			return 1
		}
		return 0
	}
	if l.peek() == '>' && l.peekN(1) != '>' {
		l.advance()
		right := l.parseConstAdd()
		if left > right {
			return 1
		}
		return 0
	}

	return left
}

func (l *Lexer) parseConstAdd() int64 {
	left := l.parseConstMult()
	for {
		l.skipWhitespace()
		ch := l.peek()
		if ch == '+' {
			l.advance()
			left = left + l.parseConstMult()
		} else if ch == '-' && l.peekN(1) != '>' {
			l.advance()
			left = left - l.parseConstMult()
		} else if ch == '|' && l.peekN(1) != '|' {
			l.advance()
			left = left | l.parseConstMult()
		} else if ch == '^' {
			l.advance()
			left = left ^ l.parseConstMult()
		} else {
			break
		}
	}
	return left
}

func (l *Lexer) parseConstMult() int64 {
	left := l.parseConstUnary()
	for {
		l.skipWhitespace()
		ch := l.peek()
		if ch == '*' {
			l.advance()
			left = left * l.parseConstUnary()
		} else if ch == '/' {
			l.advance()
			right := l.parseConstUnary()
			if right == 0 {
				l.error("division by zero in constant expression")
			}
			left = left / right
		} else if ch == '%' {
			l.advance()
			right := l.parseConstUnary()
			if right == 0 {
				l.error("modulo by zero in constant expression")
			}
			left = left % right
		} else if ch == '&' && l.peekN(1) != '&' {
			l.advance()
			left = left & l.parseConstUnary()
		} else if ch == '<' && l.peekN(1) == '<' {
			l.advance()
			l.advance()
			left = left << l.parseConstUnary()
		} else if ch == '>' && l.peekN(1) == '>' {
			l.advance()
			l.advance()
			left = left >> l.parseConstUnary()
		} else {
			break
		}
	}
	return left
}

func (l *Lexer) parseConstUnary() int64 {
	l.skipWhitespace()
	ch := l.peek()

	if ch == '-' {
		l.advance()
		return -l.parseConstUnary()
	}
	if ch == '~' {
		l.advance()
		return ^l.parseConstUnary()
	}
	if ch == '!' {
		l.advance()
		val := l.parseConstUnary()
		if val == 0 {
			return 1
		}
		return 0
	}

	// Check for sizeof or other identifier
	if isLetter(ch) {
		ident := l.scanIdentifier()
		if ident == "sizeof" {
			l.skipWhitespace()
			if l.peek() != '(' {
				l.error("expected '(' after sizeof")
			}
			l.advance()
			l.skipWhitespace()

			// Parse type specifier
			typeName := l.scanIdentifier()
			size := l.sizeofType(typeName)

			l.skipWhitespace()
			if l.peek() != ')' {
				l.error("expected ')' after sizeof type")
			}
			l.advance()
			return size
		}
		// Not sizeof, pass pre-read identifier to parseConstPrimary
		return l.parseConstPrimaryWithIdent(ident)
	}

	return l.parseConstPrimaryWithIdent("")
}

func (l *Lexer) sizeofType(typeName string) int64 {
	switch typeName {
	case "byte", "uint8":
		return 1
	case "int16", "uint16":
		return 2
	case "block32":
		return 4 // 32 bits = 4 bytes
	case "block64":
		return 8 // 64 bits = 8 bytes
	case "block128":
		return 16 // 128 bits = 16 bytes
	default:
		// Pointer types
		if strings.HasPrefix(typeName, "@") {
			return 2 // pointers are 16-bit
		}
		l.error(fmt.Sprintf("unknown type in sizeof: %s", typeName))
		return 0
	}
}

func (l *Lexer) parseConstPrimaryWithIdent(preIdent string) int64 {
	l.skipWhitespace()

	// If we already have a pre-read identifier, handle it directly
	if preIdent != "" {
		return l.resolveConstIdent(preIdent)
	}

	ch := l.peek()

	if ch == '(' {
		l.advance()
		val := l.parseConstExpr()
		l.skipWhitespace()
		if l.peek() != ')' {
			l.error("expected ')' in constant expression")
		}
		l.advance()
		return val
	}

	if isDigit(ch) {
		return l.scanNumber()
	}

	if ch == '\'' {
		return l.scanCharLiteral()
	}

	if isLetter(ch) {
		ident := l.scanIdentifier()
		return l.resolveConstIdent(ident)
	}

	l.error(fmt.Sprintf("unexpected character in constant expression: %c", ch))
	return 0
}

func (l *Lexer) resolveConstIdent(ident string) int64 {
	// Check if it's a type name for type cast
	if keywords[ident] && l.isTypeName(ident) {
		l.skipWhitespace()
		if l.peek() == '(' {
			l.advance()
			val := l.parseConstExpr()
			l.skipWhitespace()
			if l.peek() != ')' {
				l.error("expected ')' after type cast")
			}
			l.advance()
			// Apply type cast (truncate to appropriate size)
			return l.applyTypeCast(ident, val)
		}
	}
	// Look up in constant table
	if val, ok := l.constants[ident]; ok {
		return val
	}
	l.error(fmt.Sprintf("undefined constant: %s", ident))
	return 0
}

func (l *Lexer) isTypeName(name string) bool {
	switch name {
	case "byte", "uint8", "int16", "uint16", "void",
		"block32", "block64", "block128":
		return true
	}
	return false
}

func (l *Lexer) applyTypeCast(typeName string, val int64) int64 {
	switch typeName {
	case "byte", "uint8":
		return val & 0xFF
	case "int16":
		v := val & 0xFFFF
		if v >= 0x8000 {
			return v - 0x10000
		}
		return v
	case "uint16":
		return val & 0xFFFF
	default:
		return val
	}
}

// Main lexer loop

func (l *Lexer) Run() {
	// Emit file directive
	fmt.Fprintf(l.output, "#file %s\n", l.filename)

	for l.peek() != 0 {
		l.skipWhitespace()
		if l.peek() == 0 {
			break
		}

		ch := l.peek()

		// Handle directives - must be processed even when skipping
		// so that #else and #endif work correctly
		if ch == '#' {
			l.handleDirective()
			continue
		}

		// Skip if we're in a false #if block
		if l.skipping {
			l.advance()
			continue
		}

		// Identifier or keyword
		if isLetter(ch) {
			ident := l.scanIdentifier()
			if keywords[ident] {
				// Check if this is 'const' - we need to track the declaration
				if ident == "const" {
					l.handleConstDecl()
					continue
				}
				// Check if this is 'var' - array dimensions need constant folding
				if ident == "var" {
					l.handleVarDecl()
					continue
				}
				// Check if this is 'struct' - member array dimensions need constant folding
				if ident == "struct" {
					l.handleStructDecl()
					continue
				}
				l.emitToken(KEY, ident)
			} else {
				l.emitToken(ID, ident)
			}
			continue
		}

		// Number
		if isDigit(ch) {
			val := l.scanNumber()
			l.emitToken(LIT, fmt.Sprintf("0x%04X", val&0xFFFF))
			continue
		}

		// Character literal
		if ch == '\'' {
			val := l.scanCharLiteral()
			l.emitToken(LIT, fmt.Sprintf("0x%04X", val&0xFFFF))
			continue
		}

		// String literal
		if ch == '"' {
			str := l.scanString()
			l.emitToken(LIT, str)
			continue
		}

		// Multi-character operators (all are exactly 2 chars)
		found := false
		for _, op := range multiCharOps {
			if l.peek() == op[0] && l.peekN(1) == op[1] {
				l.advance()
				l.advance()
				l.emitToken(PUNCT, op)
				found = true
				break
			}
		}
		if found {
			continue
		}

		// Single-character operators/punctuation
		if singleCharOps[ch] {
			l.advance()
			l.emitToken(PUNCT, string(ch))
			continue
		}

		l.error(fmt.Sprintf("unexpected character: %c (0x%02X)", ch, ch))
	}

	// Check for unclosed #if
	if len(l.ifStack) > 0 {
		l.error("unterminated #if")
	}

	l.output.Flush()
}

// Handle const declaration - parse and record the constant value
// Syntax: const TypeSpecifier identifier = ConstExpr ;
// Or: const TypeSpecifier identifier [ ] = ArrayInit ;
// Or: const TypeSpecifier identifier [ ConstExpr ] = ArrayInit ;
func (l *Lexer) handleConstDecl() {
	l.emitToken(KEY, "const")

	l.skipWhitespace()

	// Handle pointer types (@)
	for l.peek() == '@' {
		l.advance()
		l.emitToken(PUNCT, "@")
		l.skipWhitespace()
	}

	// Get type name
	if !isLetter(l.peek()) {
		l.error("expected type after 'const'")
	}
	typeName := l.scanIdentifier()
	if keywords[typeName] {
		l.emitToken(KEY, typeName)
	} else {
		l.emitToken(ID, typeName)
	}

	l.skipWhitespace()

	// Get identifier name
	if !isLetter(l.peek()) {
		l.error("expected identifier in const declaration")
	}
	name := l.scanIdentifier()
	l.emitToken(ID, name)

	l.skipWhitespace()

	// Check for array dimension
	isArray := false
	if l.peek() == '[' {
		isArray = true
		l.advance()
		l.emitToken(PUNCT, "[")
		l.skipWhitespace()

		// Check for empty brackets [] (inferred size)
		if l.peek() == ']' {
			// Empty brackets - inferred size from initializer
			l.advance()
			l.emitToken(PUNCT, "]")
			l.skipWhitespace()
		} else {
			// Parse constant expression for array dimension
			val := l.parseConstExpr()
			l.emitToken(LIT, fmt.Sprintf("0x%04X", val&0xFFFF))

			l.skipWhitespace()
			if l.peek() != ']' {
				l.error("expected ']' after array dimension")
			}
			l.advance()
			l.emitToken(PUNCT, "]")
			l.skipWhitespace()
		}
	}

	// Expect '='
	if l.peek() != '=' {
		l.error("expected '=' in const declaration")
	}
	l.advance()
	l.emitToken(PUNCT, "=")

	l.skipWhitespace()

	if isArray {
		// Const array - don't parse as constant expression, let parser handle it
		// Just continue with normal lexing for the initializer
		return
	}

	// Parse constant expression for scalar const
	val := l.parseConstExpr()

	// Store in constant table
	l.constants[name] = val

	// Emit the folded literal value
	l.emitToken(LIT, fmt.Sprintf("0x%04X", val&0xFFFF))

	l.skipWhitespace()

	// Expect ';'
	if l.peek() != ';' {
		l.error("expected ';' after const declaration")
	}
	l.advance()
	l.emitToken(PUNCT, ";")
}

// Handle var declaration - fold array dimension if present
func (l *Lexer) handleVarDecl() {
	l.emitToken(KEY, "var")

	l.skipWhitespace()

	// Handle pointer types (@)
	for l.peek() == '@' {
		l.advance()
		l.emitToken(PUNCT, "@")
		l.skipWhitespace()
	}

	// Get type name
	if !isLetter(l.peek()) {
		l.error("expected type after 'var'")
	}
	typeName := l.scanIdentifier()
	if keywords[typeName] {
		l.emitToken(KEY, typeName)
	} else {
		l.emitToken(ID, typeName)
	}

	l.skipWhitespace()

	// Get variable name
	if !isLetter(l.peek()) {
		l.error("expected identifier in var declaration")
	}
	varName := l.scanIdentifier()
	l.emitToken(ID, varName)

	l.skipWhitespace()

	// Check for array dimension
	if l.peek() == '[' {
		l.advance()
		l.emitToken(PUNCT, "[")
		l.skipWhitespace()

		// Check for empty brackets [] (inferred size)
		if l.peek() == ']' {
			// Empty brackets - inferred size from initializer
			l.advance()
			l.emitToken(PUNCT, "]")
			l.skipWhitespace()
		} else {
			// Parse constant expression for array dimension
			val := l.parseConstExpr()
			l.emitToken(LIT, fmt.Sprintf("0x%04X", val&0xFFFF))

			l.skipWhitespace()
			if l.peek() != ']' {
				l.error("expected ']' after array dimension")
			}
			l.advance()
			l.emitToken(PUNCT, "]")
			l.skipWhitespace()
		}
	}

	// Check for initializer
	if l.peek() == '=' {
		l.advance()
		l.emitToken(PUNCT, "=")
		// After '=' is a runtime expression, not a constant expression
		// So we just continue with normal lexing - the main loop will handle it
		return
	}

	// Expect ';'
	if l.peek() == ';' {
		l.advance()
		l.emitToken(PUNCT, ";")
	}
}

// Handle struct declaration - fold array dimensions in members
func (l *Lexer) handleStructDecl() {
	l.emitToken(KEY, "struct")

	l.skipWhitespace()

	// Get struct name
	if !isLetter(l.peek()) {
		l.error("expected identifier after 'struct'")
	}
	name := l.scanIdentifier()
	l.emitToken(ID, name)

	l.skipWhitespace()

	// Expect '{'
	if l.peek() != '{' {
		l.error("expected '{' in struct declaration")
	}
	l.advance()
	l.emitToken(PUNCT, "{")

	// Parse struct members
	for {
		l.skipWhitespace()

		if l.peek() == '}' {
			l.advance()
			l.emitToken(PUNCT, "}")
			break
		}

		// Handle pointer types (@)
		for l.peek() == '@' {
			l.advance()
			l.emitToken(PUNCT, "@")
			l.skipWhitespace()
		}

		// Get type name
		if !isLetter(l.peek()) {
			l.error("expected type in struct member")
		}
		typeName := l.scanIdentifier()
		if keywords[typeName] {
			l.emitToken(KEY, typeName)
		} else {
			l.emitToken(ID, typeName)
		}

		l.skipWhitespace()

		// Get member name
		if !isLetter(l.peek()) {
			l.error("expected identifier in struct member")
		}
		memberName := l.scanIdentifier()
		l.emitToken(ID, memberName)

		l.skipWhitespace()

		// Check for array dimension
		if l.peek() == '[' {
			l.advance()
			l.emitToken(PUNCT, "[")
			l.skipWhitespace()

			// Parse constant expression for array dimension
			val := l.parseConstExpr()
			l.emitToken(LIT, fmt.Sprintf("0x%04X", val&0xFFFF))

			l.skipWhitespace()
			if l.peek() != ']' {
				l.error("expected ']' after array dimension")
			}
			l.advance()
			l.emitToken(PUNCT, "]")
			l.skipWhitespace()
		}

		// Expect ';'
		if l.peek() != ';' {
			l.error("expected ';' after struct member")
		}
		l.advance()
		l.emitToken(PUNCT, ";")
	}

	l.skipWhitespace()

	// Expect ';' after struct
	if l.peek() == ';' {
		l.advance()
		l.emitToken(PUNCT, ";")
	}
}

func main() {
	// Get filename from command line argument, default to "stdin"
	filename := "stdin"
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}

	// Create buffered output
	output := bufio.NewWriter(os.Stdout)

	// Run lexer - read from stdin via buffered reader
	lexer := NewLexer(os.Stdin, filename, output)
	lexer.Run()
}
