/*
Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com)

This program is free software: you can redistribute it and/or modify it
under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful, but
WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public
License along with this program. If not, see
<http://www.gnu.org/licenses/>.
*/
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"time"
)

// The WUT-4 boots in kernel mode, so the kernel binary is mandatory.
// A user mode binary is optional.
var dflag = flag.Bool("d", false, "enable debugging")
var hflag = flag.Bool("h", false, "home cursor (don't scroll)")
var pflag = flag.Bool("p", false, "write profile to cpu.prof")
var qflag = flag.Bool("q", false, "quiet (no simulator output)")
var uflag = flag.String("u", "", "user binary")

// Functional emulator for wut4 instruction set

const K = 1024
const IOSize = 64  // 64 words of I/O space
const SprSize = 64 // 64 special registers, per Mode

const ( // Special registers
	PC      = 0 // Special register 0 is PC, read-only
	Link    = 1 // Special register 1 is Link, per Mode
	Irr     = 2 // Kernel only interrupt return register SPR
	Icr     = 3 // Kernel only interrupt cause register SPR
	Imr     = 4 // Kernel only interrupt mode register SPR
	CCLS    = 6 // Cycle counter, lower short
	CCMS    = 7 // Cycle counter, most significant short
	MmuCtl1 = 8 // MMU control register
)

const Kern = 0 // Mode = Kernel
const User = 1 // Mode = User

type word uint16

// Exception types. These must be even numbers less than 64, so
// there are 32 distinct types. The wut-4 transitions to kernel
// mode and the value of an exception becomes the value of the
// program counter when the exception occurs. Exception 0 resets
// the system. Exceptions 2..14 correspond to the prioritized
// interrupt levels 1..7. Exceptions 16..30 are hardware generated
// traps, e.g. memory access violations, etc. Exceptions 32 .. 62
// are software traps accessible as SYS 32 through SYS 62.

type exception uint16

const (
	ExNone    exception = 0  // no exception (reset is not exception)
	ExIllegal exception = 16 // illegal instruction
	ExMemory  exception = 18 // access violation (page fault)
	ExAlign   exception = 20 // alignment fault
	ExMachine exception = 30 // machine check
)

type physaddr uint32           // physical addresses are 24 bits
const PhysMemSize = 6 * 64 * K // probably 2048K in hardware
var physmem [PhysMemSize]word  // bytes require extraction

type w4reg struct { // per mode
	gen []word // general registers
	spr []word // special registers
}

type w4machine struct {
	cyc uint64  // cycle counter
	reg []w4reg // [0] is kernel, [1] is user
	io  []word  // i/o space, accessible only in kernel mode
	pc  word

	// Non-architectural state that persists beyond an instruction
	run  bool // run/stop flag
	en   bool // true if interrupts are enabled
	mode byte // current mode, user = 0, kernel = 1

	// Non-architectural state used within an instruction
	alu uint16    // temporary alu result register; memory address
	sd  word      // memory source data register set at execute
	wb  word      // writeback register set at execute or memory
	ex  exception // exception code
	ir  word      // instruction register
	hc  uint16    // hidden carry bit, 1 or 0

	// These variables are part of the combinational logic.
	// The are set at decode time and used at execute, memory,
	// or writeback time.
	op, imm                            uint16
	xop, yop, zop, vop                 uint16
	isXop, isYop, isZop, isVop, isBase bool
	ra, rb, rc                         uint16
}

var w4 w4machine = w4machine{
	reg: []w4reg{
		{gen: make([]word, 8, 8), spr: make([]word, SprSize, SprSize)}, // user
		{gen: make([]word, 8, 8), spr: make([]word, SprSize, SprSize)}, // kernel
	},
	io: make([]word, IOSize, IOSize),
}

func main() {
	var err error

	flag.Parse()
	args := flag.Args()
	if len(args) != 1 { // kernel mode binary file is mandatory
		usage()
	}

	if *pflag {
		if *dflag {
			fatal("cannot profile and debug at the same time")
		}
		f, err := os.Create("cpu.prof")
		if err != nil {
			fatal(fmt.Sprintf("could not create CPU profile: ", err))
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			fatal(fmt.Sprintf("could not start CPU profile: ", err))
		}
		defer pprof.StopCPUProfile()
	}

	dbEnabled = *dflag
	if err := w4.load(Kern, args[0]); err != nil {
		fatal(fmt.Sprintf("loading %s: %s", args[0], err.Error()))
	}
	if len(*uflag) != 0 {
		if err := w4.load(User, *uflag); err != nil {
			fatal(fmt.Sprintf("loading %s: %s", *uflag, err.Error()))
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			fmt.Printf("caught %v ('x' to exit)\n", sig)
			dbEnabled = true
		}
	}()

	dbg("start")
	w4.reset()
	err = w4.simulate()
	if err != nil {
		// Some kind of internal error, not error in simulated program
		fatal(fmt.Sprintf("error: running %s: %s", args[0], err.Error()))
	}
	dbg("done")
}

// If we block for input at any time during execution, we don't report
// the machine cycles per second at the end because it's not meaningful.
var blockedForInput bool

func (w4 *w4machine) simulate() error {
	if w4.ex != 0 {
		fatal("internal error: simulation started with an exception pending")
	}

	// The simulator is written as a rigid set of parameterless functions
	// that act on shared machine state. This will make it simpler to
	// simulate pipelining later.
	//
	// Sequential implementation: everything happens in each machine cycle.
	// It happens in the order of a pipelined machine, though, to make
	// converting this to a pipelined simulation easier in the future.

	if dbEnabled {
		w4.dump()
		w4.run = prompt(w4)
	}
	tStart := time.Now()
	for w4.cyc++; w4.run; w4.cyc++ {
		w4.fetch()
		w4.decode()
		w4.execute()
		w4.memory()
		w4.writeback()
		if w4.ex != 0 && !w4.en {
			// double fault
			break
		}
		if dbEnabled {
			w4.dump()
			w4.run = prompt(w4)
		}
	}
	d := time.Since(tStart)

	if *qflag {
		return nil
	}

	// Dump the registers. Print a line about why the simulator halted,
	// and then a line about timing unless sim was interactive.
	w4.dump()
	msg := "halt"
	if w4.ex != 0 && !w4.en {
		msg += fmt.Sprintf(": double fault: exception %d", w4.ex)
	}
	fmt.Println(msg)

	msg = fmt.Sprintf("%d cycles executed", w4.cyc)
	if !blockedForInput { // noninteractive run: print time
		msg += fmt.Sprintf(" in %s (%1.3fMHz)",
			d.Round(time.Millisecond).String(),
			(float64(w4.cyc)/1e6)/d.Seconds())
	}
	fmt.Println(msg)
	return nil
}

// Prompt the user for input and return false if the sim should halt
func prompt(w4 *w4machine) bool {
	// Disable timing if we ever prompt during a simulation run
	blockedForInput = true

	var c []byte = make([]byte, 80, 80)
	var done bool

	for !done {
		fmt.Printf("\n[h c r s x] sim> ")
		os.Stdin.Read(c)
		switch c[0] {
		case 'c':
			w4.core("core")
		case 'h':
			fmt.Printf("h - help\nc - core\nr - run\ns - single step\nx - exit\n")
		case 'r':
			dbEnabled = false
			done = true
		case 's':
			dbEnabled = true
			done = true
		case 'x':
			return false
		}
	}
	return true
}

// Dump some machine state. This method can be invoked from inside a wut-4
// program by executing the dsp instruction or the brk instruction. The brk
// instruction also makes the simulator go interactive (prompt to continue).
func (w4 *w4machine) dump() {
	if *hflag {
		// Home cursor and clear screen
		// This erases the debug output
		fmt.Printf("\033[2J\033[0;0H")
	}

	modeName := "kern"
	if w4.mode == User {
		modeName = "user"
	}
	fmt.Printf("Run %t mode %s cycle %d alu = 0x%04X pc = %d exception = 0x%04X\n",
		w4.run, modeName, w4.cyc, w4.alu, w4.pc, w4.ex)
	// disassemble the instruction at pc
	ex, codeAddr := w4.translate(false, w4.pc)
	if ex != ExNone {
		fmt.Printf("fault@pc = 0x%04X (0x%04X)\n", w4.pc, ex)
	} else {
		arg := fmt.Sprintf("0x%04X@0x%04X", physmem[codeAddr], w4.pc)
		fmt.Printf("instruction @0x%04X: %s\n", w4.pc, rundis(arg))
	}

	reg := &w4.reg[w4.mode] // user or kernel
	headerFormat := "%12s: "
	fmt.Printf(headerFormat, "reg")
	for i := range reg.gen {
		fmt.Printf("%04X%s", reg.gen[i], spOrNL(i < len(reg.gen)-1))
	}

	// Print all 64 user and kernel SPRs
	fmt.Printf(headerFormat, "kern spr")
	fmt.Println("") // hackity hack
	for row := 0; row < 8; row++ {
		start := 8 * row
		end := 8*row + 8
		fmt.Printf("%14s", "")
		for n := start; n < end; n++ {
			fmt.Printf("%04X%s", w4.reg[Kern].spr[n], spOrNL(n < end-1))
		}
	}

	fmt.Printf(headerFormat, "user spr")
	fmt.Println("") // hackity hack
	for row := 0; row < 8; row++ {
		start := 8 * row
		end := 8*row + 8
		fmt.Printf("%14s", "")
		for n := start; n < end; n++ {
			fmt.Printf("%04X%s", w4.reg[User].spr[n], spOrNL(n < end-1))
		}
	}

	codeAddr &^= 7
	fmt.Printf(headerFormat, fmt.Sprintf("imem@0x%06X", codeAddr))
	for i := physaddr(0); i < 8; i++ {
		fmt.Printf("%04X%s", physmem[codeAddr+i], spOrNL(i < 7))
	}

	// For lack of a better answer, print the memory row at 0.
	// This at least gives 8 deterministic locations for putting
	// the results of tests
	ex, dataAddr := w4.translate(true, 0)
	if ex != ExNone {
		fmt.Printf(headerFormat, fmt.Sprintf("fault@data = 0x%04X", 0))
	} else {
		fmt.Printf(headerFormat, fmt.Sprintf("dmem@0x%06X", dataAddr))
		for i := physaddr(0); i < 8; i++ {
			fmt.Printf("%04X%s", physmem[dataAddr+i], spOrNL(i < 7))
		}
	}
}

func spOrNL(sp bool) string {
	if sp {
		return " "
	}
	return "\n"
}

func usage() {
	pr("Usage: func [options] kernel-binary\nOptions:")
	flag.PrintDefaults()
	os.Exit(1)
}
