// Copyright (c) Jeff Berkowitz 2021, 2023. All Rights Reserved
// This must be kept in sync with serial_protocol.h in the Go code

#define PROTOCOL_VERSION 12
#define ACK(CMD) ((byte)~CMD)

#define STCMD_BASE      0xE0
#define STCMD_SYNC      0xE1
#define STCMD_GET_VER   0xE2
#define STCMD_POLL      0xE3

#define STCMD_PULSE     0xF0
#define STCMD_SET       0xF4
#define STCMD_GET       0xF8

#define STERR_BADCMD    0x81  // bad command byte
