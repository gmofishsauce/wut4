; Demonstration of capital letter symbol printing
.code

; These symbols start with capital letters and will be printed
.set UART_BASE 0x8000
.set SCREEN_WIDTH 80
.set MAX_BUFFER 256

Start:
	ldi r1, UART_BASE
	ldi r2, SCREEN_WIDTH
	hlt

; These symbols start with lowercase and won't be printed
loop:
	ldi r3, 0
end:
	hlt
