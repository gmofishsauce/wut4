package main

import (
	"fmt"
)

/* Process branch instructions */
func (a *Assembler) processBranch(opcode string, tokens []Token, pos int, lineNum int) error {
	/* Look up branch condition */
	cond, ok := branchCodes[opcode]
	if !ok {
		return fmt.Errorf("unknown branch instruction: %s", opcode)
	}

	args := getArgs(tokens, pos)
	if len(args) != 1 {
		return fmt.Errorf("%s expects 1 argument (label or offset)", opcode)
	}

	/* Get target - either a label or numeric offset */
	var target int
	var err error

	if args[0].typ == TOK_IDENT {
		/* Label - try to resolve it */
		if addr, ok := a.labels[args[0].value]; ok {
			/* Label already defined - calculate offset */
			/* offset = target - (PC + 2) */
			/* PC is current codeDollar */
			target = addr - (a.codeDollar + 2)
		} else {
			/* Forward reference - add fixup */
			/* For now, emit 0 and add fixup */
			target = 0
			/* TODO: Add proper fixup handling for branches */
		}
	} else {
		target, err = a.parseImmArg(args, 0)
		if err != nil {
			return err
		}
	}

	/* Check that offset fits in 10 bits (signed) */
	if fitsInSigned(target, 10) == 0 {
		return fmt.Errorf("branch offset %d does not fit in 10 bits", target)
	}

	/* Encode: [15:13]=110 [12:3]=imm10 [2:0]=cond */
	word := uint16(0xC000) /* 110 in top 3 bits */
	word |= uint16((target & 0x3FF) << 3)
	word |= uint16(cond & 0x7)

	a.emitWord(word)
	return nil
}

/* Process LDI alias */
/* ldi rT, imm16 */
func (a *Assembler) processLDI(tokens []Token, pos int, lineNum int) error {
	args := getArgs(tokens, pos)
	if len(args) != 2 {
		return fmt.Errorf("ldi expects 2 arguments")
	}

	rT, err := a.parseRegArg(&args[0])
	if err != nil {
		return err
	}

	imm, err := a.parseImmArg(args, 1)
	if err != nil {
		return err
	}

	/* Mask to 16 bits */
	imm = imm & 0xFFFF

	/* If imm < 0x40, use ADI */
	if imm < 0x40 {
		/* adi rT, r0, imm */
		word := uint16(0x8000) /* ADI opcode */
		word |= uint16((imm & 0x7F) << 6)
		word |= uint16(rT & 0x7)
		a.emitWord(word)
		return nil
	}

	/* If (imm & 0xFFC0) == imm, use LUI only */
	if (imm & 0xFFC0) == imm {
		/* lui rT, imm>>6 */
		word := uint16(0xA000) /* LUI opcode */
		word |= uint16(((imm >> 6) & 0x3FF) << 3)
		word |= uint16(rT & 0x7)
		a.emitWord(word)
		return nil
	}

	/* Otherwise, use LUI + ADI */
	/* lui rT, (imm>>6) */
	upper := (imm >> 6) & 0x3FF
	word1 := uint16(0xA000) /* LUI opcode */
	word1 |= uint16(upper << 3)
	word1 |= uint16(rT & 0x7)
	a.emitWord(word1)

	/* adi rT, r0, (imm & 0x3F) */
	lower := imm & 0x3F
	word2 := uint16(0x8000) /* ADI opcode */
	word2 |= uint16(lower << 6)
	word2 |= uint16(rT & 0x7) /* rA = rT */
	/* rB = 0 is implicit - we need to encode rT in rB position */
	/* Actually, for ADI we want: rT = rT + lower, so rB should be rT */
	word2 |= uint16(rT << 3) /* rB = rT */
	a.emitWord(word2)

	return nil
}

/* Process MV alias */
/* mv rT, rS -> adi rT, rS, 0 */
func (a *Assembler) processMV(tokens []Token, pos int, lineNum int) error {
	args := getArgs(tokens, pos)
	if len(args) != 2 {
		return fmt.Errorf("mv expects 2 arguments")
	}

	rT, err := a.parseRegArg(&args[0])
	if err != nil {
		return err
	}

	rS, err := a.parseRegArg(&args[1])
	if err != nil {
		return err
	}

	/* adi rT, rS, 0 */
	word := uint16(0x8000) /* ADI opcode */
	word |= uint16((rS & 0x7) << 3)
	word |= uint16(rT & 0x7)

	a.emitWord(word)
	return nil
}

/* Process RET alias */
/* ret [rN] -> ji rN (default rN = link = 0) */
func (a *Assembler) processRET(tokens []Token, pos int, lineNum int) error {
	args := getArgs(tokens, pos)

	rN := 0 /* Default to LINK (r0) */
	if len(args) >= 1 {
		var err error
		rN, err = a.parseRegArg(&args[0])
		if err != nil {
			return err
		}
	}

	/* ji rN */
	word := uint16(0xFFF0) /* JI opcode */
	word |= uint16(rN & 0x7)

	a.emitWord(word)
	return nil
}

/* Process shift left aliases */
/* sla rN -> adc rN, rN, rN */
/* sll rN -> add rN, rN, rN */
func (a *Assembler) processSHIFTLEFT(opcode string, tokens []Token, pos int, lineNum int) error {
	args := getArgs(tokens, pos)
	if len(args) != 1 {
		return fmt.Errorf("%s expects 1 argument", opcode)
	}

	rN, err := a.parseRegArg(&args[0])
	if err != nil {
		return err
	}

	var word uint16
	if opcode == "sla" {
		/* adc rN, rN, rN */
		word = 0xF200 /* ADC opcode */
	} else {
		/* add rN, rN, rN */
		word = 0xF600 /* ADD opcode */
	}

	word |= uint16((rN & 0x7) << 6) /* rC */
	word |= uint16((rN & 0x7) << 3) /* rB */
	word |= uint16(rN & 0x7)        /* rA */

	a.emitWord(word)
	return nil
}

/* Helper to parse register or LINK */
func (a *Assembler) parseRegOrLink(token *Token) (int, error) {
	if token.value == "link" {
		return 0, nil
	}
	return a.parseRegArg(token)
}
