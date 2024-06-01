// Copyright (c) Jeff Berkowitz 2021, 2023. All Rights Reserved
// Automatically generated by Protogen - do not edit

#define PROTOCOL_VERSION 11
#define ACK(CMD) ((byte)~CMD)

#define STCMD_BASE           0xE0
#define STCMD_GET_MCR        0xE1
#define STCMD_RUN_COST       0xE2
#define STCMD_STOP_COST      0xE3
#define STCMD_CLOCK_CTL      0xE4
#define STCMD_WR_MEM         0xE5
#define STCMD_RD_MEM         0xE6
#define STCMD_RUN_YARC       0xE7
#define STCMD_STOP_YARC      0xE8
#define STCMD_POLL           0xE9
#define STCMD_SVC_RESPONSE   0xEA
#define STCMD_DEBUG          0xEB
#define STCMD_GET_VER        0xEE
#define STCMD_SYNC           0xEF
#define STCMD_SET_ARH        0xF0
#define STCMD_SET_ARL        0xF1
#define STCMD_SET_DRH        0xF2
#define STCMD_SET_DRL        0xF3
#define STCMD_DO_CYCLE       0xF4
#define STCMD_GET_RESULT     0xF5
#define STCMD_WR_SLICE       0xF6
#define STCMD_RD_SLICE       0xF7
#define STCMD_SET_K          0xFB
#define STCMD_SET_MCR        0xFC
#define STCMD_WR_ALU         0xFD
#define STCMD_RD_ALU         0xFE

#define STERR_NOSYNC         0x80
#define STERR_PASSIVE        0x81
#define STERR_ONECLOCK       0x82
#define STERR_CANT_SS        0x83
#define STERR_CANT_PG        0x84
#define STERR_INTERNAL       0x85
#define STERR_BADCMD         0x86
