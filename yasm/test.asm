; Simple test program for WUT-4 assembler
.code

start:
	ldi r1, 42        ; Load immediate value 42 into r1
	ldi r2, 0x100     ; Load 256 into r2
	add r3, r1, r2    ; r3 = r1 + r2
	ldw r4, r3, 0     ; Load word from memory at r3
	stw r4, r3, 4     ; Store word to memory at r3+4
	br end            ; Branch to end

end:
	hlt               ; Halt

.data
	.words 0x1234, 0x5678
	.bytes "Hello, WUT-4!", 0
