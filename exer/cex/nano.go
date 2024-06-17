// Copyright (c) Jeff Berkowitz 2021, 2022. All rights reserved.

package main

// Serial port communications for Arduino Nano.

import (
	"fmt"
	"log"
	"time"
)

// Create a session with the Nano. The session need not be interactive.
//
// Errors:
// No such file or directory: the Nano is probably not plugged in
// (The USB device doesn't exist in /dev unless Nano is connected.)
//
// Connection not established: device open, but protocol broke down
func CreateSession(nano *Arduino) error {
	var err error
	tries := 3
	for i := 0; i < tries; i++ {
		log.Println("creating connection")
		if err = establishConnection(nano, i == 0); err == nil {
			return nil
		}

		log.Printf("connection setup failed: %v: sync retry %d\n", err, i+1)
		time.Sleep(interSessionDelay)
	}
	if err != nil {
		return err
	}

	log.Println("session in progress with Arduino")
	return nil
}

func establishConnection(nano *Arduino, wasReset bool) error {
	if wasReset {
		time.Sleep(3 * time.Second)
	} else {
		if err := drain(nano); err != nil {
			return err
		}
	}
	if err := getSyncResponse(nano); err != nil {
		return err
	}
	if err := checkProtocolVersion(nano); err != nil {
		return err
	}
	if debug {
		log.Println("protocol version OK")
	}
	return nil
}

// The Nano is not supposed to ever send more than an ack, a byte count < 256,
// and that many data bytes in response to any command. Also, it shouldn't go
// for more than a millisecond or so (really much less) without transmitting
// if it has anything to transmit. So we read up to 257 + fudge factor bytes,
// giving it 10mS each. If it's still transmitting, return an error.
func drain(nano *Arduino) error {
	for i := 0; i < 300; i++ {
		if _, err := nano.ReadFor(responseDelay); err != nil {
			log.Println("Nano is drained")
			return nil
		}
	}
	return fmt.Errorf("Nano is transmitting continuously")
}

// Slowly send some syncs until we see one sync ack. If one is seen,
// return success after trying to consume any delayed acks.
func getSyncResponse(nano *Arduino) error {
	nSent := 0
	tries := 3

	for i := 0; i < tries; i += 1 {
		err := doCommand(nano, CmdSync)
		nSent++
		if err != nil {
			log.Printf("sync command failed: %s\n", err.Error())
		} else {
			for nSent--; nSent > 0; nSent-- {
				nano.ReadFor(responseDelay)
			}
			return nil // success
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("failed to synchronize")
}

func checkProtocolVersion(nano *Arduino) error {
	var nanoCmd []byte = []byte{CmdGetVer}
	b, err := doFixedCommand(nano, nanoCmd, 1)
	if err != nil {
		return err
	}
	if b[0] != ProtocolVersion {
		return fmt.Errorf("protocol version mismatch: host 0x%02X, Arduino 0x%02X\n",
			ProtocolVersion, b[0])
	}
	return nil
}

func getNanoRequest(nano *Arduino) (string, error) {
	bytes, err := doCountedReceive(nano, []byte{CmdPoll})
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func doPoll(nano *Arduino) error {
	// Poll the Nano
	msg, err := getNanoRequest(nano)
	if err != nil {
		// should session end on -any- error? Yes for now.
		return err
	}
	if len(msg) != 0 {
		if isLogRequest(msg) {
			nanoLog.Printf(msg)
		} else {
		    return fmt.Errorf("unsupported request type")
		}
	}
	return nil
}

// There are two general kinds of requests from the Nano: log requests and
// other requests. Other requests are identified by a punctuation mark ('#',
// '$', '%', or '&') in column 1. Log requests are everything else. No "other"
// requests are currently defined.
func isLogRequest(req string) bool {
	if req[0] >= '#' && req[0] <= '&' {
		// syscall request of some type: '#', '$', '%', and '&' reserved
		return false
	}
	// anything else is a log request
	return true
}

type UnexpectedResponseError struct {
	Command  byte
	Response byte
}

func (u *UnexpectedResponseError) Error() string {
	return fmt.Sprintf("command 0x%X: unexpected response 0x%X", u.Command, u.Response)
}

// Do a command, returning error if no ack or bad ack.
// Only for commands with no arguments and no return.
func doCommand(nano *Arduino, cmd byte) error {
	_, err := doFixedCommand(nano, []byte{cmd}, 0)
	return err
}

func getAck(nano *Arduino, cmd byte) error {
	b, err := nano.ReadFor(responseDelay)
	if err != nil {
		return err
	}
	if b != Ack(cmd) {
		return &UnexpectedResponseError{cmd, b}
	}
	return nil
}

// Do the fixed part of a command, which may be the entire command, and optionally
// return the fixed response if any.
//
// Every command has a command byte and may have fixed argument bytes specified
// by the protocol spec, the last of which may be a count. After receipt of the
// fixed bytes, the Nano is supposed to ack or nak. If the protocol specifies a
// fixed response, it immediately follows the ack. The fixed response is not sent
// if the Nano nak's the command. If an ack is seen and the command specified a
// counted number of bytes, they follow. The counted bytes are neither ack'd nor
// nak'd.
//
// This function handles the fixed part of the command, the ack or nak, and the
// fixed response if one is specified. If the Nano nak's, the returned byte slice
// is empty and the error is non-nil. If the Nano ack's and the expected argument
// is zero, the error is nil and the returned byte slice is empty. If the Nano
// ack's and the fixed response does not arrive, an error is returned and the
// state of the returned byte slice is undefined. Otherwise, the error is nil
// and the returned byte slice contains the fixed response. If the command includes
// counted bytes, the caller must then read or write them if the error return from
// this function is nil but must not try if the error return is non-nil.
func doFixedCommand(nano *Arduino, fixed []byte, expected int) ([]byte, error) {
	var response []byte

	if len(fixed) < 1 || len(fixed) > 8 {
		return response, fmt.Errorf("invalid fixed command length")
	}
	if expected < 0 || expected > 8 {
		return response, fmt.Errorf("invalid fixed response expected")
	}
	if debug && fixed[0] != CmdPoll {
		// Don't debug poll commands - there are too many
		log.Printf("doFixedCommand: sending %v\n", fixed)
	}
	if err := nano.Write(fixed); err != nil {
		return response, err
	}
	if err := getAck(nano, fixed[0]); err != nil {
		return response, err
	}

	if expected > 0 {
		response = make([]byte, expected, expected)
		for i := 0; i < expected; i++ {
			b, err := nano.ReadFor(responseDelay)
			if err != nil {
				// If we encounter an error here,
				// the state of the response slice
				// is undefined as described above
				return response, err
			}
			response[i] = b
		}
	}
	return response, nil
}

// Send a counted set of bytes to the Nano. The count must be in the last
// byte of the fixed slice. The counted slice must be at least as long as
// the count. If the fixed command is ack'd, the counted bytes are just
// stuffed down the pipe. If a nak or other error occurs, it is returned.
// No counted command has fixed response so only the error is returned.
func doCountedSend(nano *Arduino, fixed []byte, counted []byte) error {
	count := fixed[len(fixed)-1]
	if debug {
		log.Printf("doCountedSend(): counted len = %d\n", count)
	}
	if len(counted) < int(count) {
		return fmt.Errorf("not enough data")
	}
	if _, err := doFixedCommand(nano, fixed, 0); err != nil {
		return err
	}
	if debug {
		log.Printf("doCountedSend(): sending %d\n", len(counted))
	}
	if err := nano.Write(counted); err != nil {
		return err
	}
	return nil
}

// Issue a command to the Nano and receive a counted response. As of protocol
// v8, all commands that return a counted set of bytes have a count as the
// sole fixed response byte (even though it's often just an echo of the count
// the request asked for).
func doCountedReceive(nano *Arduino, fixed []byte) ([]byte, error) {
	expected, err := doFixedCommand(nano, fixed, 1)
	if err != nil {
		return expected, err
	}
	count := expected[0]
	if debug && count > 0 {
		log.Printf("Receive %d\n", count)
	}
	response := make([]byte, count, count)
	for i := 0; i < int(count); i++ {
		response[i] = byte(i)
	}
	for i := 0; i < int(count); i++ {
		b, err := nano.ReadFor(responseDelay)
		if err != nil {
			return response, err
		}
		response[i] = b
	}
	return response, nil
}
