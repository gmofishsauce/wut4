# Host software for Nano-based chip exerciser

The **cex** (chip exerciser) has an extremely simple-minded interactive command set and a test vector command set.

The interactive mode prompts for input. There are three commands: **t** for toggle a control line, **s** for set an output register, and **g** for get an input register. This mode presents the hardware "as wired", meaning the **id** of a control line, input port, or output port is unrelated to its name. For example, `t 0` clocks input register U3 which is driven by bus B3, and `g 9` gets the value from input register U7 (B7). This mode is intended mostly for debugging the hardware.

The vector mode is entered by specifying one or more vector files on the command line. The vector file format (vector language) is described below. Vector mode is noninteractive. In vector mode, **cex** reads and applies each test vector, reports results to the standard output, and then reads the next file named on the command line, if any.

## ID Assignment (control signal wiring)

- 0x0 Clocks input register U3
- 0x1 Reads value of input register U3 (see below)
- 0x2 Clocks output register U2
- 0x3 Clocks output register U1
- 0x4 Clocks output register U4
- 0x5 Clocks output register U5
- 0x6 Clocks output register U8
- 0x7 Clocks input register U7
- 0x8 TSTCLK signal to PLCC68 pin 17 and ZIF pin 1
- 0x9 Reads value of input register U7 (see below)
- 0xA Clocks output register U10
- 0xB Clocks input register U11
- 0xC Reads value of input register U11 (see below)
- 0xD Clocks output register U14 (no connection; U14 is not implemented)
- 0xE No connection
- 0xF No connection

## Interactive language

The interactive command language is intended for experimentation and for testing the exerciser hardware. Actual chip testing should use the vector language described later.

- **t count id**: toggle (pulse) the output id (0..0xF) **count** times.
- **s id data**: set the port id (0..0xF) to the data (0..0xFF)
- **sr id data**: like **s** except the data value is bit reversed before being written to the port.
- **g id**: return the contents of the port id (0..0xF). The value is written to the standard output.
- **gr id**: Like **g** except the value is bit reversed before being returned to the host.

The input registers should not be enabled using **t** commands as this will cause conflicts on the Nano's IO bus. Instead, first clock data into the register using `t 0`, `t 7`, or `t B` and then use `g 1`, `g 9`, or `g C` (or their bit-reversed equivalents) to get the data.

## Vector language

Empty lines in the file are allowed. Files starting with any whitespace character are considered empty (all nonempty lines must be left-justified). Lines beginning with hash are commentary. Comments must be left justified (trailing comments are not allowed).

The first non-empty non-comment line must contain the keyword **socket** followed by exactly one space. This must be followed by **PLCC** to select the 68-pin PLCC socket in the exerciser or **ZIF** to select the 40-pin ZIF socket in the exerciser.

The rest of the files lines are vector lines. Each vector line specifies one test case. The test case may be combinational or clocked.

### Combinational circuits

Combinational test cases use the values 0 or 1 to specify inputs and the symbols H, L, or X to specify outputs. All inputs and outputs must be specified. The exerciser is incapable of verifying three-state or open collector outputs. If the component has a clock pin (i.e. supports both combinational and clocked behaviors), specify 0 for the clock input. Ground and power should be specified using G and V, respectively.

If the file contains `socket PLCC` there must be exactly 68 such values in the line. For `socket ZIF` there must be exactly 24. Each 0 or 1 specifies the value of an input. Each H, L, or X specifies the value of an output. The G and V serve only a placeholders, e.g. for human readers.

Values must be separated by spaces.

Multiple bits may be specified using the shorthand value [N]hexval. Example: [16]AAAA specifies the next 16 inputs as 1 0 1 0 1 ... The [N]hexval construct is a value and must be separated from its neighbors within the line by spaces. The least-significant bits of hexval are used if more than N bits are given, so [2]D sets the 2 bits to 0b01. The maximum value of N is 16.

### Clocked circuits

Exactly one character **C** may appear in a vector. This specifies that the pin is a clock line. The firmware will set the inputs, pulse the clock line low, then high, and then read and verify the outputs.

The exerciser does not control power and ground. When executing a sequence of vectors from a vector file, the exerciser does not change any state except as directed by the vectors. As a result it is possible to create multiple-step sequential tests.