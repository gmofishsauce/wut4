/*
 * Copyright (c) Jeff Berkowitz 2025.
 *
 * This file is part of wut4, which is licensed under the Affero GPL.
 * Code generated by the transpiler is owned by the schematic owner.
 */
package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

// Prefix for generated symbols. TODO: allow overriding
var UniquePrefix = "Tsp"

// The bit number of next bit to allocate
// TODO: quick and dirty impl can only handle 64 bits total.
var nextBit int

// emith and emitc take the same arguments as printf().
// They write to the output .h or .c files, respectively,
// appending a newline if one isn't passed. Emitting an
// empty string will product a blank line in the output.

var cFileName string
var hFileName string
var cFile io.Writer
var hFile io.Writer

// The output files are written to the Gen subdirectory
// if it exists, otherwise in .
func openOutputs() error {

	// TODO option for this
	dirPath := ".." // path.Join(".", "Gen")

	inf, err := os.Stat(dirPath)
	if err != nil || !inf.IsDir() {
		dirPath = "."
	}
	name := path.Join(dirPath, UniquePrefix + "Gen")
	cFileName = name + ".c"
	hFileName = name + ".h"

	cFile, err = os.Create(cFileName)
	if err != nil {
		return err
	}
	hFile, err = os.Create(hFileName)
	if err != nil {
		return err
	}
	return nil
}

func emitc(format string, a ...any) {
	n, err := emitFmt(cFile, format, a...)
	if n == 0 || err != nil {
		panic("low level I/O error formatting output")
	}
}

func emith(format string, a ...any) {
	n, err := emitFmt(hFile, format, a...)
	if n == 0 || err != nil {
		panic("low level I/O error formatting output")
	}
}

func emitFmt(out io.Writer, format string, a ...any) (int, error) {
	if len(format) == 0 {
		n, err := fmt.Fprintf(out, "\n")
		return n, err
	}
	n, err := fmt.Fprintf(out, format, a...)
	if format[len(format)-1] != '\n' {
		fmt.Fprintf(out, "\n")
		n++
	}
	return n, err
}

func isCIdentifierChar(index int, r rune) bool {
	if 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || r == '_' {
		return true
	}
	return index != 0 && '0' <= r && r <= '9'
}

// Create a C-compatible name for a node in a net. We assume the net code,
// node ref, and node pin are ASCII. The startIndex is the offset of the
// first character of s within the eventual identifier being created by
// the caller; if startIndex > 0, the first character of s may be a digit.
func makeCIdentifier(startIndex int, s string) string {
	var sb strings.Builder
	for i, r := range s {
		if isCIdentifierChar(startIndex + i, r) {
			sb.WriteRune(r)
		} else if r == '~' {
			sb.WriteString("NOT_")
		} else if r > 128 {
			sb.WriteString(fmt.Sprintf("%x", string(r)))
		}
		// else drop this ASCII non-identifier char
	}
	return sb.String()
}

// Create a C-compatible name for a node in a network
// We assume the net code, node ref, and node pin are ASCII.
/*
 Here's a typical net:

    (net (code "8") (name "Net-(U1-D0)")
      (node (ref "U1") (pin "4") (pinfunction "D0") (pintype "input"))
      (node (ref "U2") (pin "3") (pintype "output"))
      (node (ref "U2") (pin "4") (pintype "input")))

 The name of the net is not useful. The code makes it unique. We append
 the most useful piece of information, which is the ID of the driver,
 pin U2-3 in this case, and its name if it has one (this driver doesn't).
*/

func makeNetName(ni *NetInstance) string {
	var sb strings.Builder
	var drivingNode *NetNode

	for _, nn := range ni.netNodes {
		// The values of .kind are defined by KiCad; they are:
		// input, output, bidirectional, tri-state, passive, free, unspecified,
		// power input, power output, open collector, open emitter, unconnected.
		if nn.pin.kind == "output" || nn.pin.kind == "tri-state" {
			drivingNode = nn
			break
		}
		// TODO process open collector, bidirectional, maybe others as
		// fallback "driver pins" for a net if there is just one of them.
	}

	sb.WriteRune('N') // startIndex is now 1
	sb.WriteString(ni.code)

	var rawName string
	if drivingNode == nil {
		rawName = makeCIdentifier(1, ni.name)
	} else {
		sb.WriteString("_")
		sb.WriteString(drivingNode.part.ref)
		sb.WriteRune('_')
		sb.WriteString(drivingNode.pin.num)

		name := drivingNode.pin.name
		if len(name) != 0 && name != "NOTFOUND" {
			rawName = makeCIdentifier(sb.Len(), name)
		}
	}

	if len(rawName) != 0 {
		sb.WriteRune('_')
		sb.WriteString(rawName)
	}
	return sb.String()
}

// Allocate bit(s) to represent wire nets or buses of them at runtime.
// Return the shift index of the low-order allocated bit.
func allocWireBits(nBits int) (int, error) {
	// TODO for now, we support up to 64 wires total.
	if nextBit + nBits >= 64 {
		return -1, fmt.Errorf("TODO: schematic has more than 64 wires")
	}
	result := nextBit
	nextBit += nBits
	return result, nil
}

func emitNetMacros(netName string, bitPos int, fieldWidth int) error {
	// mask := (1<<(fieldWidth)) - 1	

	// The GetNNN() macros rely on the C definition of bitN_t being
	// a uint rather than an int to avoid sign-extending arithmetic
	// right shift. Otherwise, would need to & result with mask.

	emith("// net %s", netName)
	emith("#define %s %d", netName, bitPos)

	return nil
}

func makeComponentTypeName(c *ComponentType) string {
	return "C" + makeCIdentifier(1, c.lib + "_" + c.part)
}
