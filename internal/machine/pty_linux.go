//go:build linux
// +build linux

package machine

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// PTY creates a Linux pseudoterminal pair. A host terminal program opens
// SlavePath; the emulator retains Master for serial byte transfer.
type PTY struct {
	Master    *os.File
	SlavePath string
}

func ioctlNumber(fd uintptr, request uintptr, value *uint32) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, request, uintptr(unsafe.Pointer(value)))
	if errno != 0 {
		return errno
	}
	return nil
}

func NewPTY() (*PTY, error) {
	f, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, err
	}
	var unlock uint32
	if err = ioctlNumber(f.Fd(), syscall.TIOCSPTLCK, &unlock); err != nil {
		f.Close()
		return nil, fmt.Errorf("unlock PTY: %w", err)
	}
	var number uint32
	if err = ioctlNumber(f.Fd(), syscall.TIOCGPTN, &number); err != nil {
		f.Close()
		return nil, fmt.Errorf("PTY number: %w", err)
	}
	if err = syscall.SetNonblock(int(f.Fd()), true); err != nil {
		f.Close()
		return nil, err
	}
	return &PTY{Master: f, SlavePath: fmt.Sprintf("/dev/pts/%d", number)}, nil
}

func (p *PTY) Close() error { return p.Master.Close() }
