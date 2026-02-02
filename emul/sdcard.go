// Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.

package main

import (
	"fmt"
	"os"
)

// SD card state machine states
type SDState int

const (
	SD_UNINITIALIZED SDState = iota // Before any init bytes seen
	SD_INIT_CLOCKS                  // Counting 0xFF bytes (need 10+)
	SD_IDLE                         // After CMD0, waiting for init sequence
	SD_READY                        // After successful CMD55+ACMD41 loop
	SD_ERROR                        // Unrecoverable error (wrong init sequence)
	SD_RECEIVING_CMD                // Receiving 6-byte command
	SD_SENDING_RESP                 // Sending response bytes
	SD_READING_DATA                 // Sending 512-byte block + CRC
	SD_WRITING_DATA                 // Receiving 512-byte block + CRC
)

// Data transfer phases
type DataPhase int

const (
	DATA_PHASE_TOKEN DataPhase = iota // Waiting for/sending start token
	DATA_PHASE_DATA                   // Transferring data bytes
	DATA_PHASE_CRC1                   // First CRC byte
	DATA_PHASE_CRC2                   // Second CRC byte
)

// SDCard represents an emulated SPI-connected SD card
type SDCard struct {
	file     *os.File // Host file backing the SD card
	fileSize int64    // Size in bytes

	// State machine
	state     SDState // Current state
	initCount int     // Count of init 0xFF bytes
	selected  bool    // CS line state (true = selected, active low inverted)

	// Command buffer
	cmdBuf   [6]byte // 6-byte command buffer
	cmdIndex int     // Bytes received into cmdBuf

	// Response buffer
	respBuf   []byte // Response bytes to send
	respIndex int    // Next byte to send

	// Data transfer
	dataBuf   [514]byte // Data block buffer (512 data + 2 CRC)
	dataIndex int       // Current position in data buffer
	dataPhase DataPhase // Start token, data, CRC phases
	dataAddr  uint32    // Block address for current transfer

	// Init tracking
	sawCMD0  bool
	sawCMD8  bool
	sawCMD58 bool
	sawCMD55 bool // For ACMD detection
	isReady  bool // True after successful init sequence (ACMD41 returned 0x00)

	// Tracer for debug output
	tracer *Tracer
}

// NewSDCard creates a new SD card emulator backed by the given file
func NewSDCard(file *os.File, tracer *Tracer) (*SDCard, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("cannot stat SD card file: %v", err)
	}

	size := info.Size()

	// Validate size: must be 512B to 2GB and multiple of 512
	if size < 512 {
		return nil, fmt.Errorf("SD card file too small: %d bytes (minimum 512)", size)
	}
	if size > 2*1024*1024*1024 {
		return nil, fmt.Errorf("SD card file too large: %d bytes (maximum 2GB)", size)
	}
	if size%512 != 0 {
		return nil, fmt.Errorf("SD card file size not multiple of 512: %d bytes", size)
	}

	return &SDCard{
		file:     file,
		fileSize: size,
		state:    SD_UNINITIALIZED,
		tracer:   tracer,
	}, nil
}

// SetSelect sets the chip select state (active low: 0 = selected, 1 = deselected)
func (sd *SDCard) SetSelect(value byte) {
	wasSelected := sd.selected
	sd.selected = (value & 0x01) == 0

	if sd.tracer != nil {
		if sd.selected && !wasSelected {
			sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] Selected\n")))
		} else if !sd.selected && wasSelected {
			sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] Deselected\n")))
		}
	}

	// When deselected, reset command reception state
	if !sd.selected {
		sd.cmdIndex = 0
	}
}

// Transfer performs an SPI byte transfer (simultaneous read/write)
func (sd *SDCard) Transfer(txByte byte) byte {
	// If not selected, return 0xFF and ignore input
	if !sd.selected {
		// Count init clocks when deselected
		if sd.state == SD_UNINITIALIZED && txByte == 0xFF {
			sd.initCount++
			if sd.initCount >= 10 {
				sd.state = SD_INIT_CLOCKS
				if sd.tracer != nil {
					sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] Init clocks complete (%d)\n", sd.initCount)))
				}
			}
		}
		return 0xFF
	}

	// In error state, always return 0xFF
	if sd.state == SD_ERROR {
		return 0xFF
	}

	// Handle state machine
	switch sd.state {
	case SD_UNINITIALIZED:
		// Card selected before init clocks - error
		sd.state = SD_ERROR
		if sd.tracer != nil {
			sd.tracer.out.Write([]byte("[SD] ERROR: Selected before init clocks\n"))
		}
		return 0xFF

	case SD_INIT_CLOCKS, SD_IDLE, SD_READY:
		return sd.handleCommand(txByte)

	case SD_RECEIVING_CMD:
		return sd.handleCommand(txByte)

	case SD_SENDING_RESP:
		return sd.sendResponse(txByte)

	case SD_READING_DATA:
		return sd.sendDataBlock()

	case SD_WRITING_DATA:
		return sd.receiveDataBlock(txByte)
	}

	return 0xFF
}

// handleCommand accumulates command bytes and executes when complete
func (sd *SDCard) handleCommand(txByte byte) byte {
	// If sending a response, continue that first
	if sd.state == SD_SENDING_RESP {
		return sd.sendResponse(txByte)
	}

	// Look for command start (bit 7=0, bit 6=1)
	if sd.cmdIndex == 0 {
		if (txByte & 0xC0) != 0x40 {
			// Not a command byte, return 0xFF
			return 0xFF
		}
	}

	sd.cmdBuf[sd.cmdIndex] = txByte
	sd.cmdIndex++

	if sd.cmdIndex < 6 {
		// Still accumulating command
		return 0xFF
	}

	// Command complete - execute it
	sd.cmdIndex = 0
	return sd.executeCommand()
}

// executeCommand processes a complete 6-byte command
func (sd *SDCard) executeCommand() byte {
	cmdIndex := sd.cmdBuf[0] & 0x3F
	arg := uint32(sd.cmdBuf[1])<<24 | uint32(sd.cmdBuf[2])<<16 | uint32(sd.cmdBuf[3])<<8 | uint32(sd.cmdBuf[4])

	if sd.tracer != nil {
		sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] CMD%d arg=0x%08X\n", cmdIndex, arg)))
	}

	// Check if this is an application command (preceded by CMD55)
	isACMD := sd.sawCMD55
	sd.sawCMD55 = false

	if isACMD {
		return sd.executeACMD(cmdIndex, arg)
	}

	switch cmdIndex {
	case 0: // CMD0 - GO_IDLE_STATE
		return sd.cmdGoIdle()

	case 8: // CMD8 - SEND_IF_COND
		return sd.cmdSendIfCond(arg)

	case 17: // CMD17 - READ_SINGLE_BLOCK
		return sd.cmdReadSingleBlock(arg)

	case 24: // CMD24 - WRITE_SINGLE_BLOCK
		return sd.cmdWriteSingleBlock(arg)

	case 55: // CMD55 - APP_CMD
		return sd.cmdAppCmd()

	case 58: // CMD58 - READ_OCR
		return sd.cmdReadOCR()

	default:
		if sd.tracer != nil {
			sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] Unknown CMD%d\n", cmdIndex)))
		}
		// Unknown command - return illegal command error
		sd.respBuf = []byte{0x04}
		sd.respIndex = 0
		sd.state = SD_SENDING_RESP
		return 0xFF
	}
}

// executeACMD handles application-specific commands (after CMD55)
func (sd *SDCard) executeACMD(cmdIndex byte, arg uint32) byte {
	if sd.tracer != nil {
		sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] ACMD%d arg=0x%08X\n", cmdIndex, arg)))
	}

	switch cmdIndex {
	case 41: // ACMD41 - SD_SEND_OP_COND
		return sd.acmdSendOpCond(arg)

	default:
		if sd.tracer != nil {
			sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] Unknown ACMD%d\n", cmdIndex)))
		}
		// Unknown ACMD - return illegal command error
		sd.respBuf = []byte{0x04}
		sd.respIndex = 0
		sd.state = SD_SENDING_RESP
		return 0xFF
	}
}

// cmdGoIdle handles CMD0 - reset to idle state
func (sd *SDCard) cmdGoIdle() byte {
	if sd.state != SD_INIT_CLOCKS && sd.state != SD_IDLE && sd.state != SD_READY {
		sd.state = SD_ERROR
		if sd.tracer != nil {
			sd.tracer.out.Write([]byte("[SD] ERROR: CMD0 in wrong state\n"))
		}
		return 0xFF
	}

	sd.sawCMD0 = true
	sd.sawCMD8 = false
	sd.sawCMD58 = false
	sd.state = SD_IDLE

	// R1 response: 0x01 = in idle state
	sd.respBuf = []byte{0x01}
	sd.respIndex = 0
	sd.state = SD_SENDING_RESP

	if sd.tracer != nil {
		sd.tracer.out.Write([]byte("[SD] CMD0: Going to idle state\n"))
	}
	return 0xFF
}

// cmdSendIfCond handles CMD8 - send interface condition
func (sd *SDCard) cmdSendIfCond(arg uint32) byte {
	if sd.state != SD_SENDING_RESP && sd.state != SD_IDLE {
		// Accept CMD8 if we just finished sending a response or are idle
		if sd.state != SD_INIT_CLOCKS {
			sd.state = SD_ERROR
			if sd.tracer != nil {
				sd.tracer.out.Write([]byte("[SD] ERROR: CMD8 in wrong state\n"))
			}
			return 0xFF
		}
	}

	if !sd.sawCMD0 {
		sd.state = SD_ERROR
		if sd.tracer != nil {
			sd.tracer.out.Write([]byte("[SD] ERROR: CMD8 before CMD0\n"))
		}
		return 0xFF
	}

	sd.sawCMD8 = true

	// R7 response (5 bytes): R1 + 4 bytes echoing voltage and check pattern
	// Arg format: [31:12] reserved, [11:8] voltage, [7:0] check pattern
	checkPattern := byte(arg & 0xFF)
	voltage := byte((arg >> 8) & 0x0F)

	// R1=0x01 (idle), then echo voltage and check pattern
	sd.respBuf = []byte{0x01, 0x00, 0x00, voltage, checkPattern}
	sd.respIndex = 0
	sd.state = SD_SENDING_RESP

	if sd.tracer != nil {
		sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] CMD8: Echo voltage=0x%X check=0x%02X\n", voltage, checkPattern)))
	}
	return 0xFF
}

// cmdAppCmd handles CMD55 - next command is application-specific
func (sd *SDCard) cmdAppCmd() byte {
	sd.sawCMD55 = true

	// R1 response: 0x01 if idle, 0x00 if ready
	var r1 byte = 0x01
	if sd.isReady {
		r1 = 0x00
	}

	sd.respBuf = []byte{r1}
	sd.respIndex = 0
	sd.state = SD_SENDING_RESP

	if sd.tracer != nil {
		sd.tracer.out.Write([]byte("[SD] CMD55: App command prefix\n"))
	}
	return 0xFF
}

// acmdSendOpCond handles ACMD41 - send operating condition
func (sd *SDCard) acmdSendOpCond(arg uint32) byte {
	// Check HCS bit (bit 30) - we support SDHC
	// hcs := (arg & 0x40000000) != 0

	// After a few calls, report ready
	// In a real SD card, this would check if initialization is complete
	// We'll just go to ready state immediately
	sd.isReady = true

	// R1 response: 0x00 = ready (no longer in idle state)
	sd.respBuf = []byte{0x00}
	sd.respIndex = 0
	sd.state = SD_SENDING_RESP

	if sd.tracer != nil {
		sd.tracer.out.Write([]byte("[SD] ACMD41: Card ready\n"))
	}
	return 0xFF
}

// cmdReadOCR handles CMD58 - read OCR register
func (sd *SDCard) cmdReadOCR() byte {
	if !sd.sawCMD0 {
		sd.state = SD_ERROR
		if sd.tracer != nil {
			sd.tracer.out.Write([]byte("[SD] ERROR: CMD58 before CMD0\n"))
		}
		return 0xFF
	}

	sd.sawCMD58 = true

	// R3 response (5 bytes): R1 + 4-byte OCR
	// OCR bits:
	// [31] Card power up status (1 = ready)
	// [30] Card capacity status (1 = SDHC/SDXC)
	// [23:0] Voltage window (we support 3.3V)
	var r1 byte = 0x01
	if sd.isReady {
		r1 = 0x00
	}

	// OCR: power up complete, SDHC, 3.3V supported
	ocr := uint32(0xC0FF8000)
	if !sd.isReady {
		// Not yet powered up
		ocr = 0x00FF8000
	}

	sd.respBuf = []byte{
		r1,
		byte(ocr >> 24),
		byte(ocr >> 16),
		byte(ocr >> 8),
		byte(ocr),
	}
	sd.respIndex = 0
	sd.state = SD_SENDING_RESP

	if sd.tracer != nil {
		sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] CMD58: OCR=0x%08X\n", ocr)))
	}
	return 0xFF
}

// cmdReadSingleBlock handles CMD17 - read a single 512-byte block
func (sd *SDCard) cmdReadSingleBlock(arg uint32) byte {
	if !sd.isReady {
		if sd.tracer != nil {
			sd.tracer.out.Write([]byte("[SD] ERROR: CMD17 when not ready\n"))
		}
		sd.respBuf = []byte{0x04} // Illegal command
		sd.respIndex = 0
		sd.state = SD_SENDING_RESP
		return 0xFF
	}

	// For SDHC, arg is block address (multiply by 512 to get byte address)
	// For standard SD, arg is byte address
	// We'll treat it as byte address since we're emulating standard SD
	byteAddr := arg

	// Validate address
	if int64(byteAddr)+512 > sd.fileSize {
		if sd.tracer != nil {
			sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] ERROR: CMD17 address out of range: 0x%08X\n", byteAddr)))
		}
		sd.respBuf = []byte{0x40} // Parameter error
		sd.respIndex = 0
		sd.state = SD_SENDING_RESP
		return 0xFF
	}

	// Read the block from file
	_, err := sd.file.Seek(int64(byteAddr), 0)
	if err != nil {
		if sd.tracer != nil {
			sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] ERROR: Seek failed: %v\n", err)))
		}
		sd.respBuf = []byte{0x04}
		sd.respIndex = 0
		sd.state = SD_SENDING_RESP
		return 0xFF
	}

	_, err = sd.file.Read(sd.dataBuf[:512])
	if err != nil {
		if sd.tracer != nil {
			sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] ERROR: Read failed: %v\n", err)))
		}
		sd.respBuf = []byte{0x04}
		sd.respIndex = 0
		sd.state = SD_SENDING_RESP
		return 0xFF
	}

	// Add dummy CRC
	sd.dataBuf[512] = 0x00
	sd.dataBuf[513] = 0x00

	sd.dataAddr = byteAddr
	sd.dataIndex = 0
	sd.dataPhase = DATA_PHASE_TOKEN

	// R1 response: 0x00 = success
	sd.respBuf = []byte{0x00}
	sd.respIndex = 0
	sd.state = SD_SENDING_RESP

	if sd.tracer != nil {
		sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] CMD17: Read block at 0x%08X\n", byteAddr)))
	}
	return 0xFF
}

// cmdWriteSingleBlock handles CMD24 - write a single 512-byte block
func (sd *SDCard) cmdWriteSingleBlock(arg uint32) byte {
	if !sd.isReady {
		if sd.tracer != nil {
			sd.tracer.out.Write([]byte("[SD] ERROR: CMD24 when not ready\n"))
		}
		sd.respBuf = []byte{0x04} // Illegal command
		sd.respIndex = 0
		sd.state = SD_SENDING_RESP
		return 0xFF
	}

	// Byte address (see note in cmdReadSingleBlock)
	byteAddr := arg

	// Validate address
	if int64(byteAddr)+512 > sd.fileSize {
		if sd.tracer != nil {
			sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] ERROR: CMD24 address out of range: 0x%08X\n", byteAddr)))
		}
		sd.respBuf = []byte{0x40} // Parameter error
		sd.respIndex = 0
		sd.state = SD_SENDING_RESP
		return 0xFF
	}

	sd.dataAddr = byteAddr
	sd.dataIndex = 0
	sd.dataPhase = DATA_PHASE_TOKEN

	// R1 response: 0x00 = success, then wait for data
	sd.respBuf = []byte{0x00}
	sd.respIndex = 0
	sd.state = SD_SENDING_RESP

	if sd.tracer != nil {
		sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] CMD24: Write block at 0x%08X\n", byteAddr)))
	}
	return 0xFF
}

// sendResponse sends the next byte of a response
// txByte is the byte received from the host (needed for CMD24 transition)
func (sd *SDCard) sendResponse(txByte byte) byte {
	if sd.respIndex >= len(sd.respBuf) {
		// Response complete
		// Check if we have data to send (CMD17)
		if sd.dataPhase == DATA_PHASE_TOKEN && sd.dataIndex == 0 && sd.cmdBuf[0]&0x3F == 17 {
			sd.state = SD_READING_DATA
			return sd.sendDataBlock()
		}
		// Check if we're waiting for data (CMD24)
		if sd.dataPhase == DATA_PHASE_TOKEN && sd.dataIndex == 0 && sd.cmdBuf[0]&0x3F == 24 {
			sd.state = SD_WRITING_DATA
			// Process the txByte as the first byte of data reception
			return sd.receiveDataBlock(txByte)
		}
		// Return to appropriate base state
		if sd.isReady {
			sd.state = SD_READY
		} else {
			sd.state = SD_IDLE
		}
		return 0xFF
	}

	b := sd.respBuf[sd.respIndex]
	sd.respIndex++
	return b
}

// sendDataBlock sends the next byte of a data block (read operation)
func (sd *SDCard) sendDataBlock() byte {
	switch sd.dataPhase {
	case DATA_PHASE_TOKEN:
		// Send data start token (0xFE)
		sd.dataPhase = DATA_PHASE_DATA
		return 0xFE

	case DATA_PHASE_DATA:
		b := sd.dataBuf[sd.dataIndex]
		sd.dataIndex++
		if sd.dataIndex >= 512 {
			sd.dataPhase = DATA_PHASE_CRC1
		}
		return b

	case DATA_PHASE_CRC1:
		sd.dataPhase = DATA_PHASE_CRC2
		return sd.dataBuf[512] // CRC byte 1

	case DATA_PHASE_CRC2:
		sd.state = SD_READY
		sd.dataPhase = DATA_PHASE_TOKEN
		sd.dataIndex = 0
		return sd.dataBuf[513] // CRC byte 2
	}

	return 0xFF
}

// receiveDataBlock receives the next byte of a data block (write operation)
func (sd *SDCard) receiveDataBlock(txByte byte) byte {
	switch sd.dataPhase {
	case DATA_PHASE_TOKEN:
		// Wait for data start token (0xFE)
		if txByte == 0xFE {
			sd.dataPhase = DATA_PHASE_DATA
			sd.dataIndex = 0
		}
		return 0xFF

	case DATA_PHASE_DATA:
		sd.dataBuf[sd.dataIndex] = txByte
		sd.dataIndex++
		if sd.dataIndex >= 512 {
			sd.dataPhase = DATA_PHASE_CRC1
		}
		return 0xFF

	case DATA_PHASE_CRC1:
		// Ignore CRC byte 1
		sd.dataPhase = DATA_PHASE_CRC2
		return 0xFF

	case DATA_PHASE_CRC2:
		// Ignore CRC byte 2, write the data
		_, err := sd.file.Seek(int64(sd.dataAddr), 0)
		if err != nil {
			if sd.tracer != nil {
				sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] ERROR: Write seek failed: %v\n", err)))
			}
			sd.state = SD_READY
			return 0x0D // Data rejected due to write error
		}

		_, err = sd.file.Write(sd.dataBuf[:512])
		if err != nil {
			if sd.tracer != nil {
				sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] ERROR: Write failed: %v\n", err)))
			}
			sd.state = SD_READY
			return 0x0D // Data rejected due to write error
		}

		if sd.tracer != nil {
			sd.tracer.out.Write([]byte(fmt.Sprintf("[SD] Write complete at 0x%08X\n", sd.dataAddr)))
		}

		sd.state = SD_READY
		sd.dataPhase = DATA_PHASE_TOKEN
		sd.dataIndex = 0
		return 0x05 // Data accepted
	}

	return 0xFF
}

// Close closes the SD card file
func (sd *SDCard) Close() error {
	if sd.file != nil {
		return sd.file.Close()
	}
	return nil
}
