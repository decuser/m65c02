package machine

import "errors"

// RRIOT models the MCS6530's 1 KiB mask ROM, 64-byte RAM, two 8-bit
// direction-controlled ports and programmable interval timer. The chip's
// selection wiring is expressed as three separate bus-target windows.
type RRIOT struct {
	rom                         [1024]byte
	ram                         [64]byte
	port                        [2]byte
	ddr                         [2]byte
	input                       [2]byte
	count                       byte
	divider                     uint32
	remaining                   uint32
	loaded                      bool
	running, expired, irqEnable bool
}

func NewRRIOT(image []byte) (*RRIOT, error) {
	if len(image) != 1024 {
		return nil, errors.New("6530 ROM must contain exactly 1024 bytes")
	}
	r := &RRIOT{}
	copy(r.rom[:], image)
	r.Reset()
	return r, nil
}

// Reset clears the four port registers and timer control, preserving RAM and
// mask ROM. Input pins default to high (external pull-ups); hosts may change
// them using SetInputA and SetInputB.
func (r *RRIOT) Reset() {
	r.port, r.ddr = [2]byte{}, [2]byte{}
	r.input = [2]byte{0xff, 0xff}
	r.count, r.divider, r.remaining = 0, 1, 0
	r.running, r.expired, r.irqEnable, r.loaded = false, false, false, false
}

func (r *RRIOT) SetInputA(value byte) { r.input[0] = value }
func (r *RRIOT) SetInputB(value byte) { r.input[1] = value }

func (r *RRIOT) PinsA() byte { return (r.port[0] & r.ddr[0]) | (r.input[0] &^ r.ddr[0]) }
func (r *RRIOT) PinsB() byte {
	v := (r.port[1] & r.ddr[1]) | (r.input[1] &^ r.ddr[1])
	if r.irqEnable && r.expired && r.ddr[1]&0x80 == 0 {
		v &^= 0x80
	}
	return v
}
func (r *RRIOT) OutputA() (value, mask byte) { return r.port[0], r.ddr[0] }
func (r *RRIOT) OutputB() (value, mask byte) { return r.port[1], r.ddr[1] }

// TimerIRQ is the active-low PB7 timer signal, represented as a logical
// assertion. The motherboard must explicitly wire it to the CPU IRQ or NMI.
func (r *RRIOT) TimerIRQ() bool {
	return r.irqEnable && r.expired && r.ddr[1]&0x80 == 0
}
func (r *RRIOT) Interrupts() (bool, bool) { return r.TimerIRQ(), false }

// Tick advances one machine clock after bus resolution. A timer load takes
// effect on its write cycle; its first countdown tick is the next cycle.
func (r *RRIOT) Tick() {
	if !r.running {
		return
	}
	if r.loaded {
		r.loaded = false
		return
	}
	if r.expired {
		r.count--
		return
	} // free-running /1 after underflow
	if r.remaining > 0 {
		r.remaining--
	}
	if r.remaining == 0 {
		r.expired = true
		r.count = 0xff
	} else {
		r.count = byte((r.remaining - 1) / r.divider)
	}
}

func (r *RRIOT) loadTimer(value byte, reg uint16) {
	r.count = value
	r.divider = [...]uint32{1, 8, 64, 1024}[reg&3]
	r.remaining = uint32(value)*r.divider + 1
	r.irqEnable = reg&8 != 0
	r.expired, r.loaded, r.running = false, true, true
}

func (r *RRIOT) readTimer(reg uint16) byte {
	if reg&1 != 0 {
		if r.expired {
			return 0x80
		}
		return 0
	}
	v := r.count
	if r.expired {
		// A timer read after underflow restores the programmed prescale
		// from the current count and disables the IRQ output.
		r.remaining = uint32(r.count)*r.divider + 1
		r.loaded, r.expired, r.irqEnable = true, false, false
	} else {
		r.irqEnable = reg&8 != 0
	}
	return v
}

type rriotWindow struct {
	chip *RRIOT
	kind uint8
}

func (w rriotWindow) Size() uint32 {
	switch w.kind {
	case 0:
		return 1024
	case 1:
		return 64
	default:
		return 16
	}
}
func (w rriotWindow) Read(offset uint16) byte {
	switch w.kind {
	case 0:
		if offset < 1024 {
			return w.chip.rom[offset]
		}
	case 1:
		if offset < 64 {
			return w.chip.ram[offset]
		}
	case 2:
		if offset >= 16 {
			break
		}
		switch offset & 7 {
		case 0:
			return w.chip.PinsA()
		case 1:
			return w.chip.ddr[0]
		case 2:
			return w.chip.PinsB()
		case 3:
			return w.chip.ddr[1]
		default:
			return w.chip.readTimer(offset)
		}
	}
	return 0xff
}
func (w rriotWindow) Write(offset uint16, value byte) {
	switch w.kind {
	case 0:
		return // mask ROM
	case 1:
		if offset < 64 {
			w.chip.ram[offset] = value
		}
	case 2:
		if offset >= 16 {
			return
		}
		switch offset & 7 {
		case 0:
			w.chip.port[0] = value
		case 1:
			w.chip.ddr[0] = value
		case 2:
			w.chip.port[1] = value
		case 3:
			w.chip.ddr[1] = value
		default:
			w.chip.loadTimer(value, offset)
		}
	}
}

func (r *RRIOT) ROMDevice() Device      { return rriotWindow{r, 0} }
func (r *RRIOT) RAMDevice() Device      { return rriotWindow{r, 1} }
func (r *RRIOT) RegisterDevice() Device { return rriotWindow{r, 2} }

// RRIOTState retains all mutable chip state. The mask ROM remains unchanged.
type RRIOTState struct {
	RAM                                 [64]byte
	Port, DDR, Input                    [2]byte
	Count                               byte
	Divider, Remaining                  uint32
	Loaded, Running, Expired, IRQEnable bool
}

func (r *RRIOT) Snapshot() RRIOTState {
	return RRIOTState{r.ram, r.port, r.ddr, r.input, r.count,
		r.divider, r.remaining, r.loaded, r.running, r.expired, r.irqEnable}
}
func (r *RRIOT) Restore(s RRIOTState) error {
	if s.Divider != 1 && s.Divider != 8 && s.Divider != 64 && s.Divider != 1024 {
		return errors.New("invalid 6530 timer prescaler")
	}
	r.ram, r.port, r.ddr, r.input = s.RAM, s.Port, s.DDR, s.Input
	r.count, r.divider, r.remaining = s.Count, s.Divider, s.Remaining
	r.loaded, r.running, r.expired, r.irqEnable = s.Loaded, s.Running, s.Expired, s.IRQEnable
	return nil
}
