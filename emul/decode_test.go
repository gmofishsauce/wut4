// Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com)
//
// Unit tests for instruction decoder

package main

import (
	"testing"
)

// TestDecodeBase tests decoding of base instruction formats
func TestDecodeBase(t *testing.T) {
	tests := []struct {
		name   string
		inst   uint16
		opcode uint8
		rA     uint8
		rB     uint8
		imm7   int16
	}{
		{
			name:   "LDW r1, r2, 5",
			inst:   0b000_0000101_010_001, // opcode=0 (LDW), imm7=5, rB=2, rA=1
			opcode: 0,
			rA:     1,
			rB:     2,
			imm7:   5,
		},
		{
			name:   "STW r3, r4, -10",
			inst:   0b010_1110110_100_011, // opcode=2 (STW), imm7=-10 (0x76), rB=4, rA=3
			opcode: 2,
			rA:     3,
			rB:     4,
			imm7:   -10,
		},
		{
			name:   "ADI r5, r6, 63",
			inst:   0b100_0111111_110_101, // opcode=4 (ADI), imm7=63, rB=6, rA=5
			opcode: 4,
			rA:     5,
			rB:     6,
			imm7:   63,
		},
		{
			name:   "ADI r1, r2, -1",
			inst:   0b100_1111111_010_001, // opcode=4 (ADI), imm7=-1 (0x7F), rB=2, rA=1
			opcode: 4,
			rA:     1,
			rB:     2,
			imm7:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded := decode(tt.inst)

			if !decoded.isBase {
				t.Errorf("Expected base instruction, got isBase=false")
			}

			if decoded.opcode != tt.opcode {
				t.Errorf("opcode = %d, want %d", decoded.opcode, tt.opcode)
			}

			if decoded.rA != tt.rA {
				t.Errorf("rA = %d, want %d", decoded.rA, tt.rA)
			}

			if decoded.rB != tt.rB {
				t.Errorf("rB = %d, want %d", decoded.rB, tt.rB)
			}

			if decoded.imm7 != tt.imm7 {
				t.Errorf("imm7 = %d (0x%04X), want %d (0x%04X)",
					decoded.imm7, uint16(decoded.imm7), tt.imm7, uint16(tt.imm7))
			}
		})
	}
}

// TestDecodeXOP tests 3-operand extended instructions
func TestDecodeXOP(t *testing.T) {
	tests := []struct {
		name string
		inst uint16
		xop  uint8
		rA   uint8
		rB   uint8
		rC   uint8
	}{
		{
			name: "ADD r3, r1, r2",
			inst: 0b1111_011_001_010_011, // XOP marker (1111), xop=3 (ADD), rC=1, rB=2, rA=3
			xop:  3,
			rA:   3,
			rB:   2,
			rC:   1,
		},
		{
			name: "SUB r5, r6, r7",
			inst: 0b1111_010_110_111_101, // xop=2 (SUB), rC=6, rB=7, rA=5
			xop:  2,
			rA:   5,
			rB:   7,
			rC:   6,
		},
		{
			name: "AND r0, r1, r2",
			inst: 0b1111_110_001_010_000, // xop=6 (AND), rC=1, rB=2, rA=0
			xop:  6,
			rA:   0,
			rB:   2,
			rC:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded := decode(tt.inst)

			if !decoded.isXOP {
				t.Errorf("Expected XOP instruction, got isXOP=false")
			}

			if decoded.xop != tt.xop {
				t.Errorf("xop = %d, want %d", decoded.xop, tt.xop)
			}

			if decoded.rA != tt.rA {
				t.Errorf("rA = %d, want %d", decoded.rA, tt.rA)
			}

			if decoded.rB != tt.rB {
				t.Errorf("rB = %d, want %d", decoded.rB, tt.rB)
			}

			if decoded.rC != tt.rC {
				t.Errorf("rC = %d, want %d", decoded.rC, tt.rC)
			}
		})
	}
}

// TestDecodeYOP tests 2-operand extended instructions
func TestDecodeYOP(t *testing.T) {
	tests := []struct {
		name string
		inst uint16
		yop  uint8
		rA   uint8
		rB   uint8
	}{
		{
			name: "LSP r1, r2",
			inst: 0b1111111_000_010_001, // YOP marker (1111111), yop=0 (LSP), rB=2, rA=1
			yop:  0,
			rA:   1,
			rB:   2,
		},
		{
			name: "TST r3, r4",
			inst: 0b1111111_110_100_011, // yop=6 (TST), rB=4, rA=3
			yop:  6,
			rA:   3,
			rB:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded := decode(tt.inst)

			if !decoded.isYOP {
				t.Errorf("Expected YOP instruction, got isYOP=false")
			}

			if decoded.yop != tt.yop {
				t.Errorf("yop = %d, want %d", decoded.yop, tt.yop)
			}

			if decoded.rA != tt.rA {
				t.Errorf("rA = %d, want %d", decoded.rA, tt.rA)
			}

			if decoded.rB != tt.rB {
				t.Errorf("rB = %d, want %d", decoded.rB, tt.rB)
			}
		})
	}
}

// TestDecodeZOP tests 1-operand extended instructions
func TestDecodeZOP(t *testing.T) {
	tests := []struct {
		name string
		inst uint16
		zop  uint8
		rA   uint8
	}{
		{
			name: "NOT r1",
			inst: 0b1111111111_000_001, // ZOP marker (1111111111), zop=0 (NOT), rA=1
			zop:  0,
			rA:   1,
		},
		{
			name: "NEG r5",
			inst: 0b1111111111_001_101, // zop=1 (NEG), rA=5
			zop:  1,
			rA:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded := decode(tt.inst)

			if !decoded.isZOP {
				t.Errorf("Expected ZOP instruction, got isZOP=false")
			}

			if decoded.zop != tt.zop {
				t.Errorf("zop = %d, want %d", decoded.zop, tt.zop)
			}

			if decoded.rA != tt.rA {
				t.Errorf("rA = %d, want %d", decoded.rA, tt.rA)
			}
		})
	}
}

// TestDecodeVOP tests 0-operand extended instructions
func TestDecodeVOP(t *testing.T) {
	tests := []struct {
		name string
		inst uint16
		vop  uint8
	}{
		{
			name: "HLT",
			inst: 0b1111111111111_100, // VOP marker (1111111111111), vop=4 (HLT)
			vop:  4,
		},
		{
			name: "EI",
			inst: 0b1111111111111_011, // vop=3 (EI)
			vop:  3,
		},
		{
			name: "DI",
			inst: 0b1111111111111_010, // vop=2 (DI)
			vop:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded := decode(tt.inst)

			if !decoded.isVOP {
				t.Errorf("Expected VOP instruction, got isVOP=false")
			}

			if decoded.vop != tt.vop {
				t.Errorf("vop = %d, want %d", decoded.vop, tt.vop)
			}
		})
	}
}

// TestDecodeBranch tests branch instruction decoding
func TestDecodeBranch(t *testing.T) {
	tests := []struct {
		name       string
		inst       uint16
		branchCond uint8
		imm10      uint16
	}{
		{
			name:       "BR +10",
			inst:       0b110_0000001010_000, // opcode=6 (BRx), imm10=10, cond=0 (BR)
			branchCond: 0,
			imm10:      10,
		},
		{
			name:       "BEQ +5",
			inst:       0b110_0000000101_010, // imm10=5, cond=2 (BEQ)
			branchCond: 2,
			imm10:      5,
		},
		{
			name:       "BNE -8",
			inst:       0b110_1111111000_011, // imm10=-8 (0x3F8), cond=3 (BNE)
			branchCond: 3,
			imm10:      0xFFF8, // Sign-extended to 16 bits
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded := decode(tt.inst)

			if !decoded.isBase {
				t.Errorf("Expected base instruction, got isBase=false")
			}

			if decoded.opcode != 6 {
				t.Errorf("opcode = %d, want 6 (BRx)", decoded.opcode)
			}

			if decoded.branchCond != tt.branchCond {
				t.Errorf("branchCond = %d, want %d", decoded.branchCond, tt.branchCond)
			}

			if decoded.imm10 != tt.imm10 {
				t.Errorf("imm10 = 0x%04X, want 0x%04X", decoded.imm10, tt.imm10)
			}
		})
	}
}

// TestDecodeLUI tests LUI instruction with 10-bit immediate
func TestDecodeLUI(t *testing.T) {
	tests := []struct {
		name  string
		inst  uint16
		rA    uint8
		imm10 uint16
	}{
		{
			name:  "LUI r1, 0x3FF",
			inst:  0b101_1111111111_001, // opcode=5 (LUI), imm10=0x3FF, rA=1
			rA:    1,
			imm10: 0x3FF,
		},
		{
			name:  "LUI r7, 0x100",
			inst:  0b101_0100000000_111, // imm10=0x100, rA=7
			rA:    7,
			imm10: 0x100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded := decode(tt.inst)

			if !decoded.isBase {
				t.Errorf("Expected base instruction")
			}

			if decoded.opcode != 5 {
				t.Errorf("opcode = %d, want 5 (LUI)", decoded.opcode)
			}

			if decoded.rA != tt.rA {
				t.Errorf("rA = %d, want %d", decoded.rA, tt.rA)
			}

			if decoded.imm10 != tt.imm10 {
				t.Errorf("imm10 = 0x%04X, want 0x%04X", decoded.imm10, tt.imm10)
			}
		})
	}
}
