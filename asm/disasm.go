package main

import (
	"fmt"
	"os"
)

/* Disassemble a binary file */
func disassemble(filename string) error {
	/* Read file */
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	if len(data) < 16 {
		return fmt.Errorf("file too small to be valid WUT-4 binary")
	}

	/* Parse header */
	magic := readU16(data, 0)
	if magic != 0xDDD1 {
		return fmt.Errorf("invalid magic number: 0x%04X (expected 0xDDD1)", magic)
	}

	codeSize := int(readU16(data, 2))
	dataSize := int(readU16(data, 4))

	fmt.Printf("; WUT-4 Disassembly\n")
	fmt.Printf("; Code size: %d bytes\n", codeSize)
	fmt.Printf("; Data size: %d bytes\n", dataSize)
	fmt.Printf("\n")

	/* Disassemble code segment */
	if codeSize > 0 {
		fmt.Printf(".code\n")
		offset := 16
		addr := 0
		for addr < codeSize {
			if offset+1 >= len(data) {
				break
			}
			word := readU16(data, offset)
			instr := disasmInstruction(word, addr)
			fmt.Printf("%04X: %04X  %s\n", addr, word, instr)
			offset += 2
			addr += 2
		}
	}

	/* Disassemble data segment */
	if dataSize > 0 {
		fmt.Printf("\n.data\n")
		offset := 16 + codeSize
		addr := 0
		for addr < dataSize {
			if offset >= len(data) {
				break
			}
			/* Print as bytes */
			if addr%16 == 0 {
				fmt.Printf("%04X: .bytes ", addr)
			}
			fmt.Printf("0x%02X", data[offset])
			addr++
			offset++
			if addr%16 == 0 || addr >= dataSize {
				fmt.Printf("\n")
			} else {
				fmt.Printf(", ")
			}
		}
	}

	return nil
}

/* Disassemble a single instruction */
func disasmInstruction(word uint16, addr int) string {
	/* Check bits [15:13] to determine instruction type */
	opHi := (word >> 13) & 0x7

	switch opHi {
	case 0x0: /* 000 */
		return disasmLDW(word)
	case 0x1: /* 001 */
		return disasmLDB(word)
	case 0x2: /* 010 */
		return disasmSTW(word)
	case 0x3: /* 011 */
		return disasmSTB(word)
	case 0x4: /* 100 */
		return disasmADI(word)
	case 0x5: /* 101 */
		return disasmLUI(word)
	case 0x6: /* 110 */
		return disasmBRx(word, addr)
	case 0x7: /* 111 */
		/* Check bit 12 to distinguish JAL from extended */
		if (word & 0x1000) == 0 {
			return disasmJAL(word)
		}
		return disasmExtended(word)
	default:
		return fmt.Sprintf("??? (0x%04X)", word)
	}
}

/* Disassemble base instructions */
func disasmLDW(word uint16) string {
	rA := word & 0x7
	rB := (word >> 3) & 0x7
	imm := int((word >> 6) & 0x7F)
	imm = signExtend(imm, 7)
	if imm == 0 {
		return fmt.Sprintf("ldw r%d, r%d", rA, rB)
	}
	return fmt.Sprintf("ldw r%d, r%d, %d", rA, rB, imm)
}

func disasmLDB(word uint16) string {
	rA := word & 0x7
	rB := (word >> 3) & 0x7
	imm := int((word >> 6) & 0x7F)
	imm = signExtend(imm, 7)
	if imm == 0 {
		return fmt.Sprintf("ldb r%d, r%d", rA, rB)
	}
	return fmt.Sprintf("ldb r%d, r%d, %d", rA, rB, imm)
}

func disasmSTW(word uint16) string {
	rA := word & 0x7
	rB := (word >> 3) & 0x7
	imm := int((word >> 6) & 0x7F)
	imm = signExtend(imm, 7)
	if imm == 0 {
		return fmt.Sprintf("stw r%d, r%d", rA, rB)
	}
	return fmt.Sprintf("stw r%d, r%d, %d", rA, rB, imm)
}

func disasmSTB(word uint16) string {
	rA := word & 0x7
	rB := (word >> 3) & 0x7
	imm := int((word >> 6) & 0x7F)
	imm = signExtend(imm, 7)
	if imm == 0 {
		return fmt.Sprintf("stb r%d, r%d", rA, rB)
	}
	return fmt.Sprintf("stb r%d, r%d, %d", rA, rB, imm)
}

func disasmADI(word uint16) string {
	rA := word & 0x7
	rB := (word >> 3) & 0x7
	imm := int((word >> 6) & 0x7F)
	imm = signExtend(imm, 7)
	if imm == 0 {
		return fmt.Sprintf("adi r%d, r%d", rA, rB)
	}
	return fmt.Sprintf("adi r%d, r%d, %d", rA, rB, imm)
}

func disasmLUI(word uint16) string {
	rA := word & 0x7
	imm := (word >> 3) & 0x3FF
	return fmt.Sprintf("lui r%d, 0x%X", rA, imm)
}

func disasmBRx(word uint16, addr int) string {
	cond := word & 0x7
	offset := int((word >> 3) & 0x3FF)
	offset = signExtend(offset, 10)

	target := addr + 2 + offset

	condNames := []string{"br", "brl", "brz", "brnz", "brc", "brnc", "brsge", "brslt"}
	condName := "br"
	if int(cond) < len(condNames) {
		condName = condNames[cond]
	}

	return fmt.Sprintf("%s %d ; -> 0x%04X", condName, offset, target)
}

func disasmJAL(word uint16) string {
	rA := word & 0x7
	rB := (word >> 3) & 0x7
	imm := (word >> 6) & 0x3F
	if imm == 0 {
		return fmt.Sprintf("jal r%d, r%d", rA, rB)
	}
	return fmt.Sprintf("jal r%d, r%d, %d", rA, rB, imm)
}

func disasmExtended(word uint16) string {
	/* Check bits [15:9] */
	if (word & 0xFE00) != 0xFE00 {
		/* XOP */
		return disasmXOP(word)
	}

	/* Check bits [15:6] */
	if (word & 0xFFC0) != 0xFFC0 {
		/* YOP */
		return disasmYOP(word)
	}

	/* Check bits [15:3] */
	if (word & 0xFFF8) != 0xFFF8 {
		/* ZOP */
		return disasmZOP(word)
	}

	/* VOP */
	return disasmVOP(word)
}

func disasmXOP(word uint16) string {
	rA := word & 0x7
	rB := (word >> 3) & 0x7
	rC := (word >> 6) & 0x7
	xop := (word >> 9) & 0x7

	xopNames := []string{"sbb", "adc", "sub", "add", "xor", "or", "and"}
	opName := "xop"
	if int(xop) < len(xopNames) {
		opName = xopNames[xop]
	}

	return fmt.Sprintf("%s r%d, r%d, r%d", opName, rA, rB, rC)
}

func disasmYOP(word uint16) string {
	rA := word & 0x7
	rB := (word >> 3) & 0x7
	yop := (word >> 6) & 0x7

	/* Special case for SYS */
	if yop == 5 {
		if rB == 0 {
			return fmt.Sprintf("sys %d", rA)
		}
		return fmt.Sprintf("sys r%d, r%d ; invalid", rA, rB)
	}

	yopNames := []string{"lsp", "lsi", "ssp", "ssi", "lcw", "sys", "tst"}
	opName := "yop"
	if int(yop) < len(yopNames) {
		opName = yopNames[yop]
	}

	return fmt.Sprintf("%s r%d, r%d", opName, rA, rB)
}

func disasmZOP(word uint16) string {
	rA := word & 0x7
	zop := (word >> 3) & 0x7

	zopNames := []string{"not", "neg", "tst", "sxt", "sra", "srl", "ji"}
	opName := "zop"
	if int(zop) < len(zopNames) {
		opName = zopNames[zop]
	}

	return fmt.Sprintf("%s r%d", opName, rA)
}

func disasmVOP(word uint16) string {
	vop := word & 0x7

	vopNames := []string{"ccf", "scf", "di", "ei", "hlt", "brk", "rti", "die"}
	if int(vop) < len(vopNames) {
		return vopNames[vop]
	}
	return "vop"
}
