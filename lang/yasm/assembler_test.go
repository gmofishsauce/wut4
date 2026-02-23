package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImmediateValueRanges(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "16-bit immediate: -1",
			code: ".code\n    ldi r1, -1\n    hlt\n",
			wantErr: false,
		},
		{
			name: "16-bit immediate: 65535 (max unsigned)",
			code: ".code\n    ldi r1, 65535\n    hlt\n",
			wantErr: false,
		},
		{
			name: "16-bit immediate: -32768 (min signed)",
			code: ".code\n    ldi r1, -32768\n    hlt\n",
			wantErr: false,
		},
		{
			name: "16-bit immediate: out of range high",
			code: ".code\n    ldi r1, 65536\n    hlt\n",
			wantErr: true,
		},
		{
			name: "16-bit immediate: out of range low",
			code: ".code\n    ldi r1, -32769\n    hlt\n",
			wantErr: true,
		},
		{
			name: "7-bit immediate: -64 (min signed)",
			code: ".code\n    adi r1, r2, -64\n    hlt\n",
			wantErr: false,
		},
		{
			name: "7-bit immediate: 127 (max unsigned)",
			code: ".code\n    adi r1, r2, 127\n    hlt\n",
			wantErr: false,
		},
		{
			name: "7-bit immediate: out of range high",
			code: ".code\n    adi r1, r2, 128\n    hlt\n",
			wantErr: true,
		},
		{
			name: "7-bit immediate: out of range low",
			code: ".code\n    adi r1, r2, -65\n    hlt\n",
			wantErr: true,
		},
		{
			name: "10-bit immediate: -512 (min signed)",
			code: ".code\n    lui r1, -512\n    hlt\n",
			wantErr: false,
		},
		{
			name: "10-bit immediate: 1023 (max unsigned)",
			code: ".code\n    lui r1, 1023\n    hlt\n",
			wantErr: false,
		},
		{
			name: "10-bit immediate: out of range high",
			code: ".code\n    lui r1, 1024\n    hlt\n",
			wantErr: true,
		},
		{
			name: "10-bit immediate: out of range low",
			code: ".code\n    lui r1, -513\n    hlt\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp input file
			tmpDir := t.TempDir()
			inputFile := filepath.Join(tmpDir, "test.asm")
			outputFile := filepath.Join(tmpDir, "test.bin")

			err := os.WriteFile(inputFile, []byte(tt.code), 0644)
			if err != nil {
				t.Fatalf("failed to write input file: %v", err)
			}

			// Run assembler
			err = assemble(inputFile, tt.code, outputFile)

			if tt.wantErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCommasOptional(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "tst without comma",
			code: ".code\n    tst r1 r2\n    hlt\n",
			wantErr: false,
		},
		{
			name: "tst with comma",
			code: ".code\n    tst r1, r2\n    hlt\n",
			wantErr: false,
		},
		{
			name: "add without commas",
			code: ".code\n    add r1 r2 r3\n    hlt\n",
			wantErr: false,
		},
		{
			name: "add with commas",
			code: ".code\n    add r1, r2, r3\n    hlt\n",
			wantErr: false,
		},
		{
			name: "ldi without comma",
			code: ".code\n    ldi r1 42\n    hlt\n",
			wantErr: false,
		},
		{
			name: "ldi with comma",
			code: ".code\n    ldi r1, 42\n    hlt\n",
			wantErr: false,
		},
		{
			name: "ldi with negative value without comma",
			code: ".code\n    ldi r1 -1\n    hlt\n",
			wantErr: false,
		},
		{
			name: "ldi with negative value with comma",
			code: ".code\n    ldi r1, -1\n    hlt\n",
			wantErr: false,
		},
		{
			name: ".set without comma",
			code: ".code\n.set CONST 42\n    ldi r1 CONST\n    hlt\n",
			wantErr: false,
		},
		{
			name: ".set with comma",
			code: ".code\n.set CONST, 42\n    ldi r1, CONST\n    hlt\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp input file
			tmpDir := t.TempDir()
			inputFile := filepath.Join(tmpDir, "test.asm")
			outputFile := filepath.Join(tmpDir, "test.bin")

			err := os.WriteFile(inputFile, []byte(tt.code), 0644)
			if err != nil {
				t.Fatalf("failed to write input file: %v", err)
			}

			// Run assembler
			err = assemble(inputFile, tt.code, outputFile)

			if tt.wantErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestExpressions(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "expression with spaces",
			code: ".code\n    ldi r1 2 + 3\n    hlt\n",
			wantErr: false,
		},
		{
			name: "expression without spaces",
			code: ".code\n    ldi r1 2+3\n    hlt\n",
			wantErr: false,
		},
		{
			name: "negative number vs subtraction",
			code: ".code\n.set A 5\n    ldi r1 A\n    ldi r2 -1\n    hlt\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp input file
			tmpDir := t.TempDir()
			inputFile := filepath.Join(tmpDir, "test.asm")
			outputFile := filepath.Join(tmpDir, "test.bin")

			err := os.WriteFile(inputFile, []byte(tt.code), 0644)
			if err != nil {
				t.Fatalf("failed to write input file: %v", err)
			}

			// Run assembler
			err = assemble(inputFile, tt.code, outputFile)

			if tt.wantErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestAllInstructions tests every regular (non-pseudo) instruction by assembling,
// disassembling, and verifying the encoding round-trips correctly.
func TestAllInstructions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // expected disassembly lines (without addresses)
	}{
		// Base instructions (FMT_BASE)
		{
			name:  "ldw with all registers",
			input: "ldw r1, r2, 10\nldw r7, r0, -5\nldw r3, r4, 0",
			expected: []string{
				"ldw r1, r2, 0xa",
				"ldw r7, r0, 0x7b",
				"ldw r3, r4",
			},
		},
		{
			name:  "ldb with various immediates",
			input: "ldb r1, r2, 10\nldb r7, r0, -64\nldb r3, r4, 127",
			expected: []string{
				"ldb r1, r2, 0xa",
				"ldb r7, r0, 0x40",
				"ldb r3, r4, 0x7f",
			},
		},
		{
			name:  "stw instructions",
			input: "stw r1, r2, 5\nstw r0, r7, 0\nstw r5, r5, -32",
			expected: []string{
				"stw r1, r2, 0x5",
				"stw r0, r7",
				"stw r5, r5, 0x60",
			},
		},
		{
			name:  "stb instructions",
			input: "stb r2, r3, 15\nstb r1, r1, 0\nstb r6, r7, -10",
			expected: []string{
				"stb r2, r3, 0xf",
				"stb r1, r1",
				"stb r6, r7, 0x76",
			},
		},
		{
			name:  "adi instructions",
			input: "adi r1, r2, 10\nadi r7, r0, -64\nadi r3, r4, 0",
			expected: []string{
				"adi r1, r2, 0xa",
				"adi r7, r0, 0x40",
				"adi r3, r4",
			},
		},
		// LUI (FMT_LUI)
		{
			name:  "lui instructions",
			input: "lui r1, 0x100\nlui r7, 0x3FF\nlui r3, 0",
			expected: []string{
				"lui r1, 0x100",
				"lui r7, 0x3ff",
				"lui r3, 0x0",
			},
		},
		// Branch instructions (FMT_BRX)
		{
			name:  "br unconditional",
			input: "start:\n  br start\n  br end\nend:\n  hlt",
			expected: []string{
				"br 0x0",
				"br 0x4",
				"hlt",
			},
		},
		{
			name:  "conditional branches",
			input: "loop:\n  brz loop\n  brnz loop\n  brc loop\n  brnc loop\n  hlt",
			expected: []string{
				"brz 0x0",
				"brnz 0x0",
				"brc 0x0",
				"brnc 0x0",
				"hlt",
			},
		},
		{
			name:  "signed branches",
			input: "test:\n  brsge test\n  brslt test\n  hlt",
			expected: []string{
				"brsge 0x0",
				"brslt 0x0",
				"hlt",
			},
		},
		// JAL (FMT_JAL) - always uses 2 words (lui + jal) for consistent sizing
		{
			name:  "jal with small immediate",
			input: "jal r1, r2, 0\njal r3, r4, 32\njal r7, r0, 63",
			expected: []string{
				// jal always uses 2 words for consistent sizing between passes
				"lui r1, 0x0",
				"jal r1, r2",
				"lui r3, 0x0",
				"jal r3, r4, 0x20",
				"lui r7, 0x0",
				"jal r7, r0, 0x3f",
			},
		},
		// XOPs (FMT_XOP)
		{
			name:  "arithmetic XOPs",
			input: "add r1, r2, r3\nsub r4, r5, r6\nadc r7, r0, r1\nsbb r2, r3, r4",
			expected: []string{
				"add r1, r2, r3",
				"sub r4, r5, r6",
				"adc r7, r0, r1",
				"sbb r2, r3, r4",
			},
		},
		{
			name:  "logical XOPs",
			input: "and r1, r2, r3\nor r4, r5, r6\nxor r7, r0, r1",
			expected: []string{
				"and r1, r2, r3",
				"or r4, r5, r6",
				"xor r7, r0, r1",
			},
		},
		// YOPs (FMT_YOP)
		{
			name:  "YOP instructions",
			input: "tst r1, r2\nlsp r3, r4\nlsi r5, r6\nssp r7, r0\nssi r1, r2\nlcw r3, r4\nsys r5, r6",
			expected: []string{
				"tst r1, r2",
				"lsp r3, r4",
				"lsi r5, r6",
				"ssp r7, r0",
				"ssi r1, r2",
				"lcw r3, r4",
				"sys r5, r6",
			},
		},
		// ZOPs (FMT_ZOP)
		{
			name:  "ZOP instructions",
			input: "not r1\nneg r2\ndub r3\nsxt r4\nsra r5\nsrl r6\nji r7",
			expected: []string{
				"not r1",
				"neg r2",
				"dub r3",
				"sxt r4",
				"sra r5",
				"srl r6",
				"ji r7",
			},
		},
		// VOPs (FMT_VOP)
		{
			name:  "VOP instructions",
			input: "hlt\nbrk\nrti\ndie\nccf\nscf\ndi\nei",
			expected: []string{
				"hlt",
				"brk",
				"rti",
				"die",
				"ccf",
				"scf",
				"di",
				"ei",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputFile := filepath.Join(tmpDir, "test.asm")
			outputFile := filepath.Join(tmpDir, "test.bin")

			// Create full source file
			source := ".code\n" + tt.input + "\n"
			err := os.WriteFile(inputFile, []byte(source), 0644)
			if err != nil {
				t.Fatalf("failed to write input file: %v", err)
			}

			// Assemble
			err = assemble(inputFile, source, outputFile)
			if err != nil {
				t.Fatalf("assembly failed: %v", err)
			}

			// Read the binary
			data, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}

			// Verify it has proper header
			if len(data) < HEADER_SIZE {
				t.Fatalf("output file too short")
			}

			// Extract code segment
			codeSize := int(uint16(data[2]) | (uint16(data[3]) << 8))
			codeBuf := data[HEADER_SIZE : HEADER_SIZE+codeSize]

			// Disassemble each instruction
			var disasm []string
			pc := 0
			for pc < codeSize {
				word := readWord(codeBuf, pc)
				instr := disassembleInstruction(word, pc)
				disasm = append(disasm, instr)
				pc += 2
			}

			// Compare
			if len(disasm) != len(tt.expected) {
				t.Errorf("expected %d instructions, got %d\nExpected: %v\nGot: %v",
					len(tt.expected), len(disasm), tt.expected, disasm)
				return
			}

			for i := range disasm {
				if disasm[i] != tt.expected[i] {
					t.Errorf("instruction %d mismatch:\n  expected: %s\n  got:      %s",
						i, tt.expected[i], disasm[i])
				}
			}
		})
	}
}

// TestObjectMode tests that -c (object mode) produces a WOF file with
// correct relocation entries for external symbol references.
func TestObjectMode(t *testing.T) {
	// caller.asm: calls Foo (defined in another file) via jal
	callerSrc := ".code\nMain:\n    jal Foo\n    hlt\n"
	// callee.asm: defines Foo
	calleeSrc := ".code\nFoo:\n    ldi r1, 99\n    ret\n"

	tests := []struct {
		name       string
		src        string
		wantRelocs int
		wantSyms   int // global symbols in WOF
	}{
		{"callee (no external refs)", calleeSrc, 0, 1},
		{"caller (one external ref)", callerSrc, 1, 2}, // Main (defined) + Foo (undef)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputFile := tmpDir + "/test.asm"
			outputFile := tmpDir + "/test.wo"

			if err := os.WriteFile(inputFile, []byte(tt.src), 0644); err != nil {
				t.Fatalf("write input: %v", err)
			}

			if err := assembleMode(inputFile, tt.src, outputFile, true); err != nil {
				t.Fatalf("assembleMode: %v", err)
			}

			data, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("read output: %v", err)
			}

			// Validate WOF magic
			magic := uint16(data[0]) | uint16(data[1])<<8
			if magic != MAGIC_WOF {
				t.Errorf("expected magic 0x%04X, got 0x%04X", MAGIC_WOF, magic)
			}

			// Parse header fields
			symCount := int(uint16(data[8]) | uint16(data[9])<<8)
			relocCount := int(uint16(data[10]) | uint16(data[11])<<8)

			if symCount != tt.wantSyms {
				t.Errorf("expected %d symbols, got %d", tt.wantSyms, symCount)
			}
			if relocCount != tt.wantRelocs {
				t.Errorf("expected %d relocations, got %d", tt.wantRelocs, relocCount)
			}
		})
	}
}

// TestPseudoInstructions tests pseudo-instructions and verifies their
// optimizations produce the correct machine code.
func TestPseudoInstructions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // expected disassembly
	}{
		// LDI pseudo-instruction optimizations
		{
			name:  "ldi small immediate (uses adi)",
			input: "ldi r1, 0\nldi r2, 10\nldi r3, 63",
			expected: []string{
				// ldi always uses 2 words (lui + adi) for consistent sizing
				"lui r1, 0x0",     // ldi r1, 0 -> lui r1, 0
				"adi r1, r1",      //            + adi r1, r1, 0
				"lui r2, 0x0",     // ldi r2, 10 -> lui r2, 0
				"adi r2, r2, 0xa", //             + adi r2, r2, 10
				"lui r3, 0x0",     // ldi r3, 63 -> lui r3, 0
				"adi r3, r3, 0x3f", //            + adi r3, r3, 63
			},
		},
		{
			name:  "ldi aligned to 64 (uses lui + adi)",
			input: "ldi r1, 64\nldi r2, 128\nldi r3, 0x3FC0",
			expected: []string{
				// ldi always uses 2 words (lui + adi) for consistent sizing
				"lui r1, 0x1",  // ldi r1, 64 -> lui r1, 1
				"adi r1, r1",   //             + adi r1, r1, 0
				"lui r2, 0x2",  // ldi r2, 128 -> lui r2, 2
				"adi r2, r2",   //              + adi r2, r2, 0
				"lui r3, 0xff", // ldi r3, 0x3FC0 -> lui r3, 0xFF
				"adi r3, r3",   //                 + adi r3, r3, 0
			},
		},
		{
			name:  "ldi non-aligned (uses lui + adi)",
			input: "ldi r1, 65\nldi r2, 200\nldi r3, 0x1234",
			expected: []string{
				"lui r1, 0x1",      // ldi r1, 65 -> lui r1, 1 (64)
				"adi r1, r1, 0x1",  //            + adi r1, r1, 1
				"lui r2, 0x3",      // ldi r2, 200 -> lui r2, 3 (192)
				"adi r2, r2, 0x8",  //             + adi r2, r2, 8
				"lui r3, 0x48",     // ldi r3, 0x1234 -> lui r3, 0x48
				"adi r3, r3, 0x34", //                + adi r3, r3, 52
			},
		},
		{
			name:  "ldi negative values",
			input: "ldi r1, -1\nldi r2, -100",
			expected: []string{
				"lui r1, 0x3ff",    // ldi r1, -1 -> lui r1, 0x3FF
				"adi r1, r1, 0x3f", //            + adi r1, r1, 63
				"lui r2, 0x3fe",    // ldi r2, -100 -> lui r2, 0x3FE
				"adi r2, r2, 0x1c", //              + adi r2, r2, 28
			},
		},
		// MV pseudo-instruction
		{
			name:  "mv (uses adi)",
			input: "mv r1, r2\nmv r7, r0",
			expected: []string{
				"adi r1, r2",  // mv r1, r2 -> adi r1, r2, 0
				"adi r7, r0",  // mv r7, r0 -> adi r7, r0, 0
			},
		},
		// RET pseudo-instruction
		{
			name:  "ret (uses ji)",
			input: "ret\nret r3",
			expected: []string{
				"ji r0",  // ret -> ji r0 (ji link)
				"ji r3",  // ret r3 -> ji r3
			},
		},
		// SLA pseudo-instruction
		{
			name:  "sla (uses adc)",
			input: "sla r1\nsla r7",
			expected: []string{
				"adc r1, r1, r1",  // sla r1 -> adc r1, r1, r1
				"adc r7, r7, r7",  // sla r7 -> adc r7, r7, r7
			},
		},
		// SLL pseudo-instruction
		{
			name:  "sll (uses add)",
			input: "sll r2\nsll r6",
			expected: []string{
				"add r2, r2, r2",  // sll r2 -> add r2, r2, r2
				"add r6, r6, r6",  // sll r6 -> add r6, r6, r6
			},
		},
		// SRR pseudo-instruction
		{
			name:  "srr small immediate",
			input: "srr r1, r2, 5",
			expected: []string{
				// ldi always uses 2 words (lui + adi) for consistent sizing
				"lui r2, 0x0",     // srr r1, r2, 5 -> ldi r2, 5 (lui r2, 0)
				"adi r2, r2, 0x5", //                           + adi r2, r2, 5
				"lsp r1, r2",      //               -> lsp r1, r2
			},
		},
		{
			name:  "srr large immediate",
			input: "srr r3, r4, 0x100",
			expected: []string{
				// ldi always uses 2 words (lui + adi) for consistent sizing
				"lui r4, 0x4", // srr r3, r4, 0x100 -> ldi r4, 0x100 (lui r4, 4)
				"adi r4, r4",  //                                   + adi r4, r4, 0
				"lsp r3, r4",  //                   -> lsp r3, r4
			},
		},
		// SRW pseudo-instruction
		{
			name:  "srw small immediate",
			input: "srw r5, r6, 10",
			expected: []string{
				// ldi always uses 2 words (lui + adi) for consistent sizing
				"lui r6, 0x0",     // srw r5, r6, 10 -> ldi r6, 10 (lui r6, 0)
				"adi r6, r6, 0xa", //                            + adi r6, r6, 10
				"ssp r5, r6",      //                -> ssp r5, r6
			},
		},
		// JAL pseudo-instruction (1 and 2 operand forms)
		{
			name:  "jal with 1 operand",
			input: "jal 0x200",
			expected: []string{
				"lui r0, 0x8",  // jal 0x200 -> lui r0, 8 (upper)
				"jal r0, r0",   //           -> jal r0, r0, 0 (lower) - disassembler omits 0
			},
		},
		{
			name:  "jal with 2 operands",
			input: "jal r3, 0x180",
			expected: []string{
				"lui r3, 0x6",    // jal r3, 0x180 -> lui r3, 6 (upper)
				"jal r3, r3",     //               -> jal r3, r3, 0 (lower = 0)
			},
		},
		{
			name:  "jal with 3 operands (large address)",
			input: "jal r2, r5, 0x200",
			expected: []string{
				"lui r2, 0x8",    // jal r2, r5, 0x200 -> lui r2, 8
				"jal r2, r5",     //                   -> jal r2, r5, 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			inputFile := filepath.Join(tmpDir, "test.asm")
			outputFile := filepath.Join(tmpDir, "test.bin")

			// Create full source file
			source := ".code\n" + tt.input + "\n"
			err := os.WriteFile(inputFile, []byte(source), 0644)
			if err != nil {
				t.Fatalf("failed to write input file: %v", err)
			}

			// Assemble
			err = assemble(inputFile, source, outputFile)
			if err != nil {
				t.Fatalf("assembly failed: %v", err)
			}

			// Read the binary
			data, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}

			// Verify it has proper header
			if len(data) < HEADER_SIZE {
				t.Fatalf("output file too short")
			}

			// Extract code segment
			codeSize := int(uint16(data[2]) | (uint16(data[3]) << 8))
			codeBuf := data[HEADER_SIZE : HEADER_SIZE+codeSize]

			// Disassemble each instruction
			var disasm []string
			pc := 0
			for pc < codeSize {
				word := readWord(codeBuf, pc)
				instr := disassembleInstruction(word, pc)
				disasm = append(disasm, instr)
				pc += 2
			}

			// Compare
			if len(disasm) != len(tt.expected) {
				t.Errorf("expected %d instructions, got %d\nExpected: %v\nGot: %v",
					len(tt.expected), len(disasm), tt.expected, disasm)
				return
			}

			for i := range disasm {
				if disasm[i] != tt.expected[i] {
					t.Errorf("instruction %d mismatch:\n  expected: %s\n  got:      %s",
						i, tt.expected[i], disasm[i])
				}
			}
		})
	}
}
