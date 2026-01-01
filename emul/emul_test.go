// Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Test harness for WUT-4 emulator regression tests
// Tests use a convention: programs write 0x0000 to physical address 0xFFFE (word 0x7FFF)
// to indicate success, or any non-zero value to indicate failure code.

const (
	// Test result convention: write result to last word of physical memory
	TEST_RESULT_ADDR = 0x7FFF // Word address (byte address 0xFFFE)
	TEST_SUCCESS     = 0x0000
	MAX_TEST_CYCLES  = 100000 // Prevent infinite loops in tests
)

// runTestBinary loads and runs a binary, returning the test result and any error
func runTestBinary(t *testing.T, binaryPath string) (uint16, error) {
	t.Helper()

	// Load binary file
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return 0xFFFF, fmt.Errorf("failed to read binary: %w", err)
	}

	// Create CPU with test I/O (discard output)
	cpu := NewCPU()
	cpu.consoleIn = bytes.NewReader([]byte{}) // Empty input
	cpu.consoleOut = &bytes.Buffer{}          // Capture output

	// Load binary
	err = cpu.LoadBinary(data)
	if err != nil {
		return 0xFFFF, fmt.Errorf("failed to load binary: %w", err)
	}

	// Reset and run
	cpu.Reset()

	// Run with cycle limit to prevent infinite loops
	for cpu.running && cpu.cycles < MAX_TEST_CYCLES {
		inst, err := cpu.fetch()
		if err != nil {
			return 0xFFFF, err
		}

		if cpu.pendingException {
			cpu.handleException()
			cpu.cycles++
			continue
		}

		decoded := decode(inst)
		err = cpu.execute(decoded)
		if err != nil {
			return 0xFFFF, err
		}

		if cpu.pendingException {
			cpu.handleException()
		}

		cpu.cycles++
	}

	// Check if we hit cycle limit
	if cpu.cycles >= MAX_TEST_CYCLES {
		return 0xFFFF, fmt.Errorf("test exceeded cycle limit (%d)", MAX_TEST_CYCLES)
	}

	// Read test result from physical memory
	result := cpu.physMem[TEST_RESULT_ADDR]
	return result, nil
}

// TestIntegration runs all integration tests from testdata directory
func TestIntegration(t *testing.T) {
	// Test structure: testdata/<category>/<test_name>.out
	categories := []string{"arithmetic", "memory", "branch", "exceptions"}

	for _, category := range categories {
		categoryPath := filepath.Join("testdata", category)
		if _, err := os.Stat(categoryPath); os.IsNotExist(err) {
			t.Logf("Skipping category %s (directory does not exist)", category)
			continue
		}

		// Find all .out files in this category
		files, err := filepath.Glob(filepath.Join(categoryPath, "*.out"))
		if err != nil {
			t.Fatalf("Failed to glob test files in %s: %v", category, err)
		}

		for _, binPath := range files {
			testName := filepath.Base(binPath[:len(binPath)-4]) // Remove .out extension

			t.Run(fmt.Sprintf("%s/%s", category, testName), func(t *testing.T) {
				result, err := runTestBinary(t, binPath)
				if err != nil {
					t.Fatalf("Test execution failed: %v", err)
				}

				if result != TEST_SUCCESS {
					t.Errorf("Test failed with result code: 0x%04X (expected 0x%04X)", result, TEST_SUCCESS)
				}
			})
		}
	}
}

// TestArithmeticBasic tests basic arithmetic operations
func TestArithmeticBasic(t *testing.T) {
	testPath := "testdata/arithmetic/add_basic.out"
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Skip("Test binary not found (run build_tests.sh first)")
	}

	result, err := runTestBinary(t, testPath)
	if err != nil {
		t.Fatalf("Test execution failed: %v", err)
	}

	if result != TEST_SUCCESS {
		t.Errorf("Test failed with result code: 0x%04X", result)
	}
}

// TestMemoryOperations tests load/store operations
func TestMemoryOperations(t *testing.T) {
	testPath := "testdata/memory/load_store.out"
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Skip("Test binary not found (run build_tests.sh first)")
	}

	result, err := runTestBinary(t, testPath)
	if err != nil {
		t.Fatalf("Test execution failed: %v", err)
	}

	if result != TEST_SUCCESS {
		t.Errorf("Test failed with result code: 0x%04X", result)
	}
}

// TestBranchOperations tests conditional and unconditional branches
func TestBranchOperations(t *testing.T) {
	testPath := "testdata/branch/conditional.out"
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Skip("Test binary not found (run build_tests.sh first)")
	}

	result, err := runTestBinary(t, testPath)
	if err != nil {
		t.Fatalf("Test execution failed: %v", err)
	}

	if result != TEST_SUCCESS {
		t.Errorf("Test failed with result code: 0x%04X", result)
	}
}
