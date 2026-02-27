; loader.asm - WUT-4 bootstrap loader (hand-written assembly)
;
; Loads the OS from the SD card boot image and transfers control to it.
; Boot image layout (512-byte sectors):
;   Sector 0:                 header (magic 0xDDDD, code_sectors u16, data_sectors u16)
;   Sectors 1..code_sectors:  OS code, zero-padded to sector boundary
;   Sectors 1+code_sectors..: OS data, zero-padded to sector boundary
;
; Physical frame allocation:
;   Frame 0:   loader (code)
;   Frame 1:   stack (data page 15, mapped below)
;   Frame 2:   trampoline + staging
;   Frames 3+: OS pages
;
; boot.asm preamble: map physical frame 1 at virtual data 0xF000..0xFFFF
; MMU entry format: bits[11:0]=physical frame, bits[15:12]=perms (0=RWX)
; srw clobbers r1 and r2
    ldi r1, 0x0001
    srw r1, r2, 95

; Bootstrap entry
_start:
    ldi r7, 0
    jal Main
    hlt

; -----------------------------------------------------------------------
; Leaf I/O primitives (no stack frame needed)
; r0 is hardwired zero; LINK = SPR[0]
; -----------------------------------------------------------------------

; Putc(byte b) - r1=b - write to console TX FIFO (SPR 96)
Putc:
    ldi r2, 96
    ssp r1, r2
    ret

; spiSend(byte tx) - r1=tx - write to SPI data (SPR 100)
spiSend:
    ldi r2, 100
    ssp r1, r2
    ret

; pump() / spiRecv() / spiRx() - read one byte from SPI data (SPR 100)
; Three names, one body: pump discards the result, spiRecv/spiRx return it in r1
pump:
spiRecv:
spiRx:
    ldi r2, 100
    lsp r1, r2
    ret

; spiSel(byte val) - r1=val - write chip-select (SPR 101); 0=sel, 1=desel
spiSel:
    ldi r2, 101
    ssp r1, r2
    ret

; storeByte(uint16 addr, byte val) - r1=addr, r2=val
storeByte:
    stb r2, r1, 0
    ret

; storeWord(uint16 addr, uint16 val) - r1=addr, r2=val
storeWord:
    stw r2, r1, 0
    ret

; loadWord(uint16 addr) - r1=result
loadWord:
    ldw r1, r1, 0
    ret

; writeMMU(uint16 spr, uint16 frame) - r1=spr, r2=frame
; ssp r2, r1 stores r2 to SPR[r1]
writeMMU:
    ssp r2, r1
    ret

; -----------------------------------------------------------------------
; pump5: discard five SPI response bytes
; Non-leaf: saves LINK only (2-byte frame)
; -----------------------------------------------------------------------
pump5:
    adi r7, r7, -2
    lsp r3, r0
    stw r3, r7, 0
    jal pump
    jal pump
    jal pump
    jal pump
    jal pump
    ldw r3, r7, 0
    ssp r3, r0
    adi r7, r7, 2
    ret

; -----------------------------------------------------------------------
; waitResp: poll SPI up to 32 times for a non-0xFF response byte
; Returns: r1 = response byte (0xFF on timeout)
; Frame: 6 bytes (LINK@0, r4@2, r5@4)
; r4 = i (loop counter), r5 = resp
; -----------------------------------------------------------------------
waitResp:
    adi r7, r7, -6
    lsp r3, r0
    stw r3, r7, 0
    stw r4, r7, 2
    stw r5, r7, 4
    ldi r4, 0               ; i = 0
l_wr_loop:
    ldi r3, 32
    tst r4, r3              ; i - 32
    brult l_wr_body         ; if i < 32, do body
    ldi r1, 255             ; timeout: return 0xFF
    br l_wr_done
l_wr_body:
    jal spiRx               ; r1 = spiRx()
    or r5, r1, r0           ; r5 = resp
    ldi r3, 255
    tst r1, r3              ; resp - 255
    brnz l_wr_found         ; if resp != 0xFF, return it
    adi r4, r4, 1           ; i++
    br l_wr_loop
l_wr_found:
    or r1, r5, r0           ; r1 = resp
l_wr_done:
    ldw r3, r7, 0
    ssp r3, r0
    ldw r4, r7, 2
    ldw r5, r7, 4
    adi r7, r7, 6
    ret

; -----------------------------------------------------------------------
; sdCmd: send 6-byte SD SPI command, return R1 response byte
; Args: r1=cmd, r2=argHi (bits 31:16), r3=argLo (bits 15:0)
; Returns: r1 = R1 response byte
; Frame: 8 bytes (LINK@0, r4@2, r5@4, r6@6)
; r4=cmd, r5=argHi, r6=argLo held across spiSend calls
; r3 used for crc (spiSend is leaf; doesn't touch r3)
; -----------------------------------------------------------------------
sdCmd:
    adi r7, r7, -8
    stw r4, r7, 2           ; save r4 before clobbering
    stw r5, r7, 4
    stw r6, r7, 6
    or r4, r1, r0           ; r4 = cmd
    or r5, r2, r0           ; r5 = argHi
    or r6, r3, r0           ; r6 = argLo  (before lsp clobbers r3)
    lsp r3, r0
    stw r3, r7, 0           ; save LINK
    ; compute crc into r3
    ldi r3, 255             ; default crc = 0xFF
    tst r4, r0              ; cmd - 0
    brnz l_crc_not0
    ldi r3, 149             ; CMD0 crc = 0x95
    br l_crc_done
l_crc_not0:
    ldi r2, 8
    tst r4, r2              ; cmd - 8
    brnz l_crc_done
    ldi r3, 135             ; CMD8 crc = 0x87
l_crc_done:
    ; spiSend(cmd | 0x40)
    ldi r2, 64              ; 0x40
    or r1, r4, r2
    jal spiSend
    ; spiSend(argHi >> 8): copy argHi to r1, shift right 8
    or r1, r5, r0
    srl r1
    srl r1
    srl r1
    srl r1
    srl r1
    srl r1
    srl r1
    srl r1
    jal spiSend
    ; spiSend(argHi & 0xFF)
    ldi r2, 255
    and r1, r5, r2
    jal spiSend
    ; spiSend(argLo >> 8)
    or r1, r6, r0
    srl r1
    srl r1
    srl r1
    srl r1
    srl r1
    srl r1
    srl r1
    srl r1
    jal spiSend
    ; spiSend(argLo & 0xFF)
    ldi r2, 255
    and r1, r6, r2
    jal spiSend
    ; spiSend(crc)
    or r1, r3, r0
    jal spiSend
    ; return byte(waitResp())
    jal waitResp
    ldw r3, r7, 0
    ssp r3, r0
    ldw r4, r7, 2
    ldw r5, r7, 4
    ldw r6, r7, 6
    adi r7, r7, 8
    ret

; -----------------------------------------------------------------------
; sdInit0: send 80 SPI clock pulses (deselected) then CMD0 (reset)
; Returns: r1 = 0 on success, 1 on error
; Frame: 8 bytes (LINK@0, r4@2, r5@4, r6@6)
; r4 = loop counter i
; -----------------------------------------------------------------------
sdInit0:
    adi r7, r7, -8
    lsp r3, r0
    stw r3, r7, 0
    stw r4, r7, 2
    stw r5, r7, 4
    stw r6, r7, 6
    ldi r1, 1               ; spiSel(1) - deselect
    jal spiSel
    ldi r4, 0               ; i = 0
l_i0_loop:
    ldi r3, 12
    tst r4, r3              ; i - 12
    brult l_i0_body
    br l_i0_loop_done
l_i0_body:
    ldi r1, 255             ; spiSend(0xFF)
    jal spiSend
    adi r4, r4, 1           ; i++
    br l_i0_loop
l_i0_loop_done:
    ldi r1, 0               ; spiSel(0) - select
    jal spiSel
    ldi r1, 0               ; sdCmd(0, 0, 0)
    ldi r2, 0
    ldi r3, 0
    jal sdCmd               ; r1 = CMD0 response
    ldi r3, 1
    tst r1, r3              ; r1 - 1
    brz l_i0_ok
    ldi r1, 1               ; return 1 (error)
    br l_i0_ret
l_i0_ok:
    jal pump
    ldi r1, 0               ; return 0
l_i0_ret:
    ldw r3, r7, 0
    ssp r3, r0
    ldw r4, r7, 2
    ldw r5, r7, 4
    ldw r6, r7, 6
    adi r7, r7, 8
    ret

; -----------------------------------------------------------------------
; sdInit1: CMD8 (interface condition) + CMD58 (OCR read)
; Returns: r1 = 0 on success, 2 on CMD8 error, 3 on CMD58 error
; Frame: 8 bytes (LINK@0, r4@2, r5@4, r6@6)
; -----------------------------------------------------------------------
sdInit1:
    adi r7, r7, -8
    lsp r3, r0
    stw r3, r7, 0
    stw r4, r7, 2
    stw r5, r7, 4
    stw r6, r7, 6
    ldi r1, 8               ; sdCmd(8, 0, 0x01AA)
    ldi r2, 0
    ldi r3, 426             ; 0x01AA
    jal sdCmd
    ldi r3, 1
    tst r1, r3              ; r1 - 1
    brz l_i1_cmd8_ok
    ldi r1, 2               ; return 2
    br l_i1_ret
l_i1_cmd8_ok:
    jal pump5
    ldi r1, 58              ; sdCmd(58, 0, 0)
    ldi r2, 0
    ldi r3, 0
    jal sdCmd
    ldi r3, 1
    tst r1, r3              ; r1 - 1
    brz l_i1_cmd58_ok
    ldi r1, 3               ; return 3
    br l_i1_ret
l_i1_cmd58_ok:
    jal pump5
    ldi r1, 0               ; return 0
l_i1_ret:
    ldw r3, r7, 0
    ssp r3, r0
    ldw r4, r7, 2
    ldw r5, r7, 4
    ldw r6, r7, 6
    adi r7, r7, 8
    ret

; -----------------------------------------------------------------------
; sdInit2: CMD55 + ACMD41 (SD card init)
; Returns: r1 = 0 on success, 4 on CMD55 error, 5 on ACMD41 error
; Frame: 8 bytes (LINK@0, r4@2, r5@4, r6@6)
; -----------------------------------------------------------------------
sdInit2:
    adi r7, r7, -8
    lsp r3, r0
    stw r3, r7, 0
    stw r4, r7, 2
    stw r5, r7, 4
    stw r6, r7, 6
    ldi r1, 55              ; sdCmd(55, 0, 0)
    ldi r2, 0
    ldi r3, 0
    jal sdCmd
    ldi r3, 1
    tst r1, r3              ; r1 - 1
    brz l_i2_cmd55_ok
    ldi r1, 4               ; return 4
    br l_i2_ret
l_i2_cmd55_ok:
    jal pump
    ldi r1, 41              ; sdCmd(41, 0x4000, 0)
    ldi r2, 16384           ; 0x4000
    ldi r3, 0
    jal sdCmd
    tst r1, r0              ; r1 - 0
    brz l_i2_ok
    ldi r1, 5               ; return 5
    br l_i2_ret
l_i2_ok:
    jal pump
    ldi r1, 0               ; return 0
l_i2_ret:
    ldw r3, r7, 0
    ssp r3, r0
    ldw r4, r7, 2
    ldw r5, r7, 4
    ldw r6, r7, 6
    adi r7, r7, 8
    ret

; -----------------------------------------------------------------------
; sdInit: full SD card init (CMD0 -> CMD8 -> CMD58 -> ACMD41)
; Returns: r1 = 0 on success, 1-5 on error
; Frame: 2 bytes (LINK@0 only)
; -----------------------------------------------------------------------
sdInit:
    adi r7, r7, -2
    lsp r3, r0
    stw r3, r7, 0
    jal sdInit0
    tst r1, r0
    brnz l_si_ret           ; return r1 if nonzero
    jal sdInit1
    tst r1, r0
    brnz l_si_ret
    jal sdInit2
l_si_ret:
    ldw r3, r7, 0
    ssp r3, r0
    adi r7, r7, 2
    ret

; -----------------------------------------------------------------------
; readSector: read one 512-byte SD sector into memory at dest
; Args: r1=sector (0-based), r2=dest (virtual data address)
; Returns: r1 = 0 on success, nonzero on error
; Sector byte address: argHi = sector>>7, argLo = (sector & 0x7F) << 9
; Frame: 8 bytes (LINK@0, r4@2, r5@4, r6@6)
; r4 = loop counter i (0..511), r5 = dest, r6 = argHi then loop limit
; -----------------------------------------------------------------------
readSector:
    adi r7, r7, -8
    lsp r3, r0
    stw r3, r7, 0
    stw r4, r7, 2
    stw r5, r7, 4
    stw r6, r7, 6
    or r4, r1, r0           ; r4 = sector
    or r5, r2, r0           ; r5 = dest
    ; argHi = sector >> 7: copy sector to r6, shift right 7
    or r6, r4, r0
    srl r6
    srl r6
    srl r6
    srl r6
    srl r6
    srl r6
    srl r6                  ; r6 = sector >> 7 = argHi
    ; argLo = (sector & 0x7F) << 9
    ldi r3, 127             ; 0x7F
    and r1, r4, r3          ; r1 = sector & 0x7F
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1          ; r1 = (sector & 0x7F) << 9 = argLo
    ; sdCmd(17, argHi, argLo)
    or r3, r1, r0           ; r3 = argLo
    ldi r1, 17
    or r2, r6, r0           ; r2 = argHi
    jal sdCmd               ; r1 = CMD17 response
    tst r1, r0
    brnz l_rs_cmd_err
    ; wait for data token 0xFE
    jal waitResp            ; r1 = token
    ldi r4, 254             ; 0xFE
    tst r1, r4
    brz l_rs_tok_ok
    ldi r1, 255             ; return 0xFF (bad token)
    br l_rs_done
l_rs_tok_ok:
    ; read 512 bytes: i counts 0..511, dest in r5
    ldi r4, 0               ; i = 0
    ldi r6, 512             ; loop limit
l_rs_loop:
    tst r4, r6              ; i - 512
    brult l_rs_body
    br l_rs_loop_done
l_rs_body:
    jal spiRecv             ; r1 = byte (r4, r5, r6 preserved: leaf)
    or r3, r1, r0           ; r3 = byte
    add r1, r5, r4          ; r1 = dest + i (addr)
    or r2, r3, r0           ; r2 = byte (val)
    stb r2, r1, 0           ; storeByte inline
    adi r4, r4, 1           ; i++
    br l_rs_loop
l_rs_loop_done:
    jal pump                ; discard CRC byte 1
    jal pump                ; discard CRC byte 2
    ldi r1, 0               ; return 0
    br l_rs_done
l_rs_cmd_err:
    ; r1 already holds the error code from sdCmd
l_rs_done:
    ldw r3, r7, 0
    ssp r3, r0
    ldw r4, r7, 2
    ldw r5, r7, 4
    ldw r6, r7, 6
    adi r7, r7, 8
    ret

; -----------------------------------------------------------------------
; die: print "ERR" + one-byte error code + newline, then halt
; Args: r1 = error code byte
; Never returns (ends with hlt)
; No frame needed; r4 holds code across Putc leaf calls
; ASCII: E=69 R=82 I=73 H=72 M=77 C=67 D=68 LF=10
; -----------------------------------------------------------------------
die:
    or r4, r1, r0           ; r4 = error code
    ldi r1, 69              ; 'E'
    jal Putc
    ldi r1, 82              ; 'R'
    jal Putc
    ldi r1, 82              ; 'R'
    jal Putc
    or r1, r4, r0           ; error code
    jal Putc
    ldi r1, 10              ; '\n'
    jal Putc
    hlt

; -----------------------------------------------------------------------
; Main: bootstrap entry point
; Maps staging page, inits SD, reads header, loads OS code+data pages,
; writes trampoline to frame 2, jumps to trampoline at 0xF000.
;
; Stack frame: 18 bytes
;   r7+0:  codeSectors
;   r7+2:  dataSectors
;   r7+4:  codePages
;   r7+6:  dataPages
;   r7+8:  sector (current sector index)
;   r7+10: saved LINK
;   r7+12: saved r4
;   r7+14: saved r5
;   r7+16: saved r6
;
; Registers during loops:
;   r4 = p (outer loop: page index)
;   r5 = s (inner loop: sector within page)
;   r6 = physFrame (current physical frame)
; -----------------------------------------------------------------------
Main:
    adi r7, r7, -18
    lsp r3, r0
    stw r3, r7, 10
    stw r4, r7, 12
    stw r5, r7, 14
    stw r6, r7, 16

    ; Map data page 1 (SPR 81) -> frame 2 for header staging
    ldi r1, 81
    ldi r2, 2
    jal writeMMU

    ; Initialize SD card
    jal sdInit
    tst r1, r0
    brz l_main_sd_ok
    ldi r1, 73              ; die('I') - init error
    jal die
l_main_sd_ok:

    ; Read boot image header (sector 0) into virtual data 0x1000
    ldi r1, 0
    ldi r2, 0x1000
    jal readSector
    tst r1, r0
    brz l_main_hdr_ok
    ldi r1, 72              ; die('H') - header read error
    jal die
l_main_hdr_ok:

    ; Verify boot image magic (must be 0xDDDD = 56797)
    ldi r1, 0x1000
    jal loadWord
    ldi r3, 56797           ; 0xDDDD
    tst r1, r3
    brz l_main_magic_ok
    ldi r1, 77              ; die('M') - bad magic
    jal die
l_main_magic_ok:

    ; Read codeSectors and dataSectors from header
    ldi r1, 0x1002
    jal loadWord
    stw r1, r7, 0           ; codeSectors

    ldi r1, 0x1004
    jal loadWord
    stw r1, r7, 2           ; dataSectors

    ; codePages = (codeSectors + 7) >> 3
    ldw r1, r7, 0
    adi r1, r1, 7
    srl r1
    srl r1
    srl r1
    stw r1, r7, 4           ; codePages

    ; dataPages = (dataSectors + 7) >> 3
    ldw r1, r7, 2
    adi r1, r1, 7
    srl r1
    srl r1
    srl r1
    stw r1, r7, 6           ; dataPages

    ; ---- Load OS code pages into physical frames 3, 4, ... ----
    ; Data page 2 (SPR 82, virtual 0x2000-0x2FFF) is the staging window.
    ; Code page 0 (SPR 64) is NOT remapped here (loader executes from it).
    ldi r1, 1
    stw r1, r7, 8           ; sector = 1
    ldi r4, 0               ; p = 0

l_main_code_loop:
    ldw r3, r7, 4           ; r3 = codePages
    tst r4, r3              ; p - codePages
    brult l_main_code_body
    br l_main_code_done

l_main_code_body:
    ; physFrame = 3 + p
    ldi r6, 3
    add r6, r6, r4          ; r6 = physFrame

    ; writeMMU(82, physFrame) - remap data page 2 for this code page
    ldi r1, 82
    or r2, r6, r0
    jal writeMMU

    ; if (p != 0): writeMMU(64+p, physFrame) - map code page p
    tst r4, r0              ; p - 0
    brz l_main_skip_code_mmu
    ldi r1, 64
    add r1, r1, r4          ; r1 = 64 + p
    or r2, r6, r0
    jal writeMMU
l_main_skip_code_mmu:

    ; inner loop: s = 0..7, load up to 8 sectors
    ldi r5, 0               ; s = 0

l_main_code_inner:
    ldi r3, 8
    tst r5, r3              ; s - 8
    brult l_main_code_inner_body
    br l_main_code_inner_done

l_main_code_inner_body:
    ; if (sector <= codeSectors): load it
    ldw r1, r7, 8           ; r1 = sector
    ldw r2, r7, 0           ; r2 = codeSectors
    tst r2, r1              ; codeSectors - sector
    brult l_main_code_skip  ; skip when codeSectors < sector

    ; dest = 0x2000 + (s << 9)
    or r1, r5, r0           ; r1 = s
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1          ; r1 = s << 9
    ldi r2, 0x2000
    add r2, r2, r1          ; r2 = dest = 0x2000 + (s<<9)
    ldw r1, r7, 8           ; r1 = sector
    jal readSector
    tst r1, r0
    brz l_main_code_read_ok
    ldi r1, 67              ; die('C') - code read error
    jal die
l_main_code_read_ok:
    ; sector++
    ldw r1, r7, 8
    adi r1, r1, 1
    stw r1, r7, 8

l_main_code_skip:
    adi r5, r5, 1           ; s++
    br l_main_code_inner

l_main_code_inner_done:
    adi r4, r4, 1           ; p++
    br l_main_code_loop

l_main_code_done:

    ; ---- Load OS data pages ----
    ; Data page 3 (SPR 83, virtual 0x3000-0x3FFF) is the staging window.
    ldw r1, r7, 0           ; codeSectors
    adi r1, r1, 1           ; 1 + codeSectors
    stw r1, r7, 8           ; sector = 1 + codeSectors
    ldi r4, 0               ; p = 0

l_main_data_loop:
    ldw r3, r7, 6           ; r3 = dataPages
    tst r4, r3              ; p - dataPages
    brult l_main_data_body
    br l_main_data_done

l_main_data_body:
    ; physFrame = 3 + codePages + p
    ldw r6, r7, 4           ; r6 = codePages
    ldi r3, 3
    add r6, r6, r3          ; r6 = 3 + codePages
    add r6, r6, r4          ; r6 = 3 + codePages + p

    ; writeMMU(80+p, physFrame) - map OS kernel data page p
    ldi r1, 80
    add r1, r1, r4          ; r1 = 80 + p
    or r2, r6, r0
    jal writeMMU

    ; writeMMU(83, physFrame) - remap data page 3 (staging)
    ldi r1, 83
    or r2, r6, r0
    jal writeMMU

    ; inner loop: s = 0..7
    ldi r5, 0               ; s = 0

l_main_data_inner:
    ldi r3, 8
    tst r5, r3              ; s - 8
    brult l_main_data_inner_body
    br l_main_data_inner_done

l_main_data_inner_body:
    ; if (sector <= codeSectors + dataSectors): load it
    ldw r1, r7, 8           ; r1 = sector
    ldw r2, r7, 0           ; r2 = codeSectors
    ldw r3, r7, 2           ; r3 = dataSectors
    add r2, r2, r3          ; r2 = codeSectors + dataSectors
    tst r2, r1              ; total - sector
    brult l_main_data_skip  ; skip when total < sector

    ; dest = 0x3000 + (s << 9)
    or r1, r5, r0           ; r1 = s
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1
    add r1, r1, r1          ; r1 = s << 9
    ldi r2, 0x3000
    add r2, r2, r1          ; r2 = dest = 0x3000 + (s<<9)
    ldw r1, r7, 8           ; r1 = sector
    jal readSector
    tst r1, r0
    brz l_main_data_read_ok
    ldi r1, 68              ; die('D') - data read error
    jal die
l_main_data_read_ok:
    ; sector++
    ldw r1, r7, 8
    adi r1, r1, 1
    stw r1, r7, 8

l_main_data_skip:
    adi r5, r5, 1           ; s++
    br l_main_data_inner

l_main_data_inner_done:
    adi r4, r4, 1           ; p++
    br l_main_data_loop

l_main_data_done:

    ; ---- Write trampoline to physical frame 2 ----
    ; Map data page 14 (SPR 94) -> frame 2.
    ; Data pages 1 (SPR 81, from header staging) and 14 both map to frame 2;
    ; writing to 0xE000 overwrites the now-unneeded header at frame 2's start.
    ldi r1, 94
    ldi r2, 2
    jal writeMMU

    ; Write trampoline words to virtual data 0xE000..0xE00E
    ; These instructions execute from virtual CODE 0xF000 (code page 15 -> frame 2):
    ;   ldi r1, 3     0xA001, 0x80C9  (OS first code frame)
    ;   ldi r2, 64    0xA00A, 0x8012  (SPR 64 = kernel code page 0)
    ;   ssp r1, r2    0xFE91          (remap code page 0 -> frame 3)
    ;   ldi r1, 0     0xA001, 0x8009  (jump target = VA 0)
    ;   ji r1         0xFFF1          (jump to OS)
    ldi r1, 0xE000
    ldi r2, 0xA001
    jal storeWord
    ldi r1, 0xE002
    ldi r2, 0x80C9
    jal storeWord
    ldi r1, 0xE004
    ldi r2, 0xA00A
    jal storeWord
    ldi r1, 0xE006
    ldi r2, 0x8012
    jal storeWord
    ldi r1, 0xE008
    ldi r2, 0xFE91
    jal storeWord
    ldi r1, 0xE00A
    ldi r2, 0xA001
    jal storeWord
    ldi r1, 0xE00C
    ldi r2, 0x8009
    jal storeWord
    ldi r1, 0xE00E
    ldi r2, 0xFFF1
    jal storeWord

    ; Map code page 15 (SPR 79) -> frame 2 to make trampoline executable
    ldi r1, 79
    ldi r2, 2
    jal writeMMU

    ; Jump to trampoline at virtual code 0xF000
    ldi r1, 0xF000
    ji r1
