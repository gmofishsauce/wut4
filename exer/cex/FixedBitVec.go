package main

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

