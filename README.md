# Wholly Unnecessary Technologies (WUT) model 4 computer (WUT-4 or WUT4)

The content of this repo is [licensed](./LICENSE).

## [SAVE](./SAVE) - abandoned work

The contents may be useful in the future.

## [asm](./asm) - wut4 assembler

Assembler based on [customasm](https://github.com/hlorenzi/customasm)

## [dig](./dig) - simulations in hneeman/Digital

Digital is a graphical logic simulator. Some parts (or all)
of the WUT-4 may be simulated in Digital.

## [dsm](./dsm) - wut4 disassembler

Disassembler written in Golang

## [emul](./emul) - wut4 emulator

Functional emulator written in Golang

## [exer/cex](./exer/cex) - chip exerciser/tester

CLI tool that communicates via USB serial with a Nano. The Nano
is configured as a chip exerciser / tester. Also in `exer` are
the KiCad schematic for the chip exerciser in `ki` and the Nano
firmware in `fw`. The firmware is a simplified version of the
YARC firmware.

## [itf](./itf) - test framework for assembler and disassembler

A simple test framework that assembles a test file, disassembles
the binary, reassembles the disassembly creating a second binary,
and checks that the first and second binaries are identical.

## [ki](./ki) - KiCad files for the WUT-4

Schematics and board designs, some merely experiments with KiCad,
board layout practice, surface mount technology, etc.

## [yapl-1](./yapl-1) - wut4 compiler version 1

The first version of the compiler for YAPL, Yet Another Programming
Language, targeted at the WUT-4.

## [yapl-2](./yapl-2) - wut4 compiler version 2

Second version of YAPL compiler.
