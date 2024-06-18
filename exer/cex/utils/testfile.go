package utils

import (
	"fmt"
	"strings"
)

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
	socket   string       // "PLCC" or "ZIF"
	size     int          // number of bits, 0 .. size-1
	clockPin int          // PIN NUMBER 1..n of clock, or 0
	nano     *Arduino     // open Arduino device
	toUUT    *FixedBitVec // bits that are UUT inputs
	fromUUT  *FixedBitVec // bits that are UUT outputs
}

// Allocate a test file object. The returned value may be defined
// as PLCC or ZIF. Otherwise, the allocator returns nil.
func NewTestFile(socket string, nano *Arduino) *TestFile {
	var size int
	if socket == "PLCC" {
		size = 68
	} else if socket == "ZIF" {
		size = 24
	} else {
		return nil
	}
	return &TestFile{
		socket:   socket,
		size:     size,
		clockPin: 0, // means "no clock"
		nano:     nano,
		toUUT:    NewFixedBitVec(size),
		fromUUT:  NewFixedBitVec(size),
	}
}

func (tf *TestFile) Size() int {
	return tf.size
}

func (tf *TestFile) Socket() string {
	return tf.socket
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
// We make the clock pin high so we can output the byte
// containing the pin. We then check for it and toggle
// it before reading the fromUUT() bits. In other words
// we assume a positive edge clock.
func (tf *TestFile) SetClock(bit BitPosition) {
	tf.clockPin = 1 + int(bit)
	tf.SetToUUT(bit)
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

func (tf *TestFile) GetByteFromUUT(bit BitPosition) {
	tf.fromUUT.GetByte(bit)
}

func (tf *TestFile) GetByteToUUT(bit BitPosition) {
	tf.toUUT.GetByte(bit)
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
