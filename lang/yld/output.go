package main

import (
	"os"
)

func writeExecutable(filename string, codeBuf, dataBuf []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	/* Write header */
	header := make([]byte, HEADER_SIZE)

	/* Magic number 0xDDD1 (little endian) */
	header[0] = byte(MAGIC_EXE & 0xFF)
	header[1] = byte((MAGIC_EXE >> 8) & 0xFF)

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

	if codeSize > 0 {
		if _, err := file.Write(codeBuf); err != nil {
			return err
		}
	}

	if dataSize > 0 {
		if _, err := file.Write(dataBuf); err != nil {
			return err
		}
	}

	return nil
}
