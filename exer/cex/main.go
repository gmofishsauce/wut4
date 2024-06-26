// Copyright (c) Jeff Berkowitz 2021, 2022. All rights reserved.

package main

// Serial port communications for Arduino Nano.

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"cex/dev"
)

var debug = false
var nanoLog *log.Logger

// When the Arduino (the "Nano") is connected by USB-serial, opening the port
// from the Mac side forces a hard reset to the device (the Arduino restarts).

const arduinoNanoDevice = "/dev/cu.usbserial-AQ0169PT"
const baudRate = 115200 // Note: change requires updating the Arduino firmware

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
	nano, err := dev.NewArduino(arduinoNanoDevice, baudRate, nanoLog, debug)
	if err != nil {
		log.Printf("opening Arduino device %s: %v", arduinoNanoDevice, err)
		return 2
	}
	defer nano.Close()

	// Create a protocol connection to the Nano
	if err := dev.CreateSession(nano); err != nil {
		log.Printf("creating session with Arduino device %s: %v", arduinoNanoDevice, err)
		return 2
	}

	// If there are vector files, process them and done
	if len(vectorFiles) > 0 {
		totalFailures := 0
		for _, vf := range vectorFiles {
			log.Printf("processing vector file %s", vf)
			failureCount, err := DoVectorFile(vf, nano)
			if err != nil {
				log.Printf("vector file %s: error: %s\n", vf, err)
				return 2
			}
			log.Printf("vector file %s: %d failure(s)", vf, failureCount)
			totalFailures += failureCount
		}
		outcome := 0
		if totalFailures == 0 {
			log.Printf("success")
		} else {
			log.Printf("%d failures", totalFailures)
			outcome = 3
		}
		return outcome
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
func interactiveSession(input *Input, nano *dev.Arduino) error {
	var err error
	for {
		if err = dev.DoPoll(nano); err != nil {
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

// These doSomeCmd functions are used by both interactive mode
// and vector mode. Since they are used interactively, the just
// print messages for syntax errors, but don't return errors
// because those would end the session with the Nano.

func doToggleCmd(line string, nano *dev.Arduino) error {
	var cmd []byte = make([]byte, 3, 3)
	// t id count
	n, err := fmt.Sscanf(line[2:], "%x %x", &cmd[1], &cmd[2])
	if n != 2 { // print a message and do nothing
		log.Printf("usage: t hexct hexid")
		return nil
	}
	cmd[0] = dev.CmdPulse
	_, err = dev.DoFixedCommand(nano, cmd, 0)
	if err == nil && debug {
		log.Printf("%s", line)
	}
	return err
}

func doSetCmd(line string, nano *dev.Arduino) error {
	var cmd []byte = make([]byte, 3, 3)
	// s id data or sr id data for bit-reversed set
	var cmdByte byte = dev.CmdSet
	offset := 2
	if line[1] == 'r' {
		cmdByte = dev.CmdSetR
		offset = 3
	}
	if n, _ := fmt.Sscanf(line[offset:], "%x %x", &cmd[1], &cmd[2]); n != 2 {
		log.Printf("usage: s hexid hexdata or sr hexid hexdata")
		return nil
	}
	cmd[0] = cmdByte
	_, err := dev.DoFixedCommand(nano, cmd, 0)
	if err == nil && debug {
		log.Printf("%s", line)
	}
	return err
}

func doGetCmd(line string, nano *dev.Arduino) (byte, error) {
	// Note: this just reads the reads the input register.
	// It must be separately clocked using a "t" command.
	var cmd []byte = make([]byte, 2, 2)
	// g id or gr id for bit-reversed get
	var cmdByte byte = dev.CmdGet
	offset := 2
	if line[1] == 'r' {
		cmdByte = dev.CmdGetR
		offset = 3
	}
	n, err := fmt.Sscanf(line[offset:], "%x", &cmd[1])
	if n != 1 {
		log.Printf("usage: g hexid or gr hexid")
		return 0, nil
	}
	cmd[0] = cmdByte
	sl, err := dev.DoFixedCommand(nano, cmd, 1)
	if err != nil {
		return 0, err
	}
	result := sl[0]
	if debug {
		log.Printf("%s = %02X", line, result)
	}
	return result, err
}

// Process a line of user input. Returning error is fatal,
// so we don't do that for typos, etc. We just print messages.
func process(line string, nano *dev.Arduino) error {
	switch line[0] {
	case 't': // toggle a control line, e.g. a clock
		if err := doToggleCmd(line, nano); err != nil {
			log.Printf("command %s: %v", line, err)
			return err
		}
	case 's': // set a register - 'sr' reverses the bits
		if err := doSetCmd(line, nano); err != nil {
			log.Printf("command %s: %v", line, err)
			return err
		}
	case 'g': // get a register - 'gr' reverses the bits
		result, err := doGetCmd(line, nano)
		if err != nil {
			log.Printf("command %s: %v", line, err)
			return err
		}
		log.Printf("read 0x%02X\n", result)
	default:
		log.Printf("%s: unknown command\n", line)
	}
	return nil
}
