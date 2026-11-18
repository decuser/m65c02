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

// BaudRate reads the slave's termios speed through the PTY master.
// A PTY carries bytes without electrical pacing, but terminal programs
// still configure this metadata (e.g. Minicom's 1200-baud setting).
func (p *PTY) BaudRate() (uint64, error) {
	var term syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, p.Master.Fd(), syscall.TCGETS, uintptr(unsafe.Pointer(&term)))
	if errno != 0 {
		return 0, errno
	}
	// Linux's standard termios speed bits occupy CBAUD's low 4 bits.
	speeds := map[uint32]uint64{
		syscall.B50: 50, syscall.B75: 75,
		syscall.B110: 110, syscall.B300: 300, syscall.B600: 600,
		syscall.B134: 134, syscall.B150: 150, syscall.B200: 200,
		syscall.B1200: 1200, syscall.B1800: 1800, syscall.B2400: 2400,
		syscall.B4800: 4800, syscall.B9600: 9600, syscall.B19200: 19200,
		syscall.B38400: 38400,
	}
	baud, ok := speeds[term.Cflag&0xf]
	if !ok {
		return 0, fmt.Errorf("unsupported PTY baud encoding %#x", term.Cflag&0xf)
	}
	return baud, nil
}
