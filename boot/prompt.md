Hi Claude. Let's write a bootstrap loader for the WUT-4.  First
let's define a bootstrap program. On this computer, the hardware
initializes with the first code MMU register pointing to page frames
zero, and the second MMU register pointing to the same physical
page. Bootstrap programs run in this state, so they must have special
initialization that starts the stack at 0x4096 (the end of the first
and only mapped page.) And they must be compiled so that all the
data is in the code section, since code and data map to the same
page. These compiler behaviors are triggered bye "pragma bootstrap".
After the system is initialized by system software, code and data
MMU entries will not point to the same page at any time. So the
hardware will be used like a Harvard machine even though it isn't.

Additionally, I will use the word "frame" to refer to a physical
page of memory and the word "page" to refer to a virtual page in
an address space.

So a bootstrap loader is a bootstrap program that loads another
program from the SD disk.  The bootstrap loader itself will be
supplied to the emulator on the command line  as the program to
run.  The file supplied to the emulator as the SD disk would be a
WUT-4 binary program that has not been compiled as a bootstrap. The
bootstrap loader will load this program into pages it allocates and
maps by setting up the MMU registers. These pages will include one
page of stack at the top of the address space and also code and
data pages at zero sufficient to execute the program.  The bootstrap
loader runs in privileged mode and transfers control to the program
it loads after setting up memory. The bootstrap loader does not
switch the computer into user mode. I believe that in order to do
this the bootstrap loader will need to read the header of the file
from the SD card so that he knows how many page frames to map at
virtual 0 for a code and also how many to map at virtual 0 for data.
The file supplied as the SD data on the command line to the emulator
must be specially prepared. It must contain a binary program at
file offset zero but must also be padded to a multiple of 512 bytes
as required by the emulated SD card implementation.

The bootstrap loader must fit entirely in physical page frame 0.  It
must not attempt to reuse physical page frames zero for the program
it loads.

Please discuss with me before writing code and plan to create th
program in the empty directory wut4/bootloader. Also plan to use
wut4/lib/nio for console and I/O needs.
