package main

import (
	"fmt"
	"os"
)

/* Process a directive */
func (a *Assembler) processDirective(directive string, tokens []Token, pos int, lineNum int) error {
	args := getArgs(tokens, pos)

	switch directive {
	case ".align":
		return a.dirAlign(args, lineNum)
	case ".bytes":
		return a.dirBytes(args, lineNum)
	case ".words":
		return a.dirWords(args, lineNum)
	case ".space":
		return a.dirSpace(args, lineNum)
	case ".code":
		return a.dirCode(args, lineNum)
	case ".data":
		return a.dirData(args, lineNum)
	case ".set":
		return a.dirSet(args, lineNum)
	default:
		return fmt.Errorf("unknown directive: %s", directive)
	}
}

/* .align directive */
func (a *Assembler) dirAlign(args []Token, lineNum int) error {
	if len(args) != 1 {
		return fmt.Errorf(".align expects 1 argument")
	}

	align, err := a.parseImmArg(args, 0)
	if err != nil {
		return err
	}

	if align <= 0 {
		return fmt.Errorf("alignment must be positive")
	}

	/* Get current position */
	var dollar *int
	var bytes *[]byte
	if a.inCodeSeg != 0 {
		dollar = &a.codeDollar
		bytes = &a.codeBytes
	} else {
		dollar = &a.dataDollar
		bytes = &a.dataBytes
	}

	/* Calculate padding */
	remainder := *dollar % align
	if remainder != 0 {
		padding := align - remainder
		for i := 0; i < padding; i++ {
			*bytes = append(*bytes, 0)
		}
		*dollar += padding
	}

	return nil
}

/* .bytes directive */
func (a *Assembler) dirBytes(args []Token, lineNum int) error {
	for i := 0; i < len(args); i++ {
		if args[i].typ == TOK_STRING {
			/* String literal */
			str := parseString(args[i].value)
			for j := 0; j < len(str); j++ {
				a.emitByte(byte(str[j]))
			}
		} else {
			/* Numeric value */
			val, err := a.parseImmArg(args, i)
			if err != nil {
				return err
			}
			if val < -128 || val > 255 {
				fmt.Fprintf(os.Stderr, "Warning: byte value %d truncated\n", val)
			}
			a.emitByte(byte(val & 0xFF))
		}
	}
	return nil
}

/* .words directive */
func (a *Assembler) dirWords(args []Token, lineNum int) error {
	for i := 0; i < len(args); i++ {
		val, err := a.parseImmArg(args, i)
		if err != nil {
			return err
		}
		a.emitWord(uint16(val & 0xFFFF))
	}
	return nil
}

/* .space directive */
func (a *Assembler) dirSpace(args []Token, lineNum int) error {
	if len(args) != 1 {
		return fmt.Errorf(".space expects 1 argument")
	}

	size, err := a.parseImmArg(args, 0)
	if err != nil {
		return err
	}

	if size < 0 {
		return fmt.Errorf("space size must be non-negative")
	}

	for i := 0; i < size; i++ {
		a.emitByte(0)
	}

	return nil
}

/* .code directive */
func (a *Assembler) dirCode(args []Token, lineNum int) error {
	if len(args) != 0 {
		return fmt.Errorf(".code expects no arguments")
	}
	a.inCodeSeg = 1
	return nil
}

/* .data directive */
func (a *Assembler) dirData(args []Token, lineNum int) error {
	if len(args) != 0 {
		return fmt.Errorf(".data expects no arguments")
	}
	a.inCodeSeg = 0
	return nil
}

/* .set directive */
func (a *Assembler) dirSet(args []Token, lineNum int) error {
	if len(args) < 2 {
		return fmt.Errorf(".set expects symbol name and value")
	}

	if args[0].typ != TOK_IDENT {
		return fmt.Errorf(".set expects symbol name")
	}

	name := args[0].value

	/* Check if symbol already defined */
	if _, ok := a.symbols[name]; ok {
		return fmt.Errorf("symbol %s already defined", name)
	}

	/* Evaluate value expression */
	val, err := a.evalExpr(args, 1, len(args))
	if err != nil {
		return err
	}

	a.symbols[name] = val
	return nil
}
