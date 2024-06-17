package main

import (
	"log"
	"time"
)

// I put all the global symbols here while I move them where they belong.
// There was a lot of laziness about globals in the code I'm refactoring
// (came from yarc/host).
var debug = false
const responseDelay = 5000 * time.Millisecond
const interSessionDelay = 3000 * time.Millisecond
const arduinoNanoDevice = "/dev/cu.usbserial-AQ0169PT"
const baudRate = 115200 // Note: change requires updating the Arduino firmware
var nanoLog *log.Logger

