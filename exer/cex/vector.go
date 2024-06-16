package main

// The acronym "UUT" stands for "unit under test". We output the bits
// that control the UUT's inputs, and input the results that come from
// the UUT's outputs.

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// As conventionally used in this tester, component pin "n" (1..package_count)
// is at position (n-1) (0..package_count-1). Values in the range of 0..n-1
// are consistently called "positions".
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
	fbv.bitArray[theByte] |= theBit
}

// Clear the bit at the position and mark it "used"
func (fbv *FixedBitVec) Reset(bit BitPosition) {
	theByte := bit / 8
	theBit := byte(bit % 8)
	fbv.bitArray[theByte] &^= theBit
}

// Get the bit at position and return as a 1 or a 0
func (fbv *FixedBitVec) Get(bit BitPosition) byte {
	theByte := bit / 8
	theBit := byte(bit % 8)
	return fbv.bitArray[theByte] >> theBit
}

// Get the byte containing the bit argument
func (fbv *FixedBitVec) GetByte(bit BitPosition) byte {
	return fbv.bitArray[bit / 8]
}

// Each test file is parsed into of these. The scalar fields
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
// The getter panics if there is no clock pin defined.
type TestFile struct {
	socket string		 // "PLCC" or "ZIF"
	size int			 // number of bits, 0 .. size-1
	clockPin int         // PIN NUMBER 1..n of clock, or 0
	toUUT *FixedBitVec	 // bits that are UUT inputs
	fromUUT *FixedBitVec // bits that are UUT outputs
	used *FixedBitVec	 // bits that are used are set to 1
}

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
		used: NewFixedBitVec(size),
	}
}

func (tf *TestFile) SetToUUT(bit BitPosition) {
	tf.toUUT.Set(bit)
	tf.used.Set(bit)
}

func (tf *TestFile) SetFromUUT(bit BitPosition) {
	tf.fromUUT.Set(bit)
	tf.used.Set(bit)
}

func (tf *TestFile) ResetToUUT(bit BitPosition) {
	tf.toUUT.Reset(bit)
	// A zero bit is still "used"
	tf.used.Set(bit)
}

func (tf *TestFile) ResetFromUUT(bit BitPosition) {
	tf.fromUUT.Reset(bit)
	// A zero bit is still "used"
	tf.used.Set(bit)
}

// Set the clock pin. The argument is a position 0..n-1
func (tf *TestFile) SetClock(bit BitPosition) {
	tf.clockPin = 1 + int(bit)
}

// Get the clock pin. The argument is a position.
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
	tf.used = NewFixedBitVec(tf.size)
}

// Return true if this position is in use. It's used
// if it was ever set or cleared. There is no way to
// clear or "forget" the used state of a position.
func (tf *TestFile) Used(bit BitPosition) bool {
	return tf.used.Get(bit) != 0
}

const SPACE = ' '

// Process one vector file
func DoVectorFile(filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
		return err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
	var tf *TestFile
    for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' || line[0] == SPACE {
			continue
		}

		tokens := strings.Split(line, " ")
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

		// The tokens should be a line specifying 24 or 68 bits

		for _, s := range tokens {
			fmt.Println(s)
		}
    }

    if err := scanner.Err(); err != nil {
        return err
    }
	return nil
}
