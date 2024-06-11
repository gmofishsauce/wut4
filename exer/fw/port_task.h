// Copyright (c) Jeff Berkowitz 2021. All rights reserved.
//
// 456789012345678901234567890123456789012345678901234567890123456789012
//
// The lowest layer of the port support (roughly, the nanoXYZ() functions)
// has been moved into port_utils.h because this file got too large. There
// is an extensive comment in that file.

namespace PortPrivate {

  // Initialization: yarc_fw.ino::setup() calls InitTasks() which calls the
  // init functions of all the tasks, in the order specified in the static
  // task definition table in task_runner.h. At this point all the system
  // facilities are usable. Then InitTasks() calls postInit() which
  // just calls PortPrivate::internalPostInit() in this file. If postInit()
  // returns false, InitTasks() calls panic(). internalPostInit() has some
  // built-in functionality and calls out to the following three functions.

  void callWhenAnyReset(void);      // Called from the top of postInit() always
  void callWhenPowerOnReset(void);  // Called only when power-on reset occurring
  void callAfterPostInit(void);     // Called from the end of postInit() always
  
  // Because of the order of initialization, this is basically
  // the very first code executed on either a hard or soft reset.
  // This (and all the init() functions) should be fast.
  
  void internalPortInit() {
    // Set the two decoder select pins to outputs. Delay after making
    // any change to this register.
    DDRC = DDRC | (_BV(DDC3) | _BV(DDC4));
    delayMicroseconds(2);
  
    // Turn off both of the decoder select lines so no decoder outputs
    // are active.
    PORTC &= ~(_BV(PORTC3) | _BV(PORTC4));
    
    nanoSetMode(portData,   OUTPUT);
    nanoSetMode(portSelect, OUTPUT);
  }

  // PostInit() is called from setup after the init() functions are called for all the firmware tasks.
  // The name is a pun, because POST stands for Power On Self Test in addition to meaning "after". But
  // the "power on" part is a misleading pun, because postInit() runs on both power-on resets and "soft"
  // resets (of the Nano only) that occur when the host opens the serial port.
  //
  // The hardware allows the Nano to detect power-on reset by reading bit 0x08 of the MCR. A 0 value
  // means the YARC is in the reset state. This state lasts at least two seconds after power-on, much
  // longer than it takes the Nano to initialize. The Nano detects this and performs initialization
  // steps both before and after the YARC comes out of the POR state as can be seen in the code below.
    
  // Power on self test and initialization. Startup will hang if this function returns false.

  bool internalPostInit() {

    // All the internalInit functions have been called, so all
    // the Nano's system facilities are supposed to be available.

    callWhenAnyReset();

    // Not clear if there will be an equivalent to YARC's power
    // on reset circuitry in the chip exerciser.
    // if (YarcIsPowerOnReset()) {
    //   callWhenPowerOnReset();
    // }
                
    pinMode(LED_PIN, OUTPUT);
    digitalWrite(LED_PIN, HIGH);

    // Now do some other tests, which can panic.
    callAfterPostInit();
    return true;
  }
} // End of PortPrivate section

// Public interface to ports

void portInit() {
  PortPrivate::internalPortInit();
}

int portTask() {
  return 171;
}

bool postInit() {
  return PortPrivate::internalPostInit();
}

// Public interface to the write-only 8-bit Display Register (DR)

void SetDisplay(byte b) {
  // There's no display register in the chip exerciser (may need one)
  // PortPrivate::nanoSetRegister(PortPrivate::DisplayRegister, b);
}

// These are convenience functions. Making them functions allows me to stash them
// at the very bottom of the file.
namespace PortPrivate {

  void callWhenAnyReset() {
    SerialReset();
  }

  void callAfterPostInit() {
  }
}


