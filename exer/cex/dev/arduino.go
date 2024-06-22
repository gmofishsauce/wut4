// Copyright (c) Jeff Berkowitz 2021. All rights reserved.

// Type arduino provides a synchronous byte I/O interface to an Arduino. This
// implementation uses the default USB serial port provided by an Arduino Nano.
// Opening a standard USB serial port activates the DTR signal which resets the
// Arduino, necessitating a full reconnect.

package dev

import (
	"fmt"
	"go.bug.st/serial"
	"log"
	"syscall"
	"time"
)

// The Nano snoops the serial line during the first few seconds of a reset,
// looking for a byte pattern that identifiers a download from the IDE. It's
// safest to just wait for this snoop period to end. This value works.
const resetDelay time.Duration = 4 * time.Second

// Types

type Arduino struct {
	port  serial.Port
	log   *log.Logger
	debug bool
}

type NoResponseError time.Duration

func (nre NoResponseError) Error() string {
	return fmt.Sprintf("read from Arduino: no response after %v", time.Duration(nre))
}

// Public interface

func NewArduino(deviceName string, baudRate int, log *log.Logger, debug bool) (*Arduino, error) {
	var arduino Arduino
	var err error

	mode := &serial.Mode{BaudRate: baudRate, DataBits: 8,
		Parity: serial.NoParity, StopBits: serial.OneStopBit}
	arduino.port, err = serial.Open(deviceName, mode)
	if err != nil {
		return nil, err
	}

	arduino.log = log
	arduino.debug = false // = debug FOR NOW
	log.Printf("serial port is open - delaying %.0f seconds for Nano reset", resetDelay.Seconds())
	time.Sleep(resetDelay)
	return &arduino, nil
}

// Read the Arduino until a byte is received or a timeout occurs
func (arduino *Arduino) ReadFor(timeout time.Duration) (byte, error) {
	return arduino.readByte(timeout)
}

// Write bytes to the Arduino.
func (arduino *Arduino) Write(b []byte) error {
	return arduino.writeBytes(b)
}

// Close the connection to the Arduino.
func (arduino *Arduino) Close() error {
	return arduino.closeSerialPort()
}

// Implementation

// Read a byte
func (arduino *Arduino) readByte(readTimeout time.Duration) (byte, error) {
	b := make([]byte, 1, 1)
	var n int
	var err error

	// The for-loop is -solely- to handle EINTR, which occurs constantly
	// as a result of Golang's Goroutine-level context switching mechanism.
	arduino.port.SetReadTimeout(readTimeout)
	for {
		n, err = arduino.port.Read(b)
		// Break loop unless EINTR.
		if !isRetryableSyscallError(err) {
			break
		}
		if n != 0 {
			panic("bytes returned despite EINTR")
		}
	}
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, NoResponseError(readTimeout)
	}
	return b[0], nil
}

// Write bytes
func (arduino *Arduino) writeBytes(toWrite []byte) error {
	var n int
	var err error

	// The for-loop is -solely- to handle EINTR, which occurs constantly
	// as a result of Golang's Goroutine-level context switching mechanism.
	for {
		n, err = arduino.port.Write(toWrite)
		// Drop out of the loop on success
		// or error, but not on EINTR.
		if !isRetryableSyscallError(err) {
			break
		}
		if n != 0 {
			panic("bytes written despite EINTR")
		}
	}
	if err != nil {
		return err
	}
	if n != len(toWrite) {
		return fmt.Errorf("write didn't consume all the bytes")
	}
	return nil
}

func (arduino *Arduino) closeSerialPort() error {
	if arduino.port == nil {
		return fmt.Errorf("internal error: close(): port not open")
	}
	if err := arduino.port.Close(); err != nil {
		log.Printf("close serial port: %s", err)
		return err
	}
	log.Println("serial port closed")
	arduino.port = nil
	return nil
}

func isRetryableSyscallError(err error) bool {
	const eIntr = 4
	if errno, ok := err.(syscall.Errno); ok {
		return errno == eIntr
	}
	return false
}
