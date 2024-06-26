// Copyright (c) Jeff Berkowitz 2021. All rights reserved.
//
// 456789012345678901234567890123456789012345678901234567890123456789012
//
// This file was split out of port_task.h because it got too large.
//
// There have been two version of this code. The first version used the
// Arduino library (digitalWrite, pinMode, etc.) while the second version,
// below, uses direct referencs to the ATmega328P's internal registers.
// Ths second version saved a couple of thousand bytes of program memory
// and runs things like a full scan of memory more than 10 times as fast.
//
// This change introduced ambiguity into the word "register". Originally,
// the word referred to the registers I've constructed outside the Nano.
// Now, it may refer to these or to the ATmega328P's internal registers
// (PORTB, PINB, DDRB, etc.) used to manipulate the Nano's external pins.
// This is unavoidable and must be resolved by context.
//
// There is also the standard Arduino confusion over the concept of "pins".
// There are 30 physical pins on the Arduino Nano, and the "data port" I've
// defined, for example, is on physical pins 8..15. But Arduino defines a
// sort of "logical pin" pin concept across all the many Arduino board
// types. So these pins of the "data port" are labelled D5, D6, ..., D12
// on the silkscreen and in most figures documenting the pins.
//
// The logical pin numbers 5, 6, ..., 12 are used with the Arduino library
// functions like pinMode() and digitalWrite(). But since this code has
// been rewritten to use the ATmega's internal registers, the logical pin
// numbers are no longer used at all. Instead, there is a mapping from the
// ATmega's internal registers (used for manipulating the pins) to the
// physical pins.
//
// I've defined two "ports" for communicating with external registers.
// The "data port" is physical pins 8..15 on the Nano and the "select
// port" is physical pins 19..21 and 22,23. The data port is used in
// both read and write mode. It drives (or receives from) the internal
// Nano I/O bus to all the external registers. The select port is used
// to clock and/or enable these registers and to directly control the
// IC under test. Here are the mappings:
//
// Internal register  Port Name     Physical pin on Nano (1..30)
// ~~~~~~~~~~~~~~~~~  ~~~~~~~~~     ~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// PORTB 5:7          Data 0..2     5..7
// PORTD 0:4          Data 3..7     8..12
// PORTC 0:2          Select 0..2   19..21
//
// Physical pins 22 and 23 (PORTC:3 and PORTC:4) are used to "strobe" the
// decoder line select by the select port, which is bussed to two decoders.
//
// Writing to an external register requires coordinating both ports. First,
// the data port must be switched to output and a value set on its pins.
// Next, the select port must be set the index of 1 of 8 output strobes on
// the decoders (their three A-lines are bus-connected). Finally, one of the
// two decoders must be enabled, then disabled, via physical pins 22 or 23,
// producing a low-going pulse that ends with a rising edge to clock one of
// the output registers.
//
// Read is similar, except the port must be set to input and the actual read
// of the port must occur while the enable line is low, since the enable line
// is connected to output enable pin on register or transceiver that drives
// the Nano's internal bus.
//
// We don't manage the LED port here, because its port assignment (pin 13)
// is pretty standard across all Arduinos and clones. We leave that to a
// separate purpose-built LED task that can play various patterns.


const byte NOT_PIN = 0;
typedef const byte PinList[];

// Identifiers for the data port (Arduino logical pins 5, 6, ... 12) and the
// select port (Arduino logical pins 14, 15, 16, plus 17 and 18 as explained
// below).
//
// In the original version of this code which used digitalWrite(), the pin
// values were were essential and were stored in these two arrays:
//
// PinList portData   = {5, 6, 7, 8, 9, 10, 11, 12, NOT_PIN};
// PinList portSelect = {14, 15, 16, NOT_PIN};
//
// In the new Nano-specific version of this code which writes directly to
// the ATmega's PORTB, PORTC, and PORTD registers, the Nano pin numbers
// no longer referenced; but there is still code that requires portData
// and portSelect be defined as -something- for logical tests. So we use
// empty arrays to save a little space.

PinList portData = {};
PinList portSelect = {};

// Outside the Nano there are two 3-to-8 decoder chips, providing a total
// of 16 pulse outputs. The pulse outputs are used to clock input and output
// registers, enable input registers to the Nano's I/O bus, and as direct
// controls to the IC under test.
//
// The three bit address on the decoders is bused from the Nano to both
// decoders. But there are two distinct select pins, one for each decoder,
// allowing for a 17th state where none of the 16 pulse outputs are active.
//
// As a result there are two ways of representing the "address" of one of
// the pulse outputs. In both representations, bits 2:0 go to the address
// input lines (A-lines) of both decoders. But in the REGISTER_ID, bit 3
// is 0 for the "low" (0-7) decoder and 1 for the "high" (8-15) decoder.
// Later, which doing the actual I/O operation, we need to either toggle
// PORTC:3 for the low decoder, or PORTC:4 for the high decoder.
//
// Finally, note that the two toggles run to the active HIGH enable inputs
// of the decoder chips. This was done because the Nano initializes output
// pins to LOW by default. But the active HIGH enables cause negative-going
// pulses on the decoder outputs, because that's how 74XX138s work, always.

constexpr int PIN_SELECT_0_7 = _BV(PORTC3);
constexpr int PIN_SELECT_8_15 = _BV(PORTC4);
constexpr int BOTH_DECODERS = (PIN_SELECT_0_7 | PIN_SELECT_8_15);
      
constexpr byte DECODER_ADDRESS_MASK = 7;
constexpr byte DECODER_SELECT_MASK  = 8;

typedef byte REGISTER_ID;

// Addresses on low decoder
constexpr byte B3_CLK = 0;              // Input port
constexpr byte B3_OE = 1;               // Read input
constexpr byte B2_CLK = 2;              // B2_OE is a port bit
constexpr byte B1_CLK = 3;              // B1_OE is a port bit
constexpr byte B4_CLK = 4;              // Output is always enabled
constexpr byte B5_CLK = 5;              // Output is always enabled
constexpr byte B8_CLK = 6;              // B8_OE is a port bit
constexpr byte B7_CLK = 7;              // Input port

// Addresses on high decoder
constexpr byte TSTCLK = 0;              // Clock the unit under test
constexpr byte B7_OE = 1;               // Read input
constexpr byte B10_CLK = 2;             // Output is always enabled
constexpr byte B11_CLK = 3;             // Input port
constexpr byte B11_OE = 4;              // Read input
constexpr byte UN_HI_5 = 5;             // unused        
constexpr byte UN_HI_6 = 6;             // unused          
constexpr byte UN_HI_7 = 7;             // unused          

// Register IDs on low decoder are just their address
constexpr REGISTER_ID RI_B3_CLK = B3_CLK;
constexpr REGISTER_ID RI_B3_OE = B3_OE;
constexpr REGISTER_ID RI_B2_CLK = B2_CLK;
constexpr REGISTER_ID RI_B1_CLK = B1_CLK;
constexpr REGISTER_ID RI_B4_CLK = B4_CLK;
constexpr REGISTER_ID RI_B5_CLK = B5_CLK;
constexpr REGISTER_ID RI_B8_CLK = B8_CLK;
constexpr REGISTER_ID RI_B7_CLK = B7_CLK;

// Register IDs on high decoder need bit 3 set
constexpr REGISTER_ID RI_TSTCLK = DECODER_SELECT_MASK|TSTCLK;
constexpr REGISTER_ID RI_B7_OE = DECODER_SELECT_MASK|B7_OE;
constexpr REGISTER_ID RI_U10_CLK = DECODER_SELECT_MASK|B10_CLK;
constexpr REGISTER_ID RI_U11_CLK = DECODER_SELECT_MASK|B11_CLK;
constexpr REGISTER_ID RI_U11_OE = DECODER_SELECT_MASK|B11_OE;

constexpr byte getAddressFromRegisterID(REGISTER_ID reg) {
  return reg & DECODER_ADDRESS_MASK;
}

// This function returns the select pin for a register ID as above.
constexpr int getDecoderSelectPinFromRegisterID(REGISTER_ID reg) {
  return (reg & DECODER_SELECT_MASK) ? PIN_SELECT_8_15 : PIN_SELECT_0_7;
}

// === start of lowest level code for writing to ports ===

// For convenience, some buses may be wired backwards. This function allows
// us to reverse bits. The table is in program memory, which we have plenty of.
// The code came from StackOverflow and probably cannot be covered by copyright.

byte reverse_byte(byte b) {
  static const PROGMEM byte table[] = {
      0x00, 0x80, 0x40, 0xc0, 0x20, 0xa0, 0x60, 0xe0,
      0x10, 0x90, 0x50, 0xd0, 0x30, 0xb0, 0x70, 0xf0,
      0x08, 0x88, 0x48, 0xc8, 0x28, 0xa8, 0x68, 0xe8,
      0x18, 0x98, 0x58, 0xd8, 0x38, 0xb8, 0x78, 0xf8,
      0x04, 0x84, 0x44, 0xc4, 0x24, 0xa4, 0x64, 0xe4,
      0x14, 0x94, 0x54, 0xd4, 0x34, 0xb4, 0x74, 0xf4,
      0x0c, 0x8c, 0x4c, 0xcc, 0x2c, 0xac, 0x6c, 0xec,
      0x1c, 0x9c, 0x5c, 0xdc, 0x3c, 0xbc, 0x7c, 0xfc,
      0x02, 0x82, 0x42, 0xc2, 0x22, 0xa2, 0x62, 0xe2,
      0x12, 0x92, 0x52, 0xd2, 0x32, 0xb2, 0x72, 0xf2,
      0x0a, 0x8a, 0x4a, 0xca, 0x2a, 0xaa, 0x6a, 0xea,
      0x1a, 0x9a, 0x5a, 0xda, 0x3a, 0xba, 0x7a, 0xfa,
      0x06, 0x86, 0x46, 0xc6, 0x26, 0xa6, 0x66, 0xe6,
      0x16, 0x96, 0x56, 0xd6, 0x36, 0xb6, 0x76, 0xf6,
      0x0e, 0x8e, 0x4e, 0xce, 0x2e, 0xae, 0x6e, 0xee,
      0x1e, 0x9e, 0x5e, 0xde, 0x3e, 0xbe, 0x7e, 0xfe,
      0x01, 0x81, 0x41, 0xc1, 0x21, 0xa1, 0x61, 0xe1,
      0x11, 0x91, 0x51, 0xd1, 0x31, 0xb1, 0x71, 0xf1,
      0x09, 0x89, 0x49, 0xc9, 0x29, 0xa9, 0x69, 0xe9,
      0x19, 0x99, 0x59, 0xd9, 0x39, 0xb9, 0x79, 0xf9,
      0x05, 0x85, 0x45, 0xc5, 0x25, 0xa5, 0x65, 0xe5,
      0x15, 0x95, 0x55, 0xd5, 0x35, 0xb5, 0x75, 0xf5,
      0x0d, 0x8d, 0x4d, 0xcd, 0x2d, 0xad, 0x6d, 0xed,
      0x1d, 0x9d, 0x5d, 0xdd, 0x3d, 0xbd, 0x7d, 0xfd,
      0x03, 0x83, 0x43, 0xc3, 0x23, 0xa3, 0x63, 0xe3,
      0x13, 0x93, 0x53, 0xd3, 0x33, 0xb3, 0x73, 0xf3,
      0x0b, 0x8b, 0x4b, 0xcb, 0x2b, 0xab, 0x6b, 0xeb,
      0x1b, 0x9b, 0x5b, 0xdb, 0x3b, 0xbb, 0x7b, 0xfb,
      0x07, 0x87, 0x47, 0xc7, 0x27, 0xa7, 0x67, 0xe7,
      0x17, 0x97, 0x57, 0xd7, 0x37, 0xb7, 0x77, 0xf7,
      0x0f, 0x8f, 0x4f, 0xcf, 0x2f, 0xaf, 0x6f, 0xef,
      0x1f, 0x9f, 0x5f, 0xdf, 0x3f, 0xbf, 0x7f, 0xff,
    };
  // Get the reversed by from program (flash) memory
    return pgm_read_ptr_near(&table[b]);
}

// Set the data port to the byte b. The data port is made from pieces of
// the Nano's internal PORTB and PORTD.
void nanoPutDataPort(byte b) {
  // The "data port" is made of Nano physical pins 8 through 15. The
  // three low order bits are in Nano PORTD. The five higher order are
  // in the low-order bits of PORTB.
  // First set PD5, PD6, and PD7 to the three low order bits of b.
  int portDlowOrderBits = PORTD & 0x1F; // note: may sign extend
  PORTD = byte(portDlowOrderBits | ((b & 0x07) << 5));

  // Now set the low order 5 bits of PORTB to the high 5 bits of b.
  // These PORTB outputs appear on pins 11 through 15 inclusive.
  int portBhighOrderBits = PORTB & 0xE0; // note: may sign extend
  PORTB = byte(portBhighOrderBits | ((b & 0xF8) >> 3));
}

// Set PORTC bits 0, 1, and 2 to the three-bit address of one of eight
// outputs on a 74HC138 decoder. Do not change the 5 high order bits of
// PORTC. The choice of which decoder is made seprately in nanoTogglePort().
void nanoPutSelectPort(byte b) {
    int portChighOrder5bits = PORTC & 0xF8; // note: may sign extend
    PORTC = byte(portChighOrder5bits | (b & 0x07));
}

void nanoPutPort(PinList port, int value) {
  if (port == portData) {
    nanoPutDataPort(value);
  } else {
    nanoPutSelectPort(value);
  }
}

// We take advantage of the fact that we only ever call get()
// on the data port.
byte nanoGetPort(PinList port) {
  // The "data port" is made of Nano physical pins 8..15. The three
  // low order bits are in ATmega PORTD. The five higher order, PORTB.
  // First get PD7:5 and put them in the low order bits of the result.
  byte portDbits = (PIND >> 5) & 0x07;
  byte portBbits = (PINB & 0x1F) << 3;
  return byte(portDbits | portBbits);
}

// Set the data port to be output or input. Delays in this file are
// critical and must not be altered; some of them handle documented
// issues with the ATmega, and some handle registrictions imposed by
// the design of the external registers. This one is the first kind.
void nanoSetDataPortMode(int mode) {
    if (mode == OUTPUT) {
      DDRD = DDRD | 0xE0;
      DDRB = DDRB | 0x1F;
    } else {
      DDRD = DDRD & ~0xE0;
      DDRB = DDRB & ~0x1F;
    }
    delayMicroseconds(2);
}

// Set the select port to be output (it's always output). Again,
// delays in this file are critical and must not be altered.
void nanoSetSelectPortMode(int mode) {
  DDRC |= DDRC | 0x07;
  delayMicroseconds(2);
}

void nanoSetMode(PinList port, int mode) {
  if (port == portData) {
    nanoSetDataPortMode(mode);
  } else {
    nanoSetSelectPortMode(mode);
  }
}

// This is a critical function that serves to pulse one of the 16
// decoder outputs. To do this, it has to put the 3-bit address of
// one of 8 data ports on to the 3-bit select port which is bussed
// to the address (A) lines of the decoders. Then it has to enable
// the correct decoder by togging either PORTC:3 or PORTC:4. One
// of these values is returned by getDecoderSelectPinFromRegisterID().
void nanoTogglePulse(REGISTER_ID reg) {
  // Bug fix (although no symptoms were ever seen): to prevent glitches
  // and overlap on busses, we must disable both decoders before enabling
  // either one.
  PORTC &= ~BOTH_DECODERS;
  
  byte decoderAddress = getAddressFromRegisterID(reg);
  nanoPutPort(portSelect, decoderAddress);
  
  byte decoderEnablePin = getDecoderSelectPinFromRegisterID(reg);
  PORTC = PORTC | decoderEnablePin;
  PORTC = PORTC & ~decoderEnablePin;
}

#if 0 
// This function is only for use during debugging. It causes a toggle
// to instead go low and stay that way.
void nanoStartToggle(REGISTER_ID reg) {
  PORTC &= ~BOTH_DECODERS;
  
  byte decoderAddress = getAddressFromRegisterID(reg);
  nanoPutPort(portSelect, decoderAddress);
  
  byte decoderEnablePin = getDecoderSelectPinFromRegisterID(reg);
  PORTC = PORTC | decoderEnablePin;    
}
#endif

// Enable the specified register for input and call getPort() to read
// the value. We cannot use nanoTogglePulse() here because we have to
// read the value after setting the enable line low and before setting
// it high again. As always, the delays are the result of careful
// experimentation and are absolutely required.
byte nanoGetRegister(REGISTER_ID reg) {    
  byte decoderAddress = getAddressFromRegisterID(reg);
  nanoPutPort(portSelect, decoderAddress);
  
  nanoSetMode(portData, INPUT);
  
  byte result;
  byte decoderEnablePin = getDecoderSelectPinFromRegisterID(reg);
  PORTC |= decoderEnablePin;
  delayMicroseconds(2);
  result = nanoGetPort(portData);
  PORTC &= ~decoderEnablePin;
  
  nanoSetMode(portData, OUTPUT);
  return result;
}

void nanoSetRegister(REGISTER_ID reg, byte data) {
  // The low order 4 bits of U10 must always stay low for now, because they
  // are output enables for other output registers. Setting them high would
  // make it possible to share some test lines as either inputs or outputs.
  // This would require additional hardware that is in the design but is not
  // implemented.
  if (reg == RI_U10_CLK) {
    data &= 0xF0;
  }

  // Some ports are bit reversed as a wiring convenience.
  if (reg == RI_U10_CLK) {
    data = reverse_byte(data);
  }

  nanoSetMode(portData, OUTPUT);
  nanoPutPort(portData, data);    
  nanoTogglePulse(reg);
}

