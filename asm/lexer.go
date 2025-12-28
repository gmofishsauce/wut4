package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

/* Tokenize a single line */
func tokenizeLine(line string, lineNum int) []Token {
	tokens := make([]Token, 0, 16)
	i := 0
	n := len(line)

	for i < n {
		/* Skip whitespace */
		if line[i] == ' ' || line[i] == '\t' {
			i++
			continue
		}

		/* Comment - rest of line */
		if line[i] == ';' {
			break
		}

		/* Colon */
		if line[i] == ':' {
			tokens = append(tokens, Token{typ: TOK_COLON, line: lineNum, col: i})
			i++
			continue
		}

		/* Comma */
		if line[i] == ',' {
			tokens = append(tokens, Token{typ: TOK_COMMA, line: lineNum, col: i})
			i++
			continue
		}

		/* Operators */
		if line[i] == '+' {
			tokens = append(tokens, Token{typ: TOK_PLUS, line: lineNum, col: i})
			i++
			continue
		}
		if line[i] == '-' {
			/* Check if this is a negative number */
			if i+1 < n && unicode.IsDigit(rune(line[i+1])) {
				/* Fall through to number parsing below */
			} else {
				tokens = append(tokens, Token{typ: TOK_MINUS, line: lineNum, col: i})
				i++
				continue
			}
		}
		if line[i] == '*' {
			tokens = append(tokens, Token{typ: TOK_STAR, line: lineNum, col: i})
			i++
			continue
		}
		if line[i] == '/' {
			tokens = append(tokens, Token{typ: TOK_SLASH, line: lineNum, col: i})
			i++
			continue
		}
		if line[i] == '(' {
			tokens = append(tokens, Token{typ: TOK_LPAREN, line: lineNum, col: i})
			i++
			continue
		}
		if line[i] == ')' {
			tokens = append(tokens, Token{typ: TOK_RPAREN, line: lineNum, col: i})
			i++
			continue
		}

		/* String literal */
		if line[i] == '"' {
			start := i
			i++
			for i < n && line[i] != '"' {
				if line[i] == '\\' && i+1 < n {
					i += 2
				} else {
					i++
				}
			}
			if i >= n {
				/* Unterminated string */
				tokens = append(tokens, Token{typ: TOK_STRING, value: line[start:], line: lineNum, col: start})
			} else {
				i++ /* Skip closing quote */
				tokens = append(tokens, Token{typ: TOK_STRING, value: line[start:i], line: lineNum, col: start})
			}
			continue
		}

		/* Number */
		if unicode.IsDigit(rune(line[i])) || (line[i] == '-' && i+1 < n && unicode.IsDigit(rune(line[i+1]))) {
			start := i
			if line[i] == '-' {
				i++
			}
			/* Hex number */
			if i+1 < n && line[i] == '0' && (line[i+1] == 'x' || line[i+1] == 'X') {
				i += 2
				for i < n && isHexDigit(rune(line[i])) {
					i++
				}
			} else {
				/* Decimal number */
				for i < n && unicode.IsDigit(rune(line[i])) {
					i++
				}
			}
			numStr := line[start:i]
			val := parseNumber(numStr)
			tokens = append(tokens, Token{typ: TOK_NUMBER, value: numStr, intval: val, line: lineNum, col: start})
			continue
		}

		/* Identifier or label */
		if unicode.IsLetter(rune(line[i])) || line[i] == '_' || line[i] == '.' {
			start := i
			for i < n && (unicode.IsLetter(rune(line[i])) || unicode.IsDigit(rune(line[i])) || line[i] == '_' || line[i] == '.') {
				i++
			}
			ident := line[start:i]
			tokens = append(tokens, Token{typ: TOK_IDENT, value: ident, line: lineNum, col: start})
			continue
		}

		/* Unknown character - skip it */
		i++
	}

	return tokens
}

func isHexDigit(r rune) bool {
	if unicode.IsDigit(r) {
		return true
	}
	if r >= 'a' && r <= 'f' {
		return true
	}
	if r >= 'A' && r <= 'F' {
		return true
	}
	return false
}

func parseNumber(s string) int {
	s = strings.TrimSpace(s)
	var val int64
	var err error

	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		val, err = strconv.ParseInt(s[2:], 16, 64)
	} else if strings.HasPrefix(s, "-0x") || strings.HasPrefix(s, "-0X") {
		val, err = strconv.ParseInt("-"+s[3:], 16, 64)
	} else {
		val, err = strconv.ParseInt(s, 10, 64)
	}

	if err != nil {
		return 0
	}
	return int(val)
}

/* Parse a string literal, handling escape sequences */
func parseString(s string) string {
	if len(s) < 2 {
		return s
	}
	/* Remove quotes */
	s = s[1 : len(s)-1]

	/* Handle escape sequences */
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			case '"':
				result.WriteByte('"')
			case '0':
				result.WriteByte(0)
			default:
				result.WriteByte(s[i+1])
			}
			i += 2
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

/* Print tokens for debugging */
func printTokens(tokens []Token) {
	for i := 0; i < len(tokens); i++ {
		t := &tokens[i]
		fmt.Printf("Token[%d]: ", i)
		switch t.typ {
		case TOK_EOF:
			fmt.Printf("EOF\n")
		case TOK_IDENT:
			fmt.Printf("IDENT '%s'\n", t.value)
		case TOK_NUMBER:
			fmt.Printf("NUMBER %d (0x%x)\n", t.intval, t.intval)
		case TOK_STRING:
			fmt.Printf("STRING %s\n", t.value)
		case TOK_COMMA:
			fmt.Printf("COMMA\n")
		case TOK_COLON:
			fmt.Printf("COLON\n")
		case TOK_PLUS:
			fmt.Printf("PLUS\n")
		case TOK_MINUS:
			fmt.Printf("MINUS\n")
		case TOK_STAR:
			fmt.Printf("STAR\n")
		case TOK_SLASH:
			fmt.Printf("SLASH\n")
		default:
			fmt.Printf("OTHER (type %d)\n", t.typ)
		}
	}
}
