  ---                                                                                        
  Review: WUT-4 Emulator vs. Architecture Specification                                      
                                                                                            
  Summary: The emulator is well-structured and covers the broad shape of the architecture,   
  but has several correctness bugs—two of which (MMU permission bit extraction and LSI     
  operand swap) would cause complete functional failure if the relevant features are         
  exercised, and one (BRL return address) would break all subroutine calls using the
  branch-and-link instruction. There are also a handful of spec compliance gaps worth        
  discussing before finalizing.                                                              
                                                                                             
  ---                                                                                        
  Critical Issues — Must Fix

  [critical] MMU permission bits extracted from wrong bit position

  File: memory.go:65 (and duplicated at memory.go:183, memory.go:218)

  The spec describes the MMU register layout as "RRPP" where bits [15:14] are reserved and
  bits [13:12] are the PP permission field (with bits [11:0] as the physical page number).
  The emulator does:

  perm := (mmuEntry >> 14) & 0x03  // reads bits [15:14] = RR (reserved!)

  It should be:

  perm := (mmuEntry >> 12) & 0x03  // reads bits [13:12] = PP (permission)

  Because all MMU entries are initialized to zero, PERM_RWX (00) works accidentally. But when
   any kernel code tries to mark a page execute-only (PP=01, i.e., mmuEntry = 0x1000) or
  invalid (PP=11, i.e., mmuEntry = 0x3000), the emulator reads bits [15:14] which are both 0,
   yielding PERM_RWX and silently granting all permissions. Memory protection is entirely
  non-functional for any non-default page configuration.

  The duplicate MMU lookup in storeWord/storeByte (re-fetching the entry to check write
  permission after translate() already ran) has the same bug and is also redundant — consider
   moving the write-permission check into translate() by passing the access type.

  ---
  [critical] BRL saves branch target to LINK instead of return address

  File: execute.go:121-133

  // condition is true
  offset := int32(int16(inst.imm10))
  cpu.pc = uint16(int32(cpu.pc+2) + offset)   // cpu.pc is now the TARGET

  if inst.branchCond == BR_LINK {
      cpu.spr[cpu.mode][SPR_LINK] = cpu.pc    // saves target, not return addr!
  }

  LINK should hold the address of the instruction after BRL (original_pc + 2). The fix:

  returnAddr := cpu.pc + 2
  cpu.pc = uint16(int32(cpu.pc+2) + offset)
  if inst.branchCond == BR_LINK {
      cpu.spr[cpu.mode][SPR_LINK] = returnAddr
  }

  Every subroutine called via brl will return to a garbage address with the current code.

  ---
  [critical] LSI operands are swapped

  File: execute.go:282-293

  Spec: LSI stores SPR[rB] into memory at address rA. The emulator does the opposite:

  case 1: // LSI
      sprAddr := regs[inst.rA]   // spec says rB selects the SPR
      memAddr := regs[inst.rB]   // spec says rA is the memory address
      value, err := cpu.loadSPR(sprAddr)
      ...
      err = cpu.storeWord(memAddr, value)

  SSI (case 3) is correct. Only LSI is swapped.

  ---
  [critical] XOP with rA=0 writes to gen[0] instead of discarding result

  File: execute.go:263

  The spec is explicit: "These instructions (XOPs) never read or update the LINK special
  register; a zero register specifier produces the value 0 and a 0 target specifier discards
  the result of the operation."

  The emulator unconditionally executes regs[inst.rA] = uint16(result). When rA=0, this
  overwrites gen[mode][0]. Since LDW/LDB/STW/STB read regs[inst.rB] directly without a
  zero-check, a subsequent ldw r3, r0+0 will use a corrupted base register. The same
  black-hole issue exists for LDW/LDB result writes (rA=0), ZOP results (rA=0), and YOP loads
   (rA=0). ADI, LUI, and JAL handle rA=0 correctly; all others need the check.

  ---
  Warning Issues — Should Fix

  [warning] SYS instruction saves wrong PC to IRR

  File: cpu.go: handleException() + execute.go:326-334

  The spec says "The PC of the first instruction not completed is stored in IRR." For a page
  fault (instruction cannot complete), IRR should point to the faulting instruction so it can
   be retried. For SYS (a deliberate, completed trap), IRR should point to SYS+2 so RTI
  returns to the caller. Currently, no exception path advances PC before calling
  raiseException, so IRR is always the faulting/calling instruction's address. Returning from
   a SYS call will re-execute the SYS instruction.

  Fix: in the SYS case, either advance cpu.pc += 2 before calling raiseException, or give
  handleException a way to distinguish "retry" (page fault, alignment) from "continue" (SYS)
  exception types.

  ---
  [warning] ICR and IDR incorrectly writable

  File: spr.go:152-153

  The spec explicitly states: "ICR: Writes are ignored" and "IDR: Writes are ignored." The
  emulator writes to both in the SSP path:

  case SPR_IRR, SPR_ICR, SPR_IDR, SPR_ISR:
      cpu.spr[ModeKernel][spr] = value   // ICR and IDR should be no-ops

  ---
  [warning] User mode SPR writes restricted to LINK only

  File: spr.go:127-133

  The spec says user mode can access SPRs 0–7 (both read and write). The emulator allows only
   SPR 0 (LINK) for user-mode writes; any write to SPR 1 (FLAGS) from user mode generates an
  illegal instruction fault. A user process should be able to set/clear condition flags via
  SSP to FLAGS.

  ---
  [warning] FLAGS read masks out T flag even in kernel mode

  File: spr.go:46

  flags := cpu.spr[cpu.mode][SPR_FLAGS] & 0x00FF

  The mask 0x00FF covers bits 7..0, but the Trap flag T is at bit 8 (FLAG_T = 0x0100). In
  kernel mode, reading FLAGS should return the T flag as well. The mask should be 0x03FF in
  kernel mode (bits 9..0, excluding read-only IE bit 9 which is overlaid from intEnabled).
  User mode reads should return only 0x000F (the four arithmetic flags).

  ---
  [warning] Exception vector spacing and SYS vector base appear incorrect

  File: constants.go:69-80

  The spec gives the concrete example "Vector 0 at address 0, vector 1 at address 4, vector 3
   at 0xC", which describes 4-byte (2-word) spacing—consistent with each vector holding an
  LUI+JAL pair. The current constants:

  EX_PAGE_FAULT      = 0x0012  // byte address 18: between vectors 4 and 5!
  EX_MACHINE_CHECK   = 0x001E  // byte address 30: between vectors 7 and 8!
  EX_SYSCALL_BASE    = 0x0010  // should be 0x0020 (vectors 8..15 per spec TODO #7 DONE)

  EX_PAGE_FAULT and EX_MACHINE_CHECK land at non-4-byte-aligned addresses, which can't
  accommodate the expected 2-instruction vector stubs. The SYS vector base should be 0x0020
  (vector 8) through 0x003C (vector 15), not 0x0010 through 0x001E. I'm flagging this as a
  question (see below) since the spec doesn't enumerate all vector assignments explicitly.

  ---
  [warning] Per-context general registers not implemented

  File: cpu.go:43-44

  The spec states: "The hardware provides a minimum of 256 complete sets of MMU registers,
  just as it does for general registers" and "A properly designed kernel may maintain up to
  255 user processes and switch between them simply by writing the context register (no
  per-process state need be saved to data memory when switching between contexts)."

  The emulator has gen[2][8] (kernel and one user bank). Writing the CONTEXT register changes
   cpu.context but does not switch register banks—all user contexts share the same eight
  registers. True per-context switching is not possible without gen[256][8] (or equivalent).
  This is a significant architectural gap for any multi-process workload.

  ---
  [warning] UART interrupt enable bits not implemented

  File: spr.go:256-257, io.go

  The spec defines bit 7 of u2 (SPR 98) and u3 (SPR 99) as interrupt enable flags, with
  interrupt delivery on vector 3 (address 0xC). Writes to SPR 98/99 are silently dropped, the
   enable bits are never stored, and reads always return 0 for bit 7. UART-driven interrupts
  cannot work.

  ---
  [warning] Duplicate fetch-decode-execute loop in runEmulator

  File: main.go:209-261

  runEmulator contains a full copy of the loop already in cpu.Run(). The two paths can
  diverge (the max-cycles path has a slightly different double-fault check at line 220 that
  differs from raiseException's logic). Prefer calling cpu.Run() from runEmulator and
  tracking max-cycles in the CPU's Run loop itself, or at minimum share a single
  execute-one-step helper.

  ---
  Nits

  - [nit] Magic numbers for exception vectors in cpu.go (0x0012, 0x0014) and memory.go — use
  the named EX_PAGE_FAULT, EX_ALIGNMENT_FAULT constants that already exist.
  - [nit] Debug fmt.Fprintf calls in execute.go (lines 149–156, 413–415) emit "JAL DEBUG:"
  and "RTN DEBUG:" to the trace file unconditionally when a tracer is attached. These should
  be removed or (<== human note: the code review was truncated at this point)

