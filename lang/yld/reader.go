package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

func readObjectFile(path string) (*ObjectFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %v", path, err)
	}

	if len(data) < 16 {
		return nil, fmt.Errorf("%s: file too short for WOF header", path)
	}

	obj := &ObjectFile{path: path}

	/* Parse header */
	obj.header.Magic = binary.LittleEndian.Uint16(data[0:2])
	if obj.header.Magic != MAGIC_WOF {
		return nil, fmt.Errorf("%s: bad magic 0x%04X (expected 0x%04X)", path, obj.header.Magic, MAGIC_WOF)
	}
	obj.header.Version = data[2]
	obj.header.Flags = data[3]
	obj.header.CodeSize = binary.LittleEndian.Uint16(data[4:6])
	obj.header.DataSize = binary.LittleEndian.Uint16(data[6:8])
	obj.header.SymCount = binary.LittleEndian.Uint16(data[8:10])
	obj.header.RelocCount = binary.LittleEndian.Uint16(data[10:12])
	obj.header.StringTableSize = binary.LittleEndian.Uint16(data[12:14])
	obj.header.Reserved = binary.LittleEndian.Uint16(data[14:16])

	/* Compute section offsets within file */
	codeStart := 16
	dataStart := codeStart + int(obj.header.CodeSize)
	symStart := dataStart + int(obj.header.DataSize)
	relocStart := symStart + int(obj.header.SymCount)*8
	strtabStart := relocStart + int(obj.header.RelocCount)*8
	strtabEnd := strtabStart + int(obj.header.StringTableSize)

	if strtabEnd > len(data) {
		return nil, fmt.Errorf("%s: file truncated (need %d bytes, have %d)", path, strtabEnd, len(data))
	}

	/* Code section */
	obj.code = make([]byte, obj.header.CodeSize)
	copy(obj.code, data[codeStart:dataStart])

	/* Data section */
	obj.data = make([]byte, obj.header.DataSize)
	copy(obj.data, data[dataStart:symStart])

	/* String table */
	strtab := data[strtabStart:strtabEnd]

	/* Symbol table */
	obj.symbols = make([]WOFSymbol, obj.header.SymCount)
	for i := range obj.symbols {
		base := symStart + i*8
		obj.symbols[i].NameOffset = binary.LittleEndian.Uint16(data[base : base+2])
		obj.symbols[i].Value = binary.LittleEndian.Uint16(data[base+2 : base+4])
		obj.symbols[i].Section = data[base+4]
		obj.symbols[i].Visibility = data[base+5]
		obj.symbols[i].Reserved = binary.LittleEndian.Uint16(data[base+6 : base+8])
		/* Decode name from string table */
		off := int(obj.symbols[i].NameOffset)
		if off >= len(strtab) {
			return nil, fmt.Errorf("%s: symbol %d name offset %d out of range", path, i, off)
		}
		end := off
		for end < len(strtab) && strtab[end] != 0 {
			end++
		}
		obj.symbols[i].Name = string(strtab[off:end])
	}

	/* Relocation table */
	obj.relocations = make([]WOFReloc, obj.header.RelocCount)
	for i := range obj.relocations {
		base := relocStart + i*8
		obj.relocations[i].Section = data[base]
		obj.relocations[i].Type = data[base+1]
		obj.relocations[i].Offset = binary.LittleEndian.Uint16(data[base+2 : base+4])
		obj.relocations[i].SymIndex = binary.LittleEndian.Uint16(data[base+4 : base+6])
		obj.relocations[i].Reserved = binary.LittleEndian.Uint16(data[base+6 : base+8])
	}

	return obj, nil
}
