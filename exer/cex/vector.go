package main

// This is an IC component exerciser. It drives patterns of digital
// signals to component inputs and checks values from component outputs.
//
// The acronym "UUT" stands for "unit under test". We output the bits
// that control the UUT's inputs, and check the results that come from
// the UUT's outputs using our inputs. Since this makes the terms
// "input" and "output" confusing, I usually use "toUUT" and "fromUUT".

import (
	"bufio"
	"fmt"
	"log"
	"math/bits" // Reverse16
	"os"
	"strconv"
	"strings"

	"cex/dev"
	"cex/utils"
)

const SPACE = ' '

// Process one vector file. The file format and structure
// are described in the README in this directory. Processing
// involves parsing the file, applying each vector to the
// hardware, and reporting the results.
//
// TODO track line numbers else troubleshooting bad vectors
// will be next to impossible.
func DoVectorFile(filePath string, nano *dev.Arduino) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return scan(bufio.NewScanner(file), nano)
}

// Scan one vector file. Return number of hardware failures
// detected and an error. Hardware failures are not "errors".
func scan(scanner *bufio.Scanner, nano *dev.Arduino) (int, error) {
	var tf *utils.TestFile
	var totalErrors int

	for scanner.Scan() {
		// First check for empty lines and comments. Lines must
		// be left-justified. Lines starting with spaces are empty.
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' || line[0] == SPACE {
			continue
		}

		// Output log comment
		if line[0] == '>' {
			log.Printf("%s", line[1:])
			continue
		}

		tokens := strings.Split(line, string(SPACE))

		// Handle the exactly-once-per-file "socket" statement
		if tokens[0] == "socket" {
			if tf != nil || len(tokens) != 2 {
				return 0, fmt.Errorf("bad 'socket' statement")
			}
			tf = utils.NewTestFile(tokens[1], nano)
			if tf == nil {
				return 0, fmt.Errorf("bad socket type")
			}
			continue
		}

		// Must be a vector or error. We have per-file stuff
		// and per-vector stuff in the same data structure, the
		// TestFile. Sort of ugly, meh. Clear the per-vector
		// fields now.
		tf.Clear()

		if err := parseVector(tf, tokens); err != nil {
			return 0, err
		}
		if err := scanner.Err(); err != nil {
			return 0, err
		}
		if debug {
			log.Printf("Parsed: %s\n", tf)
		}

		// Successfully parsed one vector; apply it
		errorCount, err := applyVector(tf)
		if err != nil {
			return errorCount, err
		}
		totalErrors += errorCount
	}

	return totalErrors, nil
}

// Parse the next vector from the file into the bit vectors in tf.
// The tokens should be a line specifying exactly 24 or 68 bits as
// specified in the socket statement and stored in the size field.
// The vector representations must have been cleared (reallocated)
// by the caller.
func parseVector(tf *utils.TestFile, tokens []string) error {
	var pos utils.BitPosition
	for _, t := range tokens {
		if len(t) == 0 {
			// Golang's split() function splits on every
			// occurrence of the split character, rather
			// than consuming groups of them. Being here
			// just means there were multiple spaces.
			continue
		}

		switch t[0] {
		case '0':
			tf.ResetToUUT(pos)
			pos++
		case '1':
			tf.SetToUUT(pos)
			pos++
		case 'L':
			tf.ResetFromUUT(pos)
			pos++
		case 'H':
			tf.SetFromUUT(pos)
			pos++
		case 'C':
			tf.SetClock(pos)
			pos++
		case 'X', 'G', 'V': // place holders
			tf.SetIgnored(pos)
			pos++
		case '%':
			// Set 16 bits of the vector from four digits of hex.
			// This and the next case are similar, but not quite
			// similar enough to bother making a function.
			v, err := strconv.ParseUint(t[1:5], 16, 16)
			if err != nil {
				return fmt.Errorf("parse %s: %v", t, err)
			}
			for i := 0; i < 16; i++ {
				if v&1 != 0 {
					tf.SetToUUT(pos)
				}
				pos++
				v >>= 1
			}
		case '@':
			// Set 16 bits reversed from four digits of hex.
			v, err := strconv.ParseUint(t[1:5], 16, 16)
			if err != nil {
				return fmt.Errorf("parse %s: %v", t, err)
			}
			v = uint64(bits.Reverse16(uint16(v)))
			for i := 0; i < 16; i++ {
				if v&1 != 0 {
					tf.SetFromUUT(pos)
				}
				pos++
				v >>= 1
			}
		default:
			return fmt.Errorf("unknown token %s", t)
		}
	}

	if int(pos) != tf.Size() {
		return fmt.Errorf("expected %d bits in vector, got %d", tf.Size(), pos)
	}

	return nil
}

// Apply the vector stored in the tf structure to the hardware.

func applyVector(tf *utils.TestFile) (int, error) {
	if tf.Socket() == "PLCC" {
		return applyPLCC(tf)
	} else if tf.Socket() == "ZIF" {
		return applyZIF(tf)
	} else {
		return 0, fmt.Errorf("unknown socket type %s", tf.Socket())
	}
}

func pinToPos(pin int) utils.BitPosition {
	return utils.BitPosition(pin - 1)
}

// Apply one vector, which is stored in the TestFile, to the hardware.
// Return a count of hardware failures and an error value. Hardware
// failures do not cause an "error".
func applyPLCC(tf *utils.TestFile) (int, error) {
	var b byte

	// Pins 1 - 8: U4:0..7
	if err := doSetCmd(fmt.Sprintf("s 4 %02X", tf.GetByteToUUT(0)), tf.Nano()); err != nil {
		return 0, err
	}

	// Pins 9 - 16: U5:0..7
	if err := doSetCmd(fmt.Sprintf("s 5 %02X", tf.GetByteToUUT(8)), tf.Nano()); err != nil {
		return 0, err
	}

	// Pins 17, 18, 19 - Clk, Vcc, and Gnd
	// Pins 20..24 Cout, P, G, Z, V outputs from UUT.
	// Pins 20 through 24 are inputs, captured below.
	// So tf.GetByteToUUT(16) is not a thing because
	// pin positions 17..24 are not "toUUT" pins.

	// Pins 25 - 27: U8:2..0 (bit reversed)
	// Pins 44 - 48: U8:3..7
	b = 0
	b |= byte(tf.GetToUUT(pinToPos(27)))      // ENF
	b |= byte(tf.GetToUUT(pinToPos(26))) << 1 // FTF
	b |= byte(tf.GetToUUT(pinToPos(25))) << 2 // OE#
	b |= byte(tf.GetToUUT(pinToPos(44))) << 3 // Cin to UUT
	b |= byte(tf.GetToUUT(pinToPos(45))) << 4 // S0
	b |= byte(tf.GetToUUT(pinToPos(46))) << 5 // S1
	b |= byte(tf.GetToUUT(pinToPos(47))) << 6 // S2
	b |= byte(tf.GetToUUT(pinToPos(48))) << 7 // OSA
	if err := doSetCmd(fmt.Sprintf("s 6 %02X", b), tf.Nano()); err != nil {
		return 0, err
	}

	// Pins 49 - 52: B10:3..0 (bit reversed)
	b = 0
	b |= byte(tf.GetToUUT(pinToPos(49))) << 4 // OSB
	b |= byte(tf.GetToUUT(pinToPos(50))) << 5 // FTAB
	b |= byte(tf.GetToUUT(pinToPos(51))) << 6 // ENB#
	b |= byte(tf.GetToUUT(pinToPos(52))) << 7 // ENA#
	if err := doSetCmd(fmt.Sprintf("s A %02X", b), tf.Nano()); err != nil {
		return 0, err
	}

	// Pins 53 - 60: B1:0..7 (B input low byte)
	b = 0
	shift := 0
	for i := pinToPos(53); i <= pinToPos(60); i++ {
		b |= byte(tf.GetToUUT(i)) << shift
		shift++
	}
	if err := doSetCmd(fmt.Sprintf("s 3 %02X", b), tf.Nano()); err != nil {
		return 0, err
	}

	// Pins 61 - 68: B2:0..7 (B input high byte)
	b = 0
	shift = 0
	for i := pinToPos(61); i <= pinToPos(68); i++ {
		b |= byte(tf.GetToUUT(i)) << shift
		shift++
	}
	if err := doSetCmd(fmt.Sprintf("s 2 %02X", b), tf.Nano()); err != nil {
		return 0, err
	}

	// All the static toUUT pins on the PLCC have been set. Now, the
	// ALU device may be used in a clocked way or combinationally.
	// If this vector is clocked, toggle PLCC Pin 17, which is wired
	// to TSTCLK, Nano toggle 8.
	if tf.HasClock() {
		doToggleCmd("t 1 8", tf.Nano())
	}

	// Read the chip's outputs through our inputs. We need to clock
	// each register with a toggle (t) command before we read it with
	// a get (g) command. The addresses are completely arbitrary. The
	// first argument to the t command is a count.

	// Pins 20 - 24: U11:0..4: clock bit 0xB, read port 0xC.
	// Carry out (c), carry propagate (p), carry generate (g), zero
	// flag (z), and overflow flag (v) plus three unused input bits.
	// U3/B3: clock pin 0 - high byte of F (result)
	// U7/B7: clock pin 7 - low byte of F

	if err := doToggleCmd("t 1 B", tf.Nano()); err != nil {
		return 0, err
	}
	if err := doToggleCmd("t 1 0", tf.Nano()); err != nil {
		return 0, err
	}
	if err := doToggleCmd("t 1 7", tf.Nano()); err != nil {
		return 0, err
	}
	cpgzvXXX, err := doGetCmd("g C", tf.Nano())
	if err != nil {
		return 0, err
	}
	bHigh, err := doGetCmd("g 1", tf.Nano())
	if err != nil {
		return 0, err
	}
	bLow, err := doGetCmd("g 9", tf.Nano())
	if err != nil {
		return 0, err
	}

	// We have the device outputs, whether clocked or combinational,
	// in cpgzvXXX, bHigh, and bLow.

	errorCount := 0
	got := int(bHigh)<<8 | int(bLow)
	expected := 0
	shift = 15
	for i := pinToPos(28); i < pinToPos(44); i++ {
		expected |= (tf.GetFromUUT(i) & 1) << shift
		shift--
	}
	if expected != got {
		log.Printf("  fail expected 0x%04X, got 0x%04X", expected, got)
		errorCount++
	}

	shift = 0
	names := "CPGZV"
	for i := pinToPos(20); i <= pinToPos(24); i++ {
		if (cpgzvXXX>>shift)&1 != byte(tf.GetFromUUT(i)) {
			// Since we don't chain ALU chips to make a 32-bit ALU, we
			// usually ignore the carry generate and propagate outputs.
			// Indent the error printf beneath the indented fail line
			// for the operation, if there was one.
			if tf.IsIgnored(i) == 0 {
				name := names[i-pinToPos(20)]
				log.Printf("    fail pin '%c' expected %d", name, tf.GetFromUUT(i))
				errorCount++
			}
		}
		shift++
	}

	return errorCount, nil
}

// Apply one vector, which is stored in the TestFile, to the hardware.
// Return a count of hardware failures and an error value. Hardware
// failures do not cause an "error".
func applyZIF(tf *utils.TestFile) (int, error) {
    // Pins 1 - 8: U5:0..7
    if err := doSetCmd(fmt.Sprintf("s 5 %02X", tf.GetByteToUUT(0)), tf.Nano()); err != nil {
        return 0, err
    }

	// The next 7 pins 9..14 are inputs except that pin 12 is ground.
	// The part is wired so the bits line up, e.g. bit 3 of the port
	// is not connected to anything so bit 4 is on pin 13.
	//
	// Circuits in 22v10s may be clocked or combinational. So there may
	// be a clock. If there is a clock, by my own convention it's on pin
	// 13, which is bit 3 of port U4. So we have to set this now, and then
	// toggle it.
	//
    // Pins 9 - 11, 13 - 15: U4:0..2, 4..6 (U4:3, U4:7 not connected)
	b := tf.GetByteToUUT(1)
	if (tf.HasClock()) {
		b |= 0x10
	}
    if err := doSetCmd(fmt.Sprintf("s 4 %02X", b), tf.Nano()); err != nil {
        return 0, err
    }

	if (tf.HasClock()) {
		b &^= 0x10
		if err := doSetCmd(fmt.Sprintf("s 4 %02X", b), tf.Nano()); err != nil {
			return 0, err
		}
		b |= 0x10
		if err := doSetCmd(fmt.Sprintf("s 4 %02X", b), tf.Nano()); err != nil {
			return 0, err
		}
	}

	// Pins 16 - 23: U11:0..4: clock bit 0xB, read port 0xC.
	if err := doToggleCmd("t 1 B", tf.Nano()); err != nil {
		return 0, err
	}
	results, err := doGetCmd("g C", tf.Nano())
	if err != nil {
		return 0, err
	}

	errorCount := 0
	shift := 0
	for i := pinToPos(15); i <= pinToPos(23); i++ {
		if (results>>shift)&1 != byte(tf.GetFromUUT(i)) {
			if tf.IsIgnored(i) == 0 {
				log.Printf("    fail pin %d expected %d", i, tf.GetFromUUT(i))
				errorCount++
			}
		}
		shift++
	}

	return 0, nil
}
