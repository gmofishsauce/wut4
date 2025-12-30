package main

import (
	"fmt"
	"strings"
)

/* Instruction table */
var instructions = []Instruction{
	/* Base instructions */
	{"ldw", FMT_BASE, 0x0000, 3},
	{"ldb", FMT_BASE, 0x2000, 3},
	{"stw", FMT_BASE, 0x4000, 3},
	{"stb", FMT_BASE, 0x6000, 3},
	{"adi", FMT_BASE, 0x8000, 3},
	{"lui", FMT_LUI, 0xA000, 2},
	{"br", FMT_BRX, 0xC000, 1},
	{"brl", FMT_BRX, 0xC001, 1},
	{"brz", FMT_BRX, 0xC002, 1},
	{"breq", FMT_BRX, 0xC002, 1},
	{"brnz", FMT_BRX, 0xC003, 1},
	{"brneq", FMT_BRX, 0xC003, 1},
	{"brc", FMT_BRX, 0xC004, 1},
	{"bruge", FMT_BRX, 0xC004, 1},
	{"brnc", FMT_BRX, 0xC005, 1},
	{"brult", FMT_BRX, 0xC005, 1},
	{"brsge", FMT_BRX, 0xC006, 1},
	{"brslt", FMT_BRX, 0xC007, 1},
	{"jal", FMT_JAL, 0xE000, 3},
	/* XOPs */
	{"sbb", FMT_XOP, 0xF000, 3},
	{"adc", FMT_XOP, 0xF200, 3},
	{"sub", FMT_XOP, 0xF400, 3},
	{"add", FMT_XOP, 0xF600, 3},
	{"xor", FMT_XOP, 0xF800, 3},
	{"or", FMT_XOP, 0xFA00, 3},
	{"and", FMT_XOP, 0xFC00, 3},
	/* YOPs */
	{"lsp", FMT_YOP, 0xFE00, 2},
	{"lsi", FMT_YOP, 0xFE20, 2},
	{"ssp", FMT_YOP, 0xFE40, 2},
	{"ssi", FMT_YOP, 0xFE60, 2},
	{"lcw", FMT_YOP, 0xFE80, 2},
	{"sys", FMT_YOP, 0xFEA0, 2},
	{"tst", FMT_YOP, 0xFEC0, 2},
	/* ZOPs */
	{"not", FMT_ZOP, 0xFFC0, 1},
	{"neg", FMT_ZOP, 0xFFC8, 1},
	{"dub", FMT_ZOP, 0xFFD0, 1},
	{"sxt", FMT_ZOP, 0xFFD8, 1},
	{"sra", FMT_ZOP, 0xFFE0, 1},
	{"srl", FMT_ZOP, 0xFFE8, 1},
	{"ji", FMT_ZOP, 0xFFF0, 1},
	/* VOPs */
	{"ccf", FMT_VOP, 0xFFF8, 0},
	{"scf", FMT_VOP, 0xFFF9, 0},
	{"di", FMT_VOP, 0xFFFA, 0},
	{"ei", FMT_VOP, 0xFFFB, 0},
	{"hlt", FMT_VOP, 0xFFFC, 0},
	{"brk", FMT_VOP, 0xFFFD, 0},
	{"rti", FMT_VOP, 0xFFFE, 0},
	{"die", FMT_VOP, 0xFFFF, 0},
	/* Pseudoinstructions */
	{"ldi", FMT_BASE, 0, 2}, /* handled specially */
	{"mv", FMT_BASE, 0, 2},  /* handled specially */
	{"ret", FMT_ZOP, 0, 1},  /* handled specially */
	{"srr", FMT_YOP, 0, 3},  /* handled specially */
	{"srw", FMT_YOP, 0, 3},  /* handled specially */
	{"sla", FMT_XOP, 0, 1},  /* handled specially */
	{"sll", FMT_XOP, 0, 1},  /* handled specially */
}

func lookupInstr(name string) *Instruction {
	name = strings.ToLower(name)
	for i := range instructions {
		if instructions[i].mnemonic == name {
			return &instructions[i]
		}
	}
	return nil
}

func parseRegister(arg string) (int, error) {
	arg = strings.ToLower(arg)
	if strings.HasPrefix(arg, "r") {
		regNum := 0
		if len(arg) == 2 && arg[1] >= '0' && arg[1] <= '7' {
			regNum = int(arg[1] - '0')
			return regNum, nil
		}
	}
	if arg == "link" || arg == "r0" {
		return 0, nil
	}
	return -1, fmt.Errorf("invalid register: %s", arg)
}

func (asm *Assembler) evaluateExpr(exprStr string, allowFwd bool) (int, error) {
	/* Tokenize the expression */
	lex := newLexer(exprStr)
	var tokens []*Token
	for {
		tok, err := lex.nextToken()
		if err != nil {
			return 0, err
		}
		if tok.typ == TOK_EOF || tok.typ == TOK_NEWLINE {
			break
		}
		tokens = append(tokens, tok)
	}

	if len(tokens) == 0 {
		return 0, fmt.Errorf("empty expression")
	}

	/* Parse the expression */
	ep := newExprParser(tokens, asm, allowFwd)
	return ep.parse()
}

func (asm *Assembler) emitWord(word uint16) {
	if asm.currentSeg == SEG_CODE {
		asm.ensureCodeCapacity(asm.codeSize + 2)
		asm.codeBuf[asm.codeSize] = byte(word & 0xFF)
		asm.codeBuf[asm.codeSize+1] = byte((word >> 8) & 0xFF)
		asm.codeSize += 2
		asm.codePC += 2
	} else {
		asm.ensureDataCapacity(asm.dataSize + 2)
		asm.dataBuf[asm.dataSize] = byte(word & 0xFF)
		asm.dataBuf[asm.dataSize+1] = byte((word >> 8) & 0xFF)
		asm.dataSize += 2
		asm.dataPC += 2
	}
}

func (asm *Assembler) emitByte(b byte) {
	if asm.currentSeg == SEG_CODE {
		asm.ensureCodeCapacity(asm.codeSize + 1)
		asm.codeBuf[asm.codeSize] = b
		asm.codeSize++
		asm.codePC++
	} else {
		asm.ensureDataCapacity(asm.dataSize + 1)
		asm.dataBuf[asm.dataSize] = b
		asm.dataSize++
		asm.dataPC++
	}
}

func (asm *Assembler) generateInstruction(stmt *Statement) error {
	instr := lookupInstr(stmt.instr)
	if instr == nil {
		return fmt.Errorf("unknown instruction: %s", stmt.instr)
	}

	/* Handle pseudoinstructions */
	switch stmt.instr {
	case "ldi":
		return asm.genLDI(stmt)
	case "mv":
		return asm.genMV(stmt)
	case "ret":
		return asm.genRET(stmt)
	case "srr":
		return asm.genSRR(stmt)
	case "srw":
		return asm.genSRW(stmt)
	case "sla":
		return asm.genSLA(stmt)
	case "sll":
		return asm.genSLL(stmt)
	case "jal":
		return asm.genJAL(stmt)
	}

	switch instr.format {
	case FMT_BASE:
		return asm.genBase(stmt, instr)
	case FMT_LUI:
		return asm.genLUI(stmt, instr)
	case FMT_BRX:
		return asm.genBRX(stmt, instr)
	case FMT_XOP:
		return asm.genXOP(stmt, instr)
	case FMT_YOP:
		return asm.genYOP(stmt, instr)
	case FMT_ZOP:
		return asm.genZOP(stmt, instr)
	case FMT_VOP:
		return asm.genVOP(stmt, instr)
	}

	return fmt.Errorf("unhandled instruction format")
}

func (asm *Assembler) genBase(stmt *Statement, instr *Instruction) error {
	if stmt.numArgs < 2 {
		return fmt.Errorf("%s requires at least 2 arguments", instr.mnemonic)
	}

	rA, err := parseRegister(stmt.args[0])
	if err != nil {
		return err
	}

	rB, err := parseRegister(stmt.args[1])
	if err != nil {
		return err
	}

	imm := 0
	if stmt.numArgs >= 3 {
		imm, err = asm.evaluateExpr(stmt.args[2], false)
		if err != nil {
			return err
		}
	}

	/* Check for special case: LDW r0, r0+0 */
	if instr.mnemonic == "ldw" && rA == 0 && rB == 0 && imm == 0 {
		return fmt.Errorf("ldw r0, r0+0 is not allowed (use die for 0x0000)")
	}

	/* 7-bit immediate: allow -64 to 127 (full signed/unsigned range) */
	if imm < -64 || imm > 127 {
		return fmt.Errorf("immediate value %d out of range for %s", imm, instr.mnemonic)
	}
	imm7 := imm & 0x7F

	word := instr.opcode | uint16((imm7&0x7F)<<6) | uint16((rB&0x7)<<3) | uint16(rA&0x7)
	asm.emitWord(word)
	return nil
}

func (asm *Assembler) genLUI(stmt *Statement, instr *Instruction) error {
	if stmt.numArgs != 2 {
		return fmt.Errorf("lui requires 2 arguments")
	}

	rA, err := parseRegister(stmt.args[0])
	if err != nil {
		return err
	}

	imm, err := asm.evaluateExpr(stmt.args[1], false)
	if err != nil {
		return err
	}

	/* 10-bit immediate: allow -512 to 1023 (full signed/unsigned range) */
	if imm < -512 || imm > 1023 {
		return fmt.Errorf("immediate value %d out of range for lui", imm)
	}

	word := instr.opcode | uint16((imm&0x3FF)<<3) | uint16(rA&0x7)
	asm.emitWord(word)
	return nil
}

func (asm *Assembler) genBRX(stmt *Statement, instr *Instruction) error {
	if stmt.numArgs != 1 {
		return fmt.Errorf("%s requires 1 argument", instr.mnemonic)
	}

	/* Evaluate target address */
	target, err := asm.evaluateExpr(stmt.args[0], true)
	if err != nil {
		return err
	}

	/* Calculate relative offset: offset = target - (PC + 2) */
	currentPC := asm.codePC
	offset := target - (currentPC + 2)

	/* Check offset range: -512 to 511 */
	if offset < -512 || offset > 511 {
		return fmt.Errorf("branch offset %d out of range", offset)
	}

	imm10 := offset & 0x3FF
	word := instr.opcode | uint16((imm10&0x3FF)<<3)
	asm.emitWord(word)
	return nil
}

func (asm *Assembler) genXOP(stmt *Statement, instr *Instruction) error {
	if stmt.numArgs != 3 {
		return fmt.Errorf("%s requires 3 arguments", instr.mnemonic)
	}

	rA, err := parseRegister(stmt.args[0])
	if err != nil {
		return err
	}

	rB, err := parseRegister(stmt.args[1])
	if err != nil {
		return err
	}

	rC, err := parseRegister(stmt.args[2])
	if err != nil {
		return err
	}

	word := instr.opcode | uint16((rC&0x7)<<6) | uint16((rB&0x7)<<3) | uint16(rA&0x7)
	asm.emitWord(word)
	return nil
}

func (asm *Assembler) genYOP(stmt *Statement, instr *Instruction) error {
	if stmt.numArgs != 2 {
		return fmt.Errorf("%s requires 2 arguments", instr.mnemonic)
	}

	rA, err := parseRegister(stmt.args[0])
	if err != nil {
		return err
	}

	rB, err := parseRegister(stmt.args[1])
	if err != nil {
		return err
	}

	word := instr.opcode | uint16((rB&0x7)<<3) | uint16(rA&0x7)
	asm.emitWord(word)
	return nil
}

func (asm *Assembler) genZOP(stmt *Statement, instr *Instruction) error {
	if stmt.numArgs != 1 {
		return fmt.Errorf("%s requires 1 argument", instr.mnemonic)
	}

	rA, err := parseRegister(stmt.args[0])
	if err != nil {
		return err
	}

	word := instr.opcode | uint16(rA&0x7)
	asm.emitWord(word)
	return nil
}

func (asm *Assembler) genVOP(stmt *Statement, instr *Instruction) error {
	if stmt.numArgs != 0 {
		return fmt.Errorf("%s requires no arguments", instr.mnemonic)
	}

	asm.emitWord(instr.opcode)
	return nil
}

/* Pseudoinstructions */

func (asm *Assembler) genLDI(stmt *Statement) error {
	if stmt.numArgs != 2 {
		return fmt.Errorf("ldi requires 2 arguments")
	}

	rT, err := parseRegister(stmt.args[0])
	if err != nil {
		return err
	}

	imm, err := asm.evaluateExpr(stmt.args[1], false)
	if err != nil {
		return err
	}

	/* 16-bit immediate: allow -32768 to 65535 (full signed/unsigned range) */
	if imm < -32768 || imm > 65535 {
		return fmt.Errorf("immediate value %d out of range for ldi", imm)
	}

	/* Normalize to unsigned 16-bit for encoding selection */
	uimm := uint16(imm & 0xFFFF)

	/* Choose encoding based on value */
	if uimm < 0x40 {
		/* ADI rT, r0, imm6 */
		word := uint16(0x8000) | uint16((uimm&0x7F)<<6) | uint16(rT&0x7)
		asm.emitWord(word)
	} else if (uimm & 0xFFC0) == uimm {
		/* LUI rT, imm10 */
		imm10 := (uimm >> 6) & 0x3FF
		word := uint16(0xA000) | uint16((imm10&0x3FF)<<3) | uint16(rT&0x7)
		asm.emitWord(word)
	} else {
		/* LUI rT, upper; ADI rT, rT, lower */
		upper := (uimm >> 6) & 0x3FF
		lower := uimm & 0x3F
		word1 := uint16(0xA000) | uint16((upper&0x3FF)<<3) | uint16(rT&0x7)
		asm.emitWord(word1)
		word2 := uint16(0x8000) | uint16((lower&0x7F)<<6) | uint16((rT&0x7)<<3) | uint16(rT&0x7)
		asm.emitWord(word2)
	}

	return nil
}

func (asm *Assembler) genMV(stmt *Statement) error {
	if stmt.numArgs != 2 {
		return fmt.Errorf("mv requires 2 arguments")
	}

	rT, err := parseRegister(stmt.args[0])
	if err != nil {
		return err
	}

	rS, err := parseRegister(stmt.args[1])
	if err != nil {
		return err
	}

	/* ADI rT, rS, 0 */
	word := uint16(0x8000) | uint16((rS&0x7)<<3) | uint16(rT&0x7)
	asm.emitWord(word)
	return nil
}

func (asm *Assembler) genRET(stmt *Statement) error {
	rN := 0 /* default to LINK */
	if stmt.numArgs == 1 {
		var err error
		rN, err = parseRegister(stmt.args[0])
		if err != nil {
			return err
		}
	}

	/* JI rN */
	word := uint16(0xFFF0) | uint16(rN&0x7)
	asm.emitWord(word)
	return nil
}

func (asm *Assembler) genSRR(stmt *Statement) error {
	if stmt.numArgs != 3 {
		return fmt.Errorf("srr requires 3 arguments")
	}

	rA, err := parseRegister(stmt.args[0])
	if err != nil {
		return err
	}

	rB, err := parseRegister(stmt.args[1])
	if err != nil {
		return err
	}

	/* LDI rB, imm7; LSP rA, rB */
	if err := asm.genLDI(&Statement{args: []string{stmt.args[1], stmt.args[2]}, numArgs: 2, instr: "ldi"}); err != nil {
		return err
	}
	word := uint16(0xFE00) | uint16((rB&0x7)<<3) | uint16(rA&0x7)
	asm.emitWord(word)
	return nil
}

func (asm *Assembler) genSRW(stmt *Statement) error {
	if stmt.numArgs != 3 {
		return fmt.Errorf("srw requires 3 arguments")
	}

	rA, err := parseRegister(stmt.args[0])
	if err != nil {
		return err
	}

	rB, err := parseRegister(stmt.args[1])
	if err != nil {
		return err
	}

	/* LDI rB, imm7; SSP rA, rB */
	if err := asm.genLDI(&Statement{args: []string{stmt.args[1], stmt.args[2]}, numArgs: 2, instr: "ldi"}); err != nil {
		return err
	}
	word := uint16(0xFE40) | uint16((rB&0x7)<<3) | uint16(rA&0x7)
	asm.emitWord(word)
	return nil
}

func (asm *Assembler) genSLA(stmt *Statement) error {
	if stmt.numArgs != 1 {
		return fmt.Errorf("sla requires 1 argument")
	}

	rN, err := parseRegister(stmt.args[0])
	if err != nil {
		return err
	}

	/* ADC rN, rN, rN */
	word := uint16(0xF200) | uint16((rN&0x7)<<6) | uint16((rN&0x7)<<3) | uint16(rN&0x7)
	asm.emitWord(word)
	return nil
}

func (asm *Assembler) genSLL(stmt *Statement) error {
	if stmt.numArgs != 1 {
		return fmt.Errorf("sll requires 1 argument")
	}

	rN, err := parseRegister(stmt.args[0])
	if err != nil {
		return err
	}

	/* ADD rN, rN, rN */
	word := uint16(0xF600) | uint16((rN&0x7)<<6) | uint16((rN&0x7)<<3) | uint16(rN&0x7)
	asm.emitWord(word)
	return nil
}

func (asm *Assembler) genJAL(stmt *Statement) error {
	/* Handle multiple forms of JAL */
	if stmt.numArgs == 1 {
		/* jal label -> jal link, label */
		target, err := asm.evaluateExpr(stmt.args[0], true)
		if err != nil {
			return err
		}
		upper := (target >> 6) & 0x3FF
		lower := target & 0x3F
		/* LUI r0, upper; JAL r0, r0, lower */
		word1 := uint16(0xA000) | uint16((upper&0x3FF)<<3)
		asm.emitWord(word1)
		word2 := uint16(0xE000) | uint16((lower&0x3F)<<6)
		asm.emitWord(word2)
	} else if stmt.numArgs == 2 {
		/* jal rT, label */
		rT, err := parseRegister(stmt.args[0])
		if err != nil {
			return err
		}
		target, err := asm.evaluateExpr(stmt.args[1], true)
		if err != nil {
			return err
		}
		upper := (target >> 6) & 0x3FF
		lower := target & 0x3F
		/* LUI rT, upper; JAL rT, rT, lower */
		word1 := uint16(0xA000) | uint16((upper&0x3FF)<<3) | uint16(rT&0x7)
		asm.emitWord(word1)
		word2 := uint16(0xE000) | uint16((lower&0x3F)<<6) | uint16((rT&0x7)<<3) | uint16(rT&0x7)
		asm.emitWord(word2)
	} else if stmt.numArgs == 3 {
		/* jal rT, rS, label or jal rT, rS, imm6 */
		rT, err := parseRegister(stmt.args[0])
		if err != nil {
			return err
		}
		rS, err := parseRegister(stmt.args[1])
		if err != nil {
			return err
		}

		/* Check if third arg is a small immediate or label */
		imm, err := asm.evaluateExpr(stmt.args[2], true)
		if err == nil && imm >= 0 && imm < 64 {
			/* Direct JAL with imm6 */
			word := uint16(0xE000) | uint16((imm&0x3F)<<6) | uint16((rS&0x7)<<3) | uint16(rT&0x7)
			asm.emitWord(word)
		} else {
			/* Full address */
			target, err := asm.evaluateExpr(stmt.args[2], true)
			if err != nil {
				return err
			}
			upper := (target >> 6) & 0x3FF
			lower := target & 0x3F
			/* LUI rT, upper; JAL rT, rS, lower */
			word1 := uint16(0xA000) | uint16((upper&0x3FF)<<3) | uint16(rT&0x7)
			asm.emitWord(word1)
			word2 := uint16(0xE000) | uint16((lower&0x3F)<<6) | uint16((rS&0x7)<<3) | uint16(rT&0x7)
			asm.emitWord(word2)
		}
	} else {
		return fmt.Errorf("jal requires 1-3 arguments")
	}

	return nil
}
