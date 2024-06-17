// Copyright (c) Jeff Berkowitz 2021, 2022. All rights reserved.

package main

// Serial port communications for Arduino Nano.

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

// When the Arduino (the "Nano") is connected by USB-serial, opening the port
// from the Mac side forces a hard reset to the device (the Arduino restarts).

// About calls to time.Sleep() in this code: sleeps occur only during session
// setup and teardown, and they are long (seconds). There are no millisecond
// delays imposed by code in this file. As of March 2022, the only millisecond
// sleep is in the terminal input code.

func main() {
	os.Exit(submain())
}

func submain() int { // return exit code
	// User logger to standard output for rough timing
	log.SetFlags(log.Lmsgprefix | log.Lmicroseconds)
	log.SetPrefix("cex: ")
	log.Println("firing up")

	flag.BoolVar(&debug, "d", false, "enable debug output")
	flag.Parse()
	vectorFiles := flag.Args()

	// Open the Nano's log file (not the Nano itself)
	nanoLogFile, err := os.OpenFile("Nano.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("opening Nano.log: %v", err)
		return 2
	}
	defer nanoLogFile.Close()
	nanoLog = log.New(nanoLogFile, "", log.Lmsgprefix|log.Lmicroseconds)

	// Now open the Nano (serial device)
	nano, err := NewArduino(arduinoNanoDevice, baudRate)
	if err != nil {
		log.Printf("opening Arduino device %s: %v", arduinoNanoDevice, err)
		return 2
	}
	defer nano.Close()

	// Create a protocol connection to the Nano
	if err := CreateSession(nano); err != nil {
		log.Printf("creating session with Arduino device %s: %v", arduinoNanoDevice, err)
		return 2
	}

	// If there are vector files, process them and done
	if len(vectorFiles) > 0 {
		for _, vf := range vectorFiles {
			err := DoVectorFile(vf)
			if err != nil {
				log.Printf("vector file %s: %s\n", vf, err)
				return 2
			}
		}
		return 0
	}

	// No vector files on the command line - interactive mode
	log.Println("starting interactive session")
	input := NewInput()
	err = interactiveSession(input, nano)
	if err == io.EOF {
		log.Printf("user quit\n")
		return 0
	}

	log.Printf("session aborted: %v\n", err)
	return 2
}

// Conduct an interactive session with the Nano.
//
// Errors:
// No such file or directory: the Nano is probably not plugged in
// (The USB device doesn't exist in /dev unless Nano is connected.)
//
// Connection not established: device open, but protocol broke down
// Various I/O errors
func interactiveSession(input *Input, nano *Arduino) error {
	var err error
	for {
		if err = doPoll(nano); err != nil {
			return err
		}

		var line string
		if line, err = input.CheckFor(); err != nil {
			return err
		}
		if len(line) > 1 { // 1 for the newline
			if err = process(line, nano); err != nil {
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
			log.Printf("usage: t hexct hexid\n")
			return nil
		}
		cmd[0] = CmdPulse
		if _, err := doFixedCommand(nano, cmd, 0); err != nil {
			log.Printf("cmd t 0x%02X 0x%02X: %v\n", cmd[1], cmd[2], err);
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
            log.Printf("usage: s hexid hexdata or sr hexid hexdata\n")
            return nil
        }
        cmd[0] = cmdByte
        if _, err := doFixedCommand(nano, cmd, 0); err != nil {
            log.Printf("error: cmd s 0x%02X 0x%02X: %v\n", cmd[1], cmd[2], err);
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
            log.Printf("usage: g hexid or gr hexid\n")
            return nil
        }
        cmd[0] = cmdByte
        sl, err := doFixedCommand(nano, cmd, 1)
		if err != nil {
            log.Printf("cmd g 0x%02X: %v\n", cmd[1], err);
			break
        }
		log.Printf("in(0x%02X) = 0x%02X\n", cmd[1], sl[0])

	default:
		log.Printf("%s: unknown command\n", line)
	}
	return nil
}
