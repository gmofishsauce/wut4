/* Copyright © 2024 Jeff Berkowitz (pdxjjb@gmail.com) - Affero GPL v3 */

package main

import "syscall"

// Input/output compatible with the tiny kernel on the WUT-4. The
// WUT-4 has no native idea of signedness or signs, so the I/O
// functions are coded somewhat clumsily using unsigned values.

const E_EOF Word = 0xFFFF
const E_IOERR Word = 0xFFFE
const E_UNKNOWN Word = 0xFFFD

// Get a byte from "channel" fd. If the MS byte of the return value
// is nonzero, an error has occurred. Otherwise, the input byte is
// in the LS byte of the return value.
func Getb(fd Word) Word {
	b := []byte{0x00}
	for {
		n, err := syscall.Read(int(fd), b)
		if err != nil {
			return E_IOERR
		}
		if n == 0 {
			return E_EOF
		}
		return Word(b[0]) // success
	}
}

// Put a byte on "channel" fd. If the MS byte of the return value
// is nonzero, an error has occurred. Otherwise, the return value
// is 1 indicating that the byte was written.
func Putb(fd Word, val Byte) Word {
	b := []byte{byte(val)}
	n, err := syscall.Write(int(fd), b)
	if err != nil {
		return E_IOERR
	}
	if n != 1 {
		return E_EOF
	}
	return 1
}

func Exit(code Word) {
	syscall.Exit(int(code))
}

func Printf(s string, args ...any) {
	fmt := false
	argN := 0

	for _, c := range s {
		if fmt {
			if c == 'n' {
				Putb(STDOUT, '\n')
			} else if argN >= len(args) {
				printOops(0)
			} else if c == 's' {
				printS(args[argN])
			} else if c == 'x' {
				printX(args[argN])
			} else {
				printOops(args[argN])
			}
			argN++
			fmt = false
			continue
		}
		if c == '%' {
			fmt = true
		} else {
			Putb(STDOUT, Byte(c))
			continue
		}
	}
}

func printOops(arg any) {
	Putb(STDOUT, Byte('?'))
	Putb(STDOUT, Byte('?'))
	Putb(STDOUT, Byte('?'))
}

func printS(arg any) {
	s, ok := arg.(string)
	if !ok {
		Putb(STDOUT, Byte('s'))
		Putb(STDOUT, Byte('?'))
		return
	}
	for _, c := range s {
		Putb(STDOUT, Byte(c))
	}
}

const hexChars = "0123456789ABCDEF"

func printX(arg any) {
	if w, ok := arg.(Word); ok {
		Putb(STDOUT, Byte(hexChars[w>>12]))
		Putb(STDOUT, Byte(hexChars[(w>>8)&0xF]))
		Putb(STDOUT, Byte(hexChars[(w>>4)&0xF]))
		Putb(STDOUT, Byte(hexChars[(w)&0xF]))
	} else if w, ok := arg.(Byte); ok {
		Putb(STDOUT, Byte(hexChars[(w>>4)&0xF]))
		Putb(STDOUT, Byte(hexChars[(w)&0xF]))
	} else {
		Putb(STDOUT, '?')
		Putb(STDOUT, '?')
		Putb(STDOUT, '?')
		Putb(STDOUT, '?')
	}
}
