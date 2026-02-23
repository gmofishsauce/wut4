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

// Page permissions
const (
	PERM_RWX     = 0 // 00 = Read/Write/Execute (all permissions)
	PERM_EXEC    = 1 // 01 = Execute only (for code), Read only (for data)
	PERM_RSVD    = 2 // 10 = Reserved
	PERM_INVALID = 3 // 11 = Invalid (any access causes page fault)
)

const (
	PAGE_SIZE  = 4096 // 4KB pages
	PAGE_SHIFT = 12   // log2(PAGE_SIZE)
	PAGE_MASK  = 0x0FFF
)

// translateCode translates a virtual code address (byte address) to a physical byte address
func (cpu *CPU) translateCode(virtAddr uint16) (uint32, error) {
	return cpu.translate(virtAddr, false)
}

// translateData translates a virtual data address (byte address) to a physical byte address
func (cpu *CPU) translateData(virtAddr uint16) (uint32, error) {
	return cpu.translate(virtAddr, true)
}

// translate converts a virtual address to physical byte address through the MMU
// Both code and data virtAddr are byte addresses
func (cpu *CPU) translate(virtAddr uint16, isData bool) (uint32, error) {
	// Extract page number (bits 15-12)
	pageNum := (virtAddr >> PAGE_SHIFT) & 0x0F // 0-15

	// Determine MMU slot
	var slot int
	if isData {
		slot = 16 + int(pageNum) // Data pages are slots 16-31
	} else {
		slot = int(pageNum) // Code pages are slots 0-15
	}

	// Get MMU entry for current context
	var mmuEntry uint16
	if cpu.mode == ModeKernel {
		mmuEntry = cpu.mmu[0][slot]
	} else {
		mmuEntry = cpu.mmu[cpu.context][slot]
	}

	// Extract physical page number (bits 11-0) and permissions (bits 14-15)
	physPage := mmuEntry & 0x0FFF
	perm := (mmuEntry >> 12) & 0x03

	// Check permissions
	if perm == PERM_INVALID {
		return 0, fmt.Errorf("page fault: invalid page at vaddr=0x%04X", virtAddr)
	}

	// For code pages with PERM_EXEC (01), execution is allowed
	// For data pages with PERM_EXEC (01), only read is allowed (write will fault)
	// PERM_RWX (00) allows everything
	// PERM_RSVD (10) is reserved and treated as fault

	if perm == PERM_RSVD {
		return 0, fmt.Errorf("page fault: reserved permission at vaddr=0x%04X", virtAddr)
	}

	// Compute physical byte address
	// Physical pages are 4KB (4096 bytes = 2048 words)
	// Both code and data addresses are byte addresses
	pageOffset := virtAddr & PAGE_MASK // bits 11-0, byte offset (0-4095)
	physByteAddr := (uint32(physPage) << PAGE_SHIFT) | uint32(pageOffset)

	// Ensure physical address is within bounds
	// physMem is word-indexed, so multiply length by 2 for byte comparison
	if physByteAddr >= uint32(len(cpu.physMem))*2 {
		return 0, fmt.Errorf("physical address 0x%06X out of bounds", physByteAddr)
	}

	return physByteAddr, nil
}

// loadWord loads a 16-bit word from virtual data address
func (cpu *CPU) loadWord(virtAddr uint16) (uint16, error) {
	// Check alignment
	if virtAddr&1 != 0 {
		cpu.raiseException(EX_ALIGNMENT_FAULT, virtAddr) // Alignment fault
		return 0, nil
	}

	physByteAddr, err := cpu.translateData(virtAddr)
	if err != nil {
		cpu.raiseException(EX_PAGE_FAULT, virtAddr) // Page fault
		return 0, nil
	}

	// Convert byte address to word index
	wordIndex := physByteAddr >> 1
	value := cpu.physMem[wordIndex]

	// Trace memory read
	if cpu.tracer != nil {
		cpu.tracer.TraceMemoryRead(virtAddr, physByteAddr, value, false)
	}

	return value, nil
}

// loadByte loads an 8-bit byte from virtual data address (sign extended)
func (cpu *CPU) loadByte(virtAddr uint16) (uint16, error) {
	// Translate the byte address directly
	physByteAddr, err := cpu.translateData(virtAddr)
	if err != nil {
		cpu.raiseException(EX_PAGE_FAULT, virtAddr) // Page fault
		return 0, nil
	}

	// Convert byte address to word index and determine which byte
	wordIndex := physByteAddr >> 1
	isOdd := (physByteAddr & 1) != 0

	word := cpu.physMem[wordIndex]
	var value uint16

	if isOdd {
		// High byte
		value = (word >> 8) & 0xFF
	} else {
		// Low byte
		value = word & 0xFF
	}

	// Sign extend
	if value&0x80 != 0 {
		value |= 0xFF00
	}

	// Trace memory read
	if cpu.tracer != nil {
		cpu.tracer.TraceMemoryRead(virtAddr, physByteAddr, value, true)
	}

	return value, nil
}

// storeWord stores a 16-bit word to virtual data address
func (cpu *CPU) storeWord(virtAddr uint16, value uint16) error {
	// Check alignment
	if virtAddr&1 != 0 {
		cpu.raiseException(EX_ALIGNMENT_FAULT, virtAddr) // Alignment fault
		return nil
	}

	physByteAddr, err := cpu.translateData(virtAddr)
	if err != nil {
		cpu.raiseException(EX_PAGE_FAULT, virtAddr) // Page fault
		return nil
	}

	// Check write permission
	// Get MMU entry
	pageNum := (virtAddr >> PAGE_SHIFT) & 0x0F
	slot := 16 + int(pageNum) // Data pages
	var mmuEntry uint16
	if cpu.mode == ModeKernel {
		mmuEntry = cpu.mmu[0][slot]
	} else {
		mmuEntry = cpu.mmu[cpu.context][slot]
	}
	perm := (mmuEntry >> 12) & 0x03

	// PERM_EXEC (01) for data means read-only
	if perm == PERM_EXEC {
		cpu.raiseException(EX_PAGE_FAULT, virtAddr) // Page fault (write to read-only)
		return nil
	}

	// Trace memory write
	if cpu.tracer != nil {
		cpu.tracer.TraceMemoryWrite(virtAddr, physByteAddr, value, false)
	}

	// Convert byte address to word index
	wordIndex := physByteAddr >> 1
	cpu.physMem[wordIndex] = value
	return nil
}

// storeByte stores an 8-bit byte to virtual data address
func (cpu *CPU) storeByte(virtAddr uint16, value uint16) error {
	physByteAddr, err := cpu.translateData(virtAddr)
	if err != nil {
		cpu.raiseException(EX_PAGE_FAULT, virtAddr) // Page fault
		return nil
	}

	// Check write permission
	pageNum := (virtAddr >> PAGE_SHIFT) & 0x0F
	slot := 16 + int(pageNum)
	var mmuEntry uint16
	if cpu.mode == ModeKernel {
		mmuEntry = cpu.mmu[0][slot]
	} else {
		mmuEntry = cpu.mmu[cpu.context][slot]
	}
	perm := (mmuEntry >> 12) & 0x03

	if perm == PERM_EXEC {
		cpu.raiseException(EX_PAGE_FAULT, virtAddr)
		return nil
	}

	// Convert byte address to word index and determine which byte
	wordIndex := physByteAddr >> 1
	isOdd := (physByteAddr & 1) != 0

	// Read-modify-write
	word := cpu.physMem[wordIndex]
	if isOdd {
		// High byte
		word = (word & 0x00FF) | ((value & 0xFF) << 8)
	} else {
		// Low byte
		word = (word & 0xFF00) | (value & 0xFF)
	}

	// Trace memory write
	if cpu.tracer != nil {
		cpu.tracer.TraceMemoryWrite(virtAddr, physByteAddr, value&0xFF, true)
	}

	cpu.physMem[wordIndex] = word
	return nil
}

// loadCodeWord loads a word from code space (LCW instruction)
func (cpu *CPU) loadCodeWord(virtAddr uint16) (uint16, error) {
	// virtAddr is a byte address in code space
	physByteAddr, err := cpu.translateCode(virtAddr)
	if err != nil {
		cpu.raiseException(EX_PAGE_FAULT, virtAddr)
		return 0, nil
	}

	// Convert byte address to word index
	wordIndex := physByteAddr >> 1
	value := cpu.physMem[wordIndex]

	if cpu.tracer != nil {
		cpu.tracer.TraceMemoryRead(virtAddr, physByteAddr, value, false)
	}

	return value, nil
}

// LoadBinary loads a binary file into physical memory
// The new file format has a 16-byte header:
//   - Offset 0-1: Magic number 0xDDD1 (uint16 little-endian)
//   - Offset 2-3: Code size in bytes (uint16 little-endian)
//   - Offset 4-5: Data size in bytes (uint16 little-endian)
//   - Offset 6-15: Reserved (10 bytes)
//   - Code section starts at offset 16
//   - Data section follows code section immediately
// Both code and data load to physical address 0 (overlapping)
// The MMU separates code and data address spaces via different page mappings
func (cpu *CPU) LoadBinary(data []byte) error {
	const HEADER_SIZE = 16
	const MAGIC_NUMBER = 0xDDD1

	// Validate minimum size (must have header)
	if len(data) < HEADER_SIZE {
		return fmt.Errorf("binary file too small: %d bytes (minimum %d)", len(data), HEADER_SIZE)
	}

	// Parse header (little-endian)
	magic := uint16(data[0]) | (uint16(data[1]) << 8)
	codeSize := uint16(data[2]) | (uint16(data[3]) << 8)
	dataSize := uint16(data[4]) | (uint16(data[5]) << 8)

	// Validate magic number
	if magic != MAGIC_NUMBER {
		return fmt.Errorf("invalid magic number: 0x%04X (expected 0x%04X)", magic, MAGIC_NUMBER)
	}

	// Validate code size is non-zero
	if codeSize == 0 {
		return fmt.Errorf("binary file has no code (code size is 0)")
	}

	// Validate sizes
	expectedSize := HEADER_SIZE + int(codeSize) + int(dataSize)
	if len(data) < expectedSize {
		return fmt.Errorf("binary file size mismatch: got %d bytes, expected %d (header=%d + code=%d + data=%d)",
			len(data), expectedSize, HEADER_SIZE, codeSize, dataSize)
	}

	// Load code section (file offset 16+) into physical memory at address 0
	codeOffset := HEADER_SIZE
	for i := 0; i < int(codeSize)/2; i++ {
		offset := codeOffset + i*2
		low := uint16(data[offset])
		high := uint16(data[offset+1])
		cpu.physMem[i] = (high << 8) | low
	}
	if codeSize%2 != 0 {
		// Odd trailing byte goes into the low byte of the next word
		cpu.physMem[int(codeSize)/2] = uint16(data[codeOffset+int(codeSize)-1])
	}

	// Load data section (follows code) into physical memory at address 0 (overlapping with code)
	// The MMU will separate these via different page mappings
	if dataSize > 0 {
		dataOffset := codeOffset + int(codeSize)

		// Validate data doesn't exceed physical memory
		if dataSize > uint16(len(cpu.physMem)*2) {
			return fmt.Errorf("data section too large: %d bytes (max %d)", dataSize, len(cpu.physMem)*2)
		}

		for i := 0; i < int(dataSize)/2; i++ {
			offset := dataOffset + i*2
			low := uint16(data[offset])
			high := uint16(data[offset+1])
			cpu.physMem[i] = (high << 8) | low
		}
		if dataSize%2 != 0 {
			// Odd trailing byte goes into the low byte of the next word
			cpu.physMem[int(dataSize)/2] = uint16(data[dataOffset+int(dataSize)-1])
		}
	}

	return nil
}
