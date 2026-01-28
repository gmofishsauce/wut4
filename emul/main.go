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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/term"
)

var (
	traceFile   = flag.String("trace", "", "Write execution trace to file")
	maxCycles   = flag.Uint64("max-cycles", 0, "Stop after N cycles (0 = unlimited)")
	showVersion = flag.Bool("version", false, "Show version and exit")
)

const version = "1.0.0"

var savedTermState *term.State

// setupTerminal puts the terminal in raw mode for the UART emulation
func setupTerminal() error {
	// Only set raw mode if stdin is a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil
	}

	// Save current terminal state
	state, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to get terminal state: %v", err)
	}
	savedTermState = state

	// Put terminal in raw mode (non-canonical mode, no echo, etc.)
	_, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to set raw mode: %v", err)
	}

	return nil
}

// restoreTerminal restores the terminal to its original state
func restoreTerminal() {
	if savedTermState != nil && term.IsTerminal(int(os.Stdin.Fd())) {
		term.Restore(int(os.Stdin.Fd()), savedTermState)
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Printf("WUT-4 Emulator v%s\n", version)
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) != 1 {
		usage()
		os.Exit(1)
	}

	binaryFile := args[0]

	// Load binary file before setting up terminal raw mode so that
	// any errors are reported cleanly in normal terminal mode.
	data, err := os.ReadFile(binaryFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading binary file: %v\n", err)
		os.Exit(1)
	}

	// Create CPU
	cpu := NewCPU()
	cpu.consoleIn = os.Stdin
	cpu.consoleOut = os.Stderr

	// Set up tracing if requested
	if *traceFile != "" {
		f, err := os.Create(*traceFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating trace file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		cpu.tracer = NewTracer(f)
		fmt.Fprintf(f, "WUT-4 Emulator Trace\n")
		fmt.Fprintf(f, "Binary: %s\n", binaryFile)
		fmt.Fprintf(f, "Size: %d bytes (%d words)\n", len(data), len(data)/2)
		fmt.Fprintf(f, "========================================\n\n")
	}

	// Load binary into memory
	err = cpu.LoadBinary(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading binary: %v\n", err)
		os.Exit(1)
	}

	// Set up terminal in raw mode for UART emulation
	err = setupTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up terminal: %v\n", err)
		os.Exit(1)
	}

	// Ensure terminal is restored on exit
	defer restoreTerminal()

	// Set up signal handler to restore terminal on interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		restoreTerminal()
		os.Exit(130) // Standard exit code for SIGINT
	}()

	// Reset CPU to initial state
	cpu.Reset()

	// Start UART I/O goroutines
	cpu.startUART()

	// Write startup message to trace file if available
	if cpu.tracer != nil {
		fmt.Fprintf(cpu.tracer.out, "Loaded: %s (%d bytes, %d words)\n", binaryFile, len(data), len(data)/2)
		if *traceFile != "" {
			fmt.Fprintf(cpu.tracer.out, "Trace: %s\n", *traceFile)
		}
		if *maxCycles > 0 {
			fmt.Fprintf(cpu.tracer.out, "Max cycles: %d\n", *maxCycles)
		}
		fmt.Fprintf(cpu.tracer.out, "\n")
	}

	// Run the emulator
	startTime := time.Now()
	err = runEmulator(cpu, *maxCycles)
	elapsed := time.Since(startTime)

	// Restore terminal before printing statistics so output isn't garbled
	restoreTerminal()

	// Print statistics
	fmt.Fprintf(os.Stderr, "\n========================================\n")
	fmt.Fprintf(os.Stderr, "Execution completed\n")
	fmt.Fprintf(os.Stderr, "Cycles: %d\n", cpu.cycles)
	fmt.Fprintf(os.Stderr, "Time: %v\n", elapsed.Round(time.Millisecond))

	if elapsed.Seconds() > 0 {
		mhz := (float64(cpu.cycles) / 1_000_000.0) / elapsed.Seconds()
		fmt.Fprintf(os.Stderr, "Speed: %.3f MHz\n", mhz)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Exit: normal\n")
}

func runEmulator(cpu *CPU, maxCycles uint64) error {
	// If max cycles specified, wrap the execution loop
	if maxCycles > 0 {
		for cpu.running {
			if cpu.cycles >= maxCycles {
				fmt.Fprintf(os.Stderr, "\nMax cycles reached (%d)\n", maxCycles)
				cpu.running = false
				return nil
			}

			// Execute one instruction cycle
			if cpu.pendingException && !cpu.intEnabled && cpu.mode == ModeKernel {
				return fmt.Errorf("double fault: exception 0x%04X in kernel mode with interrupts disabled", cpu.exceptionVector)
			}

			if cpu.tracer != nil {
				cpu.tracer.TracePreInstruction(cpu)
			}

			inst, err := cpu.fetch()
			if err != nil {
				return err
			}

			if cpu.pendingException {
				cpu.handleException()
				cpu.cycles++
				continue
			}

			decoded := decode(inst)

			err = cpu.execute(decoded)
			if err != nil {
				return err
			}

			if cpu.pendingException {
				cpu.handleException()
			}

			cpu.cycles++

			if cpu.tracer != nil {
				cpu.tracer.TracePostInstruction(cpu, decoded)
			}
		}
		return nil
	}

	// No max cycles - use normal Run() method
	return cpu.Run()
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] <binary-file>\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "WUT-4 Emulator - Execute WUT-4 binaries\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nArguments:\n")
	fmt.Fprintf(os.Stderr, "  <binary-file>    WUT-4 binary file to execute\n")
	fmt.Fprintf(os.Stderr, "\nThe emulator executes the binary and connects console I/O to stdin/stderr.\n")
	fmt.Fprintf(os.Stderr, "Use -trace to generate a detailed execution trace file.\n")
}
