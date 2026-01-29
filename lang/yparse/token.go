// YAPL Parser - Token Reader
// Reads token stream from Pass 1 (lexer) output

package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Token categories (from lexer)
const (
	TokKEY   = "KEY"
	TokID    = "ID"
	TokPUNCT = "PUNCT"
	TokLIT   = "LIT"
	TokEOF   = "EOF"
)

// Token represents a lexical token from Pass 1
type Token struct {
	Num      int    // token number (for debugging)
	Category string // KEY, ID, PUNCT, LIT, EOF
	Value    string // the token value
	Line     int    // source line number
	File     string // source file name
}

func (t Token) String() string {
	return fmt.Sprintf("%s:%d: %s %q", t.File, t.Line, t.Category, t.Value)
}

// IsKeyword returns true if token is the specified keyword
func (t Token) IsKeyword(kw string) bool {
	return t.Category == TokKEY && t.Value == kw
}

// IsPunct returns true if token is the specified punctuation
func (t Token) IsPunct(p string) bool {
	return t.Category == TokPUNCT && t.Value == p
}

// TokenReader reads tokens from the lexer output
type TokenReader struct {
	scanner  *bufio.Scanner
	filename string
	line     int
	current  Token
	peeked   bool
	eof      bool
}

// NewTokenReader creates a new token reader from an io.Reader
func NewTokenReader(r io.Reader) *TokenReader {
	return &TokenReader{
		scanner:  bufio.NewScanner(r),
		filename: "unknown",
		line:     1,
		peeked:   false,
		eof:      false,
	}
}

// readNextToken reads the next token from the input, handling directives
func (tr *TokenReader) readNextToken() Token {
	for tr.scanner.Scan() {
		text := strings.TrimSpace(tr.scanner.Text())

		// Skip empty lines and comments
		if text == "" || strings.HasPrefix(text, "#") && !strings.HasPrefix(text, "#file") && !strings.HasPrefix(text, "#line") {
			continue
		}

		// Handle #file directive
		if strings.HasPrefix(text, "#file ") {
			tr.filename = strings.TrimPrefix(text, "#file ")
			continue
		}

		// Handle #line directive
		if strings.HasPrefix(text, "#line ") {
			lineStr := strings.TrimPrefix(text, "#line ")
			if n, err := strconv.Atoi(lineStr); err == nil {
				tr.line = n
			}
			continue
		}

		// Parse token line: "num, CATEGORY, value"
		tok := tr.parseTokenLine(text)
		tok.File = tr.filename
		tok.Line = tr.line
		return tok
	}

	// End of input
	tr.eof = true
	return Token{
		Category: TokEOF,
		Value:    "",
		File:     tr.filename,
		Line:     tr.line,
	}
}

// parseTokenLine parses a token line like "1, KEY, const"
func (tr *TokenReader) parseTokenLine(text string) Token {
	// Format: "num, CATEGORY, value"
	// Split on ", " (comma-space)
	parts := strings.SplitN(text, ", ", 3)
	if len(parts) != 3 {
		// Malformed token line - return an error token
		return Token{
			Category: TokEOF,
			Value:    fmt.Sprintf("malformed token: %s", text),
		}
	}

	num, _ := strconv.Atoi(parts[0])
	category := parts[1]
	value := parts[2]

	return Token{
		Num:      num,
		Category: category,
		Value:    value,
	}
}

// Peek returns the next token without consuming it
func (tr *TokenReader) Peek() Token {
	if !tr.peeked {
		tr.current = tr.readNextToken()
		tr.peeked = true
	}
	return tr.current
}

// Next returns the next token and advances
func (tr *TokenReader) Next() Token {
	tok := tr.Peek()
	tr.peeked = false
	return tok
}

// AtEOF returns true if at end of input
func (tr *TokenReader) AtEOF() bool {
	return tr.Peek().Category == TokEOF
}

// Expect consumes a token and returns error if it doesn't match
func (tr *TokenReader) Expect(category, value string) (Token, error) {
	tok := tr.Next()
	if tok.Category != category || tok.Value != value {
		return tok, fmt.Errorf("%s:%d: expected %s %q, got %s %q",
			tok.File, tok.Line, category, value, tok.Category, tok.Value)
	}
	return tok, nil
}

// ExpectKeyword consumes a keyword token
func (tr *TokenReader) ExpectKeyword(kw string) (Token, error) {
	return tr.Expect(TokKEY, kw)
}

// ExpectPunct consumes a punctuation token
func (tr *TokenReader) ExpectPunct(p string) (Token, error) {
	return tr.Expect(TokPUNCT, p)
}

// ExpectID consumes an identifier token
func (tr *TokenReader) ExpectID() (Token, error) {
	tok := tr.Next()
	if tok.Category != TokID {
		return tok, fmt.Errorf("%s:%d: expected identifier, got %s %q",
			tok.File, tok.Line, tok.Category, tok.Value)
	}
	return tok, nil
}

// CurrentFile returns the current source file name
func (tr *TokenReader) CurrentFile() string {
	return tr.filename
}

// CurrentLine returns the current source line number
func (tr *TokenReader) CurrentLine() int {
	return tr.line
}
