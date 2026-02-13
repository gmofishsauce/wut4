# Wholly Unnecessary Technologies (WUT) model 4 computer (WUT-4 or WUT4)

The content of this repo is [licensed](./LICENSE).


#
## [yasm](./yasm) - wut4 assembler

Assembler written in Go by Claude Code.


## [emul](./emul) - wut4 emulator

Functional emulator written in Go by Claude Code.

## [lang] (./lang) - YAPL language)

Four pass compiler written in Go by Claude Code.

## [exer/cex](./exer/cex) - chip exerciser/tester

CLI tool that communicates via USB serial with a Nano. The Nano
is configured as a chip exerciser / tester. Also in `exer` are
the KiCad schematic for the chip exerciser in `ki` and the Nano
firmware in `fw`. The firmware is a simplified version of the
YARC firmware.

## [lib](./lib)

Eventual library of I/O routines for emulated devices.

## [pld](./pld) - 22v10 experiments

Including a general 8-bit register.
