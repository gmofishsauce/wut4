package main

import (
	"fmt"
	"os"
)

func writeOutput(filename string, codeBuf, dataBuf []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	/* Write header */
	header := make([]byte, HEADER_SIZE)

	/* Magic number 0xDDD1 (little endian) */
	header[0] = byte(MAGIC_NUMBER & 0xFF)
	header[1] = byte((MAGIC_NUMBER >> 8) & 0xFF)

	/* Code size (little endian) */
	codeSize := len(codeBuf)
	header[2] = byte(codeSize & 0xFF)
	header[3] = byte((codeSize >> 8) & 0xFF)

	/* Data size (little endian) */
	dataSize := len(dataBuf)
	header[4] = byte(dataSize & 0xFF)
	header[5] = byte((dataSize >> 8) & 0xFF)

	/* Reserved bytes (6-15) are already zero */

	if _, err := file.Write(header); err != nil {
		return err
	}

	/* Write code segment */
	if codeSize > 0 {
		if _, err := file.Write(codeBuf); err != nil {
			return err
		}
	}

	/* Write data segment */
	if dataSize > 0 {
		if _, err := file.Write(dataBuf); err != nil {
			return err
		}
	}

	return nil
}

/* writeObjectFile writes a WOF (WUT-4 Object Format) relocatable object file.
   Format:
     Header         16 bytes
     Code section   code_size bytes
     Data section   data_size bytes
     Symbol table   sym_count × 8 bytes
     Reloc table    reloc_count × 8 bytes
     String table   string_table_size bytes (null-terminated names)
*/
func writeObjectFile(filename string, asm *Assembler) error {
	/* --- Build string table --- */
	/* First byte is always \0 (null string for index 0) */
	strtab := []byte{0}
	symNameToOffset := make(map[string]uint16)

	/* Collect symbols to emit: defined globals (uppercase) + undefined externals */
	type wofSym struct {
		nameOff    uint16
		value      uint16
		section    uint8
		visibility uint8
	}
	var wofSyms []wofSym

	for i := 0; i < asm.numSymbols; i++ {
		sym := &asm.symbols[i]
		isGlobal := len(sym.name) > 0 && sym.name[0] >= 'A' && sym.name[0] <= 'Z'
		isUndefined := !sym.defined

		if !isGlobal && !isUndefined {
			continue /* skip local defined symbols */
		}

		/* Add name to string table if not already there */
		off, ok := symNameToOffset[sym.name]
		if !ok {
			off = uint16(len(strtab))
			symNameToOffset[sym.name] = off
			strtab = append(strtab, []byte(sym.name)...)
			strtab = append(strtab, 0)
		}

		var sec uint8
		var vis uint8
		var val uint16

		if isUndefined {
			sec = SEC_UNDEF
			val = 0
		} else {
			if sym.segment == SEG_CODE {
				sec = SEC_CODE_WOF
			} else {
				sec = SEC_DATA_WOF
			}
			val = uint16(sym.value)
		}

		if isGlobal {
			vis = VIS_GLOBAL
		} else {
			vis = VIS_LOCAL
		}

		wofSyms = append(wofSyms, wofSym{
			nameOff:    off,
			value:      val,
			section:    sec,
			visibility: vis,
		})
	}

	/* Build sym name → index map for relocation table */
	symIndex := make(map[string]uint16)
	for idx, ws := range wofSyms {
		/* Find the original name via name offset */
		off := int(ws.nameOff)
		end := off
		for end < len(strtab) && strtab[end] != 0 {
			end++
		}
		name := string(strtab[off:end])
		symIndex[name] = uint16(idx)
	}

	/* Validate all relocations reference known symbols */
	for _, r := range asm.relocations {
		if _, ok := symIndex[r.symName]; !ok {
			return fmt.Errorf("relocation references unknown symbol: %s", r.symName)
		}
	}

	/* --- Compute sizes --- */
	codeSize := asm.codeSize
	dataSize := asm.dataSize
	symCount := uint16(len(wofSyms))
	relocCount := uint16(len(asm.relocations))
	strtabSize := uint16(len(strtab))

	/* --- Write file --- */
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	/* Header (16 bytes) */
	header := make([]byte, HEADER_SIZE)
	header[0] = byte(MAGIC_WOF & 0xFF)
	header[1] = byte((MAGIC_WOF >> 8) & 0xFF)
	header[2] = 1    /* version */
	if asm.bootstrapMode {
		header[3] = 1 /* flags: bit 0 = bootstrap */
	}
	header[4] = byte(codeSize & 0xFF)
	header[5] = byte((codeSize >> 8) & 0xFF)
	header[6] = byte(dataSize & 0xFF)
	header[7] = byte((dataSize >> 8) & 0xFF)
	header[8] = byte(symCount & 0xFF)
	header[9] = byte((symCount >> 8) & 0xFF)
	header[10] = byte(relocCount & 0xFF)
	header[11] = byte((relocCount >> 8) & 0xFF)
	header[12] = byte(strtabSize & 0xFF)
	header[13] = byte((strtabSize >> 8) & 0xFF)
	/* bytes 14-15: reserved (zero) */

	if _, err := file.Write(header); err != nil {
		return err
	}

	/* Code section */
	if codeSize > 0 {
		if _, err := file.Write(asm.codeBuf[:codeSize]); err != nil {
			return err
		}
	}

	/* Data section */
	if dataSize > 0 {
		if _, err := file.Write(asm.dataBuf[:dataSize]); err != nil {
			return err
		}
	}

	/* Symbol table: 8 bytes per entry */
	symEntry := make([]byte, 8)
	for _, ws := range wofSyms {
		symEntry[0] = byte(ws.nameOff & 0xFF)
		symEntry[1] = byte((ws.nameOff >> 8) & 0xFF)
		symEntry[2] = byte(ws.value & 0xFF)
		symEntry[3] = byte((ws.value >> 8) & 0xFF)
		symEntry[4] = ws.section
		symEntry[5] = ws.visibility
		symEntry[6] = 0
		symEntry[7] = 0
		if _, err := file.Write(symEntry); err != nil {
			return err
		}
	}

	/* Relocation table: 8 bytes per entry */
	relocEntry := make([]byte, 8)
	for _, r := range asm.relocations {
		sec := uint8(0) /* 0 = code section */
		if r.inDataSeg {
			sec = 1
		}
		idx := symIndex[r.symName]
		relocEntry[0] = sec
		relocEntry[1] = r.rtype
		relocEntry[2] = byte(r.offset & 0xFF)
		relocEntry[3] = byte((r.offset >> 8) & 0xFF)
		relocEntry[4] = byte(idx & 0xFF)
		relocEntry[5] = byte((idx >> 8) & 0xFF)
		relocEntry[6] = 0
		relocEntry[7] = 0
		if _, err := file.Write(relocEntry); err != nil {
			return err
		}
	}

	/* String table */
	if _, err := file.Write(strtab); err != nil {
		return err
	}

	return nil
}
