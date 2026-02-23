package main

import (
	"encoding/binary"
	"fmt"
)

type Linker struct {
	objects    []*ObjectFile
	globalSyms map[string]*ResolvedSym
	verbose    bool
}

func newLinker(verbose bool) *Linker {
	return &Linker{
		globalSyms: make(map[string]*ResolvedSym),
		verbose:    verbose,
	}
}

func (ld *Linker) addObject(obj *ObjectFile) {
	ld.objects = append(ld.objects, obj)
}

/* Phase 1: Symbol resolution.
   Collect all globally-visible defined symbols.
   Verify all undefined (external) references can be satisfied. */
func (ld *Linker) resolveSymbols() error {
	/* Pass A: collect defined globals */
	for objIdx, obj := range ld.objects {
		for _, sym := range obj.symbols {
			if sym.Section == SEC_UNDEF {
				continue /* undefined reference, handled below */
			}
			if sym.Visibility != VIS_GLOBAL {
				continue /* local symbol, not exported */
			}
			if _, exists := ld.globalSyms[sym.Name]; exists {
				return fmt.Errorf("symbol %q defined in multiple object files", sym.Name)
			}
			ld.globalSyms[sym.Name] = &ResolvedSym{
				name:     sym.Name,
				value:    sym.Value,
				section:  sym.Section,
				objIndex: objIdx,
			}
			if ld.verbose {
				secName := "code"
				if sym.Section == SEC_DATA_WOF {
					secName = "data"
				}
				fmt.Printf("  global %s: %s[0x%04X] from %s\n", sym.Name, secName, sym.Value, obj.path)
			}
		}
	}

	/* Pass B: verify all undefined references */
	for _, obj := range ld.objects {
		for _, sym := range obj.symbols {
			if sym.Section != SEC_UNDEF {
				continue
			}
			if _, ok := ld.globalSyms[sym.Name]; !ok {
				return fmt.Errorf("undefined symbol %q (referenced in %s)", sym.Name, obj.path)
			}
		}
	}

	return nil
}

/* Phase 2: Compute code and data offsets for each object file.
   All code sections are placed first, then all data sections. */
func (ld *Linker) layout() {
	var codeOff, dataOff uint16
	for _, obj := range ld.objects {
		obj.codeOffset = codeOff
		codeOff += obj.header.CodeSize
		/* Align to 2-byte boundary */
		if codeOff%2 != 0 {
			codeOff++
		}

		obj.dataOffset = dataOff
		dataOff += obj.header.DataSize
		if dataOff%2 != 0 {
			dataOff++
		}

		if ld.verbose {
			fmt.Printf("  %s: code@0x%04X data@0x%04X\n", obj.path, obj.codeOffset, obj.dataOffset)
		}
	}
}

/* Phase 3: Copy sections into merged buffers and apply relocations. */
func (ld *Linker) relocate() ([]byte, []byte, error) {
	/* Compute total sizes */
	var totalCode, totalData int
	for _, obj := range ld.objects {
		totalCode += int(obj.header.CodeSize)
		if totalCode%2 != 0 {
			totalCode++
		}
		totalData += int(obj.header.DataSize)
		if totalData%2 != 0 {
			totalData++
		}
	}

	mergedCode := make([]byte, totalCode)
	mergedData := make([]byte, totalData)

	/* Copy sections */
	for _, obj := range ld.objects {
		copy(mergedCode[obj.codeOffset:], obj.code)
		if obj.header.DataSize > 0 {
			copy(mergedData[obj.dataOffset:], obj.data)
		}
	}

	/* Apply relocations */
	for objIdx, obj := range ld.objects {
		for _, r := range obj.relocations {
			if int(r.SymIndex) >= len(obj.symbols) {
				return nil, nil, fmt.Errorf("%s: relocation sym_index %d out of range", obj.path, r.SymIndex)
			}
			localSym := obj.symbols[r.SymIndex]

			/* Resolve the symbol */
			var resolved *ResolvedSym
			if localSym.Section == SEC_UNDEF {
				/* External reference: look up in global table */
				var ok bool
				resolved, ok = ld.globalSyms[localSym.Name]
				if !ok {
					return nil, nil, fmt.Errorf("relocation in %s references unresolved symbol %q", obj.path, localSym.Name)
				}
			} else {
				/* Intra-file symbol: build a ResolvedSym on the fly */
				resolved = &ResolvedSym{
					name:     localSym.Name,
					value:    localSym.Value,
					section:  localSym.Section,
					objIndex: objIdx,
				}
			}

			/* Compute final address of the symbol */
			var finalAddr uint16
			if resolved.section == SEC_CODE_WOF {
				finalAddr = ld.objects[resolved.objIndex].codeOffset + resolved.value
			} else {
				finalAddr = ld.objects[resolved.objIndex].dataOffset + resolved.value
			}

			/* Determine which merged buffer to patch and the absolute patch offset */
			var buf []byte
			var patchBase uint16
			if r.Section == 0 {
				buf = mergedCode
				patchBase = obj.codeOffset
			} else {
				buf = mergedData
				patchBase = obj.dataOffset
			}
			patchOffset := int(patchBase) + int(r.Offset)

			if ld.verbose {
				secStr := "code"
				if r.Section != 0 {
					secStr = "data"
				}
				fmt.Printf("  reloc %s+0x%04X type=0x%02X sym=%q final=0x%04X\n",
					secStr, r.Offset, r.Type, localSym.Name, finalAddr)
			}

			switch r.Type {
			case R_LDI_CODE, R_LDI_DATA:
				/* Patch 2-word LUI+ADI sequence at patchOffset */
				if err := patchLUIPlusADI(buf, patchOffset, finalAddr); err != nil {
					return nil, nil, fmt.Errorf("%s: R_LDI patch at 0x%04X: %v", obj.path, r.Offset, err)
				}

			case R_JAL:
				/* Patch 2-word LUI+JAL sequence at patchOffset */
				if err := patchLUIPlusJAL(buf, patchOffset, finalAddr); err != nil {
					return nil, nil, fmt.Errorf("%s: R_JAL patch at 0x%04X: %v", obj.path, r.Offset, err)
				}

			case R_WORD16_CODE, R_WORD16_DATA:
				/* Patch a bare 16-bit word */
				if patchOffset+2 > len(buf) {
					return nil, nil, fmt.Errorf("%s: R_WORD16 patch at 0x%04X out of bounds", obj.path, r.Offset)
				}
				binary.LittleEndian.PutUint16(buf[patchOffset:], finalAddr)

			default:
				return nil, nil, fmt.Errorf("%s: unknown relocation type 0x%02X", obj.path, r.Type)
			}
		}
	}

	return mergedCode, mergedData, nil
}

/* patchLUIPlusADI patches a 2-word LUI+ADI sequence with finalAddr.
   word1 = 0xA000 | (upper10 << 3) | rT
   word2 = 0x8000 | (lower6  << 6) | (rT << 3) | rT
   We preserve rT from word1. */
func patchLUIPlusADI(buf []byte, offset int, addr uint16) error {
	if offset+4 > len(buf) {
		return fmt.Errorf("offset %d+4 out of bounds (len=%d)", offset, len(buf))
	}
	word1 := binary.LittleEndian.Uint16(buf[offset:])
	rT := word1 & 0x7
	upper := (addr >> 6) & 0x3FF
	lower := addr & 0x3F
	newWord1 := uint16(0xA000) | uint16(upper<<3) | rT
	newWord2 := uint16(0x8000) | uint16(lower<<6) | uint16(rT<<3) | rT
	binary.LittleEndian.PutUint16(buf[offset:], newWord1)
	binary.LittleEndian.PutUint16(buf[offset+2:], newWord2)
	return nil
}

/* patchLUIPlusJAL patches a 2-word LUI+JAL sequence with finalAddr.
   word1 = 0xA000 | (upper10 << 3) | rT
   word2 = 0xE000 | (lower6  << 6) | (rS << 3) | rT
   We preserve rT from word1 and rS from word2. */
func patchLUIPlusJAL(buf []byte, offset int, addr uint16) error {
	if offset+4 > len(buf) {
		return fmt.Errorf("offset %d+4 out of bounds (len=%d)", offset, len(buf))
	}
	word1 := binary.LittleEndian.Uint16(buf[offset:])
	word2 := binary.LittleEndian.Uint16(buf[offset+2:])
	rT := word1 & 0x7
	rS := (word2 >> 3) & 0x7
	upper := (addr >> 6) & 0x3FF
	lower := addr & 0x3F
	newWord1 := uint16(0xA000) | uint16(upper<<3) | rT
	newWord2 := uint16(0xE000) | uint16(lower<<6) | uint16(rS<<3) | rT
	binary.LittleEndian.PutUint16(buf[offset:], newWord1)
	binary.LittleEndian.PutUint16(buf[offset+2:], newWord2)
	return nil
}

/* link performs all four phases and returns the merged code and data buffers */
func (ld *Linker) link() ([]byte, []byte, error) {
	if ld.verbose {
		fmt.Println("Phase 1: Symbol resolution")
	}
	if err := ld.resolveSymbols(); err != nil {
		return nil, nil, err
	}

	if ld.verbose {
		fmt.Println("Phase 2: Layout")
	}
	ld.layout()

	if ld.verbose {
		fmt.Println("Phase 3: Relocation")
	}
	return ld.relocate()
}
