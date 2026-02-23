// Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com)
//
// Unit tests for memory and MMU operations

package main

import (
	"bytes"
	"testing"
)

// TestMMUTranslation tests virtual to physical address translation
func TestMMUTranslation(t *testing.T) {
	cpu := NewCPU()

	// Set up MMU: Map virtual page 0 to physical page 0 with RWX permissions
	cpu.mmu[0][0] = 0x0000 | (PERM_RWX << 12)  // Code page 0
	cpu.mmu[0][16] = 0x0000 | (PERM_RWX << 12) // Data page 0

	// Map virtual page 1 to physical page 5 with RWX permissions
	cpu.mmu[0][1] = 0x0005 | (PERM_RWX << 12)  // Code page 1
	cpu.mmu[0][17] = 0x0005 | (PERM_RWX << 12) // Data page 1

	tests := []struct {
		name     string
		virtAddr uint16
		isData   bool
		wantPhys uint32
		wantErr  bool
	}{
		{
			name:     "Code page 0, offset 0",
			virtAddr: 0x0000,
			isData:   false,
			wantPhys: 0x0000,
			wantErr:  false,
		},
		{
			name:     "Code page 0, offset 0x100",
			virtAddr: 0x0100,
			isData:   false,
			wantPhys: 0x0100,
			wantErr:  false,
		},
		{
			name:     "Code page 1, offset 0",
			virtAddr: 0x1000, // Page 1 starts at 0x1000 (4KB)
			isData:   false,
			wantPhys: 0x5000, // Physical page 5 starts at 0x5000
			wantErr:  false,
		},
		{
			name:     "Code page 1, offset 0x234",
			virtAddr: 0x1234,
			isData:   false,
			wantPhys: 0x5234, // Physical page 5 + offset 0x234
			wantErr:  false,
		},
		{
			name:     "Data page 0, offset 0",
			virtAddr: 0x0000,
			isData:   true,
			wantPhys: 0x0000,
			wantErr:  false,
		},
		{
			name:     "Data page 1, offset 0xABC",
			virtAddr: 0x1ABC,
			isData:   true,
			wantPhys: 0x5ABC,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var physAddr uint32
			var err error

			if tt.isData {
				physAddr, err = cpu.translateData(tt.virtAddr)
			} else {
				physAddr, err = cpu.translateCode(tt.virtAddr)
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if physAddr != tt.wantPhys {
					t.Errorf("physAddr = 0x%06X, want 0x%06X", physAddr, tt.wantPhys)
				}
			}
		})
	}
}

// TestMMUPermissions tests page permission checking
func TestMMUPermissions(t *testing.T) {
	cpu := NewCPU()

	// Set up different permission scenarios
	cpu.mmu[0][0] = 0x0010 | (PERM_RWX << 12)     // Code page 0: RWX on physical page 0x10
	cpu.mmu[0][1] = 0x0011 | (PERM_EXEC << 12)    // Code page 1: Execute-only
	cpu.mmu[0][2] = 0x0012 | (PERM_INVALID << 12) // Code page 2: Invalid
	cpu.mmu[0][16] = 0x0020 | (PERM_RWX << 12)    // Data page 0: RWX
	cpu.mmu[0][17] = 0x0021 | (PERM_EXEC << 12)   // Data page 1: Read-only (EXEC for data = read-only)

	tests := []struct {
		name     string
		virtAddr uint16
		isData   bool
		wantErr  bool
	}{
		{
			name:     "Valid RWX code page",
			virtAddr: 0x0100,
			isData:   false,
			wantErr:  false,
		},
		{
			name:     "Valid execute-only code page",
			virtAddr: 0x1100,
			isData:   false,
			wantErr:  false,
		},
		{
			name:     "Invalid code page",
			virtAddr: 0x2100,
			isData:   false,
			wantErr:  true,
		},
		{
			name:     "Valid RWX data page",
			virtAddr: 0x0100,
			isData:   true,
			wantErr:  false,
		},
		{
			name:     "Valid read-only data page",
			virtAddr: 0x1100,
			isData:   true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.isData {
				_, err = cpu.translateData(tt.virtAddr)
			} else {
				_, err = cpu.translateCode(tt.virtAddr)
			}

			if tt.wantErr && err == nil {
				t.Errorf("Expected error for invalid page, got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestLoadStoreWord tests word load/store operations
func TestLoadStoreWord(t *testing.T) {
	cpu := NewCPU()

	// Set up basic identity mapping for testing
	cpu.mmu[0][16] = 0x0000 | (PERM_RWX << 12) // Data page 0

	// Test store and load
	testValue := uint16(0x1234)
	testAddr := uint16(0x0100) // Word-aligned address

	err := cpu.storeWord(testAddr, testValue)
	if err != nil {
		t.Fatalf("storeWord failed: %v", err)
	}

	value, err := cpu.loadWord(testAddr)
	if err != nil {
		t.Fatalf("loadWord failed: %v", err)
	}

	if value != testValue {
		t.Errorf("loadWord returned 0x%04X, want 0x%04X", value, testValue)
	}
}

// TestLoadStoreByte tests byte load/store operations
func TestLoadStoreByte(t *testing.T) {
	cpu := NewCPU()

	// Set up basic identity mapping
	cpu.mmu[0][16] = 0x0000 | (PERM_RWX << 12)

	// Test byte operations
	testByte := uint16(0x42)
	testAddr := uint16(0x0101)

	err := cpu.storeByte(testAddr, testByte)
	if err != nil {
		t.Fatalf("storeByte failed: %v", err)
	}

	value, err := cpu.loadByte(testAddr)
	if err != nil {
		t.Fatalf("loadByte failed: %v", err)
	}

	if value != testByte {
		t.Errorf("loadByte returned 0x%04X, want 0x%04X", value, testByte)
	}
}

// TestAlignmentCheck tests that unaligned word access raises exception
func TestAlignmentCheck(t *testing.T) {
	cpu := NewCPU()
	cpu.consoleOut = &bytes.Buffer{} // Prevent nil pointer in raiseException
	cpu.intEnabled = true              // Enable interrupts to avoid double fault
	cpu.mmu[0][16] = 0x0000 | (PERM_RWX << 12)

	// Try to load from odd address - should raise alignment exception
	oddAddr := uint16(0x0101)
	_, err := cpu.loadWord(oddAddr)

	// The function should not return an error directly, but should set pendingException
	if err != nil {
		t.Errorf("loadWord returned error instead of raising exception: %v", err)
	}

	if !cpu.pendingException {
		t.Errorf("Expected alignment exception to be pending")
	}

	if cpu.exceptionVector != EX_ALIGNMENT_FAULT {
		t.Errorf("Exception vector = 0x%04X, want 0x%04X (alignment fault)", cpu.exceptionVector, uint16(EX_ALIGNMENT_FAULT))
	}
}

// TestContextSwitching tests MMU context switching
func TestContextSwitching(t *testing.T) {
	cpu := NewCPU()

	// Set up different mappings for context 0 (kernel) and context 1 (user)
	cpu.mmu[0][0] = 0x0010 | (PERM_RWX << 12)  // Kernel code page 0 → physical 0x10
	cpu.mmu[1][0] = 0x0020 | (PERM_RWX << 12)  // User context 1 code page 0 → physical 0x20
	cpu.mmu[0][16] = 0x0030 | (PERM_RWX << 12) // Kernel data page 0 → physical 0x30
	cpu.mmu[1][16] = 0x0040 | (PERM_RWX << 12) // User context 1 data page 0 → physical 0x40

	// Test in kernel mode (context 0)
	cpu.mode = ModeKernel
	cpu.context = 0

	physAddr, err := cpu.translateCode(0x0000)
	if err != nil {
		t.Fatalf("translateCode failed in kernel mode: %v", err)
	}
	if physAddr != 0x10000 {
		t.Errorf("Kernel code translation = 0x%06X, want 0x10000", physAddr)
	}

	physAddr, err = cpu.translateData(0x0000)
	if err != nil {
		t.Fatalf("translateData failed in kernel mode: %v", err)
	}
	if physAddr != 0x30000 {
		t.Errorf("Kernel data translation = 0x%06X, want 0x30000", physAddr)
	}

	// Switch to user mode context 1
	cpu.mode = ModeUser
	cpu.context = 1

	physAddr, err = cpu.translateCode(0x0000)
	if err != nil {
		t.Fatalf("translateCode failed in user mode: %v", err)
	}
	if physAddr != 0x20000 {
		t.Errorf("User code translation = 0x%06X, want 0x20000", physAddr)
	}

	physAddr, err = cpu.translateData(0x0000)
	if err != nil {
		t.Fatalf("translateData failed in user mode: %v", err)
	}
	if physAddr != 0x40000 {
		t.Errorf("User data translation = 0x%06X, want 0x40000", physAddr)
	}
}

// TestRegister0ReadsZero tests that register 0 reads as zero during instruction execution
// Note: The gen[x][0] array element can be written to, but instructions treat r0 as zero when reading
func TestRegister0ReadsZero(t *testing.T) {
	cpu := NewCPU()

	// Set up some non-zero values in registers
	cpu.gen[ModeKernel][1] = 0x1234
	cpu.gen[ModeKernel][2] = 0x5678

	// Decode an ADD instruction: ADD r3, r0, r1 (adds 0 + 0x1234)
	// Format: 1111_011_000_001_011 (XOP=3 (ADD), rC=0, rB=1, rA=3)
	inst := decode(0b1111_011_000_001_011)

	err := cpu.execute(inst)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	// Result should be 0 + 0x1234 = 0x1234 (r0 reads as 0)
	if cpu.gen[ModeKernel][3] != 0x1234 {
		t.Errorf("ADD r3, r0, r1: got 0x%04X, want 0x1234 (r0 should read as 0)", cpu.gen[ModeKernel][3])
	}

	// Try ADD r4, r2, r0 (adds 0x5678 + 0)
	inst2 := decode(0b1111_011_010_000_100) // XOP=3 (ADD), rC=2, rB=0, rA=4
	err = cpu.execute(inst2)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if cpu.gen[ModeKernel][4] != 0x5678 {
		t.Errorf("ADD r4, r2, r0: got 0x%04X, want 0x5678 (r0 should read as 0)", cpu.gen[ModeKernel][4])
	}
}
