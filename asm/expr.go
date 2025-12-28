package main

import (
	"fmt"
)

/* Evaluate a constant expression from tokens */
func (a *Assembler) evalExpr(tokens []Token, start int, end int) (int, error) {
	if start >= end {
		return 0, fmt.Errorf("empty expression")
	}

	/* Single token */
	if end-start == 1 {
		t := &tokens[start]
		switch t.typ {
		case TOK_NUMBER:
			return t.intval, nil
		case TOK_IDENT:
			/* Check if it's a symbol */
			if val, ok := a.symbols[t.value]; ok {
				return val, nil
			}
			/* Check if it's a label */
			if val, ok := a.labels[t.value]; ok {
				return val, nil
			}
			return 0, fmt.Errorf("undefined symbol: %s", t.value)
		default:
			return 0, fmt.Errorf("unexpected token in expression")
		}
	}

	/* Handle binary operators - simple left-to-right evaluation */
	/* Find lowest precedence operator */
	depth := 0
	opPos := -1
	opPrec := 999

	for i := start; i < end; i++ {
		t := &tokens[i]
		if t.typ == TOK_LPAREN {
			depth++
		} else if t.typ == TOK_RPAREN {
			depth--
		} else if depth == 0 {
			prec := getOpPrecedence(t.typ)
			if prec >= 0 && prec <= opPrec {
				opPos = i
				opPrec = prec
			}
		}
	}

	/* If we found an operator, split and recurse */
	if opPos >= 0 {
		left, err := a.evalExpr(tokens, start, opPos)
		if err != nil {
			return 0, err
		}
		right, err := a.evalExpr(tokens, opPos+1, end)
		if err != nil {
			return 0, err
		}

		op := tokens[opPos].typ
		switch op {
		case TOK_PLUS:
			return left + right, nil
		case TOK_MINUS:
			return left - right, nil
		case TOK_STAR:
			return left * right, nil
		case TOK_SLASH:
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		default:
			return 0, fmt.Errorf("unknown operator")
		}
	}

	/* Handle parentheses */
	if tokens[start].typ == TOK_LPAREN && tokens[end-1].typ == TOK_RPAREN {
		return a.evalExpr(tokens, start+1, end-1)
	}

	return 0, fmt.Errorf("invalid expression")
}

func getOpPrecedence(typ int) int {
	switch typ {
	case TOK_PLUS, TOK_MINUS:
		return 1
	case TOK_STAR, TOK_SLASH:
		return 2
	default:
		return -1
	}
}

/* Try to evaluate an expression, return default value if it contains undefined symbols */
func (a *Assembler) tryEvalExpr(tokens []Token, start int, end int, defaultVal int) int {
	val, err := a.evalExpr(tokens, start, end)
	if err != nil {
		return defaultVal
	}
	return val
}
