# emul - emulator for WUT-4 instruction set

The emulator executes binary programs in the file format produced by the [WUT-4 assembler](../asm). It loads a kernel mode binary and an optional user mode binary specified with `-u` and then begins execution in kernel mode at address 0 (the reset vector). When the program halts for any reason, it produces a register dump (unless `-q` was given) and then exits.

If `-d` is given, the emulator produces a register dump before each instruction executes and then pauses with a prompt. Single letter commands may be entered, including `h` for a definition of the single letter commands.

The emulator normally measures its own emulation time and produces an estimate of instructions emulated per second when it exits. This display is suppressed if the emulator has prompted during execution.

Execution of the `brk` instruction in WUT-4 code turns on the debug flag, causing an immediate register dump and prompt. The `r` command can be used to continue execution.

The emulated program halts if the `hlt` instruction is executed or if an interrupt, trap, fault, or exception (ITFE)  occurs with interrupts disabled (a "double fault"). Execution begins with interrupts disabled. ITFEs are enabled when the WUT-4 enters user mode and disabled if a TFE occurs. No hardware interrupts are currently emulated.

For more information, see the [ISA documentation](https://docs.google.com/document/d/1FGd5GkRvqJ2vp0SLA7tWmnykon00CeoPMjt_EcEqfLI/edit?usp=sharing).
