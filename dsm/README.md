# dsm - diassembler for wut4

Dsm is a disassembler for WUT-4 binaries. It produces assembly language
on its standard output. The default output cannot be reassembled directly
because it's a listing format, with memory addresses and opcodes.

Dsm understands the following command line flags:

-q quiets the addresses and opcodes. This leaves a format that the
assembler will consume.

-f 0x%04X@0x%04X, e.g. -f 0xC00D@0x1000 disassembles the literal
instruction 0xC00D as if it were located at code address 0x1000 (the
code address matters for branch instructions).
