// Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.

package main

// Instruction represents a decoded instruction
type Instruction struct {
	raw    uint16 // Raw instruction word
	opcode uint8  // Base opcode (bits 15-13)

	// Register fields
	rA uint8 // Bits 2-0
	rB uint8 // Bits 5-3
	rC uint8 // Bits 8-6

	// Immediate values
	imm7  int16  // 7-bit signed immediate
	imm10 uint16 // 10-bit unsigned immediate

	// Extended opcode fields
	xop uint8 // Bits 11-9 for XOP
	yop uint8 // Bits 8-6 for YOP
	zop uint8 // Bits 5-3 for ZOP
	vop uint8 // Bits 2-0 for VOP

	// Branch condition (overloads rA)
	branchCond uint8

	// Instruction type flags
	isBase bool
	isXOP  bool
	isYOP  bool
	isZOP  bool
	isVOP  bool
}

// decode decodes a 16-bit instruction word
func decode(inst uint16) *Instruction {
	d := &Instruction{
		raw: inst,
	}

	// Extract common fields
	d.opcode = uint8((inst >> 13) & 0x07)
	d.rA = uint8((inst >> 0) & 0x07)
	d.rB = uint8((inst >> 3) & 0x07)
	d.rC = uint8((inst >> 6) & 0x07)

	// Determine instruction type
	// VOP: bits 15-3 are all 1 (0x1FFF)
	if (inst >> 3) == 0x1FFF {
		d.isVOP = true
		d.vop = d.rA
		return d
	}

	// ZOP: bits 15-6 are all 1 (0x03FF)
	if (inst >> 6) == 0x03FF {
		d.isZOP = true
		d.zop = d.rB
		return d
	}

	// YOP: bits 15-9 are all 1 (0x007F)
	if (inst >> 9) == 0x007F {
		d.isYOP = true
		d.yop = d.rC
		return d
	}

	// XOP: bits 15-12 are all 1 (0x000F)
	if (inst >> 12) == 0x000F {
		d.isXOP = true
		d.xop = uint8((inst >> 9) & 0x07)
		return d
	}

	// Base instruction
	d.isBase = true

	// Decode immediates based on opcode
	switch d.opcode {
	case 0, 1, 2, 3, 4: // LDW, LDB, STW, STB, ADI
		// 7-bit signed immediate (bits 12-6)
		imm := uint8((inst >> 6) & 0x7F)
		// Sign extend from 7 bits
		if imm&0x40 != 0 {
			d.imm7 = int16(imm) | ^0x3F
		} else {
			d.imm7 = int16(imm)
		}

	case 5: // LUI
		// 10-bit unsigned immediate (bits 12-3)
		d.imm10 = (inst >> 3) & 0x03FF

	case 6: // BRx
		// 10-bit signed immediate (bits 12-3)
		imm := (inst >> 3) & 0x03FF
		// Sign extend from 10 bits
		if imm&0x0200 != 0 {
			d.imm10 = imm | 0xFC00
		} else {
			d.imm10 = imm
		}
		// Branch condition is in rA field
		d.branchCond = d.rA

	case 7: // JAL
		// 6-bit unsigned immediate (bits 8-3)
		d.imm10 = (inst >> 3) & 0x003F
	}

	return d
}

// Helper methods for Instruction

func (i *Instruction) String() string {
	if i.isVOP {
		return vopNames[i.vop]
	}
	if i.isZOP {
		return zopNames[i.zop]
	}
	if i.isYOP {
		return yopNames[i.yop]
	}
	if i.isXOP {
		return xopNames[i.xop]
	}
	// Base instruction
	return baseOpNames[i.opcode]
}

// Instruction names for disassembly
var baseOpNames = []string{
	"ldw", "ldb", "stw", "stb", "adi", "lui", "brx", "jal",
}

var xopNames = []string{
	"sbb", "adc", "sub", "add", "xor", "or", "and", "???",
}

var yopNames = []string{
	"lsp", "lsi", "ssp", "ssi", "lcw", "sys", "???", "???",
}

var zopNames = []string{
	"not", "neg", "zxt", "sxt", "sra", "srl", "dub", "???",
}

var vopNames = []string{
	"ccf", "scf", "di", "ei", "hlt", "brk", "rti", "die",
}

var branchCondNames = []string{
	"br", "brl", "beq", "bne", "bcs", "bcc", "bge", "blt",
}
