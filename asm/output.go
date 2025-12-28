package main

import (
	"os"
)

/* Write output file with header */
func (a *Assembler) writeOutput(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	/* Write header */
	/* Magic number: 0xDDD1 */
	err = writeU16(file, 0xDDD1)
	if err != nil {
		return err
	}

	/* Code size */
	err = writeU16(file, uint16(len(a.codeBytes)))
	if err != nil {
		return err
	}

	/* Data size */
	err = writeU16(file, uint16(len(a.dataBytes)))
	if err != nil {
		return err
	}

	/* Reserved: 5 words = 10 bytes */
	for i := 0; i < 5; i++ {
		err = writeU16(file, 0)
		if err != nil {
			return err
		}
	}

	/* Write code segment */
	_, err = file.Write(a.codeBytes)
	if err != nil {
		return err
	}

	/* Write data segment */
	_, err = file.Write(a.dataBytes)
	if err != nil {
		return err
	}

	return nil
}

/* Write a 16-bit value in little endian */
func writeU16(file *os.File, val uint16) error {
	buf := make([]byte, 2)
	buf[0] = byte(val & 0xFF)
	buf[1] = byte((val >> 8) & 0xFF)
	_, err := file.Write(buf)
	return err
}

/* Read a 16-bit value in little endian */
func readU16(data []byte, offset int) uint16 {
	if offset+1 >= len(data) {
		return 0
	}
	return uint16(data[offset]) | (uint16(data[offset+1]) << 8)
}
