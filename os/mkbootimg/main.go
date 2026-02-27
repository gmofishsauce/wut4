// mkbootimg - convert a WUT-4 executable to a bootable SD card image.
//
// Usage: mkbootimg input.out output.img
//
// Reads a WUT-4 executable (magic 0xDDD1) and writes a boot image suitable
// for loading by the WUT-4 bootstrap loader via the emulated SD card.
//
// Boot image layout (all fields little-endian):
//
//   Sector 0 (512 bytes) - header:
//     offset 0: uint16  magic = 0xDDDD
//     offset 2: uint16  code_sectors  (number of 512-byte code sectors)
//     offset 4: uint16  data_sectors  (number of 512-byte data sectors)
//     offset 6: [506 zero bytes]
//
//   Sectors 1 .. code_sectors:       code segment, zero-padded to sector boundary
//   Sectors 1+code_sectors .. end:   data segment, zero-padded to sector boundary
//
// The bootstrap loader reads sector 0 first to learn the sector counts, then
// reads each subsequent sector directly into the mapped virtual pages.

package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	magicExe  = 0xDDD1 // WUT-4 executable magic
	magicBoot = 0xDDDD // boot image magic
	exeHeader = 16     // WUT-4 executable header size in bytes
	sectorSize = 512   // SD card sector size in bytes
)

// BootImage converts the bytes of a WUT-4 executable into a boot image.
// Returns an error if the input is not a valid WUT-4 executable.
func BootImage(exe []byte) ([]byte, error) {
	if len(exe) < exeHeader {
		return nil, fmt.Errorf("file too small (%d bytes, need at least %d)", len(exe), exeHeader)
	}

	magic := binary.LittleEndian.Uint16(exe[0:2])
	if magic != magicExe {
		return nil, fmt.Errorf("bad magic 0x%04X (expected 0x%04X)", magic, magicExe)
	}

	codeSize := int(binary.LittleEndian.Uint16(exe[2:4]))
	dataSize := int(binary.LittleEndian.Uint16(exe[4:6]))

	need := exeHeader + codeSize + dataSize
	if len(exe) < need {
		return nil, fmt.Errorf("file too short: header declares %d+%d bytes but file is only %d bytes",
			codeSize, dataSize, len(exe)-exeHeader)
	}

	code := exe[exeHeader : exeHeader+codeSize]
	data := exe[exeHeader+codeSize : exeHeader+codeSize+dataSize]

	codeSectors := ceilSectors(codeSize)
	dataSectors := ceilSectors(dataSize)

	// Allocate output: 1 header sector + code sectors + data sectors.
	// make() zero-initialises, so padding bytes are automatically zero.
	out := make([]byte, (1+codeSectors+dataSectors)*sectorSize)

	// Sector 0: boot image header.
	binary.LittleEndian.PutUint16(out[0:2], magicBoot)
	binary.LittleEndian.PutUint16(out[2:4], uint16(codeSectors))
	binary.LittleEndian.PutUint16(out[4:6], uint16(dataSectors))

	// Code segment starting at sector 1.
	copy(out[sectorSize:], code)

	// Data segment immediately after code sectors.
	copy(out[sectorSize*(1+codeSectors):], data)

	return out, nil
}

// ceilSectors returns the number of 512-byte sectors needed to hold n bytes.
func ceilSectors(n int) int {
	if n == 0 {
		return 0
	}
	return (n + sectorSize - 1) / sectorSize
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: mkbootimg input.out output.img\n")
		os.Exit(1)
	}

	exe, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkbootimg: %v\n", err)
		os.Exit(1)
	}

	img, err := BootImage(exe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkbootimg: %v\n", err)
		os.Exit(1)
	}

	codeSize := int(binary.LittleEndian.Uint16(exe[2:4]))
	dataSize := int(binary.LittleEndian.Uint16(exe[4:6]))
	codeSectors := ceilSectors(codeSize)
	dataSectors := ceilSectors(dataSize)
	totalSectors := 1 + codeSectors + dataSectors

	if err := os.WriteFile(os.Args[2], img, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "mkbootimg: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("mkbootimg: code %d bytes (%d sectors), data %d bytes (%d sectors)\n",
		codeSize, codeSectors, dataSize, dataSectors)
	fmt.Printf("mkbootimg: wrote %d sectors (%d bytes) to %s\n",
		totalSectors, totalSectors*sectorSize, os.Args[2])
}
