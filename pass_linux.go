//go:build linux

package main

import (
	"syscall"
	"unsafe"
)

// linuxTermios mirrors struct termios on Linux x86_64 (and most 64-bit
// architectures). NCCS is 32.
type linuxTermios struct {
	Iflag, Oflag, Cflag, Lflag uint32
	Line                       uint8
	Cc                         [32]uint8
	Ispeed, Ospeed             uint32
}

const (
	echoOff   uint32 = 0o10   // ECHO
	echoNLOff uint32 = 0o100  // ECHONL
	tcsanow   uint32 = 0x5408 // TCSANOW
	tcsadrain uint32 = 0x540a // TCSADRAIN
)

// getPass reads a single line from the controlling terminal with echo
// disabled, so a pasted API key does not appear on screen. Returns the raw
// bytes (without trailing newline) or an error.
func getPass(prompt string) ([]byte, error) {
	if prompt != "" {
		_, _ = sysWrite([]byte(prompt))
	}
	fd := 0 // stdin
	term, err := tcgetattr(fd)
	if err != nil {
		return nil, err
	}
	noEcho := *term
	noEcho.Lflag &^= echoOff | echoNLOff
	if err := tcsetattr(fd, tcsanow, &noEcho); err != nil {
		return nil, err
	}
	defer tcsetattr(fd, tcsadrain, term) // restore original state

	buf := make([]byte, 1)
	var line []byte
	for {
		n, err := syscall.Read(fd, buf)
		if err != nil {
			return nil, err
		}
		if n == 1 && buf[0] == '\n' {
			break
		}
		if n > 0 {
			line = append(line, buf[:n]...)
		}
	}
	return line, nil
}

func sysWrite(b []byte) (int, error) {
	n, _, e := syscall.Syscall(syscall.SYS_WRITE, 1, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	var err error
	if e != 0 {
		err = e
	}
	return int(n), err
}
