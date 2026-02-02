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

// loadSPR loads a value from a special register
func (cpu *CPU) loadSPR(addr uint16) (uint16, error) {
	spr := addr & 0x7F // Only 128 special registers

	// Trace SPR read
	defer func() {
		if cpu.tracer != nil && !cpu.pendingException {
			value := cpu.spr[cpu.mode][spr]
			if spr == SPR_CYCLO {
				value = uint16(cpu.cycles & 0xFFFF)
			} else if spr == SPR_CYCHI {
				value = uint16((cpu.cycles >> 16) & 0xFFFF)
			}
			cpu.tracer.TraceSPRRead(spr, value)
		}
	}()

	// User mode can only access SPRs 0-7
	if cpu.mode == ModeUser && spr >= 8 {
		cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
		return 0, nil
	}

	switch spr {
	case SPR_LINK:
		return cpu.spr[cpu.mode][SPR_LINK], nil

	case SPR_FLAGS:
		// Return flags with IE bit reflecting interrupt enable state
		flags := cpu.spr[cpu.mode][SPR_FLAGS] & 0x00FF
		if cpu.intEnabled {
			flags |= FLAG_IE
		}
		return flags, nil

	case 2, 3, 4, 5:
		// Undefined SPRs - return 0
		return 0, nil

	case SPR_CYCLO:
		return uint16(cpu.cycles & 0xFFFF), nil

	case SPR_CYCHI:
		return uint16((cpu.cycles >> 16) & 0xFFFF), nil

	case SPR_IRR, SPR_ICR, SPR_IDR, SPR_ISR:
		// Kernel-only registers
		return cpu.spr[ModeKernel][spr], nil

	case 12, 13, 14:
		// Reserved SPRs
		return 0, nil

	case SPR_CONTEXT:
		// Context register
		return uint16(cpu.context), nil

	case 16, 17, 18, 19, 20, 21, 22, 23:
		// User general registers (kernel can access user mode regs)
		regNum := spr - SPR_USERGEN_BASE
		return cpu.gen[ModeUser][regNum], nil

	case 24, 25, 26, 27, 28, 29, 30, 31:
		// User special registers
		if spr == 25 {
			// User LINK register
			return cpu.spr[ModeUser][SPR_LINK], nil
		}
		return 0, nil

	default:
		if spr >= SPR_USER_CODE_MMU_BASE && spr < SPR_USER_DATA_MMU_BASE {
			// User code MMU (SPR 32-47)
			slot := int(spr - SPR_USER_CODE_MMU_BASE)
			return cpu.mmu[cpu.context][slot], nil

		} else if spr >= SPR_USER_DATA_MMU_BASE && spr < SPR_KERN_CODE_MMU_BASE {
			// User data MMU (SPR 48-63)
			slot := 16 + int(spr-SPR_USER_DATA_MMU_BASE)
			return cpu.mmu[cpu.context][slot], nil

		} else if spr >= SPR_KERN_CODE_MMU_BASE && spr < SPR_KERN_DATA_MMU_BASE {
			// Kernel code MMU (SPR 64-79)
			slot := int(spr - SPR_KERN_CODE_MMU_BASE)
			return cpu.mmu[0][slot], nil

		} else if spr >= SPR_KERN_DATA_MMU_BASE && spr < SPR_IO_BASE {
			// Kernel data MMU (SPR 80-95)
			slot := 16 + int(spr-SPR_KERN_DATA_MMU_BASE)
			return cpu.mmu[0][slot], nil

		} else if spr >= SPR_IO_BASE {
			// I/O registers (SPR 96-127)
			return cpu.loadIO(spr)
		}
	}

	return 0, nil
}

// storeSPR stores a value to a special register
func (cpu *CPU) storeSPR(addr uint16, value uint16) error {
	spr := addr & 0x7F

	// Trace SPR write
	if cpu.tracer != nil {
		cpu.tracer.TraceSPRWrite(spr, value)
	}

	// User mode can only access SPR 0 (LINK)
	if cpu.mode == ModeUser {
		if spr == SPR_LINK {
			cpu.spr[ModeUser][SPR_LINK] = value
			return nil
		}
		cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
		return nil
	}

	// Kernel mode
	switch spr {
	case SPR_LINK:
		cpu.spr[ModeKernel][SPR_LINK] = value

	case SPR_FLAGS:
		// Bit 8 (T flag) is writable, bit 9 (IE) is read-only
		cpu.spr[ModeKernel][SPR_FLAGS] = value & 0x01FF
		// IE flag is controlled by DI/EI/RTI instructions, not by writes

	case 2, 3, 4, 5:
		// Undefined - writes ignored

	case SPR_CYCLO, SPR_CYCHI:
		// Read-only - writes ignored

	case SPR_IRR, SPR_ICR, SPR_IDR, SPR_ISR:
		cpu.spr[ModeKernel][spr] = value

	case 12, 13, 14:
		// Reserved - writes ignored

	case SPR_CONTEXT:
		// Writing context register switches user context
		if value >= 1 && value <= 255 {
			cpu.context = uint8(value)
		}

	case 16, 17, 18, 19, 20, 21, 22, 23:
		// Set user general registers
		regNum := spr - SPR_USERGEN_BASE
		cpu.gen[ModeUser][regNum] = value

	case 25:
		// User LINK register
		cpu.spr[ModeUser][SPR_LINK] = value

	default:
		if spr >= SPR_USER_CODE_MMU_BASE && spr < SPR_USER_DATA_MMU_BASE {
			// User code MMU
			slot := int(spr - SPR_USER_CODE_MMU_BASE)
			cpu.mmu[cpu.context][slot] = value

		} else if spr >= SPR_USER_DATA_MMU_BASE && spr < SPR_KERN_CODE_MMU_BASE {
			// User data MMU
			slot := 16 + int(spr-SPR_USER_DATA_MMU_BASE)
			cpu.mmu[cpu.context][slot] = value

		} else if spr >= SPR_KERN_CODE_MMU_BASE && spr < SPR_KERN_DATA_MMU_BASE {
			// Kernel code MMU
			slot := int(spr - SPR_KERN_CODE_MMU_BASE)
			cpu.mmu[0][slot] = value

		} else if spr >= SPR_KERN_DATA_MMU_BASE && spr < SPR_IO_BASE {
			// Kernel data MMU
			slot := 16 + int(spr-SPR_KERN_DATA_MMU_BASE)
			cpu.mmu[0][slot] = value

		} else if spr >= SPR_IO_BASE {
			// I/O registers
			return cpu.storeIO(spr, value)
		}
	}

	return nil
}

// loadIO loads from an I/O register
func (cpu *CPU) loadIO(spr uint16) (uint16, error) {
	if cpu.mode != ModeKernel {
		cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
		return 0, nil
	}

	switch spr {
	case 96:
		// u0 - Write-only transmit data register
		return 0, nil

	case 97:
		// u1 - Read receive data register
		return cpu.uartReadData(), nil

	case 98:
		// u2 - Transmit status register
		return cpu.uartReadTxStatus(), nil

	case 99:
		// u3 - Receive status register
		return cpu.uartReadRxStatus(), nil

	case SPR_SPI_DATA:
		// s0 - SPI data register (read)
		return cpu.spiReadData(), nil

	case SPR_SPI_SELECT:
		// s1 - SPI select register (read returns current state)
		return cpu.spiReadSelect(), nil

	default:
		// Other I/O registers undefined
		return 0, nil
	}
}

// storeIO stores to an I/O register
func (cpu *CPU) storeIO(spr uint16, value uint16) error {
	if cpu.mode != ModeKernel {
		cpu.raiseException(EX_ILLEGAL_INST, cpu.pc)
		return nil
	}

	switch spr {
	case 96:
		// u0 - Write transmit data register
		cpu.uartWriteData(value)

	case 97:
		// u1 - Read-only register, writes ignored

	case 98, 99:
		// u2, u3 - Status registers, writes ignored

	case SPR_SPI_DATA:
		// s0 - SPI data register (write)
		cpu.spiWriteData(value)

	case SPR_SPI_SELECT:
		// s1 - SPI select register (write)
		cpu.spiWriteSelect(value)

	default:
		// Other I/O registers undefined
	}

	return nil
}

func init() {
	// Verify constant definitions at compile time
	if SPR_LINK != 0 {
		panic(fmt.Sprintf("SPR_LINK should be 0, got %d", SPR_LINK))
	}
	if SPR_FLAGS != 1 {
		panic(fmt.Sprintf("SPR_FLAGS should be 1, got %d", SPR_FLAGS))
	}
}
