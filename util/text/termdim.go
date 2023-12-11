package text

import (
	"syscall"
	"unsafe"
)

func GetTermDim() (width, height int, err error) {
	var termDim [4]uint16
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(0), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&termDim)), 0, 0, 0); err != 0 {
		return -1, -1, err
	}
	return int(termDim[1]), int(termDim[0]), nil
}
