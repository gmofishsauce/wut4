/*
Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com)

This program is free software: you can redistribute it and/or modify it
under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful, but
WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public
License along with this program. If not, see
<http://www.gnu.org/licenses/>.
*/
package main

// Fetch next instruction into ir.
func (w4 *w4machine) fetch() {
	if w4.ex != 0 {
		// double fault should have been handled in main loop.
		assert(w4.en, "double fault in fetch()")

		// an exception occurred during the previous cycle.
		w4.reg[Kern].spr[Irr] = w4.pc
		w4.reg[Kern].spr[Icr] = word(w4.ex)
		w4.reg[Kern].spr[Imr] = word(w4.mode)

		w4.mode = Kern
		w4.pc = word(w4.ex)
		w4.en = false
		w4.ex = 0
	}

	ex, addr := w4.translate(false, w4.pc)
	if ex != ExNone {
		w4.ex = ex
		return
	}
	w4.ir = physmem[addr]

	// Control flow instructions will overwrite this in a later stage.
	// This implementation is sequential (does everything each clock cycle).
	w4.pc++
}

// Pull out all the possible distinct field types into uint16s. The targets
// (op, i7, yop, etc.) are all non-architectural per-cycle and mostly mean
// e.g. multiplexer outputs in hardware. The remaining stages can act on the
// decoded values. Plausible additional decoding (which instructions have
// targets? Which target special registers?) is left to the execution code.
func (w4 *w4machine) decode() {
	if w4.ex != 0 {
		// There was an exception during fetch
		return
	}

	w4.op = w4.ir.bits(15, 13) // base opcode
	w4.imm = w4.sxtImm()

	w4.xop = w4.ir.bits(11, 9)
	w4.yop = w4.ir.bits(8, 6)
	w4.zop = w4.ir.bits(5, 3)
	w4.vop = w4.ir.bits(2, 0)

	w4.isVop = w4.ir.bits(15, 3) == 0x1FFF
	w4.isZop = !w4.isVop && w4.ir.bits(15, 6) == 0x03FF
	w4.isYop = !w4.isVop && !w4.isZop && w4.ir.bits(15, 9) == 0x007F
	w4.isXop = !w4.isVop && !w4.isZop && !w4.isYop && w4.ir.bits(15, 12) == 0x000F
	w4.isBase = !w4.isVop && !w4.isZop && !w4.isYop && !w4.isXop

	w4.ra = w4.vop
	w4.rb = w4.zop
	w4.rc = w4.yop
}

// Set the ALU output and memory (for stores) data in the
// non-architectural per-cycle machine state. Again,
// somewhat like the eventual pipelined implementation.
func (w4 *w4machine) execute() {
	if w4.ex != 0 {
		// The program counter gets modified by the execution
		// stage, so we must not proceed if there has been any
		// exception caused by the fetch or decode activities.
		return
	}
	if w4.isBase {
		baseops[w4.op]()
	} else if w4.isXop {
		w4.alu3()
	} else if w4.isYop {
		yops[w4.yop]()
	} else if w4.isZop {
		w4.alu1()
	} else {
		if !w4.isVop {
			w4.decodeFailure("vop")
		}
		vops[w4.vop]()
	}
}

// For instructions that reference memory, special register space,
// or I/O space, do the operation. The computed address is in the alu
// (alu result) register and the execute phase must also have loaded
// the store data register.
func (w4 *w4machine) memory() {
	if w4.ex != 0 { // exception pending - don't modify memory
		return
	}

	// Default the writeback register to the alu output. It gets
	// overwritten in the code below by memory, io, or spr read,
	// if any. In the writeback stage, it gets used, or it just
	// doesn't, depending on the instruction.
	w4.wb = word(w4.alu)

	if w4.op < 4 { // general register load or store
		isOdd := w4.alu&1 != 0
		ex, addr := w4.translate(true, word(w4.alu))
		if ex != ExNone {
			w4.ex = ex
			return
		}

		switch w4.op {
		case 0: // ldw
			if isOdd {
				w4.ex = ExMemory
				break
			}
			w4.wb = physmem[addr]
		case 1: // ldb
			memWord := physmem[addr]
			if isOdd {
				w4.wb = memWord >> 8
			} else {
				w4.wb = memWord & 0xFF
			}
		case 2: // stw
			if isOdd {
				w4.ex = ExMemory
				break
			}
			dbg("stw memory phase: addr, sd = 0x%04X, 0x%04X", addr, w4.sd)
			physmem[addr] = w4.sd
		case 3: // stb
			memWord := physmem[addr]
			if isOdd {
				memWord &= 0xFF
				memWord |= (w4.sd << 8)
			} else {
				memWord &= 0xFF00
				memWord |= w4.sd & 0xFF
			}
			physmem[addr] = memWord
			// no default
		}
	} else if w4.isYop { // special register or IO load or store
		switch w4.yop {
		case 0: // lsp (load special)
			w4.wb = w4.loadSpecial()
		case 1: // lio (load from io)
			w4.wb = w4.loadIO()
		case 2: // ssp (store special)
			w4.storeSpecial(w4.sd)
		case 3: // sio
			w4.storeIO(w4.sd)
			// no default
		}
	}
}

// return the value of the special register addressed by the ALU result
// from the previous stage. May set an exception, in which case the result
// value doesn't matter because it won't be written back to a register.
func (w4 *w4machine) loadSpecial() word {
	r := w4.alu & (SprSize - 1) // 0..63
	switch r {                  // no default
	case PC:
		return w4.pc
	case Link:
		return w4.reg[w4.mode].spr[Link]
	case Irr, Icr, Imr, 5:
		if w4.mode == Kern {
			return w4.reg[w4.mode].spr[r]
		}
		w4.ex = ExIllegal
		return 0
	case CCLS:
		return word(w4.cyc & 0xFFFF)
	case CCMS:
		return word((w4.cyc & 0xFFFF0000) >> 16)
	}
	if w4.mode == User {
		w4.ex = ExIllegal
		return 0
	}
	switch {
	case r == MmuCtl1:
		return w4.reg[Kern].spr[MmuCtl1]
	case r > 8 && r < 16: // unused SPRs
		return 0
	case r >= 16 && r < 24: // user general registers
		return w4.reg[User].gen[r-16]
	case r >= 24 && r < 31: // user special registers
		if r == 25 { // user link register
			// Could allow the kernel to access the PC
			// here, or CCLS/CCMS, but it's stupid.
			return w4.reg[User].spr[Link]
		}
	case r >= 32: // MMU - MmuCtl1 gives kern access to user
		if w4.reg[Kern].spr[MmuCtl1]&0x10 != 0 {
			return w4.reg[User].spr[r]
		} else {
			return w4.reg[Kern].spr[r]
		}
	default:
		w4.ex = ExIllegal
		return 0
	}
	// All the cases should have been handled,
	// so this should not be reachable.
	assert(false, "missing case in loadSpecial()")
	return 0
}

func (w4 *w4machine) loadIO() word {
	TODO()
	return 0
}

func (w4 *w4machine) storeSpecial(val word) {
	r := w4.alu & (SprSize - 1) // 0..63
	if w4.mode == User {
		// user mode can write its own link register
		if r == Link {
			w4.reg[User].spr[Link] = val
			return
		}
		w4.ex = ExIllegal
		return
	}
	switch {
	case r == Irr, r == Icr, r == Imr, r == 5, r == MmuCtl1:
		w4.reg[Kern].spr[r] = val
	case r >= 16 && r < 24: // set user general register
		w4.reg[User].gen[r-16] = val
	case r == 25: // set user link register
		w4.reg[User].spr[Link] = val
	case r >= 32: // set MMU entry, MmuCtl1&0x10 gives kernel access to user
		if w4.reg[Kern].spr[MmuCtl1]&0x10 != 0 {
			w4.reg[User].spr[r] = val
		} else {
			w4.reg[Kern].spr[r] = val
		}
	default:
		w4.ex = ExIllegal // likely double fault
	}
}

func (w4 *w4machine) storeIO(val word) {
	TODO()
}

// Write the result (including possible memory result) to a register.
// Stores and io writes are handled at memory time.
func (w4 *w4machine) writeback() {
	if w4.ex != 0 { // exception pending - don't update registers
		return
	}

	reg := w4.reg[w4.mode]
	if w4.op == 0 || // ldw
		w4.op == 1 || // ldb
		w4.op == 5 || // adi
		w4.op == 6 || // lui
		w4.isXop || // 3-operand alu
		(w4.isYop && w4.yop < 2) || // lsp or lio
		w4.isZop { // single operand alu

		if w4.ra != 0 {
			reg.gen[w4.ra] = w4.wb
		}
	}
}

// ================================================================
// === The rest of this file is the implementation of execute() ===
// ================================================================

// The opcodes basically spread out to the right, using more and
// more leading 1-bits. The bits come in groups of 3, with the
// special case that 1110... is jlr and 1111... requires decoding
// the next three (XOP) bits. After that, 1111 111... requires
// decoding the next three bits, then 1111 111 111..., etc.
//
// The decoder already figured this out and set isx, xop, isy,
// yop, and so on. We just need to switch on them and do all
// the things.

type xf func()

// We need a function with a parameter for reporting decode
// failures (internal errors). Then we need wrappers of type
// xf for the tables.
func (w4 *w4machine) decodeFailure(msg string) {
	w4.dump()
	panic("executeSequential(): decode failure: " + msg)
}

func (w4 *w4machine) baseFail() {
	w4.decodeFailure("base")
}

func (w4 *w4machine) yopFail() {
	w4.decodeFailure("yop")
}

func (w4 *w4machine) zopFail() {
	w4.decodeFailure("zop")
}

var baseops []xf = []xf{
	w4.ldw,
	w4.ldb,
	w4.stw,
	w4.stb,
	w4.beq,
	w4.adi,
	w4.lui,
	w4.jlr,
}

var yops []xf = []xf{
	w4.lsp,
	w4.lio,
	w4.ssp,
	w4.sio,
	w4.y04,
	w4.y05,
	w4.y06,
	w4.yopFail,
}

var vops []xf = []xf{
	w4.rti,
	w4.rtl,
	w4.di,
	w4.ei,
	w4.hlt,
	w4.brk,
	w4.v06,
	w4.die,
}

// base operations

func (w4 *w4machine) ldw() {
	// We end up here for zero opcodes. These try to load
	// r0 which is the black hole register. Instead of having
	// them be noops, we call them illegal instructions. This
	// prevents running uninitialized memory in the emulator,
	// which inits memory to 0 because it's written in Golang.
	if w4.ir == 0 {
		w4.ex = ExIllegal
		return
	}
	reg := w4.reg[w4.mode].gen
	w4.alu = uint16(reg[w4.rb]) + w4.imm
}

func (w4 *w4machine) ldb() {
	reg := w4.reg[w4.mode].gen
	w4.alu = uint16(reg[w4.rb]) + w4.imm
}

func (w4 *w4machine) stw() {
	reg := w4.reg[w4.mode].gen
	w4.alu = uint16(reg[w4.rb]) + w4.imm
	w4.sd = reg[w4.ra]
}

func (w4 *w4machine) stb() {
	reg := w4.reg[w4.mode].gen
	w4.alu = uint16(reg[w4.rb]) + w4.imm
	w4.sd = reg[w4.ra]
}

func (w4 *w4machine) beq() {
	reg := w4.reg[w4.mode].gen
	if reg[w4.rb] == reg[w4.ra] {
		w4.pc = word(uint16(w4.pc) + w4.imm)
	}
	// no standard register writeback
}

func (w4 *w4machine) adi() {
	reg := w4.reg[w4.mode].gen
	w4.alu = uint16(reg[w4.rb]) + w4.imm
}

func (w4 *w4machine) lui() {
	w4.alu = w4.imm
}

func (w4 *w4machine) jlr() {
	// the jlr opcode has bits [15..13] == 0b111, just like xops.
	// It's a jlr, not an xop, because bit 12, the MS bit of the
	// immediate value, has to be a 0. The decoder is supposed to
	// take care of this, but for sanity, we check here.
	if w4.ir.bits(15, 12) != 0xE {
		w4.baseFail() // internal error
	}

	// There are three flavors, determined by the rA field, which
	// is overloaded as additional opcode bits here.
	switch w4.ra {
	case 0: // sys trap
		if w4.rb != 0 || w4.imm&1 == 1 || w4.imm < 32 || w4.imm > 62 {
			w4.ex = ExIllegal
			return
		}
		w4.ex = exception(w4.imm)
	case 1: // jump and link
		w4.reg[w4.mode].spr[Link] = w4.pc
		w4.pc = word(uint16(w4.reg[w4.mode].gen[w4.rb]) + w4.imm)
	case 2: // jump
		w4.pc = word(uint16(w4.reg[w4.mode].gen[w4.rb]) + w4.imm)
	default:
		w4.ex = ExIllegal
	}
}

// xops - 3-operand ALU operations all handled here

func (w4 *w4machine) alu3() {
	reg := w4.reg[w4.mode].gen
	rs2 := uint16(reg[w4.rc])
	rs1 := uint16(reg[w4.rb])

	switch w4.xop {
	case 0: // add
		full := uint32(rs2 + rs1)
		w4.alu = uint16(full & 0xFFFF)
		w4.hc = uint16((full & 0x10000) >> 16)
	case 1: // adc
		full := uint32(rs2 + rs1 + w4.hc)
		w4.alu = uint16(full & 0xFFFF)
		w4.hc = uint16((full & 0x10000) >> 16)
	case 2: // sub
		full := uint32(rs2 - rs1)
		w4.alu = uint16(full & 0xFFFF)
		w4.hc = uint16((full & 0x10000) >> 16)
	case 3: // sbc
		full := uint32(rs2 - rs1 - w4.hc)
		w4.alu = uint16(full & 0xFFFF)
		w4.hc = uint16((full & 0x10000) >> 16)
	case 4: // bic (nand)
		full := uint32(rs2 &^ rs1)
		w4.alu = uint16(full & 0xFFFF)
		w4.hc = 0
	case 5: // bis (or)
		full := uint32(rs2 | rs1)
		w4.alu = uint16(full & 0xFFFF)
		w4.hc = 0
	case 6: // xor
		full := uint32(rs2 ^ rs1)
		w4.alu = uint16(full & 0xFFFF)
		w4.hc = 0
	case 7:
		w4.decodeFailure("alu3 op == 7")
	}
}

// yops

func (w4 *w4machine) lsp() {
	// Execution stage merely passes rB to ALU
	// Memory stage will put SPR[alu] in wb
	w4.alu = uint16(w4.reg[w4.mode].gen[w4.rb])
}

func (w4 *w4machine) lio() {
	// Execution stage merely passes rB to ALU
	// Memory stage will put SPR[alu] in wb
	w4.alu = uint16(w4.reg[w4.mode].gen[w4.rb])
}

func (w4 *w4machine) ssp() {
	// Execution stage must set both ALU and sd
	w4.alu = uint16(w4.reg[w4.mode].gen[w4.rb])
	w4.sd = w4.reg[w4.mode].gen[w4.ra]
}

func (w4 *w4machine) sio() {
	// Execution stage must set both ALU and sd
	w4.alu = uint16(w4.reg[w4.mode].gen[w4.rb])
	w4.sd = w4.reg[w4.mode].gen[w4.ra]
}

func (w4 *w4machine) y04() {
	w4.ex = ExIllegal
}

func (w4 *w4machine) y05() {
	// possible future read from code space
	w4.ex = ExIllegal
}

func (w4 *w4machine) y06() {
	// possible future write to code space
	w4.ex = ExIllegal
}

// zops - 1-operand ALU operations all handled here

func (w4 *w4machine) alu1() {
	reg := w4.reg[w4.mode].gen
	rs1 := uint16(reg[w4.ra])

	switch w4.zop {
	case 0: //not
		w4.alu = ^rs1
		w4.hc = 0
	case 1: //neg
		w4.alu = 1 + ^rs1
		w4.hc = 0 // ???
	case 2: //swb
		w4.alu = rs1>>8 | rs1<<8
		w4.hc = 0
	case 3: //sxt
		if rs1&0x80 != 0 {
			w4.alu = rs1 | 0xFF00
		} else {
			w4.alu = rs1 &^ 0xFF00
		}
		w4.hc = 0
	case 4: //lsr
		w4.hc = rs1 & 1
		w4.alu = rs1 >> 1
	case 5: //lsl
		if rs1&0x8000 == 0 {
			w4.hc = 0
		} else {
			w4.hc = 1
		}
		w4.alu = rs1 << 1
	case 6: //asr
		sign := rs1 & 0x8000
		w4.hc = rs1 & 1
		w4.alu = rs1 >> 1
		w4.alu |= sign
	case 7:
		w4.zopFail()
	}
}

// vops - 0 operand instructions

func (w4 *w4machine) rti() {
	if w4.mode == User {
		w4.ex = ExIllegal
		return
	}

	// This is acceptable because (1) the machine is not pipelined
	// and (2) the instruction doesn't do anything else but this.
	// In a pipelined implementation, this would be more complex.
	// Also note that we can enable interrupts when returning from
	// any interrupt or fault, because interrupts must have been
	// enabled for the interrupt or fault to have been taken.
	w4.ex = 0
	w4.en = true
	w4.pc = w4.reg[Kern].spr[Irr]
	w4.reg[Kern].spr[Irr] = 0
	w4.mode = byte(w4.reg[Kern].spr[Imr])
}

func (w4 *w4machine) rtl() {
	w4.pc = w4.reg[w4.mode].spr[Link]
}

func (w4 *w4machine) di() {
	if w4.mode == User {
		w4.ex = ExIllegal
		return
	}

	w4.en = false
}

func (w4 *w4machine) ei() {
	if w4.mode == User {
		w4.ex = ExIllegal
		return
	}

	w4.en = true
}

func (w4 *w4machine) hlt() {
	if w4.mode == User {
		w4.ex = ExIllegal
		return
	}

	w4.run = false
}

func (w4 *w4machine) brk() {
	if w4.mode == User {
		w4.ex = ExIllegal
		return
	}

	// for now
	w4.dump()
}

func (w4 *w4machine) v06() {
	w4.ex = ExIllegal
}

func (w4 *w4machine) die() {
	if w4.mode == User {
		w4.ex = ExIllegal
		return
	}

	w4.ex = ExIllegal
}
