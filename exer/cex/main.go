// Copyright (c) Jeff Berkowitz 2021, 2022. All rights reserved.

package main

// Serial port communications for Arduino Nano.

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

var debug = false

const responseDelay = 5000 * time.Millisecond
const interSessionDelay = 3000 * time.Millisecond

const arduinoNanoDevice = "/dev/cu.usbserial-AQ0169PT"
const baudRate = 115200 // Note: change requires updating the Arduino firmware

var nanoLog *log.Logger

// When the Arduino (the "Nano") is connected by USB-serial, opening the port
// from the Mac side forces a hard reset to the device (the Arduino restarts).

// About calls to time.Sleep() in this code: sleeps occur only during session
// setup and teardown, and they are long (seconds). There are no millisecond
// delays imposed by code in this file. As of March 2022, the only millisecond
// sleep is in the terminal input code.

func main() {
	log.SetFlags(log.Lmsgprefix | log.Lmicroseconds)
	log.SetPrefix("cex: ")
	log.Println("firing up")

	tf, err := ParseVectorFile("./t.tv")
	if err != nil {
		fmt.Printf("PVF: %v\n", err)
		os.Exit(5)
	}
	if tf == nil {
		fmt.Printf("PVF: no error but nil\n")
	}
	fmt.Printf("%v\n", tf)
	os.Exit(5)

	// The Nano's log is opened first and remains open always.
	nanoLogFile, err := os.OpenFile("Nano.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("opening Nano.log: ", err)
	}
	nanoLog = log.New(nanoLogFile, "", log.Lmsgprefix|log.Lmicroseconds)
	input := NewInput()

	for {
		log.Println("starting a session")
		nano, err := NewArduino(arduinoNanoDevice, baudRate)
		if err == nil {
			err = session(input, nano)
			if err == io.EOF {
				log.Printf("user quit\n")
				os.Exit(0)
			}
		}

		log.Printf("session aborted: %v\n", err)
		if nano != nil {
			nano.Close()
		}

		// The original design was to sleep for a few seconds here
		// and then iterate. This reopens the serial port, which
		// resets the Nano. But this design has the effect of losing
		// panic codes (because of the reset), which makes
		// troubleshooting occasional problems more difficult. So
		// now, we spin here waiting for user input. The check for
		// input sleeps for 50mS so this only executes 20 times per
		// second, which is little enough to avoid heat issues.

		log.Printf("Return to continue...\n")
		var line string
		for {
			line, err = input.CheckFor()
			if err != nil {
				break
			}
			if len(line) > 0 {
				break
			}
		}
		// To be nice, we do an EOF check here, although ^C works fine.
		if err == io.EOF {
			log.Printf("user quit\n")
			os.Exit(0)
		}
		log.Printf("Continuing...\n")
	}
}

// Conduct a session with the Nano. Ideally, this function is
// called once per execution of this program and never returns.
//
// Errors:
// No such file or directory: the Nano is probably not plugged in
// (The USB device doesn't exist in /dev unless Nano is connected.)
//
// Connection not established: device open, but protocol broke down
func session(input *Input, nano *Arduino) error {
	var err error
	tries := 3
	for i := 0; i < tries; i++ {
		log.Println("creating connection")
		if err = establishConnection(nano, i == 0); err == nil {
			break
		}

		log.Printf("connection setup failed: %v: sync retry %d\n", err, i+1)
		time.Sleep(interSessionDelay)
	}
	if err != nil {
		return err
	}

	log.Println("session in progress")

	for {
		if err := doPoll(nano); err != nil {
			return err
		}

		var line string
		if line, err = input.CheckFor(); err != nil {
			return err
		}
		if len(line) > 1 { // 1 for the newline
			if err := process(line[:len(line)-1], nano); err != nil {
				return err
			}
		}
	}
}

// Process a line of user input. Returning error is fatal,
// so we don't do that for typos, etc. We just print messages.
func process(line string, nano *Arduino) error {
	switch line[0] {
	case 't': // toggle a control line, e.g. a clock
		var cmd []byte = make([]byte, 3, 3)
		// t id count
		n, err := fmt.Sscanf(line[2:], "%x %x", &cmd[1], &cmd[2])
		if debug {
			log.Printf("ret %d %v (%d %d)\n", n, err, cmd[1], cmd[2])
		}
		if n != 2 {
			fmt.Printf("usage: t hexct hexid\n")
			return nil
		}
		cmd[0] = CmdPulse
		if _, err := doFixedCommand(nano, cmd, 0); err != nil {
			fmt.Printf("cmd t 0x%02X 0x%02X: %v\n", cmd[1], cmd[2], err);
		}
	case 's': // set a register - 'sr' reverses the bits
		var cmd []byte = make([]byte, 3, 3)
		// s id data or sr id data for bit-reversed set
		var cmdByte byte = CmdSet
		offset := 2
		if line[1] == 'r' {
			cmdByte = CmdSetR
			offset = 3
		}
        n, err := fmt.Sscanf(line[offset:], "%x %x", &cmd[1], &cmd[2])
		if debug {
			log.Printf("ret %d %v (%x %x)\n", n, err, cmd[1], cmd[2])
		}
        if n != 2 {
            fmt.Printf("usage: s hexid hexdata or sr hexid hexdata\n")
            return nil
        }
        cmd[0] = cmdByte
        if _, err := doFixedCommand(nano, cmd, 0); err != nil {
            fmt.Printf("error: cmd s 0x%02X 0x%02X: %v\n", cmd[1], cmd[2], err);
        }
	case 'g': // get a register - 'gr' reverses the bits
		// Note: this just reads the reads the input register.
		// It must be separately clocked using a "t" command.
		var cmd []byte = make([]byte, 2, 2)
		// g id or gr id for bit-reversed get
		var cmdByte byte = CmdGet
		offset := 2
		if line[1] == 'r' {
			cmdByte = CmdGetR
			offset = 3
		}
        n, err := fmt.Sscanf(line[offset:], "%x", &cmd[1])
		if debug {
			log.Printf("ret %d %v (%d)\n", n, err, cmd[1])
		}
        if n != 1 {
            fmt.Printf("usage: g hexid or gr hexid\n")
            return nil
        }
        cmd[0] = cmdByte
        sl, err := doFixedCommand(nano, cmd, 1)
		if err != nil {
            fmt.Printf("cmd g 0x%02X: %v\n", cmd[1], err);
			break
        }
		fmt.Printf("in(0x%02X) = 0x%02X\n", cmd[1], sl[0])

	default:
		fmt.Printf("%s: unknown command\n", line)
	}
	return nil
}
