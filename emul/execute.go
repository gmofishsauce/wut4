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

// execute executes a decoded instruction
func (cpu *CPU) execute(inst *Instruction) error {
	// Special case: 0x0000 is illegal instruction
	if inst.raw == 0x0000 {
		cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
		return nil
	}

	// Special case: 0xFFFF is DIE (illegal instruction trap)
	if inst.raw == 0xFFFF {
		cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
		return nil
	}

	if inst.isBase {
		return cpu.executeBase(inst)
	} else if inst.isXOP {
		return cpu.executeXOP(inst)
	} else if inst.isYOP {
		return cpu.executeYOP(inst)
	} else if inst.isZOP {
		return cpu.executeZOP(inst)
	} else if inst.isVOP {
		return cpu.executeVOP(inst)
	}

	cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
	return nil
}

// executeBase executes base instructions (opcodes 0-7)
func (cpu *CPU) executeBase(inst *Instruction) error {
	regs := &cpu.gen[cpu.mode]

	switch inst.opcode {
	case 0: // LDW - Load word
		addr := uint16(int32(regs[inst.rB]) + int32(inst.imm7))
		value, err := cpu.loadWord(addr)
		if err != nil {
			return err
		}
		regs[inst.rA] = value
		cpu.pc += 2 // Advance to next instruction (byte address)

	case 1: // LDB - Load byte (sign extended)
		addr := uint16(int32(regs[inst.rB]) + int32(inst.imm7))
		value, err := cpu.loadByte(addr)
		if err != nil {
			return err
		}
		regs[inst.rA] = value
		cpu.pc += 2

	case 2: // STW - Store word
		addr := uint16(int32(regs[inst.rB]) + int32(inst.imm7))
		err := cpu.storeWord(addr, regs[inst.rA])
		if err != nil {
			return err
		}
		cpu.pc += 2

	case 3: // STB - Store byte
		addr := uint16(int32(regs[inst.rB]) + int32(inst.imm7))
		err := cpu.storeByte(addr, regs[inst.rA])
		if err != nil {
			return err
		}
		cpu.pc += 2

	case 4: // ADI - Add immediate
		var src uint16
		if inst.rB == 0 {
			src = 0
		} else {
			src = regs[inst.rB]
		}

		result := uint32(src) + uint32(uint16(inst.imm7))
		carry := (result & 0x10000) != 0

		// Overflow: sign of operands same, result sign different
		overflow := ((src^uint16(inst.imm7))&0x8000) == 0 &&
			((src^uint16(result))&0x8000) != 0

		cpu.updateFlags(result, carry, overflow)

		// If rA is 0, store to LINK register
		if inst.rA == 0 {
			cpu.spr[cpu.mode][SPR_LINK] = uint16(result)
		} else {
			regs[inst.rA] = uint16(result)
		}
		cpu.pc += 2

	case 5: // LUI - Load upper immediate
		value := inst.imm10 << 6
		if inst.rA == 0 {
			cpu.spr[cpu.mode][SPR_LINK] = value
		} else {
			regs[inst.rA] = value
		}
		cpu.pc += 2

	case 6: // BRx - Conditional branch
		condition := cpu.evaluateBranchCondition(inst.branchCond)
		if condition {
			// Branch offset is in bytes, relative to next instruction (PC+2)
			offset := int32(int16(inst.imm10))
			cpu.pc = uint16(int32(cpu.pc+2) + offset)

			// If this is BRL (branch and link), save return address
			if inst.branchCond == BR_LINK {
				cpu.spr[cpu.mode][SPR_LINK] = cpu.pc
			}
		} else {
			cpu.pc += 2
		}

	case 7: // JAL - Jump and link
		// Construct 16-bit address: high 10 bits from rB, low 6 bits from immediate
		var baseReg uint16
		if inst.rB == 0 {
			baseReg = cpu.spr[cpu.mode][SPR_LINK]
		} else {
			baseReg = regs[inst.rB]
		}

		targetAddr := (baseReg & 0xFFC0) | (inst.imm10 & 0x003F)

		// Save return address (PC + 2 advances to next instruction in byte addressing)
		returnAddr := cpu.pc + 2
		if inst.rA == 0 {
			cpu.spr[cpu.mode][SPR_LINK] = returnAddr
		} else {
			regs[inst.rA] = returnAddr
		}

		cpu.pc = targetAddr

	default:
		cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
	}

	return nil
}

// evaluateBranchCondition evaluates a branch condition
func (cpu *CPU) evaluateBranchCondition(cond uint8) bool {
	flags := cpu.getFlags()
	c := (flags & FLAG_C) != 0
	z := (flags & FLAG_Z) != 0
	n := (flags & FLAG_N) != 0
	v := (flags & FLAG_V) != 0

	switch cond {
	case BR_ALWAYS, BR_LINK:
		return true
	case BR_EQ: // Z set
		return z
	case BR_NE: // Z clear
		return !z
	case BR_CS: // C set (unsigned >=)
		return c
	case BR_CC: // C clear (unsigned <)
		return !c
	case BR_GE: // N == V (signed >=)
		return n == v
	case BR_LT: // N != V (signed <)
		return n != v
	default:
		return false
	}
}

// executeXOP executes 3-operand ALU instructions
func (cpu *CPU) executeXOP(inst *Instruction) error {
	regs := &cpu.gen[cpu.mode]
	rB := regs[inst.rB]
	rC := regs[inst.rC]

	var result uint32
	var carry, overflow bool

	flags := cpu.getFlags()
	carryIn := (flags & FLAG_C) != 0

	switch inst.xop {
	case 0: // SBB - Subtract with borrow
		result = uint32(rB) - uint32(rC) - boolToUint32(!carryIn)
		carry = (result & 0x10000) == 0 // Borrow
		overflow = ((rB^rC)&0x8000) != 0 && ((rB^uint16(result))&0x8000) != 0

	case 1: // ADC - Add with carry
		result = uint32(rB) + uint32(rC) + boolToUint32(carryIn)
		carry = (result & 0x10000) != 0
		overflow = ((rB^rC)&0x8000) == 0 && ((rB^uint16(result))&0x8000) != 0

	case 2: // SUB - Subtract
		result = uint32(rB) - uint32(rC)
		carry = (result & 0x10000) == 0 // Borrow
		overflow = ((rB^rC)&0x8000) != 0 && ((rB^uint16(result))&0x8000) != 0

	case 3: // ADD - Add
		result = uint32(rB) + uint32(rC)
		carry = (result & 0x10000) != 0
		overflow = ((rB^rC)&0x8000) == 0 && ((rB^uint16(result))&0x8000) != 0

	case 4: // XOR
		result = uint32(rB ^ rC)
		carry = false
		overflow = false

	case 5: // OR
		result = uint32(rB | rC)
		carry = false
		overflow = false

	case 6: // AND
		result = uint32(rB & rC)
		carry = false
		overflow = false

	default:
		cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
		return nil
	}

	cpu.updateFlags(result, carry, overflow)
	regs[inst.rA] = uint16(result)
	cpu.pc += 2
	return nil
}

// executeYOP executes 2-operand instructions
func (cpu *CPU) executeYOP(inst *Instruction) error {
	regs := &cpu.gen[cpu.mode]

	switch inst.yop {
	case 0: // LSP - Load special register
		addr := regs[inst.rB]
		value, err := cpu.loadSPR(addr)
		if err != nil {
			return err
		}
		regs[inst.rA] = value
		cpu.pc += 2

	case 1: // LSI - Load special register indirect
		sprAddr := regs[inst.rA]
		memAddr := regs[inst.rB]
		value, err := cpu.loadSPR(sprAddr)
		if err != nil {
			return err
		}
		err = cpu.storeWord(memAddr, value)
		if err != nil {
			return err
		}
		cpu.pc += 2

	case 2: // SSP - Store special register
		addr := regs[inst.rB]
		value := regs[inst.rA]
		err := cpu.storeSPR(addr, value)
		if err != nil {
			return err
		}
		cpu.pc += 2

	case 3: // SSI - Store special register indirect
		sprAddr := regs[inst.rA]
		memAddr := regs[inst.rB]
		value, err := cpu.loadWord(memAddr)
		if err != nil {
			return err
		}
		err = cpu.storeSPR(sprAddr, value)
		if err != nil {
			return err
		}
		cpu.pc += 2

	case 4: // LCW - Load code word
		addr := regs[inst.rB]
		value, err := cpu.loadCodeWord(addr)
		if err != nil {
			return err
		}
		regs[inst.rA] = value
		cpu.pc += 2

	case 5: // SYS - System call
		// rB must be 0, rA selects vector 0-7
		if inst.rB != 0 || inst.rA > 7 {
			cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
			return nil
		}
		// SYS 0-7 map to exception vectors 0x0010 through 0x001E (step 2)
		vector := EX_SYSCALL_BASE + (uint16(inst.rA) * 2)
		cpu.raiseException(vector, 0)

	default:
		cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
	}

	return nil
}

// executeZOP executes single-operand instructions
func (cpu *CPU) executeZOP(inst *Instruction) error {
	regs := &cpu.gen[cpu.mode]
	value := regs[inst.rA]

	var result uint16
	var carry bool

	switch inst.zop {
	case 0: // NOT
		result = ^value
		cpu.updateFlags(uint32(result), false, false)

	case 1: // NEG
		result = uint16(-int16(value))
		cpu.updateFlags(uint32(result), false, false)

	case 2: // ZXT - Zero extend lower byte
		result = value & 0x00FF
		carry = false
		cpu.updateFlags(uint32(result), carry, false)

	case 3: // SXT - Sign extend lower byte
		if value&0x80 != 0 {
			result = value | 0xFF00
		} else {
			result = value & 0x00FF
		}
		carry = false
		cpu.updateFlags(uint32(result), carry, false)

	case 4: // SRA - Shift right arithmetic
		carry = (value & 0x0001) != 0
		result = uint16(int16(value) >> 1)
		cpu.setFlags((cpu.getFlags() & 0xFFFE) | boolToUint16(carry))

	case 5: // SRL - Shift right logical
		carry = (value & 0x0001) != 0
		result = value >> 1
		cpu.setFlags((cpu.getFlags() & 0xFFFE) | boolToUint16(carry))

	case 6: // DUB - Duplicate upper byte
		upperByte := (value >> 8) & 0xFF
		result = (upperByte << 8) | upperByte
		carry = false
		cpu.updateFlags(uint32(result), carry, false)

	default:
		cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
		return nil
	}

	regs[inst.rA] = result
	cpu.pc += 2
	return nil
}

// executeVOP executes zero-operand instructions
func (cpu *CPU) executeVOP(inst *Instruction) error {
	switch inst.vop {
	case 0: // CCF - Clear carry flag
		cpu.setFlags(cpu.getFlags() & 0xFFFE)
		cpu.pc += 2

	case 1: // SCF - Set carry flag
		cpu.setFlags(cpu.getFlags() | FLAG_C)
		cpu.pc += 2

	case 2: // DI - Disable interrupts
		if cpu.mode != ModeKernel {
			cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
			return nil
		}
		cpu.intEnabled = false
		cpu.pc += 2

	case 3: // EI - Enable interrupts
		if cpu.mode != ModeKernel {
			cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
			return nil
		}
		cpu.intEnabled = true
		cpu.pc += 2

	case 4: // HLT - Halt
		if cpu.mode != ModeKernel {
			cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
			return nil
		}
		cpu.running = false
		cpu.pc += 2

	case 5: // BRK - Breakpoint (emulator support)
		// In emulator, this can be used for debugging
		// For now, treat as NOP
		cpu.pc += 2

	case 6: // RTI - Return from interrupt
		if cpu.mode != ModeKernel {
			cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
			return nil
		}

		// Restore state from special registers
		cpu.pc = cpu.spr[ModeKernel][SPR_IRR]
		prevMode := cpu.spr[ModeKernel][SPR_ISR] & 0x01
		cpu.mode = uint8(prevMode)
		cpu.intEnabled = true

		// Trace mode switch
		if cpu.tracer != nil {
			cpu.tracer.TraceModeSwitch(cpu, ModeKernel, cpu.mode)
		}

	case 7: // DIE - Always illegal (0xFFFF)
		cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)

	default:
		cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
	}

	return nil
}

// Helper functions

func boolToUint32(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

func boolToUint16(b bool) uint16 {
	if b {
		return 1
	}
	return 0
}
