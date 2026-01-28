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

import "fmt"

// disassemble produces a human-readable disassembly of an instruction
func disassemble(inst *Instruction) string {
	if inst.raw == 0x0000 {
		return "ILLEGAL (0x0000)"
	}
	if inst.raw == 0xFFFF {
		return "DIE"
	}

	if inst.isVOP {
		return disassembleVOP(inst)
	}
	if inst.isZOP {
		return disassembleZOP(inst)
	}
	if inst.isYOP {
		return disassembleYOP(inst)
	}
	if inst.isXOP {
		return disassembleXOP(inst)
	}
	return disassembleBase(inst)
}

func disassembleBase(inst *Instruction) string {
	switch inst.opcode {
	case 0: // LDW
		return fmt.Sprintf("LDW r%d, r%d%+d", inst.rA, inst.rB, inst.imm7)

	case 1: // LDB
		return fmt.Sprintf("LDB r%d, r%d%+d", inst.rA, inst.rB, inst.imm7)

	case 2: // STW
		return fmt.Sprintf("STW r%d, r%d%+d", inst.rA, inst.rB, inst.imm7)

	case 3: // STB
		return fmt.Sprintf("STB r%d, r%d%+d", inst.rA, inst.rB, inst.imm7)

	case 4: // ADI
	/*
		if inst.rB == 0 && inst.rA == 0 {
			return fmt.Sprintf("LDI LINK, %d", inst.imm7)
		} else if inst.rB == 0 {
			return fmt.Sprintf("LDI r%d, %d", inst.rA, inst.imm7)
		} else if inst.rA == 0 {
			return fmt.Sprintf("ADI LINK, r%d, %d", inst.rB, inst.imm7)
		}
	*/
		return fmt.Sprintf("ADI r%d, r%d, %d", inst.rA, inst.rB, inst.imm7)

	case 5: // LUI
		if inst.rA == 0 {
			return fmt.Sprintf("LUI LINK, 0x%03X", inst.imm10)
		}
		return fmt.Sprintf("LUI r%d, 0x%03X", inst.rA, inst.imm10)

	case 6: // BRx
		return fmt.Sprintf("%s %+d", branchCondNames[inst.branchCond], int16(inst.imm10))

	case 7: // JAL
		if inst.rA == 0 && inst.rB == 0 {
			return fmt.Sprintf("JAL LINK, LINK, %d", inst.imm7)
		} else if inst.rA == 0 {
			return fmt.Sprintf("JAL LINK, r%d, %d", inst.rB, inst.imm7)
		} else if inst.rB == 0 {
			return fmt.Sprintf("JAL r%d, LINK, %d", inst.rA, inst.imm7)
		}
		return fmt.Sprintf("JAL r%d, r%d, %d", inst.rA, inst.rB, inst.imm7)

	default:
		return fmt.Sprintf("??? (0x%04X)", inst.raw)
	}
}

func disassembleXOP(inst *Instruction) string {
	opName := xopNames[inst.xop]
	return fmt.Sprintf("%s r%d, r%d, r%d", opName, inst.rA, inst.rB, inst.rC)
}

func disassembleYOP(inst *Instruction) string {
	opName := yopNames[inst.yop]

	switch inst.yop {
	case 0, 1, 2, 3, 4: // LSP, LSI, SSP, SSI, LCW
		return fmt.Sprintf("%s r%d, r%d", opName, inst.rA, inst.rB)

	case 5: // SYS
		if inst.rB == 0 && inst.rA <= 7 {
			return fmt.Sprintf("SYS %d", inst.rA)
		}
		return fmt.Sprintf("SYS??? r%d, r%d", inst.rA, inst.rB)

	case 6: // TST
		return fmt.Sprintf("%s r%d, r%d", opName, inst.rA, inst.rB)

	default:
		return fmt.Sprintf("??? (0x%04X)", inst.raw)
	}
}

func disassembleZOP(inst *Instruction) string {
	opName := zopNames[inst.zop]
	return fmt.Sprintf("%s r%d", opName, inst.rA)
}

func disassembleVOP(inst *Instruction) string {
	return vopNames[inst.vop]
}
