# Wholly Unnecessary Technologies (WUT) model 4 computer (WUT-4 or WUT4)

The content of this repo is [licensed](./LICENSE).

(July 2025): most of the attention for the last several months has been
on the digital simulator in the sim/ directory. Everything else in this
repo is old and needs to be revised.

#
## [asm](./asm) - wut4 assembler

Assembler based on [customasm](https://github.com/hlorenzi/customasm)


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


## pld - 22v10 experiments

including a general 8-bit register.
