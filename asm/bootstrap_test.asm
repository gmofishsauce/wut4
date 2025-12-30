.bootstrap

start:
	ldi r1, 0x1000
	ldi r2, message
	br loop

loop:
	ldb r3, r2, 0
	brz done
	adi r2, r2, 1
	br loop

done:
	hlt

message:
	.bytes "Bootstrap loaded\n", 0
