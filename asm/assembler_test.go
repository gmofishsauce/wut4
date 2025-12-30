package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImmediateValueRanges(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "16-bit immediate: -1",
			code: ".code\n    ldi r1, -1\n    hlt\n",
			wantErr: false,
		},
		{
			name: "16-bit immediate: 65535 (max unsigned)",
			code: ".code\n    ldi r1, 65535\n    hlt\n",
			wantErr: false,
		},
		{
			name: "16-bit immediate: -32768 (min signed)",
			code: ".code\n    ldi r1, -32768\n    hlt\n",
			wantErr: false,
		},
		{
			name: "16-bit immediate: out of range high",
			code: ".code\n    ldi r1, 65536\n    hlt\n",
			wantErr: true,
		},
		{
			name: "16-bit immediate: out of range low",
			code: ".code\n    ldi r1, -32769\n    hlt\n",
			wantErr: true,
		},
		{
			name: "7-bit immediate: -64 (min signed)",
			code: ".code\n    adi r1, r2, -64\n    hlt\n",
			wantErr: false,
		},
		{
			name: "7-bit immediate: 127 (max unsigned)",
			code: ".code\n    adi r1, r2, 127\n    hlt\n",
			wantErr: false,
		},
		{
			name: "7-bit immediate: out of range high",
			code: ".code\n    adi r1, r2, 128\n    hlt\n",
			wantErr: true,
		},
		{
			name: "7-bit immediate: out of range low",
			code: ".code\n    adi r1, r2, -65\n    hlt\n",
			wantErr: true,
		},
		{
			name: "10-bit immediate: -512 (min signed)",
			code: ".code\n    lui r1, -512\n    hlt\n",
			wantErr: false,
		},
		{
			name: "10-bit immediate: 1023 (max unsigned)",
			code: ".code\n    lui r1, 1023\n    hlt\n",
			wantErr: false,
		},
		{
			name: "10-bit immediate: out of range high",
			code: ".code\n    lui r1, 1024\n    hlt\n",
			wantErr: true,
		},
		{
			name: "10-bit immediate: out of range low",
			code: ".code\n    lui r1, -513\n    hlt\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp input file
			tmpDir := t.TempDir()
			inputFile := filepath.Join(tmpDir, "test.asm")
			outputFile := filepath.Join(tmpDir, "test.bin")

			err := os.WriteFile(inputFile, []byte(tt.code), 0644)
			if err != nil {
				t.Fatalf("failed to write input file: %v", err)
			}

			// Run assembler
			err = assemble(inputFile, outputFile)

			if tt.wantErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCommasOptional(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "tst without comma",
			code: ".code\n    tst r1 r2\n    hlt\n",
			wantErr: false,
		},
		{
			name: "tst with comma",
			code: ".code\n    tst r1, r2\n    hlt\n",
			wantErr: false,
		},
		{
			name: "add without commas",
			code: ".code\n    add r1 r2 r3\n    hlt\n",
			wantErr: false,
		},
		{
			name: "add with commas",
			code: ".code\n    add r1, r2, r3\n    hlt\n",
			wantErr: false,
		},
		{
			name: "ldi without comma",
			code: ".code\n    ldi r1 42\n    hlt\n",
			wantErr: false,
		},
		{
			name: "ldi with comma",
			code: ".code\n    ldi r1, 42\n    hlt\n",
			wantErr: false,
		},
		{
			name: "ldi with negative value without comma",
			code: ".code\n    ldi r1 -1\n    hlt\n",
			wantErr: false,
		},
		{
			name: "ldi with negative value with comma",
			code: ".code\n    ldi r1, -1\n    hlt\n",
			wantErr: false,
		},
		{
			name: ".set without comma",
			code: ".code\n.set CONST 42\n    ldi r1 CONST\n    hlt\n",
			wantErr: false,
		},
		{
			name: ".set with comma",
			code: ".code\n.set CONST, 42\n    ldi r1, CONST\n    hlt\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp input file
			tmpDir := t.TempDir()
			inputFile := filepath.Join(tmpDir, "test.asm")
			outputFile := filepath.Join(tmpDir, "test.bin")

			err := os.WriteFile(inputFile, []byte(tt.code), 0644)
			if err != nil {
				t.Fatalf("failed to write input file: %v", err)
			}

			// Run assembler
			err = assemble(inputFile, outputFile)

			if tt.wantErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestExpressions(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name: "expression with spaces",
			code: ".code\n    ldi r1 2 + 3\n    hlt\n",
			wantErr: false,
		},
		{
			name: "expression without spaces",
			code: ".code\n    ldi r1 2+3\n    hlt\n",
			wantErr: false,
		},
		{
			name: "negative number vs subtraction",
			code: ".code\n.set A 5\n    ldi r1 A\n    ldi r2 -1\n    hlt\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp input file
			tmpDir := t.TempDir()
			inputFile := filepath.Join(tmpDir, "test.asm")
			outputFile := filepath.Join(tmpDir, "test.bin")

			err := os.WriteFile(inputFile, []byte(tt.code), 0644)
			if err != nil {
				t.Fatalf("failed to write input file: %v", err)
			}

			// Run assembler
			err = assemble(inputFile, outputFile)

			if tt.wantErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
