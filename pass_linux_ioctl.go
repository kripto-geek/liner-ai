//go:build linux

package main

import (
	"syscall"
	"unsafe"
)

// IOCTL numbers for termios on Linux.
const (
	ioctlTCGETS = 0x5401
	ioctlTCSETS = 0x5402 // TCSANOW
	ioctlTCSETSW = 0x5403 // TCSADRAIN
)

func tcgetattr(fd int) (*linuxTermios, error) {
	var t linuxTermios
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(ioctlTCGETS),
		uintptr(unsafe.Pointer(&t)),
	)
	if errno != 0 {
		return nil, errno
	}
	return &t, nil
}

func tcsetattr(fd int, when uint32, t *linuxTermios) error {
	var ioctlNum uintptr
	switch when {
	case tcsanow:
		ioctlNum = ioctlTCSETS
	case tcsadrain:
		ioctlNum = ioctlTCSETSW
	default:
		ioctlNum = ioctlTCSETS
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		ioctlNum,
		uintptr(unsafe.Pointer(t)),
	)
	if errno != 0 {
		return errno
	}
	return nil
}
