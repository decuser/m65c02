package machine

import (
	"errors"
	"os"

	"m65c02/internal/cpu"
)

// KIM1 assembles the original 1 KiB RAM and two mask-programmed 6530s.
// The board's 8 KiB selection pattern repeats through the 64 KiB CPU space.
// Display, keypad, cassette and serial circuitry are separate future devices.
type KIM1 struct {
	*Machine
	RAM                *RAM
	RRIOT002, RRIOT003 *RRIOT
	Serial             *KIMSerial
	TTYSelect          bool
}

// NewKIM1 takes raw, 1024-byte 6530-002 and 6530-003 ROM images.
func NewKIM1(rom002, rom003 []byte) (*KIM1, error) {
	if len(rom002) != 1024 || len(rom003) != 1024 {
		return nil, errors.New("KIM-1 needs 1024-byte ROM images for both 6530s")
	}
	a, err := NewRRIOT(rom002)
	if err != nil {
		return nil, err
	}
	b, err := NewRRIOT(rom003)
	if err != nil {
		return nil, err
	}
	ram, err := NewRAM(1024)
	if err != nil {
		return nil, err
	}
	bus := NewBackplane()
	board := &kim1Decoder{ram: ram, chip002: a, chip003: b}
	if err := bus.Map(0, 0xffff, board); err != nil {
		return nil, err
	}
	bus.ConnectClock(a)
	bus.ConnectClock(b)
	// Original board wiring of the 6530 timer PB7 outputs is not assumed.
	processor := cpu.New(0)
	machine, err := New(processor, bus)
	if err != nil {
		return nil, err
	}
	k := &KIM1{Machine: machine, RAM: ram, RRIOT002: a, RRIOT003: b}
	board.jumpers = k
	k.ResetCPU()
	return k, nil
}

func NewKIM1FromFiles(path002, path003 string) (*KIM1, error) {
	a, err := os.ReadFile(path002)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path003)
	if err != nil {
		return nil, err
	}
	return NewKIM1(a, b)
}

// ResetCPU initiates the CPU's documented reset-vector bus sequence.
// The reset line stays asserted until ReleaseReset is called.
func (k *KIM1) ResetCPU() {
	k.RRIOT002.Reset()
	k.RRIOT003.Reset()
	k.Bus.Reset = true
	k.CPU.SetLines(false, false, true)
}

func (k *KIM1) ReleaseReset() { k.Bus.Reset = false; k.CPU.SetLines(false, false, false) }

// ConnectSerial attaches the KIM-1 TTY wiring to the 6530-002 and clock.
// Call once, before stepping the machine.
func (k *KIM1) ConnectSerial(bitCycles uint64) *KIMSerial {
	if k.Serial != nil {
		return k.Serial
	}
	k.Serial = NewKIMSerial(k.RRIOT002, bitCycles)
	k.Bus.ConnectClock(k.Serial)
	return k.Serial
}

type kim1Decoder struct {
	ram              *RAM
	chip002, chip003 *RRIOT
	jumpers          *KIM1
}

// A-21 connects PA0 (column A) and A-V connects row 3 of the 74145.
// A closed TTY-select jumper makes a keyboard-matrix contact, not an
// unconditional port bit or a software boot-mode switch.
func (b *kim1Decoder) updateJumper() {
	value, mask := b.chip002.OutputB()
	row3 := (mask&0x1e) == 0x1e && (value>>1)&0x0f == 3
	input := b.chip002.input[0] | 1
	if b.jumpers != nil && b.jumpers.TTYSelect && row3 {
		input &^= 1
	}
	b.chip002.SetInputA(input)
}

func (b *kim1Decoder) Read(addr uint16) byte {
	b.updateJumper()
	switch a := addr & 0x1fff; {
	case a < 0x0400:
		return b.ram.Read(a)
	case a >= 0x1700 && a < 0x1710:
		return b.chip003.RegisterDevice().Read(a - 0x1700)
	case a >= 0x1740 && a < 0x1750:
		return b.chip002.RegisterDevice().Read(a - 0x1740)
	case a >= 0x1780 && a < 0x17c0:
		return b.chip003.RAMDevice().Read(a - 0x1780)
	case a >= 0x17c0 && a < 0x1800:
		return b.chip002.RAMDevice().Read(a - 0x17c0)
	case a >= 0x1800 && a < 0x1c00:
		return b.chip003.ROMDevice().Read(a - 0x1800)
	case a >= 0x1c00:
		return b.chip002.ROMDevice().Read(a - 0x1c00)
	default:
		return 0xff
	}
}

func (b *kim1Decoder) Write(addr uint16, value byte) {
	switch a := addr & 0x1fff; {
	case a < 0x0400:
		b.ram.Write(a, value)
	case a >= 0x1700 && a < 0x1710:
		b.chip003.RegisterDevice().Write(a-0x1700, value)
	case a >= 0x1740 && a < 0x1750:
		b.chip002.RegisterDevice().Write(a-0x1740, value)
	case a >= 0x1780 && a < 0x17c0:
		b.chip003.RAMDevice().Write(a-0x1780, value)
	case a >= 0x17c0 && a < 0x1800:
		b.chip002.RAMDevice().Write(a-0x17c0, value)
	}
}
