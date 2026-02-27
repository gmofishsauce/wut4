// nio.yapl - WUT-4 Native I/O library
//
// Console I/O and SPI SD card access for privileged (kernel) code.
// Compile to a relocatable object and link with programs that need I/O:
//
//   ya -c -o lib/nio.wo lib/nio.yapl
//
// Console UART (SPR assignments):
//   SPR 96:  TX data (write)
//   SPR 97:  RX data (read)
//   SPR 98:  TX status (bit 0=overflow, bit 15=empty)
//   SPR 99:  RX status (bit 0=underflow, bit 15=data available)
//
// SPI SD card (SPR assignments):
//   SPR 100: SPI data (write=send, read=receive)
//   SPR 101: SPI chip select (write)
// --- Putc: write one byte to the console TX FIFO ---

extern func void Putc(byte b);


// --- Putstr: write null-terminated string to the console ---

extern func void Putstr(@byte bp);

// --- Getc: non-blocking console read ---
// return 0 if nothing available

extern func int16 Getc();

// --- Getstr: blocking read of one line into a buffer ---
// Reads characters until a newline is received or maxlen-1 characters

extern func void Getstr(@byte buf, uint16 maxlen);

// --- SdInit: full SD card initialization sequence ---
// Returns 0 on success, or error code 1-5 on failure.

extern func byte SdInit();

// --- SdReadSec: read one 512-byte SD sector into memory ---
// addr: SD card sector byte address
// dest: destination memory address
// Returns 0 on success, nonzero on error.

extern func byte SdReadSec(uint16 addr, uint16 dest);
