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

// As conventionally used in this tester, component pin "n" (1..package_count)
// is at position (n-1) (0..package_count-1) in bit vectors. Values in the
// range of 0..n-1 are consistently called "bit positions".
type BitPosition int

// Fixed-length bit vector with byte extraction (because the bits map to
// byte-wide ports). I looked at the Golang bitvector type but decided it
// wasn't suitable. The unit of size is bits. The implementation may
// overallocate slightly. Panic: BitPosition lies outside the allocation.
type FixedBitVec struct {
	bitArray []byte
}

func NewFixedBitVec(size int) *FixedBitVec {
	// This allocates one extra byte when the number of bits
	// is a multiple of 8 but it's by far the simplest answer.
	n := 1 + size/8
	return &FixedBitVec{ bitArray: make([]byte, n, n) }
}

// Set the bit at the position and mark it "used"
func (fbv *FixedBitVec) Set(bit BitPosition) {
	theByte := bit / 8
	theBit := byte(bit % 8)
	fbv.bitArray[theByte] |= (1<<theBit)
}

// Clear the bit at the position and mark it "used"
func (fbv *FixedBitVec) Reset(bit BitPosition) {
	theByte := bit / 8
	theBit := byte(bit % 8)
	fbv.bitArray[theByte] &^= (1<<theBit)
}

// Get the bit at position and return as a 1 or a 0
func (fbv *FixedBitVec) Get(bit BitPosition) int {
	theByte := bit / 8
	theBit := byte(bit % 8)
	return int((fbv.bitArray[theByte] >> theBit) & 1)
}

// Get the byte containing the bit argument
func (fbv *FixedBitVec) GetByte(bit BitPosition) byte {
	return fbv.bitArray[bit / 8]
}

// Each test file is parsed into one of these. The scalar fields
// are set per file. The bit vectors are cleared (dropped) and
// recreated per test vector within the file.
//
// The naming of the bit vectors that hold the bits we set and
// later get is hard. We output to the inputs of the chip under
// test, and then input from its outputs to check the results.
// So I've avoided "in", "out", set, get, etc. in the naming
// and gone for the explicit names "toUUT" and "fromUUT".
//
// The clock pin is special. It is stored in the structure as
// a value from 1 to n. Internally, 0 means "no clock" (i.e. the
// test vector is combinational). The caller must use HasClock()
// to determine whether the vector is combinational or clocked.
// The setter and getter for the clock pin return the position.
// The clock pin getter panics if there is no clock pin defined.
type TestFile struct {
	socket string		 // "PLCC" or "ZIF"
	size int			 // number of bits, 0 .. size-1
	clockPin int         // PIN NUMBER 1..n of clock, or 0
	toUUT *FixedBitVec	 // bits that are UUT inputs
	fromUUT *FixedBitVec // bits that are UUT outputs
}

// Allocate a test file object. The returned value may be defined
// as PLCC or ZIF. Otherwise, the allocator returns nil.
func NewTestFile(socket string) *TestFile {
	var size int
	if (socket == "PLCC") {
		size = 68
	} else if (socket == "ZIF") {
		size = 24
	} else {
		return nil
	}
	return &TestFile {
		socket: socket,
		size: size,
		clockPin: 0, // means "no clock"
		toUUT: NewFixedBitVec(size),
		fromUUT: NewFixedBitVec(size),
	}
}

func (tf *TestFile) SetToUUT(bit BitPosition) {
	tf.toUUT.Set(bit)
}

func (tf *TestFile) SetFromUUT(bit BitPosition) {
	tf.fromUUT.Set(bit)
}

func (tf *TestFile) ResetToUUT(bit BitPosition) {
	tf.toUUT.Reset(bit)
}

func (tf *TestFile) ResetFromUUT(bit BitPosition) {
	tf.fromUUT.Reset(bit)
}

func (tf *TestFile) GetToUUT(bit BitPosition) int {
	return tf.toUUT.Get(bit)
}

func (tf *TestFile) GetFromUUT(bit BitPosition) int {
	return tf.fromUUT.Get(bit)
}

// Set the clock pin. The argument is a position 0..n-1
func (tf *TestFile) SetClock(bit BitPosition) {
	tf.clockPin = 1 + int(bit)
}

// Get the clock pin. The result is a position.
// Panic: no clock pin. Use HasClock() first.
func (tf *TestFile) GetClock() BitPosition {
	if tf.clockPin == 0 {
		panic("no clock pin")
	}
	return BitPosition(tf.clockPin - 1)
}

func (tf *TestFile) HasClock() bool {
	return tf.clockPin != 0
}

// Clear (reallocate) the bit vectors. StackOverflow says that
// reallocation is faster than clearing, at least until/unless
// the volume of data forces heavy GC (which is hard to measure).
func (tf *TestFile) Clear() {
	tf.clockPin = 0 // default none
	tf.toUUT = NewFixedBitVec(tf.size)
	tf.fromUUT = NewFixedBitVec(tf.size)
}

func (tf *TestFile) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s-%d", tf.socket, tf.size))
	if tf.HasClock() {
		sb.WriteString(fmt.Sprintf(" clk:%d", tf.GetClock()))
	}
	var i BitPosition
	for i = 0; i < BitPosition(tf.size); i++ {
		if tf.GetFromUUT(i) > 0 || tf.GetToUUT(i) > 0 {
			sb.WriteString(" 1")
		} else {
			sb.WriteString(" 0")
		}
	}
	return sb.String()
}

const SPACE = ' '

// Process one vector file. Meaningful tokens must be separated
// by spaces. Tabs are not allowed.
func ParseVectorFile(filePath string) (*TestFile, error) {
    file, err := os.Open(filePath)
    if err != nil {
		return nil, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
	var tf *TestFile
    for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' || line[0] == SPACE {
			continue
		}

		// Handle the exactly-once-per-file "socket" statement
		tokens := strings.Split(line, " ")
		if tokens[0] == "socket" {
			if tf != nil || len(tokens) != 2 {
				return nil, fmt.Errorf("bad 'socket' statement")
			}
			tf = NewTestFile(tokens[1])
			if (tf == nil) {
				return nil, fmt.Errorf("bad socket type")
			}
			continue
		}

		// The tokens should be a line specifying 24 or 68 bits
		var pos BitPosition
		for _, t := range tokens {
			if len(t) == 0 {
				// Golang's split() function splits on every
				// occurrence of the split character, rather
				// than consuming groups of them. This just
				// means multiple spaces.
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
					return nil, fmt.Errorf("parse %s: %v", t, err)
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
					return nil, fmt.Errorf("parse %s: %v", t, err)
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
				return nil, fmt.Errorf("unknown token %s in vector file %s", t, filePath)
			}
		}
    }

    if err := scanner.Err(); err != nil {
        return nil, err
    }
	return tf, nil
}
