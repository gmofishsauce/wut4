// Copyright (c) Jeff Berkowitz 2021, 2023. All Rights Reserved
#define PROTOCOL_VERSION 11
#define ACK(CMD) ((byte)~CMD)
#define STCMD_BASE           0xE0
#define STERR_BADCMD         0x86
#define STCMD_SYNC           0xEF
