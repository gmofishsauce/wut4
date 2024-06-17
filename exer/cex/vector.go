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
	"math/bits" // Reverse16
	"os"
	"strconv"
	"strings"
)

const SPACE = ' '

// Process one vector file. The file format and structure
// are described in the README in this directory.
//
// TODO track line numbers else troubleshooting bad vectors
// will be next to impossible.
func DoVectorFile(filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
		return err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
	var tf *TestFile
    for scanner.Scan() {
		// First check for empty lines, comments, 'socket' statement
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' || line[0] == SPACE {
			// Note: we don't support tab as whitespace.
			continue
		}

		tokens := strings.Split(line, " ")

		// Handle the exactly-once-per-file "socket" statement
		if tokens[0] == "socket" {
			if tf != nil || len(tokens) != 2 {
				return fmt.Errorf("bad 'socket' statement")
			}
			tf = NewTestFile(tokens[1])
			if (tf == nil) {
				return fmt.Errorf("bad socket type")
			}
			continue
		}

		// Must be a vector or error
		tf.Clear()

		if err := parseVector(tf, tokens); err != nil {
			return err
		}
		if debug {
			fmt.Printf("%s\n", tf)
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
func parseVector(tf *TestFile, tokens []string) error {
	var pos BitPosition
	for _, t := range tokens {
		if len(t) == 0 {
			// Golang's split() function splits on every
			// occurrence of the split character, rather
			// than consuming groups of them. This just
			// means there were multiple spaces.
			continue
		}

		switch (t[0]) {
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

	if int(pos) != tf.size {
		return fmt.Errorf("expected %d bits in vector, got %d", tf.size, pos)
	}

	return nil
}

// Apply the vector stored in the tf structure to the hardware.

func applyVector(tf *TestFile) error {
	fmt.Printf("TODO: apply\n")
	return nil
}
