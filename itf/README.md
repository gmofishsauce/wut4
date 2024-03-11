# Instruction Test Framework (ITF)

This directory tries to test the assembler and disassembler by round-
tripping them. It contains some test programs with the w4a (WUT-4 assembler)
extension.

For each test program t1.w4a, t2, etc., the itf program (itf.go) first
assembles the test program, producing wut4.out. It moves the generated binary
to a temporary working directory. The binary file is disassembled producing
a disassembly in the temporary directory, which is then assembled to produce
another wut4.out file. The two assembler output binaries are then compared for
equality.

This approach removes all issues with minor textual differences such as number
bases between the original assembler source and the disassembled source.
