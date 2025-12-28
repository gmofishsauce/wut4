package main

import (
	"fmt"
)

/* Process a regular instruction */
func (a *Assembler) processInstruction(opcode string, tokens []Token, pos int, lineNum int) error {
	instr := lookupInstr(opcode)
	if instr == nil {
		return fmt.Errorf("unknown instruction: %s", opcode)
	}

	args := getArgs(tokens, pos)

	switch instr.format {
	case FMT_BASE:
		return a.genBase(instr, args, lineNum)
	case FMT_LUI:
		return a.genLUI(instr, args, lineNum)
	case FMT_JAL:
		return a.genJAL(instr, args, lineNum)
	case FMT_XOP:
		return a.genXOP(instr, args, lineNum)
	case FMT_YOP:
		return a.genYOP(instr, args, lineNum)
	case FMT_ZOP:
		return a.genZOP(instr, args, lineNum)
	case FMT_VOP:
		return a.genVOP(instr, args, lineNum)
	default:
		return fmt.Errorf("unknown instruction format")
	}
}

/* Generate base instruction: LDW, LDB, STW, STB, ADI */
/* Format: opcode rA, rB, imm7 */
func (a *Assembler) genBase(instr *InstrDef, args []Token, lineNum int) error {
	/* Can have 2 or 3 arguments (immediate defaults to 0) */
	if len(args) < 2 || len(args) > 3 {
		return fmt.Errorf("%s expects 2 or 3 arguments", instr.name)
	}

	rA, err := a.parseRegArg(&args[0])
	if err != nil {
		return err
	}

	rB, err := a.parseRegArg(&args[1])
	if err != nil {
		return err
	}

	imm := 0
	if len(args) >= 3 {
		imm, err = a.parseImmArg(args, 2)
		if err != nil {
			return err
		}
	}

	/* Check that immediate fits in 7 bits (signed) */
	if fitsInSigned(imm, 7) == 0 {
		return fmt.Errorf("immediate value %d does not fit in 7 bits", imm)
	}

	/* Encode instruction */
	/* Format: [15:13]=opcode [12:6]=imm7 [5:3]=rB [2:0]=rA */
	word := instr.opcode
	word |= uint16((imm & 0x7F) << 6)
	word |= uint16((rB & 0x7) << 3)
	word |= uint16(rA & 0x7)

	/* Special case: LDW r0, r0, 0 assembles to 0x0000 which is illegal */
	if instr.name == "ldw" && rA == 0 && rB == 0 && imm == 0 {
		return fmt.Errorf("ldw r0, r0, 0 is illegal (generates 0x0000)")
	}

	a.emitWord(word)
	return nil
}

/* Generate LUI instruction */
/* Format: lui rA, imm10 */
func (a *Assembler) genLUI(instr *InstrDef, args []Token, lineNum int) error {
	if len(args) != 2 {
		return fmt.Errorf("lui expects 2 arguments")
	}

	rA, err := a.parseRegArg(&args[0])
	if err != nil {
		return err
	}

	imm, err := a.parseImmArg(args, 1)
	if err != nil {
		return err
	}

	/* Check that immediate fits in 10 bits */
	if fitsInUnsigned(imm, 10) == 0 {
		return fmt.Errorf("immediate value %d does not fit in 10 bits", imm)
	}

	/* Encode: [15:13]=101 [12:3]=imm10 [2:0]=rA */
	word := instr.opcode
	word |= uint16((imm & 0x3FF) << 3)
	word |= uint16(rA & 0x7)

	a.emitWord(word)
	return nil
}

/* Generate JAL instruction */
/* Format: jal rA, rB, imm6 */
func (a *Assembler) genJAL(instr *InstrDef, args []Token, lineNum int) error {
	if len(args) < 2 || len(args) > 3 {
		return fmt.Errorf("jal expects 2 or 3 arguments")
	}

	rA, err := a.parseRegArg(&args[0])
	if err != nil {
		return err
	}

	rB, err := a.parseRegArg(&args[1])
	if err != nil {
		return err
	}

	imm := 0
	if len(args) >= 3 {
		imm, err = a.parseImmArg(args, 2)
		if err != nil {
			return err
		}
	}

	/* Check that immediate fits in 6 bits (must be positive/unsigned) */
	if fitsInUnsigned(imm, 6) == 0 {
		return fmt.Errorf("immediate value %d does not fit in 6 bits", imm)
	}

	/* Encode: [15:12]=1110 [11:6]=imm6 [5:3]=rB [2:0]=rA */
	word := instr.opcode
	word |= uint16((imm & 0x3F) << 6)
	word |= uint16((rB & 0x7) << 3)
	word |= uint16(rA & 0x7)

	a.emitWord(word)
	return nil
}

/* Generate XOP instruction (3-operand) */
/* Format: xop rA, rB, rC */
func (a *Assembler) genXOP(instr *InstrDef, args []Token, lineNum int) error {
	if len(args) != 3 {
		return fmt.Errorf("%s expects 3 arguments", instr.name)
	}

	rA, err := a.parseRegArg(&args[0])
	if err != nil {
		return err
	}

	rB, err := a.parseRegArg(&args[1])
	if err != nil {
		return err
	}

	rC, err := a.parseRegArg(&args[2])
	if err != nil {
		return err
	}

	/* Encode: [15:9]=1111xxx [8:6]=rC [5:3]=rB [2:0]=rA */
	word := instr.opcode
	word |= uint16((rC & 0x7) << 6)
	word |= uint16((rB & 0x7) << 3)
	word |= uint16(rA & 0x7)

	a.emitWord(word)
	return nil
}

/* Generate YOP instruction (2-operand) */
/* Format: yop rA, rB */
func (a *Assembler) genYOP(instr *InstrDef, args []Token, lineNum int) error {
	/* Special case for SYS instruction */
	if instr.name == "sys" {
		return a.genSYS(args, lineNum)
	}

	if len(args) != 2 {
		return fmt.Errorf("%s expects 2 arguments", instr.name)
	}

	rA, err := a.parseRegArg(&args[0])
	if err != nil {
		return err
	}

	rB, err := a.parseRegArg(&args[1])
	if err != nil {
		return err
	}

	/* Encode: [15:6]=1111111yyy [5:3]=rB [2:0]=rA */
	word := instr.opcode
	word |= uint16((rB & 0x7) << 3)
	word |= uint16(rA & 0x7)

	a.emitWord(word)
	return nil
}

/* Generate SYS instruction */
func (a *Assembler) genSYS(args []Token, lineNum int) error {
	if len(args) != 1 {
		return fmt.Errorf("sys expects 1 argument (sys number 0-7)")
	}

	sysNum, err := a.parseImmArg(args, 0)
	if err != nil {
		return err
	}

	if sysNum < 0 || sysNum > 7 {
		return fmt.Errorf("sys number must be 0-7, got %d", sysNum)
	}

	/* Encode: [15:6]=1111110101 [5:3]=0 [2:0]=sysNum */
	word := uint16(0xFF40)
	word |= uint16(sysNum & 0x7)

	a.emitWord(word)
	return nil
}

/* Generate ZOP instruction (1-operand) */
/* Format: zop rA */
func (a *Assembler) genZOP(instr *InstrDef, args []Token, lineNum int) error {
	if len(args) != 1 {
		return fmt.Errorf("%s expects 1 argument", instr.name)
	}

	rA, err := a.parseRegArg(&args[0])
	if err != nil {
		return err
	}

	/* Encode: [15:3]=1111111111zzz [2:0]=rA */
	word := instr.opcode
	word |= uint16(rA & 0x7)

	a.emitWord(word)
	return nil
}

/* Generate VOP instruction (0-operand) */
/* Format: vop */
func (a *Assembler) genVOP(instr *InstrDef, args []Token, lineNum int) error {
	if len(args) != 0 {
		return fmt.Errorf("%s expects no arguments", instr.name)
	}

	/* Opcode is complete */
	a.emitWord(instr.opcode)
	return nil
}
