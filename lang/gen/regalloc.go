// YAPL Code Generator - Register Allocation
// Simple linear scan register allocator

package main

// RegAllocator manages virtual to physical register mapping
type RegAllocator struct {
	// Virtual register to physical register mapping
	virtToPhys map[string]int

	// Physical register state
	regInUse [8]bool
	regVirt  [8]string // which virtual register, if any

	// Spill slots for virtual registers that don't fit in registers
	spillSlots map[string]int
	nextSpill  int // offset for next spill slot (relative to frame)

	// Frame size from function definition
	frameSize int

	// Track which callee-saved registers are used
	usedCalleeSaved map[int]bool

	// Current pending arguments for function calls
	pendingArgs map[int]string // arg index -> virtual register
}

// NewRegAllocator creates a new register allocator
func NewRegAllocator(frameSize int) *RegAllocator {
	ra := &RegAllocator{
		virtToPhys:      make(map[string]int),
		spillSlots:      make(map[string]int),
		usedCalleeSaved: make(map[int]bool),
		pendingArgs:     make(map[int]string),
		frameSize:       frameSize,
		nextSpill:       frameSize, // spill slots start after locals
	}

	// Mark R0 and R7 as always in use
	ra.regInUse[R0] = true
	ra.regInUse[R7] = true

	return ra
}

// Reset clears all allocations (for new basic block)
func (ra *RegAllocator) Reset() {
	ra.virtToPhys = make(map[string]int)
	for i := 1; i <= 6; i++ {
		ra.regInUse[i] = false
		ra.regVirt[i] = ""
	}
	ra.pendingArgs = make(map[int]string)
}

// Allocate returns a physical register for a virtual register
// If the virtual register is already allocated, returns that register
// Otherwise allocates a new register, spilling if necessary
func (ra *RegAllocator) Allocate(virt string) int {
	// Check if already allocated
	if phys, ok := ra.virtToPhys[virt]; ok {
		return phys
	}

	// Try to allocate a free register
	// Prefer callee-saved registers (R4-R6) for temporaries
	for _, r := range []int{R4, R5, R6, R3, R2, R1} {
		if !ra.regInUse[r] {
			ra.regInUse[r] = true
			ra.regVirt[r] = virt
			ra.virtToPhys[virt] = r
			if r >= R4 && r <= R6 {
				ra.usedCalleeSaved[r] = true
			}
			return r
		}
	}

	// All registers in use - need to spill
	// For simplicity, spill the virtual register itself to stack
	// and return a temporary register (we'll reload when needed)
	return ra.spillAndAllocate(virt)
}

// spillAndAllocate spills an existing register and allocates it for virt
func (ra *RegAllocator) spillAndAllocate(virt string) int {
	// Pick R6 to spill (arbitrary choice)
	spillReg := R6
	oldVirt := ra.regVirt[spillReg]

	if oldVirt != "" {
		// Record spill slot for old virtual register
		ra.spillSlots[oldVirt] = ra.nextSpill
		ra.nextSpill += 2
	}

	// Allocate to new virtual register
	ra.regInUse[spillReg] = true
	ra.regVirt[spillReg] = virt
	ra.virtToPhys[virt] = spillReg

	return spillReg
}

// Free marks a virtual register as no longer needed
func (ra *RegAllocator) Free(virt string) {
	if phys, ok := ra.virtToPhys[virt]; ok {
		ra.regInUse[phys] = false
		ra.regVirt[phys] = ""
		delete(ra.virtToPhys, virt)
	}
}

// GetPhys returns the physical register for a virtual register
// Returns -1 if not allocated
func (ra *RegAllocator) GetPhys(virt string) int {
	if phys, ok := ra.virtToPhys[virt]; ok {
		return phys
	}
	return -1
}

// IsSpilled returns true if the virtual register has been spilled
func (ra *RegAllocator) IsSpilled(virt string) bool {
	_, ok := ra.spillSlots[virt]
	return ok
}

// GetSpillSlot returns the stack offset for a spilled virtual register
func (ra *RegAllocator) GetSpillSlot(virt string) int {
	return ra.spillSlots[virt]
}

// GetUsedCalleeSaved returns which callee-saved registers were used
func (ra *RegAllocator) GetUsedCalleeSaved() []int {
	var regs []int
	for r := R4; r <= R6; r++ {
		if ra.usedCalleeSaved[r] {
			regs = append(regs, r)
		}
	}
	return regs
}

// MarkUsed marks a physical register as in use (for parameter registers)
func (ra *RegAllocator) MarkUsed(phys int) {
	ra.regInUse[phys] = true
}

// MarkFree marks a physical register as free
func (ra *RegAllocator) MarkFree(phys int) {
	if phys >= R1 && phys <= R6 {
		ra.regInUse[phys] = false
		ra.regVirt[phys] = ""
	}
}

// AllocateSpecific allocates a specific physical register for a virtual register
func (ra *RegAllocator) AllocateSpecific(virt string, phys int) {
	// Free previous allocation if any
	if oldPhys, ok := ra.virtToPhys[virt]; ok && oldPhys != phys {
		ra.regInUse[oldPhys] = false
		ra.regVirt[oldPhys] = ""
	}

	// If the target physical register is in use, we need to handle that
	if ra.regInUse[phys] && ra.regVirt[phys] != virt {
		// The register is used by someone else - spill them
		oldVirt := ra.regVirt[phys]
		if oldVirt != "" {
			ra.spillSlots[oldVirt] = ra.nextSpill
			ra.nextSpill += 2
			delete(ra.virtToPhys, oldVirt)
		}
	}

	ra.regInUse[phys] = true
	ra.regVirt[phys] = virt
	ra.virtToPhys[virt] = phys

	if phys >= R4 && phys <= R6 {
		ra.usedCalleeSaved[phys] = true
	}
}

// SetPendingArg records an argument value for an upcoming call
func (ra *RegAllocator) SetPendingArg(index int, virt string) {
	ra.pendingArgs[index] = virt
}

// GetPendingArgs returns pending arguments
func (ra *RegAllocator) GetPendingArgs() map[int]string {
	return ra.pendingArgs
}

// ClearPendingArgs clears pending arguments after a call
func (ra *RegAllocator) ClearPendingArgs() {
	ra.pendingArgs = make(map[int]string)
}

// SaveCallerSaved returns which caller-saved registers need saving before a call
func (ra *RegAllocator) SaveCallerSaved() []int {
	var regs []int
	for r := R1; r <= R3; r++ {
		if ra.regInUse[r] && ra.regVirt[r] != "" {
			regs = append(regs, r)
		}
	}
	return regs
}

// GetTotalFrameSize returns total frame size including spill slots
func (ra *RegAllocator) GetTotalFrameSize() int {
	return ra.nextSpill
}
