package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

/* Main assembler function */
func assemble(inputFile string, outputFile string) error {
	a := Assembler{
		filename:   inputFile,
		lines:      make([]string, 0, 1024),
		codeDollar: 0,
		dataDollar: 0,
		inCodeSeg:  1,
		codeBytes:  make([]byte, 0, 65536),
		dataBytes:  make([]byte, 0, 65536),
		labels:     make(map[string]int),
		symbols:    make(map[string]int),
		fixups:     make([]Fixup, 0, 256),
		errors:     make([]string, 0, 64),
	}

	/* Read input file */
	file, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		a.lines = append(a.lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	/* Pass 1: Process all lines */
	for a.lineNum = 0; a.lineNum < len(a.lines); a.lineNum++ {
		line := a.lines[a.lineNum]
		err := a.processLine(line, a.lineNum+1)
		if err != nil {
			a.errors = append(a.errors, fmt.Sprintf("Line %d: %v", a.lineNum+1, err))
		}
	}

	/* Resolve forward references */
	for i := 0; i < len(a.fixups); i++ {
		fixup := &a.fixups[i]
		addr, ok := a.labels[fixup.label]
		if !ok {
			a.errors = append(a.errors, fmt.Sprintf("Line %d: undefined label: %s", fixup.line, fixup.label))
			continue
		}

		/* Patch the address in the output */
		if fixup.isInCode != 0 {
			/* Patch code segment - this is a simplified version */
			/* For now, we'll just store the address as a 16-bit value */
			if fixup.addr+1 < len(a.codeBytes) {
				a.codeBytes[fixup.addr] = byte(addr & 0xFF)
				a.codeBytes[fixup.addr+1] = byte((addr >> 8) & 0xFF)
			}
		} else {
			/* Patch data segment */
			if fixup.addr+1 < len(a.dataBytes) {
				a.dataBytes[fixup.addr] = byte(addr & 0xFF)
				a.dataBytes[fixup.addr+1] = byte((addr >> 8) & 0xFF)
			}
		}
	}

	/* Report errors */
	if len(a.errors) > 0 {
		for i := 0; i < len(a.errors); i++ {
			fmt.Fprintf(os.Stderr, "%s\n", a.errors[i])
		}
		return fmt.Errorf("assembly failed with %d errors", len(a.errors))
	}

	/* Write output file */
	return a.writeOutput(outputFile)
}

/* Process a single line */
func (a *Assembler) processLine(line string, lineNum int) error {
	tokens := tokenizeLine(line, lineNum)
	if len(tokens) == 0 {
		return nil
	}

	pos := 0

	/* Check for label */
	if len(tokens) >= 2 && tokens[1].typ == TOK_COLON {
		label := tokens[0].value
		if a.inCodeSeg != 0 {
			a.labels[label] = a.codeDollar
		} else {
			a.labels[label] = a.dataDollar
		}
		pos = 2
	}

	/* Check for directive or instruction */
	if pos >= len(tokens) {
		return nil
	}

	opcode := tokens[pos].value
	pos++

	/* Check for directive */
	if len(opcode) > 0 && opcode[0] == '.' {
		return a.processDirective(opcode, tokens, pos, lineNum)
	}

	/* Check for branch instruction */
	if strings.HasPrefix(opcode, "br") {
		return a.processBranch(opcode, tokens, pos, lineNum)
	}

	/* Check for aliases */
	if opcode == "ldi" {
		return a.processLDI(tokens, pos, lineNum)
	}
	if opcode == "mv" {
		return a.processMV(tokens, pos, lineNum)
	}
	if opcode == "ret" {
		return a.processRET(tokens, pos, lineNum)
	}
	if opcode == "sla" || opcode == "sll" {
		return a.processSHIFTLEFT(opcode, tokens, pos, lineNum)
	}

	/* Regular instruction */
	return a.processInstruction(opcode, tokens, pos, lineNum)
}

/* Emit a 16-bit word to the current segment */
func (a *Assembler) emitWord(word uint16) {
	/* Little endian */
	lo := byte(word & 0xFF)
	hi := byte((word >> 8) & 0xFF)

	if a.inCodeSeg != 0 {
		a.codeBytes = append(a.codeBytes, lo, hi)
		a.codeDollar += 2
	} else {
		a.dataBytes = append(a.dataBytes, lo, hi)
		a.dataDollar += 2
	}
}

/* Emit a byte to the current segment */
func (a *Assembler) emitByte(b byte) {
	if a.inCodeSeg != 0 {
		a.codeBytes = append(a.codeBytes, b)
		a.codeDollar++
	} else {
		a.dataBytes = append(a.dataBytes, b)
		a.dataDollar++
	}
}

/* Get argument tokens (skip commas) */
func getArgs(tokens []Token, start int) []Token {
	args := make([]Token, 0, 8)
	for i := start; i < len(tokens); i++ {
		if tokens[i].typ != TOK_COMMA {
			args = append(args, tokens[i])
		}
	}
	return args
}

/* Parse register argument */
func (a *Assembler) parseRegArg(token *Token) (int, error) {
	reg := parseRegister(token.value)
	if reg < 0 {
		return 0, fmt.Errorf("expected register, got: %s", token.value)
	}
	return reg, nil
}

/* Parse immediate value */
func (a *Assembler) parseImmArg(tokens []Token, pos int) (int, error) {
	if pos >= len(tokens) {
		return 0, fmt.Errorf("expected immediate value")
	}
	/* Try to evaluate as expression */
	val, err := a.evalExpr(tokens, pos, pos+1)
	if err != nil {
		return 0, err
	}
	return val, nil
}
