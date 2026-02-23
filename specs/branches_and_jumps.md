  How the Compiler Handles Branch Range                                                      
                                                                                             
  The strategy: always use "inverted branch over long jump"

  The compiler never emits a direct conditional branch to a user-visible label. Instead,
  every JUMPZ and JUMPNZ in the IR — which represent all if, while, and for control flow in
  YAPL — is lowered to this fixed 4-instruction sequence:

      adi  r4, r4, 0       ; set flags from r4 (the condition)
      brnz L_skipN         ; inverted: skip the jump if condition is FALSE
      jal  <actual target> ; unconditional long jump, full 16-bit range
  L_skipN:

  For JUMPNZ (branch if true), the inversion flips to brz:
      adi  r4, r4, 0
      brz  L_skipN         ; skip if condition is TRUE
      jal  <actual target>
  L_skipN:

  The short conditional branch (brnz or brz) never targets user code. It always targets the
  skip label, which is exactly 4 bytes away (one jal instruction). It cannot overflow. The
  actual branch-like logic is delegated to jal, which encodes a full 16-bit absolute address.

  Why jal works as an unconditional jump

  The Jmp emitter method (emit.go line 214) is just a wrapper around jal:

  // Clobbers LINK, but LINK is saved/restored in every function prologue/epilogue.
  func (e *Emitter) Jmp(label string) {
      e.Instr1("jal", label)
  }

  The comment explains why this is safe: because every function unconditionally saves LINK on
   entry and restores it on exit, a jal used as a jump inside a function body simply
  overwrites and discards the LINK register, which is already safely stacked. No LINK
  corruption occurs.

  jal expands to a 2-word LUI+JAL sequence encoding the full 16-bit address. It has
  effectively unlimited range within the 64KB I-space.

  Where short branches are used — and why they're always safe

  Short conditional branches (brz, brnz, brslt, etc.) appear in only two contexts in the
  generated code, and both are provably within range:

  1. Comparison sequences (genCompare, lines 1048–1110): A comparison like a < b emits a tst
  followed immediately by a conditional branch to a trueLabel or doneLabel a few instructions
   later — at most a handful of bytes, never out of range.
  2. Runtime library functions (genRuntimeLibrary, lines 109–340): The shift, multiply,
  divide, and modulo routines are small self-contained loops (the largest is __divu16 at
  about 16 instructions ≈ 32 bytes). The branches in these loops span only a few bytes.

  There are no direct short branches to IR-level labels anywhere in the code generator.

  Unconditional jumps and returns

  - OpJump (line 666): Maps directly to Jmp → jal. Full range, no issue.
  - genReturn (line 1186): Also uses Jmp to jump to the function's epilogue label
  (L_<funcname>_epilogue). Again, full 16-bit range, regardless of function size.

  ---
  Your Concern About Relocation

  Your instinct is correct. Here is the precise picture:

  Current pattern and relocation

  The current "inverted branch over jal" pattern means that every conditional branch target
  in user code is reached through a jal, which is already a 2-word LUI+JAL sequence encoding
  an absolute 16-bit address. In linker.md this is exactly the R_JAL relocation type. So:

  - The brnz skip / brz skip short branch: targets a label 4 bytes away, always local, never
  needs relocation.
  - The jal <target>: targets the actual control-flow destination. In a single-file world
  this resolves in yasm's pass 1. In a multi-file world, if the target is in another module,
  this needs an R_JAL relocation record — already accounted for in the linker design.

  If the pattern were changed

  If in the future someone changed the code to emit a direct short branch when the target is
  close enough, then:
  - A branch within the same function body, which stays in the same object file, resolves
  completely in yasm and never needs relocation. No change to the linker design.
  - Branches can never span object-file boundaries anyway: a conditional branch target is
  always a local label within the same function. Functions are never split across files.

  So the relocation design is not affected by whether the compiler uses the "inverted branch
  over jump" pattern or an optimized "direct short branch when possible" pattern. The only
  instructions that ever reference cross-module symbols are:
  - jal FunctionName — already R_JAL in the design
  - ldi rX, GlobalVar — already R_LDI_DATA / R_LDI_CODE in the design

  The one thing to update if you change the pattern: if a future optimization removed the jal
   from within a function for an intra-function branch and replaced it with a plain short
  branch, the assembler would need to verify the offset is in range (±512 bytes). The
  assembler already does this in genBRX during pass 2. The linker design would not need to
  change because intra-function references are never external.


