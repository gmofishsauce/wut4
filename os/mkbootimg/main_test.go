package main

import (
	"encoding/binary"
	"testing"
)

// makeExe builds a synthetic WUT-4 executable with recognisable fill patterns:
//   code bytes: byte(i & 0xFF)
//   data bytes: byte((i + 0x80) & 0xFF)
func makeExe(codeSize, dataSize int) []byte {
	exe := make([]byte, exeHeader+codeSize+dataSize)
	binary.LittleEndian.PutUint16(exe[0:2], magicExe)
	binary.LittleEndian.PutUint16(exe[2:4], uint16(codeSize))
	binary.LittleEndian.PutUint16(exe[4:6], uint16(dataSize))
	for i := 0; i < codeSize; i++ {
		exe[exeHeader+i] = byte(i & 0xFF)
	}
	for i := 0; i < dataSize; i++ {
		exe[exeHeader+codeSize+i] = byte((i + 0x80) & 0xFF)
	}
	return exe
}

func TestBootImage(t *testing.T) {
	tests := []struct {
		name        string
		codeSize    int
		dataSize    int
		codeSectors int
		dataSectors int
	}{
		{"both less than one sector", 100, 200, 1, 1},
		{"both exactly one sector", 512, 512, 1, 1},
		{"code two sectors data partial", 1024, 600, 2, 2},
		{"data spans sector boundary", 513, 513, 2, 2},
		{"code multi data multi", 1024, 1537, 2, 4},
		{"no data", 100, 0, 1, 0},
		{"no code", 0, 200, 0, 1},
		{"both zero", 0, 0, 0, 0},
		{"code exact boundary", 512, 0, 1, 0},
		{"code one byte over boundary", 513, 0, 2, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exe := makeExe(tt.codeSize, tt.dataSize)
			img, err := BootImage(exe)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Total size
			wantLen := (1 + tt.codeSectors + tt.dataSectors) * sectorSize
			if len(img) != wantLen {
				t.Fatalf("len(img) = %d, want %d", len(img), wantLen)
			}

			// Header magic
			if got := binary.LittleEndian.Uint16(img[0:2]); got != magicBoot {
				t.Errorf("magic = 0x%04X, want 0x%04X", got, magicBoot)
			}

			// Sector counts
			if got := int(binary.LittleEndian.Uint16(img[2:4])); got != tt.codeSectors {
				t.Errorf("code_sectors = %d, want %d", got, tt.codeSectors)
			}
			if got := int(binary.LittleEndian.Uint16(img[4:6])); got != tt.dataSectors {
				t.Errorf("data_sectors = %d, want %d", got, tt.dataSectors)
			}

			// Header padding bytes must be zero
			for i := 6; i < sectorSize; i++ {
				if img[i] != 0 {
					t.Errorf("header byte %d = 0x%02X, want 0x00", i, img[i])
					break
				}
			}

			// Code bytes at sector 1
			codeStart := sectorSize
			for i := 0; i < tt.codeSize; i++ {
				want := byte(i & 0xFF)
				if img[codeStart+i] != want {
					t.Errorf("code[%d] = 0x%02X, want 0x%02X", i, img[codeStart+i], want)
					break
				}
			}

			// Code padding must be zero
			for i := codeStart + tt.codeSize; i < sectorSize*(1+tt.codeSectors); i++ {
				if img[i] != 0 {
					t.Errorf("code padding byte %d = 0x%02X, want 0x00", i, img[i])
					break
				}
			}

			// Data bytes after code sectors
			dataStart := sectorSize * (1 + tt.codeSectors)
			for i := 0; i < tt.dataSize; i++ {
				want := byte((i + 0x80) & 0xFF)
				if img[dataStart+i] != want {
					t.Errorf("data[%d] = 0x%02X, want 0x%02X", i, img[dataStart+i], want)
					break
				}
			}

			// Data padding must be zero
			for i := dataStart + tt.dataSize; i < len(img); i++ {
				if img[i] != 0 {
					t.Errorf("data padding byte %d = 0x%02X, want 0x00", i, img[i])
					break
				}
			}
		})
	}
}

func TestBootImageErrors(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			"file too small for header",
			[]byte{0xD1, 0xDD, 0x00}, // only 3 bytes
		},
		{
			"wrong magic",
			func() []byte {
				exe := makeExe(10, 10)
				binary.LittleEndian.PutUint16(exe[0:2], 0x1234)
				return exe
			}(),
		},
		{
			"file truncated below declared size",
			func() []byte {
				exe := makeExe(100, 100) // 216 bytes total
				return exe[:50]          // header still claims 100+100
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BootImage(tt.input)
			if err == nil {
				t.Error("expected an error but got nil")
			}
		})
	}
}
