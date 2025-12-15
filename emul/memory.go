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

// translateCode translates a virtual code address to a physical address
func (cpu *CPU) translateCode(virtAddr uint16) (uint32, error) {
	return cpu.translate(virtAddr, false)
}

// translateData translates a virtual data address to a physical address
func (cpu *CPU) translateData(virtAddr uint16) (uint32, error) {
	return cpu.translate(virtAddr, true)
}

// translate converts a virtual address to physical address through the MMU
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
	perm := (mmuEntry >> 14) & 0x03

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

	// Compute physical address
	pageOffset := virtAddr & PAGE_MASK
	physAddr := (uint32(physPage) << PAGE_SHIFT) | uint32(pageOffset)

	// Ensure physical address is within bounds
	if physAddr >= uint32(len(cpu.physMem)) {
		return 0, fmt.Errorf("physical address 0x%06X out of bounds", physAddr)
	}

	return physAddr, nil
}

// loadWord loads a 16-bit word from virtual data address
func (cpu *CPU) loadWord(virtAddr uint16) (uint16, error) {
	// Check alignment
	if virtAddr&1 != 0 {
		cpu.raiseException(0x0014, virtAddr) // Alignment fault
		return 0, nil
	}

	physAddr, err := cpu.translateData(virtAddr)
	if err != nil {
		cpu.raiseException(0x0012, virtAddr) // Page fault
		return 0, nil
	}

	value := cpu.physMem[physAddr]

	// Trace memory read
	if cpu.tracer != nil {
		cpu.tracer.TraceMemoryRead(virtAddr, physAddr, value, false)
	}

	return value, nil
}

// loadByte loads an 8-bit byte from virtual data address (sign extended)
func (cpu *CPU) loadByte(virtAddr uint16) (uint16, error) {
	// Bytes are stored in little-endian format
	// Even addresses are low byte, odd addresses are high byte
	wordAddr := virtAddr & 0xFFFE
	isOdd := (virtAddr & 1) != 0

	physAddr, err := cpu.translateData(wordAddr)
	if err != nil {
		cpu.raiseException(0x0012, virtAddr) // Page fault
		return 0, nil
	}

	word := cpu.physMem[physAddr]
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
		cpu.tracer.TraceMemoryRead(virtAddr, physAddr, value, true)
	}

	return value, nil
}

// storeWord stores a 16-bit word to virtual data address
func (cpu *CPU) storeWord(virtAddr uint16, value uint16) error {
	// Check alignment
	if virtAddr&1 != 0 {
		cpu.raiseException(0x0014, virtAddr) // Alignment fault
		return nil
	}

	physAddr, err := cpu.translateData(virtAddr)
	if err != nil {
		cpu.raiseException(0x0012, virtAddr) // Page fault
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
	perm := (mmuEntry >> 14) & 0x03

	// PERM_EXEC (01) for data means read-only
	if perm == PERM_EXEC {
		cpu.raiseException(0x0012, virtAddr) // Page fault (write to read-only)
		return nil
	}

	// Trace memory write
	if cpu.tracer != nil {
		cpu.tracer.TraceMemoryWrite(virtAddr, physAddr, value, false)
	}

	cpu.physMem[physAddr] = value
	return nil
}

// storeByte stores an 8-bit byte to virtual data address
func (cpu *CPU) storeByte(virtAddr uint16, value uint16) error {
	wordAddr := virtAddr & 0xFFFE
	isOdd := (virtAddr & 1) != 0

	physAddr, err := cpu.translateData(wordAddr)
	if err != nil {
		cpu.raiseException(0x0012, virtAddr) // Page fault
		return nil
	}

	// Check write permission
	pageNum := (wordAddr >> PAGE_SHIFT) & 0x0F
	slot := 16 + int(pageNum)
	var mmuEntry uint16
	if cpu.mode == ModeKernel {
		mmuEntry = cpu.mmu[0][slot]
	} else {
		mmuEntry = cpu.mmu[cpu.context][slot]
	}
	perm := (mmuEntry >> 14) & 0x03

	if perm == PERM_EXEC {
		cpu.raiseException(0x0012, virtAddr)
		return nil
	}

	// Read-modify-write
	word := cpu.physMem[physAddr]
	if isOdd {
		// High byte
		word = (word & 0x00FF) | ((value & 0xFF) << 8)
	} else {
		// Low byte
		word = (word & 0xFF00) | (value & 0xFF)
	}

	// Trace memory write
	if cpu.tracer != nil {
		cpu.tracer.TraceMemoryWrite(virtAddr, physAddr, value&0xFF, true)
	}

	cpu.physMem[physAddr] = word
	return nil
}

// loadCodeWord loads a word from code space (LCW instruction)
func (cpu *CPU) loadCodeWord(virtAddr uint16) (uint16, error) {
	// Check alignment
	if virtAddr&1 != 0 {
		cpu.raiseException(0x0014, virtAddr)
		return 0, nil
	}

	physAddr, err := cpu.translateCode(virtAddr)
	if err != nil {
		cpu.raiseException(0x0012, virtAddr)
		return 0, nil
	}

	value := cpu.physMem[physAddr]

	if cpu.tracer != nil {
		cpu.tracer.TraceMemoryRead(virtAddr, physAddr, value, false)
	}

	return value, nil
}

// LoadBinary loads a binary file into physical memory starting at address 0
func (cpu *CPU) LoadBinary(data []byte) error {
	// Data is little-endian 16-bit words
	if len(data)%2 != 0 {
		return fmt.Errorf("binary size must be even (got %d bytes)", len(data))
	}

	wordCount := len(data) / 2
	if wordCount > len(cpu.physMem) {
		return fmt.Errorf("binary too large: %d words (max %d)", wordCount, len(cpu.physMem))
	}

	// Load words in little-endian format
	for i := 0; i < wordCount; i++ {
		low := uint16(data[i*2])
		high := uint16(data[i*2+1])
		cpu.physMem[i] = (high << 8) | low
	}

	return nil
}
