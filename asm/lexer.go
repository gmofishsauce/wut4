package main

import (
	"fmt"
	"strconv"
	"strings"
)

type Lexer struct {
	input  string
	pos    int
	line   int
	column int
	ch     byte
}

func newLexer(input string) *Lexer {
	lex := &Lexer{
		input:  input,
		pos:    0,
		line:   1,
		column: 1,
		ch:     0,
	}
	if len(input) > 0 {
		lex.ch = input[0]
	}
	return lex
}

func (lex *Lexer) advance() {
	if lex.pos < len(lex.input)-1 {
		lex.pos++
		lex.ch = lex.input[lex.pos]
		lex.column++
		if lex.ch == '\n' {
			lex.line++
			lex.column = 1
		}
	} else {
		lex.pos = len(lex.input)
		lex.ch = 0
	}
}

func (lex *Lexer) peek() byte {
	if lex.pos < len(lex.input)-1 {
		return lex.input[lex.pos+1]
	}
	return 0
}

func (lex *Lexer) skipWhitespace() {
	for lex.ch == ' ' || lex.ch == '\t' || lex.ch == '\r' {
		lex.advance()
	}
}

func (lex *Lexer) skipComment() {
	/* Skip until end of line */
	for lex.ch != '\n' && lex.ch != 0 {
		lex.advance()
	}
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

func (lex *Lexer) readIdentifier() string {
	start := lex.pos
	/* Handle leading '.' for directives */
	if lex.ch == '.' {
		lex.advance()
	}
	for isLetter(lex.ch) || isDigit(lex.ch) {
		lex.advance()
	}
	return lex.input[start:lex.pos]
}

func (lex *Lexer) readNumber() (int, error) {
	start := lex.pos
	base := 10

	if lex.ch == '0' && lex.pos < len(lex.input)-1 {
		next := lex.peek()
		if next == 'x' || next == 'X' {
			/* Hexadecimal */
			base = 16
			lex.advance() /* skip '0' */
			lex.advance() /* skip 'x' */
			start = lex.pos
		} else if next == 'b' || next == 'B' {
			/* Binary */
			base = 2
			lex.advance() /* skip '0' */
			lex.advance() /* skip 'b' */
			start = lex.pos
		} else if next == 'o' || next == 'O' {
			/* Octal */
			base = 8
			lex.advance() /* skip '0' */
			lex.advance() /* skip 'o' */
			start = lex.pos
		}
	}

	/* Read digits, allowing underscores */
	for {
		if base == 16 && isHexDigit(lex.ch) {
			lex.advance()
		} else if base == 10 && isDigit(lex.ch) {
			lex.advance()
		} else if base == 8 && lex.ch >= '0' && lex.ch <= '7' {
			lex.advance()
		} else if base == 2 && (lex.ch == '0' || lex.ch == '1') {
			lex.advance()
		} else if lex.ch == '_' {
			lex.advance() /* skip underscores */
		} else {
			break
		}
	}

	numStr := strings.ReplaceAll(lex.input[start:lex.pos], "_", "")
	val, err := strconv.ParseInt(numStr, base, 32)
	if err != nil {
		return 0, err
	}
	return int(val), nil
}

func (lex *Lexer) readString() (string, error) {
	var result strings.Builder
	lex.advance() /* skip opening quote */

	for lex.ch != '"' && lex.ch != 0 {
		if lex.ch == '\\' {
			lex.advance()
			switch lex.ch {
			case '0':
				result.WriteByte(0)
			case 'n':
				result.WriteByte('\n')
			case 'r':
				result.WriteByte('\r')
			case 'b':
				result.WriteByte('\b')
			case 't':
				result.WriteByte('\t')
			case '\\':
				result.WriteByte('\\')
			case '"':
				result.WriteByte('"')
			default:
				/* Check for hex escape \xNN */
				if lex.ch == 'x' || lex.ch == 'X' {
					lex.advance()
					hex1 := lex.ch
					lex.advance()
					hex2 := lex.ch
					hexStr := string([]byte{hex1, hex2})
					val, err := strconv.ParseInt(hexStr, 16, 32)
					if err != nil {
						return "", fmt.Errorf("invalid hex escape: \\x%s", hexStr)
					}
					result.WriteByte(byte(val))
				} else {
					return "", fmt.Errorf("invalid escape sequence: \\%c", lex.ch)
				}
			}
			lex.advance()
		} else {
			result.WriteByte(lex.ch)
			lex.advance()
		}
	}

	if lex.ch != '"' {
		return "", fmt.Errorf("unterminated string")
	}
	lex.advance() /* skip closing quote */

	return result.String(), nil
}

func (lex *Lexer) nextToken() (*Token, error) {
	lex.skipWhitespace()

	tok := &Token{
		line:   lex.line,
		column: lex.column,
	}

	if lex.ch == 0 {
		tok.typ = TOK_EOF
		return tok, nil
	}

	if lex.ch == ';' {
		lex.skipComment()
		if lex.ch == '\n' {
			tok.typ = TOK_NEWLINE
			lex.advance()
			return tok, nil
		}
		tok.typ = TOK_EOF
		return tok, nil
	}

	if lex.ch == '\n' {
		tok.typ = TOK_NEWLINE
		lex.advance()
		return tok, nil
	}

	if lex.ch == ',' {
		tok.typ = TOK_COMMA
		tok.text = ","
		lex.advance()
		return tok, nil
	}

	if lex.ch == '(' {
		tok.typ = TOK_LPAREN
		tok.text = "("
		lex.advance()
		return tok, nil
	}

	if lex.ch == ')' {
		tok.typ = TOK_RPAREN
		tok.text = ")"
		lex.advance()
		return tok, nil
	}

	if lex.ch == '+' {
		tok.typ = TOK_PLUS
		tok.text = "+"
		lex.advance()
		return tok, nil
	}

	if lex.ch == '-' {
		tok.typ = TOK_MINUS
		tok.text = "-"
		lex.advance()
		return tok, nil
	}

	if lex.ch == '*' {
		tok.typ = TOK_STAR
		tok.text = "*"
		lex.advance()
		return tok, nil
	}

	if lex.ch == '/' {
		tok.typ = TOK_SLASH
		tok.text = "/"
		lex.advance()
		return tok, nil
	}

	if lex.ch == '&' {
		tok.typ = TOK_AMP
		tok.text = "&"
		lex.advance()
		return tok, nil
	}

	if lex.ch == '|' {
		tok.typ = TOK_PIPE
		tok.text = "|"
		lex.advance()
		return tok, nil
	}

	if lex.ch == '~' {
		tok.typ = TOK_TILDE
		tok.text = "~"
		lex.advance()
		return tok, nil
	}

	if lex.ch == '<' && lex.peek() == '<' {
		tok.typ = TOK_LSHIFT
		tok.text = "<<"
		lex.advance()
		lex.advance()
		return tok, nil
	}

	if lex.ch == '>' && lex.peek() == '>' {
		tok.typ = TOK_RSHIFT
		tok.text = ">>"
		lex.advance()
		lex.advance()
		return tok, nil
	}

	if lex.ch == '$' {
		tok.typ = TOK_DOLLAR
		tok.text = "$"
		lex.advance()
		return tok, nil
	}

	if lex.ch == '"' {
		str, err := lex.readString()
		if err != nil {
			return nil, err
		}
		tok.typ = TOK_STRING
		tok.text = str
		return tok, nil
	}

	if isLetter(lex.ch) || lex.ch == '.' {
		ident := lex.readIdentifier()
		/* Check if this is a label (followed by colon) */
		lex.skipWhitespace()
		if lex.ch == ':' {
			tok.typ = TOK_LABEL
			tok.text = ident
			lex.advance() /* skip colon */
		} else {
			tok.typ = TOK_IDENT
			tok.text = ident
		}
		return tok, nil
	}

	if isDigit(lex.ch) {
		val, err := lex.readNumber()
		if err != nil {
			return nil, err
		}
		tok.typ = TOK_NUMBER
		tok.value = val
		tok.text = fmt.Sprintf("%d", val)
		return tok, nil
	}

	return nil, fmt.Errorf("unexpected character: %c", lex.ch)
}
