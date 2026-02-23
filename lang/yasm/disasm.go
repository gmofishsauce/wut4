package main

import (
	"fmt"
	"os"
)

func readWord(buf []byte, offset int) uint16 {
	if offset+1 >= len(buf) {
		return 0
	}
	return uint16(buf[offset]) | (uint16(buf[offset+1]) << 8)
}

func unsignedImm7(val int) int {
	return val & 0x7F
}

func signExtend10(val int) int {
	val = val & 0x3FF
	if (val & 0x200) != 0 {
		return val - 1024
	}
	return val
}

func disassembleInstruction(word uint16, pc int) string {
	/* Decode based on bits 15:13 */
	op3 := (word >> 13) & 0x7

	switch op3 {
	case 0: /* 000 - LDW */
		rA := int(word & 0x7)
		rB := int((word >> 3) & 0x7)
		imm := unsignedImm7(int((word >> 6) & 0x7F))
		if word == 0x0000 {
			return "die ; 0x0000 special case"
		}
		if imm == 0 {
			return fmt.Sprintf("ldw r%d, r%d", rA, rB)
		}
		return fmt.Sprintf("ldw r%d, r%d, 0x%x", rA, rB, imm)

	case 1: /* 001 - LDB */
		rA := int(word & 0x7)
		rB := int((word >> 3) & 0x7)
		imm := unsignedImm7(int((word >> 6) & 0x7F))
		if imm == 0 {
			return fmt.Sprintf("ldb r%d, r%d", rA, rB)
		}
		return fmt.Sprintf("ldb r%d, r%d, 0x%x", rA, rB, imm)

	case 2: /* 010 - STW */
		rA := int(word & 0x7)
		rB := int((word >> 3) & 0x7)
		imm := unsignedImm7(int((word >> 6) & 0x7F))
		if imm == 0 {
			return fmt.Sprintf("stw r%d, r%d", rA, rB)
		}
		return fmt.Sprintf("stw r%d, r%d, 0x%x", rA, rB, imm)

	case 3: /* 011 - STB */
		rA := int(word & 0x7)
		rB := int((word >> 3) & 0x7)
		imm := unsignedImm7(int((word >> 6) & 0x7F))
		if imm == 0 {
			return fmt.Sprintf("stb r%d, r%d", rA, rB)
		}
		return fmt.Sprintf("stb r%d, r%d, 0x%x", rA, rB, imm)

	case 4: /* 100 - ADI */
		rA := int(word & 0x7)
		rB := int((word >> 3) & 0x7)
		imm := unsignedImm7(int((word >> 6) & 0x7F))
		if imm == 0 {
			return fmt.Sprintf("adi r%d, r%d", rA, rB)
		}
		return fmt.Sprintf("adi r%d, r%d, 0x%x", rA, rB, imm)

	case 5: /* 101 - LUI */
		rA := int(word & 0x7)
		imm := int((word >> 3) & 0x3FF)
		return fmt.Sprintf("lui r%d, 0x%x", rA, imm)

	case 6: /* 110 - BRx */
		cond := int(word & 0x7)
		imm := signExtend10(int((word >> 3) & 0x3FF))
		target := pc + 2 + imm
		condNames := []string{"br", "brl", "brz", "brnz", "brc", "brnc", "brsge", "brslt"}
		return fmt.Sprintf("%s 0x%x", condNames[cond], target)

	case 7: /* 111 - JAL or extended */
		/* Check bit 12 to distinguish JAL (0) from extended (1) */
		if (word & 0x1000) == 0 {
			/* JAL: bit 12 = 0 */
			rA := int(word & 0x7)
			rB := int((word >> 3) & 0x7)
			imm := int((word >> 6) & 0x3F)
			if imm == 0 {
				return fmt.Sprintf("jal r%d, r%d", rA, rB)
			}
			return fmt.Sprintf("jal r%d, r%d, 0x%x", rA, rB, imm)
		}
		/* Extended instructions: bit 12 = 1 */
		return disassembleExtended(word)
	}

	return fmt.Sprintf(".word 0x%04x ; unknown", word)
}

func disassembleExtended(word uint16) string {
	bits15_9 := (word >> 9) & 0x7F
	bits15_6 := (word >> 6) & 0x3FF
	bits15_3 := (word >> 3) & 0x1FFF

	/* Check for VOPs (bits 15:3 = 0x1FFF) */
	if bits15_3 == 0x1FFF {
		vop := int(word & 0x7)
		vopNames := []string{"ccf", "scf", "di", "ei", "hlt", "brk", "rti", "die"}
		return vopNames[vop]
	}

	/* Check for ZOPs (bits 15:6 = 0x3FF) */
	if bits15_6 == 0x3FF {
		zop := int((word >> 3) & 0x7)
		rA := int(word & 0x7)
		zopNames := []string{"not", "neg", "dub", "sxt", "sra", "srl", "ji", "???"}
		return fmt.Sprintf("%s r%d", zopNames[zop], rA)
	}

	/* Check for YOPs (bits 15:9 = 0x7F) */
	if bits15_9 == 0x7F {
		yop := int((word >> 6) & 0x7)
		rB := int((word >> 3) & 0x7)
		rA := int(word & 0x7)
		yopNames := []string{"lsp", "lsi", "ssp", "ssi", "lcw", "sys", "tst", "???"}
		return fmt.Sprintf("%s r%d, r%d", yopNames[yop], rA, rB)
	}

	/* XOPs (bits 15:12 = 0xF, but not YOP/ZOP/VOP) */
	xop := int((word >> 9) & 0x7)
	rC := int((word >> 6) & 0x7)
	rB := int((word >> 3) & 0x7)
	rA := int(word & 0x7)
	xopNames := []string{"sbb", "adc", "sub", "add", "xor", "or", "and", "???"}
	return fmt.Sprintf("%s r%d, r%d, r%d", xopNames[xop], rA, rB, rC)
}

func disassemble(filename string) error {
	/* Read file */
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	if len(data) < HEADER_SIZE {
		return fmt.Errorf("file too short to be valid WUT-4 binary")
	}

	/* Read header */
	magic := uint16(data[0]) | (uint16(data[1]) << 8)
	if magic != MAGIC_NUMBER {
		return fmt.Errorf("invalid magic number: 0x%04x (expected 0x%04x)", magic, MAGIC_NUMBER)
	}

	codeSize := int(uint16(data[2]) | (uint16(data[3]) << 8))
	dataSize := int(uint16(data[4]) | (uint16(data[5]) << 8))

	fmt.Printf("; WUT-4 Disassembly\n")
	fmt.Printf("; Magic: 0x%04x\n", magic)
	fmt.Printf("; Code size: %d bytes\n", codeSize)
	fmt.Printf("; Data size: %d bytes\n\n", dataSize)

	/* Disassemble code segment */
	if codeSize > 0 {
		fmt.Printf(".code\n")
		codeStart := HEADER_SIZE
		codeEnd := codeStart + codeSize
		if codeEnd > len(data) {
			return fmt.Errorf("code segment extends beyond file")
		}

		codeBuf := data[codeStart:codeEnd]
		pc := 0
		for pc < codeSize {
			if pc+1 >= codeSize {
				/* Odd byte at end */
				fmt.Printf("0x%04x: .bytes 0x%02x\n", pc, codeBuf[pc])
				pc++
			} else {
				word := readWord(codeBuf, pc)
				instr := disassembleInstruction(word, pc)
				fmt.Printf("0x%04x: %s\n", pc, instr)
				pc += 2
			}
		}
	}

	/* Disassemble data segment */
	if dataSize > 0 {
		fmt.Printf("\n.data\n")
		dataStart := HEADER_SIZE + codeSize
		dataEnd := dataStart + dataSize
		if dataEnd > len(data) {
			return fmt.Errorf("data segment extends beyond file")
		}

		dataBuf := data[dataStart:dataEnd]
		offset := 0
		for offset < dataSize {
			/* Print 16 bytes per line */
			lineEnd := offset + 16
			if lineEnd > dataSize {
				lineEnd = dataSize
			}

			fmt.Printf("0x%04x: .bytes ", offset)
			for i := offset; i < lineEnd; i++ {
				if i > offset {
					fmt.Printf(", ")
				}
				fmt.Printf("0x%02x", dataBuf[i])
			}
			fmt.Printf("\n")
			offset = lineEnd
		}
	}

	return nil
}
