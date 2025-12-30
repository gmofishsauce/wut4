.code
.set OUTPORT 96

start:
	ldi r2  OUTPORT
	ldi r3  data
unrolled:
	ldb r1 r3 0
	ssp r1 r2
	ldb r1 r3 1
	ssp r1 r2
	ldb r1 r3 2
	ssp r1 r2
	ldb r1 r3 3
	ssp r1 r2
	ldb r1 r3 4
	ssp r1 r2
	ldb r1 r3 5
	ssp r1 r2
	ldb r1 r3 6
	ssp r1 r2
	hlt

data:

	.bytes "Hello\n", 0
