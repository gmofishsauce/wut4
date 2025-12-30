package main

import (
	"fmt"
	"strings"
)

type Parser struct {
	lex    *Lexer
	current *Token
	asm    *Assembler
}

func newParser(input string, asm *Assembler) *Parser {
	p := &Parser{
		lex: newLexer(input),
		asm: asm,
	}
	p.advance()
	return p
}

func (p *Parser) advance() error {
	tok, err := p.lex.nextToken()
	if err != nil {
		return err
	}
	p.current = tok
	return nil
}

func (p *Parser) parseDirective(name string) (*Statement, error) {
	stmt := &Statement{
		line:   p.current.line,
		hasDir: true,
	}

	switch name {
	case ".align":
		stmt.directive = DIR_ALIGN
		if err := p.advance(); err != nil {
			return nil, err
		}
		arg, err := p.parseArgument()
		if err != nil {
			return nil, err
		}
		stmt.args = []string{arg}
		stmt.numArgs = 1

	case ".bytes":
		stmt.directive = DIR_BYTES
		if err := p.advance(); err != nil {
			return nil, err
		}
		for p.current.typ != TOK_NEWLINE && p.current.typ != TOK_EOF {
			arg, err := p.parseArgument()
			if err != nil {
				return nil, err
			}
			stmt.args = append(stmt.args, arg)
			stmt.numArgs++
			if p.current.typ == TOK_COMMA {
				p.advance()
			}
		}

	case ".words":
		stmt.directive = DIR_WORDS
		if err := p.advance(); err != nil {
			return nil, err
		}
		for p.current.typ != TOK_NEWLINE && p.current.typ != TOK_EOF {
			arg, err := p.parseArgument()
			if err != nil {
				return nil, err
			}
			stmt.args = append(stmt.args, arg)
			stmt.numArgs++
			if p.current.typ == TOK_COMMA {
				p.advance()
			}
		}

	case ".space":
		stmt.directive = DIR_SPACE
		if err := p.advance(); err != nil {
			return nil, err
		}
		arg, err := p.parseArgument()
		if err != nil {
			return nil, err
		}
		stmt.args = []string{arg}
		stmt.numArgs = 1

	case ".code":
		stmt.directive = DIR_CODE

	case ".data":
		stmt.directive = DIR_DATA

	case ".set":
		stmt.directive = DIR_SET
		if err := p.advance(); err != nil {
			return nil, err
		}
		if p.current.typ != TOK_IDENT {
			return nil, fmt.Errorf("expected symbol name after .set")
		}
		symName := p.current.text
		if err := p.advance(); err != nil {
			return nil, err
		}
		if p.current.typ == TOK_COMMA {
			p.advance()
		}
		arg, err := p.parseArgument()
		if err != nil {
			return nil, err
		}
		stmt.args = []string{symName, arg}
		stmt.numArgs = 2

	default:
		return nil, fmt.Errorf("unknown directive: %s", name)
	}

	return stmt, nil
}

func (p *Parser) parseArgument() (string, error) {
	var tokens []*Token

	/* Handle string literals - keep quotes for later identification */
	if p.current.typ == TOK_STRING {
		arg := "\"" + p.current.text + "\""
		p.advance()
		return arg, nil
	}

	/* Collect tokens for expression */
	parenDepth := 0
	for {
		if p.current.typ == TOK_NEWLINE || p.current.typ == TOK_EOF {
			break
		}
		if p.current.typ == TOK_COMMA && parenDepth == 0 {
			break
		}

		/* Check if we've collected a complete simple argument */
		if len(tokens) > 0 && parenDepth == 0 {
			lastTok := tokens[len(tokens)-1]

			/* If we collected a single identifier or number (complete arg) and next is not a binary operator */
			if len(tokens) == 1 && (lastTok.typ == TOK_IDENT || lastTok.typ == TOK_NUMBER) {
				/* Check if current token would continue the expression */
				isBinaryOp := p.current.typ == TOK_PLUS || p.current.typ == TOK_STAR ||
					p.current.typ == TOK_SLASH || p.current.typ == TOK_AMP ||
					p.current.typ == TOK_PIPE || p.current.typ == TOK_LSHIFT ||
					p.current.typ == TOK_RSHIFT
				/* Minus is tricky - it's binary after an operand, but could start a new negative number */
				/* For simplicity, treat minus after a simple arg as starting a new argument */
				if !isBinaryOp && p.current.typ != TOK_LPAREN {
					break
				}
			}

			/* If we have tokens and next is an identifier or number, check if it's a new argument */
			if p.current.typ == TOK_IDENT || p.current.typ == TOK_NUMBER {
				/* Check if previous token was an operator (meaning this is part of expression) */
				isOperator := lastTok.typ == TOK_PLUS || lastTok.typ == TOK_MINUS ||
					lastTok.typ == TOK_STAR || lastTok.typ == TOK_SLASH ||
					lastTok.typ == TOK_AMP || lastTok.typ == TOK_PIPE ||
					lastTok.typ == TOK_TILDE || lastTok.typ == TOK_LSHIFT ||
					lastTok.typ == TOK_RSHIFT || lastTok.typ == TOK_LPAREN
				if !isOperator {
					/* This is a new argument, stop here */
					break
				}
			}
		}

		if p.current.typ == TOK_LPAREN {
			parenDepth++
		}
		if p.current.typ == TOK_RPAREN {
			parenDepth--
			if parenDepth < 0 {
				break
			}
		}
		tokens = append(tokens, p.current)
		if err := p.advance(); err != nil {
			return "", err
		}
	}

	if len(tokens) == 0 {
		return "", fmt.Errorf("expected argument")
	}

	/* If it's a simple identifier, return it */
	if len(tokens) == 1 && tokens[0].typ == TOK_IDENT {
		return tokens[0].text, nil
	}

	/* Otherwise, it's an expression - encode it */
	var sb strings.Builder
	for _, tok := range tokens {
		if tok.typ == TOK_NUMBER {
			sb.WriteString(fmt.Sprintf("%d", tok.value))
		} else {
			sb.WriteString(tok.text)
		}
		sb.WriteString(" ")
	}
	return strings.TrimSpace(sb.String()), nil
}

func (p *Parser) parseStatement() (*Statement, error) {
	/* Skip blank lines and comments */
	for p.current.typ == TOK_NEWLINE {
		if err := p.advance(); err != nil {
			return nil, err
		}
	}

	if p.current.typ == TOK_EOF {
		return nil, nil
	}

	stmt := &Statement{
		line: p.current.line,
	}

	/* Check for label */
	if p.current.typ == TOK_LABEL {
		stmt.label = p.current.text
		if err := p.advance(); err != nil {
			return nil, err
		}
		/* Skip whitespace to next token */
		if p.current.typ == TOK_NEWLINE || p.current.typ == TOK_EOF {
			return stmt, nil
		}
	}

	/* Check for directive or instruction */
	if p.current.typ == TOK_IDENT {
		name := p.current.text
		if strings.HasPrefix(name, ".") {
			/* It's a directive */
			dirStmt, err := p.parseDirective(name)
			if err != nil {
				return nil, err
			}
			dirStmt.label = stmt.label
			return dirStmt, nil
		}

		/* It's an instruction */
		stmt.instr = name
		stmt.hasInstr = true
		if err := p.advance(); err != nil {
			return nil, err
		}

		/* Parse instruction arguments */
		for p.current.typ != TOK_NEWLINE && p.current.typ != TOK_EOF {
			arg, err := p.parseArgument()
			if err != nil {
				return nil, err
			}
			stmt.args = append(stmt.args, arg)
			stmt.numArgs++
			if p.current.typ == TOK_COMMA {
				p.advance()
			}
		}
	}

	return stmt, nil
}
