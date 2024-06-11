// Copyright (c) Jeff Berkowitz 2021, 2023. All Rights Reserved
// Automatically generated by Protogen - do not edit

package main

const ProtocolVersion = 12

func Ack(b byte) byte {
	return ^b
}

const CmdBase     = 0xE0
const CmdSync     = 0xE1
const CmdGetVer   = 0xE2
const CmdPoll     = 0xE3

const CmdPulse    = 0xF0
const CmdSet      = 0xF4
const CmdGet      = 0xF8

const ErrBadcmd   = 0x81
