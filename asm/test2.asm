.code

start:
	ldi r1, 42
	ldi r2, 0x100
	add r3, r1, r2
	ldw r4, r3, 0
	stw r4, r3, 4
	br end

end:
	hlt

.data
	.words 0x1234, 0x5678
	.bytes "Hello", 0
