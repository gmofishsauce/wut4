# Exer - chip exerciser

This is an Arduino Nano chip "exerciser". It drives a 68-pin PLCC socket
with a hardwired pinout designed to fit a Logic Devices L4C381 which is
a 16-bit ALU. The wires then run to a 40-pin socket that's intended for
ATF22V10C CMOS GALs. This is a little more complicated since pins can be
either inputs or outputs, so multiple 3-state output ports are connected
to I/O pins.

The subdirectories are:

A KiCad schematic in ki/

A Nano sketch of the controller firmware in fw/

A Golang host program for communicating with the controller in host/

A Fritzing sketch of the solderless breadboard layout in Chiptester.fzz.
This is symlink because Fritzing is fussy about grouping files.

