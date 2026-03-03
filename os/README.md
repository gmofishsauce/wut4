# WUT-4 OS Bootstrap — Design and Conventions

This README documents the bootstrap loading process for the WUT-4. The most
important reader of this document is a future Claude instance working on the
bootloader, the OS, or the mkbootimg tool. Read the whole thing before making
changes to any of those.

## Overview

The WUT-4 starts in kernel mode (privileged, context 0) with only two MMU
slots live: kernel code page 0 and kernel data page 0 both map physical frame
0. The bootloader (loader.asm) runs from that single page. Its job is to load
a boot image from the SD card and transfer control to it in a fully-mapped
address space.

A boot image is produced by mkbootimg from an ordinary WUT-4 executable
(magic 0xDDD1). The image is a flat sequence of 512-byte SD sectors: one
header sector, then code sectors, then data sectors.

## Physical Frame Allocation

    Frame 0   loader code (kernel code page 0, read-only after boot.asm runs)
    Frame 1   loader stack (kernel data page 15, mapped by boot.asm preamble)
    Frame 2   trampoline + staging scratch (explained below)
    Frame 3   OS code page 0
    Frame 4   OS code page 1
    ...
    Frame 18  OS code page 15
    Frame 19  OS data page 0
    Frame 20  OS data page 1
    ...
    Frame 34  OS data page 15

The OS occupies at most 16 code pages (64 KiB) and 16 data pages (64 KiB),
which is the full virtual address space for a kernel-mode process.

## How Loading Works (the Staging Window)

The loader reads sectors into physical memory using a **data-side staging
window**, not through the code MMU. For each OS code page p (0..15):

  1. Map kernel data page 2 (SPR 82) → physical frame (3 + p).
  2. Read up to 8 sectors (one 4 KiB page worth) into virtual data address
     0x2000, which lands in physical frame (3 + p) via the data MMU.

For each OS data page p (0..15):

  1. Map kernel data page 3 (SPR 83) → physical frame (19 + p).
  2. Read up to 8 sectors into virtual data address 0x3000.

The critical point: **the loader never needs to set up the kernel code MMU
during loading**. All sector I/O goes through the data MMU staging window.
Physical frame 18 (OS code page 15) is fully and correctly loaded this way
even though the kernel code MMU slot 15 (SPR 79) still points to the
trampoline frame (frame 2) throughout the entire load phase.

## The Trampoline

After loading, the loader writes a small trampoline into physical frame 2 and
jumps to it at virtual code address 0xF000 (kernel code page 15 → frame 2,
set up by Main before the jump).

The trampoline's job is to install the final MMU mapping and hand off to the
OS. It does this sequence:

  1. Map kernel code pages 0–14 (SPRs 64–78) → OS frames 3–17.
  2. Map kernel data pages 0–15 (SPRs 80–95) → OS data frames 19–34.
     (Remapping data page 15 is safe here because the trampoline is a
     non-returning stub and never touches the stack after this point.)
  3. Jump to virtual code address 0x0000.

The trampoline does **not** remap kernel code page 15 (SPR 79). It cannot
safely do so because it is executing from that page. Remapping the page you
are fetching from causes the very next instruction fetch to come from the new
(OS) physical frame at the same virtual offset — behaviour the architecture
explicitly warns against and which the emulator may not handle predictably.

## The OS Entry Convention

Because the trampoline cannot remap code page 15, **the OS is responsible for
completing its own MMU setup as the first act of its entry point**.

When the OS receives control at VA 0x0000:

  - Kernel mode is active (privileged).
  - Interrupts are disabled.
  - Kernel code pages 0–14 map correctly to OS frames 3–17.
  - Kernel data pages 0–15 map correctly to OS data frames 19–34.
  - Kernel code page 15 (VA 0xF000–0xFFFF) still maps to physical frame 2
    (the trampoline). OS code page 15 is physically present in frame 18 but
    is not yet reachable through the code MMU.

The OS entry sequence (crt0 or equivalent) must perform this remap before
executing anything in VA 0xF000–0xFFFF or before enabling interrupts (since
interrupt vectors at high addresses would be unreachable):

    ; At OS entry (VA 0x0000), in kernel mode:
    ldi  r1, 18        ; physical frame 18, PP=00 (full permissions)
    srw  r1, r2, 79    ; write SPR 79 = KERN CODEMMU[15] -> frame 18

After those two instructions all 16 code pages and all 16 data pages are
correctly mapped and the OS has the full 64 KiB + 64 KiB virtual address
space available.

## Why Not Other Approaches

**User context (RTI trick):** The OS must run in kernel context (context 0).
User context MMU registers (SPRs 32–63) configure a user-mode process, not
the kernel. RTI transitions to user mode, which is wrong for the OS.

**PC-wrap trick:** One could place a single `ssp r1, r2` instruction at VA
0xFFFE so that after remapping code page 15, the PC wraps to 0x0000 via
16-bit overflow. This would make the bootloader fully autonomous. It was
rejected because the architecture should be free to generate a fault on PC
overflow in future hardware revisions, and depending on wrap-to-zero would
permanently close that option.

**Dual-mapping / two-stage trampoline:** Any scheme that uses two code page
slots for the same physical trampoline frame just moves the "last unmap"
problem to the second slot. The recursion bottoms out at the same constraint.

## Files

    os/bootloader/loader.asm    Hand-written bootstrap loader (assembly)
    os/bootloader/loader.yapl   (unused) YAPL source placeholder
    os/mkbootimg/main.go        Converts WUT-4 executable to boot image
    lib/boot.asm                Preamble: maps stack page before _start
    specs/wut4arch.md           Full ISA and MMU specification
    specs/fix-boot.txt          Notes on emulator MMU initialization fix needed
