package main

// This is an IC component exerciser. It drives patterns of digital
// signals to component inputs and checks values from component outputs.
//
// The acronym "UUT" stands for "unit under test". We output the bits
// that control the UUT's inputs, and check the results that come from
// the UUT's outputs.

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
func DoVectorFile(filePath string, nano *dev.Arduino) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return scan(bufio.NewScanner(file), nano)
}

// Scan one vector file.
func scan(scanner *bufio.Scanner, nano *dev.Arduino) error {
	var tf *utils.TestFile
	for scanner.Scan() {
		// First check for empty lines and comments. Lines must
		// be left-justified. Lines starting with spaces are empty.
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' || line[0] == SPACE {
			continue
		}

		tokens := strings.Split(line, string(SPACE))

		// Handle the exactly-once-per-file "socket" statement
		if tokens[0] == "socket" {
			if tf != nil || len(tokens) != 2 {
				return fmt.Errorf("bad 'socket' statement")
			}
			tf = utils.NewTestFile(tokens[1], nano)
			if tf == nil {
				return fmt.Errorf("bad socket type")
			}
			continue
		}

		// Must be a vector or error. Clear the per-vector
		// fields in the TestFile. Sort of ugly, meh.
		tf.Clear()

		if err := parseVector(tf, tokens); err != nil {
			return err
		}
		if debug {
			log.Printf("%s\n", tf)
		}
		if err := scanner.Err(); err != nil {
			return err
		}

		// Successfully parsed one vector; apply it
		if err := applyVector(tf); err != nil {
			return err
		}
	}

	return nil
}

// Parse the next vector from the file into the bit vectors in tf.
// The tokens should be a line specifying exactly 24 or 68 bits as
// specified in the socket statement and stored in the size field.
func parseVector(tf *utils.TestFile, tokens []string) error {
	var pos utils.BitPosition
	for _, t := range tokens {
		if len(t) == 0 {
			// Golang's split() function splits on every
			// occurrence of the split character, rather
			// than consuming groups of them. This just
			// means there were multiple spaces.
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
		case 'X', 'G', 'V': // place holders
			pos++
		case '%':
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

func applyVector(tf *utils.TestFile) error {
	if tf.Socket() == "PLCC" {
		return applyPLCC(tf)
	} else if tf.Socket() == "ZIF" {
		return applyZIF(tf)
	} else {
		return fmt.Errorf("unknown socket type %s", tf.Socket())
	}
}

func pinToPos(pin int) utils.BitPosition {
	return utils.BitPosition(pin - 1)
}

// Apply one vector, which is stored in the TestFile, to the hardware.
// Return the results.
func applyPLCC(tf *utils.TestFile) error {
	// Pins 1 - 8: U4:0..7
	if err := doSetCmd(fmt.Sprintf("s 4 %02X", tf.GetByteToUUT(0)), tf.Nano()); err != nil {
		return err
	}

	// Pins 9 - 16: U5:0..7
	if err := doSetCmd(fmt.Sprintf("s 5 %02X", tf.GetByteToUUT(8)), tf.Nano()); err != nil {
		return err
	}

	// Pins 17, 18, 19 - Clk, Vcc, and Gnd
	// Pins 20 through 24 are inputs, captured below.
	// So tf.GetByteToUUT(16) is not a thing because
	// pin positions 17..24 are not controls or data.

	// Pins 25 - 27: U8:2..0 (bit reversed)
	// Pins 44 - 48: U8:3..7
	// We have to construct the entire U8/B8 byte from the various
	// vector bits that drive it; this one is mis-ordered enough
	// that it's much easier to do it a bit at a time.
	var b byte
	b |= byte(tf.GetToUUT(pinToPos(27)))
	b |= byte(tf.GetToUUT(pinToPos(26))) << 1
	b |= byte(tf.GetToUUT(pinToPos(25))) << 2
	b |= byte(tf.GetToUUT(pinToPos(44))) << 3
	b |= byte(tf.GetToUUT(pinToPos(45))) << 4
	b |= byte(tf.GetToUUT(pinToPos(46))) << 5
	b |= byte(tf.GetToUUT(pinToPos(47))) << 6
	b |= byte(tf.GetToUUT(pinToPos(48))) << 7
	if err := doSetCmd(fmt.Sprintf("s 6 %02X", b), tf.Nano()); err != nil {
		return err
	}

	// Pins 49 - 52: B10:3..0 (bit reversed)
	b = 0
	b |= byte(tf.GetToUUT(pinToPos(49))) << 4
	b |= byte(tf.GetToUUT(pinToPos(50))) << 5
	b |= byte(tf.GetToUUT(pinToPos(51))) << 6
	b |= byte(tf.GetToUUT(pinToPos(52))) << 7
	if err := doSetCmd(fmt.Sprintf("s A %02X", b), tf.Nano()); err != nil {
		return err
	}

	// Pins 53 - 60: B1:0..7
	b = 0
	shift := 0
	for i := pinToPos(53); i <= pinToPos(60); i++ {
		b |= byte(tf.GetToUUT(i)) << shift
		shift++
	}
	if err := doSetCmd(fmt.Sprintf("s 3 %02X", b), tf.Nano()); err != nil {
		return err
	}

	// Pins 61 - 68: B2:0..7
	b = 0
	shift = 0
	for i := pinToPos(61); i <= pinToPos(68); i++ {
		b |= byte(tf.GetToUUT(i)) << shift
		shift++
	}
	if err := doSetCmd(fmt.Sprintf("s 2 %02X", b), tf.Nano()); err != nil {
		return err
	}

	// if this vector is clocked, toggle PLCC Pin 17,
	// which is wired to TSTCLK, Nano toggle 8
	if tf.HasClock() {
		doToggleCmd("t 1 8", tf.Nano())
	}

	// Read the chip's outputs through our inputs. We need to clock
	// each register with a toggle (t) command before we read it.

	// Pins 20 - 24: U11:0..4
	// Carry out (c), carry propagate (p), carry generate (g), zero
	// flag (z), and overflow flag (v) plus three unused input bits.
	if err := doToggleCmd("t 1 B", tf.Nano()); err != nil {
		return err
	}
	cpgzvXXX, err := doGetCmd("g C", tf.Nano())
	if err != nil {
		return err
	}
	b = 0
	shift = 0
	for i := pinToPos(20); i <= pinToPos(24); i++ {
		if (cpgzvXXX>>shift)&1 != byte(tf.GetFromUUT(i)) {
			log.Printf("  fail pin %d expected %d", 1+i, tf.GetFromUUT(i))
			shift++
		}
	}

	// 16 bit result on input ports B3 (MS byte) and B7 (LS byte).
	// The pins are conceptually bit reversed (high order bit on
	// low number pin).
	// Pins 28 - 35: B3:7..0
	// Pins 36 - 43: B7:7..0
	if err := doToggleCmd("t 1 0", tf.Nano()); err != nil {
		return err
	}
	if err := doToggleCmd("t 1 7", tf.Nano()); err != nil {
		return err
	}
	bHigh, err := doGetCmd("g 1", tf.Nano())
	if err != nil {
		return err
	}
	bLow, err := doGetCmd("g 9", tf.Nano())
	if err != nil {
		return err
	}
	log.Printf("  read result 0x%04X", int(bHigh) << 8 | int(bLow))

	return nil
}

func applyZIF(tf *utils.TestFile) error {
	log.Println("applyZIF()")
	return nil
}
