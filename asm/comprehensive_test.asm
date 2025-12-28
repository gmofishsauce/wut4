; Comprehensive test of WUT-4 assembler
; Tests all instruction types and directives

.code

; Define some constants
.set STACK_TOP, 0x1000
.set UART_DATA, 96

start:
	; Test base instructions
	ldw r1, r2, 10
	ldb r3, r4, -5
	stw r5, r6, 0
	stb r7, r1, 7
	adi r2, r3, 42

	; Test LUI
	lui r1, 0x3FF

	; Test LDI alias (various forms)
	ldi r1, 5          ; Small immediate -> ADI
	ldi r2, 0x100      ; Aligned immediate -> LUI
	ldi r3, 0x12345    ; Full immediate -> LUI + ADI (truncated to 16 bits)

	; Test XOP instructions (3-operand ALU)
	add r1, r2, r3
	sub r4, r5, r6
	adc r1, r1, r0
	sbb r2, r2, r3
	and r3, r4, r5
	or r5, r6, r7
	xor r7, r1, r2

	; Test YOP instructions (2-operand)
	lsp r1, r2
	ssp r3, r4
	tst r5, r6

	; Test ZOP instructions (1-operand)
	not r1
	neg r2
	sxt r3
	sra r4
	srl r5

	; Test VOP instructions (0-operand)
	ccf
	scf

	; Test move alias
	mv r1, r2

	; Test shift left aliases
	sla r3
	sll r4

	; Test branches
loop:
	adi r1, r1, -1
	brnz loop
	br end

	; Test conditional branches
	brz zero_label
	breq zero_label
	brc carry_label
	brnc no_carry_label
	brsge signed_ge_label
	brslt signed_lt_label

zero_label:
carry_label:
no_carry_label:
signed_ge_label:
signed_lt_label:
	; Jump indirect
	ji r7

end:
	hlt

.data
	.align 2
	.words 0x1234, 0x5678, 0xABCD
	.bytes "Test string", 10, 0
	.space 16
	.bytes 0xFF, 0xFE, 0xFD
