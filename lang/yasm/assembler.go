package main

import (
	"fmt"
	"os"
	"strings"
)

func newAssembler(inputFile, outputFile string) *Assembler {
	asm := &Assembler{
		inputFile:  inputFile,
		outputFile: outputFile,
		symbols:    make([]Symbol, 1024),
		numSymbols: 0,
		codePC:     0,
		dataPC:     0,
		currentSeg: SEG_CODE,
		codeBuf:    make([]byte, 4096),
		dataBuf:    make([]byte, 4096),
		codeSize:   0,
		dataSize:   0,
		codeCap:    4096,
		dataCap:    4096,
		pass:       1,
		errors:     0,
	}
	return asm
}

func (asm *Assembler) ensureCodeCapacity(needed int) {
	for needed > asm.codeCap {
		newCap := asm.codeCap * 2
		newBuf := make([]byte, newCap)
		copy(newBuf, asm.codeBuf[:asm.codeSize])
		asm.codeBuf = newBuf
		asm.codeCap = newCap
	}
}

func (asm *Assembler) ensureDataCapacity(needed int) {
	for needed > asm.dataCap {
		newCap := asm.dataCap * 2
		newBuf := make([]byte, newCap)
		copy(newBuf, asm.dataBuf[:asm.dataSize])
		asm.dataBuf = newBuf
		asm.dataCap = newCap
	}
}

func (asm *Assembler) addSymbol(name string, value int, segment int) error {
	/* Check if symbol already exists */
	for i := 0; i < asm.numSymbols; i++ {
		if asm.symbols[i].name == name {
			if asm.symbols[i].defined {
				return fmt.Errorf("symbol %s already defined", name)
			}
			/* Update forward reference */
			asm.symbols[i].value = value
			asm.symbols[i].defined = true
			asm.symbols[i].segment = segment
			return nil
		}
	}

	/* Add new symbol */
	if asm.numSymbols >= len(asm.symbols) {
		/* Grow symbol table */
		newSymbols := make([]Symbol, len(asm.symbols)*2)
		copy(newSymbols, asm.symbols)
		asm.symbols = newSymbols
	}

	asm.symbols[asm.numSymbols] = Symbol{
		name:    name,
		value:   value,
		defined: true,
		segment: segment,
	}
	asm.numSymbols++
	return nil
}

func (asm *Assembler) lookupSymbol(name string) *Symbol {
	for i := 0; i < asm.numSymbols; i++ {
		if asm.symbols[i].name == name {
			return &asm.symbols[i]
		}
	}
	return nil
}

func (asm *Assembler) processDirective(stmt *Statement) error {
	switch stmt.directive {
	case DIR_ALIGN:
		if stmt.numArgs != 1 {
			return fmt.Errorf(".align requires 1 argument")
		}
		align, err := asm.evaluateExpr(stmt.args[0], false)
		if err != nil {
			return err
		}
		if align <= 0 {
			return fmt.Errorf("alignment must be positive")
		}

		currentPC := asm.codePC
		if asm.currentSeg == SEG_DATA {
			currentPC = asm.dataPC
		}

		remainder := currentPC % align
		if remainder != 0 {
			padding := align - remainder
			for i := 0; i < padding; i++ {
				asm.emitByte(0)
			}
		}

	case DIR_BYTES:
		for i := 0; i < stmt.numArgs; i++ {
			arg := stmt.args[i]
			/* Check if it's a string (starts and ends with quotes) */
			if len(arg) >= 2 && arg[0] == '"' && arg[len(arg)-1] == '"' {
				/* It's a string - strip quotes and emit bytes */
				str := arg[1 : len(arg)-1]
				for j := 0; j < len(str); j++ {
					asm.emitByte(str[j])
				}
			} else {
				/* It's a numeric expression */
				val, err := asm.evaluateExpr(arg, false)
				if err != nil {
					return err
				}
				if val < -128 || val > 255 {
					fmt.Fprintf(os.Stderr, "Warning: byte value %d truncated\n", val)
				}
				asm.emitByte(byte(val & 0xFF))
			}
		}

	case DIR_WORDS:
		for i := 0; i < stmt.numArgs; i++ {
			val, err := asm.evaluateExpr(stmt.args[i], false)
			if err != nil {
				return err
			}
			asm.emitWord(uint16(val & 0xFFFF))
		}

	case DIR_SPACE:
		if stmt.numArgs != 1 {
			return fmt.Errorf(".space requires 1 argument")
		}
		count, err := asm.evaluateExpr(stmt.args[0], false)
		if err != nil {
			return err
		}
		for i := 0; i < count; i++ {
			asm.emitByte(0)
		}

	case DIR_CODE:
		if asm.bootstrapMode {
			return fmt.Errorf(".code directive not permitted in .bootstrap programs")
		}
		asm.currentSeg = SEG_CODE

	case DIR_DATA:
		if asm.bootstrapMode {
			return fmt.Errorf(".data directive not permitted in .bootstrap programs")
		}
		asm.currentSeg = SEG_DATA

	case DIR_BOOTSTRAP:
		if asm.seenCode {
			return fmt.Errorf(".bootstrap must appear first in the file")
		}
		asm.bootstrapMode = true
		asm.currentSeg = SEG_CODE

	case DIR_SET:
		if asm.pass != 1 {
			/* Only process .set in pass 1 to avoid duplicate symbol errors */
			return nil
		}
		if stmt.numArgs != 2 {
			return fmt.Errorf(".set requires 2 arguments")
		}
		symName := stmt.args[0]
		val, err := asm.evaluateExpr(stmt.args[1], false)
		if err != nil {
			return err
		}
		if err := asm.addSymbol(symName, val, -1); err != nil {
			return err
		}
	}

	return nil
}

func (asm *Assembler) processStatement(stmt *Statement) error {
	/* Mark that we've seen code (unless it's just a label or .bootstrap) */
	if stmt.hasDir && stmt.directive != DIR_BOOTSTRAP {
		asm.seenCode = true
	}
	if stmt.hasInstr {
		asm.seenCode = true
	}

	/* Handle label - only in pass 1 */
	if stmt.label != "" && asm.pass == 1 {
		/* In bootstrap mode, everything uses codePC */
		if asm.bootstrapMode {
			if err := asm.addSymbol(stmt.label, asm.codePC, SEG_CODE); err != nil {
				return err
			}
		} else {
			value := asm.codePC
			segment := SEG_CODE
			if asm.currentSeg == SEG_DATA {
				value = asm.dataPC
				segment = SEG_DATA
			}
			if err := asm.addSymbol(stmt.label, value, segment); err != nil {
				return err
			}
		}
	}

	/* Handle directive */
	if stmt.hasDir {
		return asm.processDirective(stmt)
	}

	/* Handle instruction */
	if stmt.hasInstr {
		return asm.generateInstruction(stmt)
	}

	return nil
}

func (asm *Assembler) pass1(input string) error {
	asm.pass = 1
	asm.codePC = 0
	asm.dataPC = 0
	asm.currentSeg = SEG_CODE
	asm.codeSize = 0
	asm.dataSize = 0
	asm.seenCode = false

	lines := strings.Split(input, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parser := newParser(line, asm)
		stmt, err := parser.parseStatement()
		if err != nil {
			return err
		}
		if stmt == nil {
			continue
		}

		if err := asm.processStatement(stmt); err != nil {
			return fmt.Errorf("line %d: %v", stmt.line, err)
		}
	}

	return nil
}

func (asm *Assembler) pass2(input string) error {
	asm.pass = 2
	asm.codePC = 0
	asm.dataPC = 0
	asm.currentSeg = SEG_CODE
	asm.codeSize = 0
	asm.dataSize = 0
	asm.seenCode = false

	lines := strings.Split(input, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parser := newParser(line, asm)
		stmt, err := parser.parseStatement()
		if err != nil {
			return err
		}
		if stmt == nil {
			continue
		}

		/* Skip labels in pass 2 - they were handled in pass 1 */
		if stmt.label != "" && !stmt.hasDir && !stmt.hasInstr {
			continue
		}

		if err := asm.processStatement(stmt); err != nil {
			return fmt.Errorf("line %d: %v", stmt.line, err)
		}
	}

	return nil
}

func (asm *Assembler) printCapitalSymbols() {
	hasCapitalSymbols := false

	/* First pass: check if there are any capital symbols */
	for i := 0; i < asm.numSymbols; i++ {
		sym := &asm.symbols[i]
		if len(sym.name) > 0 && sym.name[0] >= 'A' && sym.name[0] <= 'Z' {
			hasCapitalSymbols = true
			break
		}
	}

	if !hasCapitalSymbols {
		return
	}

	fmt.Println("\nSymbols:")
	for i := 0; i < asm.numSymbols; i++ {
		sym := &asm.symbols[i]
		if len(sym.name) > 0 && sym.name[0] >= 'A' && sym.name[0] <= 'Z' {
			fmt.Printf("  %s = 0x%04x (%d)\n", sym.name, sym.value, sym.value)
		}
	}
}

func assemble(inputName, input, outputFile string) error {
	asm := newAssembler(inputName, outputFile)

	/* Pass 1: collect labels and allocate space */
	if err := asm.pass1(input); err != nil {
		return fmt.Errorf("pass 1: %v", err)
	}

	/* Pass 2: generate code */
	if err := asm.pass2(input); err != nil {
		return fmt.Errorf("pass 2: %v", err)
	}

	/* Write output file */
	if err := writeOutput(outputFile, asm.codeBuf[:asm.codeSize], asm.dataBuf[:asm.dataSize]); err != nil {
		return err
	}

	fmt.Printf("Assembly successful: %s -> %s\n", inputName, outputFile)
	fmt.Printf("Code: %d bytes, Data: %d bytes\n", asm.codeSize, asm.dataSize)

	/* Print symbols that start with capital letters */
	asm.printCapitalSymbols()

	return nil
}
