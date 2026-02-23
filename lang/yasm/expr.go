package main

import (
	"fmt"
)

/* Pratt parser for constant expressions */

/* Precedence levels */
const (
	PREC_LOWEST = iota
	PREC_BITWISE_OR
	PREC_ADDITIVE
	PREC_MULTIPLICATIVE
	PREC_UNARY
)

type ExprParser struct {
	tokens    []*Token
	pos       int
	current   *Token
	asm       *Assembler
	allowFwd  bool /* allow forward references */
}

func newExprParser(tokens []*Token, asm *Assembler, allowFwd bool) *ExprParser {
	ep := &ExprParser{
		tokens:   tokens,
		pos:      0,
		asm:      asm,
		allowFwd: allowFwd,
	}
	if len(tokens) > 0 {
		ep.current = tokens[0]
	}
	return ep
}

func (ep *ExprParser) advance() {
	if ep.pos < len(ep.tokens)-1 {
		ep.pos++
		ep.current = ep.tokens[ep.pos]
	}
}

func (ep *ExprParser) precedence(tok *Token) int {
	switch tok.typ {
	case TOK_PIPE:
		return PREC_BITWISE_OR
	case TOK_PLUS, TOK_MINUS:
		return PREC_ADDITIVE
	case TOK_STAR, TOK_SLASH, TOK_AMP, TOK_LSHIFT, TOK_RSHIFT:
		return PREC_MULTIPLICATIVE
	default:
		return PREC_LOWEST
	}
}

func (ep *ExprParser) parseExpr(prec int) (int, error) {
	left, err := ep.parsePrimary()
	if err != nil {
		return 0, err
	}

	for ep.current != nil && ep.precedence(ep.current) > prec {
		op := ep.current.typ
		opPrec := ep.precedence(ep.current)
		ep.advance()

		right, err := ep.parseExpr(opPrec)
		if err != nil {
			return 0, err
		}

		/* Apply operator */
		switch op {
		case TOK_PLUS:
			left = left + right
		case TOK_MINUS:
			left = left - right
		case TOK_STAR:
			left = left * right
		case TOK_SLASH:
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			left = left / right
		case TOK_AMP:
			left = left & right
		case TOK_PIPE:
			left = left | right
		case TOK_LSHIFT:
			left = left << uint(right)
		case TOK_RSHIFT:
			left = left >> uint(right)
		default:
			return 0, fmt.Errorf("unexpected operator in expression")
		}
	}

	return left, nil
}

func (ep *ExprParser) parsePrimary() (int, error) {
	tok := ep.current

	if tok == nil {
		return 0, fmt.Errorf("unexpected end of expression")
	}

	switch tok.typ {
	case TOK_NUMBER:
		val := tok.value
		ep.advance()
		return val, nil

	case TOK_DOLLAR:
		ep.advance()
		/* In bootstrap mode, only one PC value exists */
		if ep.asm.bootstrapMode {
			return ep.asm.codePC, nil
		}
		if ep.asm.currentSeg == SEG_CODE {
			return ep.asm.codePC, nil
		}
		return ep.asm.dataPC, nil

	case TOK_IDENT:
		name := tok.text
		ep.advance()
		/* Look up symbol */
		sym := ep.lookupSymbol(name)
		if sym == nil {
			/* In object mode pass 2, undefined symbols are external references.
			   Check this BEFORE allowFwd so we record the reference even when
			   the caller used allowFwd=true (which is used for pass-1 forward refs). */
			if ep.asm.objectMode && ep.asm.pass == 2 {
				ep.asm.addExternalSymbol(name)
				ep.asm.lastExternalRef = name
				return 0, nil
			}
			if ep.allowFwd {
				/* Forward reference in pass 1 - return 0 for now */
				return 0, nil
			}
			return 0, fmt.Errorf("undefined symbol: %s", name)
		}
		if !sym.defined && !ep.allowFwd {
			return 0, fmt.Errorf("symbol used before definition: %s", name)
		}
		return sym.value, nil

	case TOK_LPAREN:
		ep.advance()
		val, err := ep.parseExpr(PREC_LOWEST)
		if err != nil {
			return 0, err
		}
		if ep.current == nil || ep.current.typ != TOK_RPAREN {
			return 0, fmt.Errorf("expected ')' in expression")
		}
		ep.advance()
		return val, nil

	case TOK_MINUS:
		/* Unary minus */
		ep.advance()
		val, err := ep.parseExpr(PREC_UNARY)
		if err != nil {
			return 0, err
		}
		return -val, nil

	case TOK_TILDE:
		/* Bitwise NOT */
		ep.advance()
		val, err := ep.parseExpr(PREC_UNARY)
		if err != nil {
			return 0, err
		}
		return ^val, nil

	default:
		return 0, fmt.Errorf("unexpected token in expression: %s", tok.text)
	}
}

func (ep *ExprParser) lookupSymbol(name string) *Symbol {
	for i := 0; i < ep.asm.numSymbols; i++ {
		if ep.asm.symbols[i].name == name {
			return &ep.asm.symbols[i]
		}
	}
	return nil
}

func (ep *ExprParser) parse() (int, error) {
	return ep.parseExpr(PREC_LOWEST)
}
