/*
 * Copyright (c) Jeff Berkowitz 2025.
 *
 * This file is part of wut4, which is licensed under the Affero GPL.
 * Code generated by the transpiler is owned by the schematic owner.
 */
package main

import (
	"fmt"
	"strings"
)

// Token represents a token in an S-expression
type Token struct {
	Type  int
	Line  int
	Value string
}

const (
	SYMBOL = 1
	LPAREN = 2
	RPAREN = 3
)

// Tokenize takes an S-expression string and returns a slice of Tokens
// Unlike most S-expr tokenizers, this one includes an (approximate)
// source line number in each token for better error messages. 
func Tokenize(input string) ([]Token, error) {
	var tokens []Token
	var currentToken string
	var currentLine = 1
	var inString bool
	var escapeSeen bool

	// N.B. the escapeSeen state is managed by a single line of code near the
	// very end of this loop. It's easy to miss, but otherwise we'd need to
	// clear the escape state in many places, and it would be easy to miss one.
	for _, r := range input {
		switch r {
		case '(':
			if inString {
				currentToken += string(r)
			} else if currentToken != "" {
				// Open parens need to be preceded by white space. This rule is likely
				// incorrect for S-exprs in generally, but helps with KiCad exports.
				return nil, fmt.Errorf("open paren in symbol near line %d", currentLine)
			} else {
                tokens = append(tokens, Token{LPAREN, currentLine, "("})
            }
		case ')':
			if inString {
				currentToken += string(r)
			} else if currentToken != "" {
				tokens = append(tokens, Token{SYMBOL, currentLine, currentToken})
				currentToken = ""
				tokens = append(tokens, Token{RPAREN, currentLine, ")"})
			} else {
                tokens = append(tokens, Token{RPAREN, currentLine, ")"})
            }
		case '"':
			if escapeSeen {
				currentToken += string(r)
			} else {
				inString = !inString
				if currentToken != "" {
					tokens = append(tokens, Token{SYMBOL, currentLine, currentToken})
					currentToken = ""
				}
			}
		case ' ', '\t', '\n', '\r':
			if inString {
				currentToken += string(r)
			} else if currentToken != "" {
				tokens = append(tokens, Token{SYMBOL, currentLine, currentToken})
				currentToken = ""
			}
			if r == '\n' {
				currentLine++
			}
		case 'n':
			if inString && escapeSeen {
				currentToken += string('\n')
			} else {
				currentToken += string(r)
			}
		case 't':
			if inString && escapeSeen {
				currentToken += string('\t')
			} else {
				currentToken += string(r)
			}
		case '\\':
			if !inString {
				currentToken += string(r)
			} else if inString && escapeSeen {
				currentToken += string(r)
			}
		default:
			currentToken += string(r)
		}

		escapeSeen = inString && !escapeSeen && (r == '\\')
	}

    if currentToken != "" {
		return nil, fmt.Errorf("netlist has mismatched parens")
	}

	return tokens, nil
}

// Parser. Creates a tree of ModelNodes.

type ModelNode struct {
	Name string
	Value []string
	Parent *ModelNode
	Children []*ModelNode
}

func (m *ModelNode) String() string {
	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(m.Name)
	sb.WriteString(" ")
	if len(m.Value) == 0 {
		sb.WriteString("- ")
	} else {
		sb.WriteString("\"")
		sb.WriteString(m.Value[0])
		if len(m.Value) > 1 {
			sb.WriteString("...")
		}
		sb.WriteString("\" ")
	}
	if m.Parent == nil {
		sb.WriteString("- ")
	} else {
		sb.WriteString(m.Parent.Name)
	}
	// don't bother with children
	sb.WriteString("]")
	return sb.String()
}

const (
	INITIAL = 0
	OPENING = 1
	NEED_ANY = 2
)

// Parse hides the tokenizer from callers.
// The returned value is tree of ModelNode.
func parse(netlist string) (*ModelNode, error) {
	tokens, err := Tokenize(netlist)
	if err != nil {
		return nil, err
	}

	var state = INITIAL
	var currentNode *ModelNode
	var previousNode *ModelNode

	for _, token := range tokens {
		//fmt.Printf("Type: %d, Value: %s\n", token.Type, token.Value)

		switch (token.Type) {
		case LPAREN:
			if state == OPENING {
				// left paren following left paren with no "key" symbol between
				return nil, fmt.Errorf("unbalanced open near line %d", token.Line)
			}
			state = OPENING
		case RPAREN:
			if state != NEED_ANY || currentNode == nil {
				// This error will occur for "()", which may not be an error
				return nil, fmt.Errorf("unbalanced close near line %d", token.Line)
			}
			previousNode = currentNode
			currentNode = currentNode.Parent
		case SYMBOL:
			if state == OPENING {
				c := &ModelNode{token.Value, []string{}, currentNode, []*ModelNode{}}
				if currentNode != nil {
					// I hate these if's that are only true once...
					currentNode.Children = append(currentNode.Children, c)
				}
				currentNode = c
				dbg("ModelNode: %s line %d\n", currentNode, token.Line)
			} else {
				currentNode.Value = append(currentNode.Value, token.Value)
			}
			state = NEED_ANY
		}
	}
	// As we climbed back up the tree due to one or more closing parens at the end,
	// the current node became nil but the prevous node at that point is the root.
	return previousNode, nil
}

// Debug function
func dump(m *ModelNode, indent int) {
	msg("%s%s\n", strings.Repeat(" ", 2*indent), m)
	for _, c := range m.Children {
		dump(c, 1+indent)
	}
}

