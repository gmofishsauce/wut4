  ---                                                                                        
  WUT-4 Architecture & Assembler Spec Review                                                 
                                                                                             
  Summary: Both documents are well-designed and internally coherent at the structural level. 
  A competent implementor could build most of the machine from them. However, there are two
  genuine correctness gaps (carry semantics, VOP count), one direct contradiction (MMU
  execute-only vs. readable), several notation ambiguities that would trip up a second
  implementor, and a handful of minor inaccuracies. No issues were found with the core RISC
  encoding or the pseudo-instruction expansions.

  ---
  Critical Issues

  [critical] Carry flag semantics after subtraction are never defined (wut4arch.md, BRx table
   + XOP/YOP descriptions)

  The BRx table says brc = bruge (branch if unsigned ≥) and brnc = brult (branch if unsigned
  <). This implies C = 1 means "no borrow" (ARM-style, NOT-borrow). But the spec never states
   this. The TST, SUB, and SBB descriptions never say whether C is set as carry-out or as the
   inverted borrow. For a subtraction rA − rB, these two conventions give opposite C values
  for most inputs. A second hardware implementor would have to guess, and the assembler,
  compiler, and runtime library depend critically on getting this right. The MEMORY.md from a
   previous session records tst rX, rX → C=1, confirming NOT-borrow, but that fact must
  appear in the architecture spec itself.

  Fix: Add one sentence to the XOP section: "Subtraction operations (SUB, SBB, TST) set the
  carry flag to the logical NOT of borrow; C=1 means the result was ≥ 0 unsigned (no borrow
  occurred)."

  ---
  [critical] VOP count arithmetic error (wut4arch.md, intro paragraph)

  "7 3-operand, 7 2-operand, 7 single-operand, and 8 0-operand instructions … (28
  instructions total plus 0xFFFF)"

  7 + 7 + 7 + 8 = 29, not 28. The VOP table does have 8 entries (CCF, SCF, DI, EI, HLT, BRK,
  RTI, DIE). Either the intro should say "7 0-operand instructions … (28 total) plus DIE
  (0xFFFF)" and demote DIE to a separate category, or it should say "29 total." DIE is
  already called out specially as "generates an illegal instruction fault," so changing it to
   "7 VOPs plus DIE" is cleaner. Either way, the current text is self-contradictory.

  ---
  Warning Issues

  [warning] "Execute-only" code pages are simultaneously described as readable (wut4arch.md,
  MMU section)

  "For code space, 01 is execute-only … Execute-only code pages are readable to the process,
  e.g. using the LCW instruction."

  These two statements directly contradict each other. "Execute-only" conventionally means
  reads are forbidden. Pick a name that matches the actual semantics: "no-write" or
  "read-execute" would be accurate. The current wording will cause confusion whenever someone
   implements a security monitor that relies on the named permission.

  ---
  [warning] Flag name inconsistency: "N" vs. "s" (wut4arch.md, FLAGS SPR table vs. BRx table)

  The FLAGS SPR description names the four flags C, Z, N, V (bit positions 0–3). The BRx
  condition table refers to the same flag as "s flag" (brsge: "s flag == v flag; s XNOR v").
  These are the same bit. Pick one name and use it throughout. The MEMORY.md and compiler
  code consistently use N, which should be canonical.

  ---
  [warning] LSI and SSI have opposite rA/rB roles (wut4arch.md, YOP section)

  For every other YOP, rA is the "first" operand and rB is the "second." The spec describes:
  - LSI: "loads the special register addressed by rB into the memory location addressed by
  rA" → mem[rA] = SPR[rB]
  - SSI: "stores to the special register addressed by rA from the memory location addressed
  by rB" → SPR[rA] = mem[rB]

  The SPR field is rB in LSI but rA in SSI; the memory field is rA in LSI but rB in SSI. The
  roles of the two register fields swap across these two instructions, which is internally
  inconsistent and an easy source of bugs. The spec should explicitly flag this asymmetry and
   explain the rationale (if any), or the encoding should be made consistent.

  ---
  [warning] BRL does not specify what value is written to LINK (wut4arch.md, BRx table)

  The BRx table entry for b2:b0=1 says brl: "unconditional branch to subroutine; writes link
  register". It doesn't say what value is written. By analogy with JAL this should be PC+2
  (the return address), but that must be stated explicitly. A hardware implementor who writes
   the wrong value here silently breaks every subroutine call made through BRL.

  ---
  [warning] "ret rN is modified" is incorrect (wut4arch.md and wut4asm.pdf,
  pseudo-instruction section)

  Both documents say: "ret [rN] — Implemented as JI rN … rN is modified." But JI reads rN to
  load the PC; it does not write to rN. The ZOP description confirms this: "JI loads all 16
  bits of the PC from the specified register." No modification of rN occurs. This note should
   be removed. (If the intent was "LINK is consumed/zeroed on return," that is also not
  described for JI anywhere and would be a separate architectural decision to document.)

  ---
  [warning] BRx branch offset sign extension is never stated (wut4arch.md, BRx description)

  "if the condition … is true, branch to PC + 2 + imm10"

  The spec never states whether imm10 is sign-extended. Backward branches require a negative
  offset, so it must be. This should say explicitly: "imm10 is a 10-bit signed
  two's-complement offset, giving a range of −512 to +511 bytes from the instruction
  following the branch."

  ---
  [warning] JAL uses "+" to mean bit concatenation (wut4arch.md, JAL description)

  "PC = rB15:6 + imm5:0"

  This uses arithmetic addition notation for bit concatenation. The text immediately after
  explains "taking the high-order 10 bits … and appending the low order 6 bits," which is
  concatenation, not addition. Write it as {rB[15:6], imm[5:0]} or rB[15:6] : imm[5:0] to
  eliminate the ambiguity. As written, a reader could mistake this for an actual 16-bit add,
  producing nonsensical PC behavior.

  ---
  [warning] ZOP flag behavior is incompletely specified (wut4arch.md, ZOP section)

  The spec specifies flag effects for SXT (clears C, sets Z) and SRA/SRL (C = bit shifted
  out). It is silent on NOT, NEG, DUB, and JI. Presumably NEG sets N, Z, V, C (it's
  negation), NOT sets N and Z, DUB has some flag effect, and JI leaves flags unchanged. This
  must be specified; the compiler and hand-written assembly both need to know whether they
  can rely on flags after these instructions.

  ---
  [warning] STB: which byte of the source register is stored? (wut4arch.md, STW/STB
  description)

  LDB specifies that a loaded byte is sign-extended. STB has no corresponding note.
  Presumably the low byte of rA is written, but this must be stated. Without it, storing the
  high byte vs. low byte is implementation-defined.

  ---
  [warning] Vector list skips vector 2 (wut4arch.md, Interrupts section)

  "Vector 0 is at kernel code address 0, vector 1 at address 4, vector 3 at 0xC, and so on."

  Vector 2 (expected at address 0x8) is skipped. This is almost certainly a typo for "vector
  2 at 0x8, vector 3 at 0xC." If the gap is intentional (e.g., vector 2 is reserved for a
  specific purpose), that needs to be documented.

  ---
  [warning] Assembler spec describes only the executable format, not the object format
  (wut4asm.pdf, Output File Format)

  The assembler spec describes the 0xDDD1 executable format. Since yasm -c now produces
  0xDDD2 WOF object files (documented in linker.md), the assembler spec is incomplete for the
   current implementation. A developer reading only wut4asm.pdf would not know the -c flag or
   the object file format exists. At minimum, a forward reference to linker.md should be
  added.

  ---
  Nits

  [nit] "brnzn" is a typo (wut4arch.md, BRx table, row 3)
  Should be brnz (matching the MEMORY.md and every other reference). The spurious trailing n
  appears only in this table.

  [nit] Encoding table dash notation is unexplained (wut4arch.md, instruction tables)
  The table uses i6 and i0 to mark the MSB and LSB of an immediate field, with - for the bits
   between them. This is non-standard notation and is never explained. Add a one-line key:
  "In the tables, iN marks the most-significant and i0 the least-significant bit of a
  contiguous N+1-bit immediate; dashes represent the intervening bits."

  [nit] mystringlen example is off by one (wut4asm.pdf, page 1–2)
  "some string\n\0" contains 11 printable characters + \n (1 byte) + \0 (1 byte) = 13 bytes,
  not 12. The .byte directive does not auto-null-terminate, and both escape sequences are
  listed explicitly.

  [nit] .byte vs. .bytes directive name mismatch (wut4asm.pdf)
  The overview example uses .byte (singular); the directives table names it .bytes (plural).
  One of these is wrong. Most assemblers use .byte.

  [nit] Assembler pseudo-instruction notation uses undefined mnemonic "LI" (wut4asm.pdf,
  srr/srw)
  srr rA, rB, imm7 (LI rB, imm7 ; LSP rA, rB) — "LI" is not defined anywhere in either spec.
  The intended pseudo is ldi. Use ldi consistently.

  [nit] "W4" instead of "WUT-4" (wut4arch.md, ITFE section)
  "the W4 halts" — should be "the WUT-4 halts."

  [nit] "dignotic" in .bootstrap description (wut4asm.pdf, Directives table)
  Typo for "diagnostic."

  [nit] "begin with the letter" (wut4asm.pdf, Symbols section)
  Missing article: "begin with a letter."

  ---
  Questions

  1. XOP result with rA=0: The XOP section says a zero destination "discards the result." Are
   the condition codes still updated when the result is discarded, or is the entire
  instruction a no-op for observable state? This matters for code that uses, e.g., add r0,
  r1, r2 as a flag-setting operation.
  2. LDB from unaligned address: The spec allows byte addressing but doesn't say whether byte
   loads must be byte-aligned (they trivially can be since bytes have no alignment
  requirement) nor what happens with word-unaligned LDW/STW — is that a fault or
  implementation-defined?
  3. JAL read-before-write ordering: The pseudo-instruction jal rT, label expands to lui rT,
  high ; jal rT, rT, low where rT is both the address source (rB) and the return-address
  destination (rA). The correctness of this sequence requires JAL to read rB before writing
  rA. Should this ordering guarantee be stated explicitly in the spec?


