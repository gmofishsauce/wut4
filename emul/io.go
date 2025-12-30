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

import "os"

// readConsole reads a single byte from console input (stdin)
// Returns 0 if no input is available (non-blocking)
func (cpu *CPU) readConsole() uint16 {
	if cpu.consoleIn == nil {
		return 0
	}

	buf := make([]byte, 1)
	n, err := cpu.consoleIn.Read(buf)
	if err != nil || n == 0 {
		return 0
	}

	value := uint16(buf[0])

	// Trace console input
	if cpu.tracer != nil {
		cpu.tracer.TraceConsoleInput(value)
	}

	return value
}

// writeConsole writes a single byte to console output (stdout)
func (cpu *CPU) writeConsole(value uint16) {
	if cpu.consoleOut == nil {
		return
	}

	buf := []byte{byte(value & 0xFF)}
	cpu.consoleOut.Write(buf)

	// Flush output immediately so character appears without waiting for newline
	if f, ok := cpu.consoleOut.(*os.File); ok {
		f.Sync()
	}

	// Trace console output
	if cpu.tracer != nil {
		cpu.tracer.TraceConsoleOutput(value & 0xFF)
	}
}
