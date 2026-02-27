; crt0.asm - Normal program startup for WUT-4
; SP = 0x0000 so stack grows downward from top of 64KB D-space.
_start:
    ldi r7, 0
    jal Main
    hlt
