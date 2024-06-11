# Host software for Nano-based chip exerciser

The "cex" (chip exerciser) has an extremely simple-minded command set.
It prompts for input. There are three commands: t for toggle a control
line, s for set an output register, and g for get an input register.
Neither the Nano nor cex has any idea about the associations between
signals. For one example, the input registers are are U3, U7, and U11.
They must be clocked by toggling a control line using the t command and
then gotten using the g command. The ID values of the control signals
required to clock the register and then read from it may be unrelated.

## Details (ID, meaning)

- 0x0 Clocks input register U3
- 0x1 Reads value of input register U3 (see note below)
- 0x2 Clocks output register U2
- 0x3 Clocks output register U1
- 0x4 Clocks output register U4
- 0x5 Clocks output register U5
- 0x6 Clocks output register U8
- 0x7 Clocks input register U7
- 0x8 TSTCLK signal to PLCC pin 17 and ZIF pin 1
- 0x9 Reads value of input register U7 (see note below)
- 0xA Clocks output register U10
- 0xB Clocks output register U11
- 0xC Reads value of input register U11 (see note below)
- 0xD Clocks output register U14 (no connection; U14 is not implemented)
- 0xE No connection
- 0xF No connection

## Notes

The input registers should not be clocked using 't' commands as this may
cause conflicts on the Nano's IO bus. Instead, use 'g' (get input data)
with the input register's ID (0x1, 0x9, or 0xC).

