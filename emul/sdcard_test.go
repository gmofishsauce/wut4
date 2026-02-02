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
	"os"
	"testing"
)

func TestSDCardInit(t *testing.T) {
	// Create a temporary file for the SD card
	f, err := os.CreateTemp("", "sdcard_test_*.img")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	// Write 1KB of zeros
	data := make([]byte, 1024)
	_, err = f.Write(data)
	if err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	f.Sync()

	// Create SD card
	sd, err := NewSDCard(f, nil)
	if err != nil {
		t.Fatalf("Failed to create SD card: %v", err)
	}

	// Test init sequence
	// 1. Send 10+ 0xFF bytes while deselected
	for i := 0; i < 12; i++ {
		result := sd.Transfer(0xFF)
		if result != 0xFF {
			t.Errorf("Init byte %d: expected 0xFF, got 0x%02X", i, result)
		}
	}

	// 2. Select the card
	sd.SetSelect(0x00)

	// 3. Send CMD0 (go idle)
	sendCommand(sd, 0, 0)
	r1 := readR1(sd)
	if r1 != 0x01 {
		t.Errorf("CMD0: expected R1=0x01 (idle), got 0x%02X", r1)
	}
	_ = sd.Transfer(0xFF) // pump

	// 4. Send CMD8 (send interface condition)
	sendCommand(sd, 8, 0x000001AA)
	r1 = readR1(sd)
	if r1 != 0x01 {
		t.Errorf("CMD8: expected R1=0x01, got 0x%02X", r1)
	}
	// Read remaining 4 bytes of R7
	r7 := make([]byte, 4)
	for i := 0; i < 4; i++ {
		r7[i] = sd.Transfer(0xFF)
	}
	if r7[3] != 0xAA {
		t.Errorf("CMD8: expected check pattern 0xAA, got 0x%02X", r7[3])
	}
	_ = sd.Transfer(0xFF) // pump

	// 5. Send CMD58 (read OCR)
	sendCommand(sd, 58, 0)
	r1 = readR1(sd)
	if r1 != 0x01 {
		t.Errorf("CMD58: expected R1=0x01, got 0x%02X", r1)
	}
	// Read OCR (4 bytes)
	for i := 0; i < 4; i++ {
		sd.Transfer(0xFF)
	}
	_ = sd.Transfer(0xFF) // pump

	// 6. Send CMD55 + ACMD41 loop
	sendCommand(sd, 55, 0)
	r1 = readR1(sd)
	if r1 != 0x01 {
		t.Errorf("CMD55: expected R1=0x01, got 0x%02X", r1)
	}
	_ = sd.Transfer(0xFF) // pump

	sendCommand(sd, 41, 0x40000000)
	r1 = readR1(sd)
	if r1 != 0x00 {
		t.Errorf("ACMD41: expected R1=0x00 (ready), got 0x%02X", r1)
	}
	_ = sd.Transfer(0xFF) // pump

	// Card should now be ready
	if sd.state != SD_READY {
		t.Errorf("Expected SD_READY state, got %d", sd.state)
	}
	if !sd.isReady {
		t.Errorf("Expected isReady=true")
	}
}

func TestSDCardReadWrite(t *testing.T) {
	// Create a temporary file for the SD card
	f, err := os.CreateTemp("", "sdcard_rw_test_*.img")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	// Write 2KB of zeros
	data := make([]byte, 2048)
	_, err = f.Write(data)
	if err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	f.Sync()

	// Create and initialize SD card
	sd, err := NewSDCard(f, nil)
	if err != nil {
		t.Fatalf("Failed to create SD card: %v", err)
	}
	initSDCard(sd)

	// Write a block to sector 1 (byte address 512)
	testData := make([]byte, 512)
	for i := range testData {
		testData[i] = byte(i & 0xFF)
	}

	// CMD24 - write single block
	sendCommand(sd, 24, 512)
	r1 := readR1(sd)
	if r1 != 0x00 {
		t.Fatalf("CMD24: expected R1=0x00, got 0x%02X", r1)
	}

	// Send data start token
	sd.Transfer(0xFE)

	// Send data
	for i := 0; i < 512; i++ {
		sd.Transfer(testData[i])
	}

	// Send CRC (dummy)
	sd.Transfer(0x00)
	resp := sd.Transfer(0x00) // This returns data response token
	if (resp & 0x1F) != 0x05 {
		t.Errorf("Write: expected data accepted (0x05), got 0x%02X", resp&0x1F)
	}

	// Wait for write to complete
	for i := 0; i < 10; i++ {
		if sd.Transfer(0xFF) == 0xFF {
			break
		}
	}

	// Read the block back
	sendCommand(sd, 17, 512)
	r1 = readR1(sd)
	if r1 != 0x00 {
		t.Fatalf("CMD17: expected R1=0x00, got 0x%02X", r1)
	}

	// Wait for data start token
	var token byte
	for i := 0; i < 100; i++ {
		token = sd.Transfer(0xFF)
		if token == 0xFE {
			break
		}
	}
	if token != 0xFE {
		t.Fatalf("CMD17: expected data token 0xFE, got 0x%02X", token)
	}

	// Read data
	readData := make([]byte, 512)
	for i := 0; i < 512; i++ {
		readData[i] = sd.Transfer(0xFF)
	}

	// Read CRC
	sd.Transfer(0xFF)
	sd.Transfer(0xFF)

	// Verify data
	for i := 0; i < 512; i++ {
		if readData[i] != testData[i] {
			t.Errorf("Data mismatch at offset %d: expected 0x%02X, got 0x%02X", i, testData[i], readData[i])
		}
	}
}

func TestSDCardFileSizeValidation(t *testing.T) {
	// Test file too small
	f, _ := os.CreateTemp("", "sdcard_small_*.img")
	defer os.Remove(f.Name())
	f.Write(make([]byte, 256)) // Only 256 bytes
	f.Close()

	f, _ = os.Open(f.Name())
	_, err := NewSDCard(f, nil)
	f.Close()
	if err == nil {
		t.Error("Expected error for file too small")
	}

	// Test file not multiple of 512
	f, _ = os.CreateTemp("", "sdcard_unaligned_*.img")
	defer os.Remove(f.Name())
	f.Write(make([]byte, 1000)) // Not multiple of 512
	f.Close()

	f, _ = os.Open(f.Name())
	_, err = NewSDCard(f, nil)
	f.Close()
	if err == nil {
		t.Error("Expected error for file not multiple of 512")
	}
}

// Helper functions

func sendCommand(sd *SDCard, cmd byte, arg uint32) {
	sd.Transfer(0x40 | cmd)                  // Command byte
	sd.Transfer(byte((arg >> 24) & 0xFF))    // Arg byte 3
	sd.Transfer(byte((arg >> 16) & 0xFF))    // Arg byte 2
	sd.Transfer(byte((arg >> 8) & 0xFF))     // Arg byte 1
	sd.Transfer(byte(arg & 0xFF))            // Arg byte 0
	sd.Transfer(0x95)                        // CRC (valid for CMD0)
}

func readR1(sd *SDCard) byte {
	// Wait for response (up to 8 bytes)
	for i := 0; i < 8; i++ {
		r := sd.Transfer(0xFF)
		if r != 0xFF {
			return r
		}
	}
	return 0xFF
}

func initSDCard(sd *SDCard) {
	// Send init clocks
	for i := 0; i < 12; i++ {
		sd.Transfer(0xFF)
	}

	// Select card
	sd.SetSelect(0x00)

	// CMD0
	sendCommand(sd, 0, 0)
	readR1(sd)
	sd.Transfer(0xFF) // pump

	// CMD8
	sendCommand(sd, 8, 0x1AA)
	readR1(sd)
	for i := 0; i < 4; i++ {
		sd.Transfer(0xFF)
	}
	sd.Transfer(0xFF) // pump

	// CMD58
	sendCommand(sd, 58, 0)
	readR1(sd)
	for i := 0; i < 4; i++ {
		sd.Transfer(0xFF)
	}
	sd.Transfer(0xFF) // pump

	// CMD55 + ACMD41
	sendCommand(sd, 55, 0)
	readR1(sd)
	sd.Transfer(0xFF) // pump
	sendCommand(sd, 41, 0x40000000)
	readR1(sd)
	sd.Transfer(0xFF) // pump
}
