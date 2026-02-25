; boot.asm - the very first instructions executed by a bootstrap loader
; or diagnostic program after hardware initialization. When entered,
; only one page is mapped: physical page frame 0 as virtual code page 0
; AND ALSO virtual data page 0. This is not a safe environment for execution
; because stack growth can overwrite the code. But we are running in privileged
; mode so we can just move the stack. Our first action is to map physical page
; frame one at virtual address 0xF000 so there is a mapped page at the standard
; stack location (the highest process virtual addresses.) crt0 can then run
; normally.
;
; MMU register format: bits [11:0] = physical page frame, bits [15:12] = RRPP.
; Physical page frame 1, PP=00 (full permissions) = 0x0001.
; KERN DATAMMU registers are SPR 80..95; slot 15 (0xF) = SPR 95.
; srw clobbers r1 (MMU value) and r2 (scratch for SPR address).
    ldi r1, 0x0001      ; physical page frame 1, PP=00 (full permissions)
    srw r1, r2, 95      ; KERN DATAMMU[15] = r1, maps virt 0xF000..0xFFFF to phys page 1
