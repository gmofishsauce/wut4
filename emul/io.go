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
	"io"
	"os"
)

// startUART starts the UART I/O goroutines
func (cpu *CPU) startUART() {
	if cpu.uart == nil {
		return
	}

	// Goroutine to read from stdin and send to receive channel
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil || n == 0 {
				continue
			}

			// Send byte to receive channel (blocking)
			// If the channel is full, this will block until space is available
			cpu.uart.rxChan <- buf[0]
		}
	}()

	// Goroutine to receive from transmit channel and write to stdout
	go func() {
		for b := range cpu.uart.txChan {
			os.Stdout.Write([]byte{b})
		}
	}()
}

// uartWriteData writes a byte to the transmit FIFO (u0, SPR 96)
func (cpu *CPU) uartWriteData(value uint16) {
	if cpu.uart == nil {
		return
	}

	b := byte(value & 0xFF)

	cpu.uart.mu.Lock()
	defer cpu.uart.mu.Unlock()

	// Try non-blocking send to transmit channel
	select {
	case cpu.uart.txChan <- b:
		// Success - byte added to FIFO
		if cpu.tracer != nil {
			cpu.tracer.TraceConsoleOutput(uint16(b))
		}
	default:
		// Channel full - set overflow flag
		cpu.uart.txOverflow = true
	}
}

// uartReadData reads a byte from the receive FIFO (u1, SPR 97)
func (cpu *CPU) uartReadData() uint16 {
	if cpu.uart == nil {
		return 0
	}

	cpu.uart.mu.Lock()
	defer cpu.uart.mu.Unlock()

	// Try non-blocking receive from receive channel
	select {
	case b := <-cpu.uart.rxChan:
		// Success - got a byte
		if cpu.tracer != nil {
			cpu.tracer.TraceConsoleInput(uint16(b))
		}
		return uint16(b)
	default:
		// Channel empty - set underflow flag and return 0
		cpu.uart.rxUnderflow = true
		return 0
	}
}

// uartReadTxStatus reads transmit status register (u2, SPR 98)
// Returns bit 0 = overflow, bit 15 = FIFO empty
// Reading clears the overflow flag
func (cpu *CPU) uartReadTxStatus() uint16 {
	if cpu.uart == nil {
		return 0x8000 // Report empty
	}

	cpu.uart.mu.Lock()
	defer cpu.uart.mu.Unlock()

	var status uint16

	// Bit 0: overflow (clear after reading)
	if cpu.uart.txOverflow {
		status |= 0x0001
		cpu.uart.txOverflow = false
	}

	// Bit 15: transmit FIFO empty
	if len(cpu.uart.txChan) == 0 {
		status |= 0x8000
	}

	return status
}

// uartReadRxStatus reads receive status register (u3, SPR 99)
// Returns bit 0 = underflow, bit 15 = data available
// Reading clears the underflow flag
func (cpu *CPU) uartReadRxStatus() uint16 {
	if cpu.uart == nil {
		return 0
	}

	cpu.uart.mu.Lock()
	defer cpu.uart.mu.Unlock()

	var status uint16

	// Bit 0: underflow (clear after reading)
	if cpu.uart.rxUnderflow {
		status |= 0x0001
		cpu.uart.rxUnderflow = false
	}

	// Bit 15: receive FIFO not empty (data available)
	if len(cpu.uart.rxChan) > 0 {
		status |= 0x8000
	}

	return status
}

// spiSelectState tracks the current SPI select register value
var spiSelectState uint16 = 0xFFFF // All devices deselected by default

// spiWriteData writes a byte to the SPI data register (s0, SPR 100)
// This performs an SPI transfer - simultaneously sends and receives
func (cpu *CPU) spiWriteData(value uint16) {
	if cpu.sdcard == nil {
		return
	}

	// Perform SPI transfer (write also triggers a read internally)
	txByte := byte(value & 0xFF)
	cpu.sdcard.Transfer(txByte)
}

// spiReadData reads a byte from the SPI data register (s0, SPR 100)
// This performs an SPI transfer - sends 0xFF and returns the received byte
func (cpu *CPU) spiReadData() uint16 {
	if cpu.sdcard == nil {
		return 0xFFFF
	}

	// SPI read sends 0xFF and returns received byte
	rxByte := cpu.sdcard.Transfer(0xFF)
	return uint16(rxByte)
}

// spiWriteSelect writes to the SPI select register (s1, SPR 101)
// Bit 0 controls SD card select (active low: 0=selected, 1=deselected)
func (cpu *CPU) spiWriteSelect(value uint16) {
	spiSelectState = value

	if cpu.sdcard != nil {
		// Pass bit 0 to SD card (active low)
		cpu.sdcard.SetSelect(byte(value & 0x01))
	}
}

// spiReadSelect reads the SPI select register (s1, SPR 101)
func (cpu *CPU) spiReadSelect() uint16 {
	return spiSelectState
}
