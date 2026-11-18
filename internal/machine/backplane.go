// Package machine connects the cycle-driven CPU to address-mapped components.
// It models logical bus transactions, not analog electrical behavior.
package machine

import (
	"errors"

	"m65c02/internal/cpu"
)

// Device is a bus target. Addresses are offsets relative to its mapped base.
// Read and Write each represent one resolved physical bus transaction.
type Device interface {
	Read(offset uint16) byte
	Write(offset uint16, value byte)
}

// Ticker is optional. A connected clock sink advances once per physical
// machine cycle, including a repeated read cycle caused by RDY being low.
type Ticker interface{ Tick() }

// InterruptSource is optional. The backplane combines device signals with
// externally asserted lines; it does not choose CPU interrupt priorities.
type InterruptSource interface{ Interrupts() (irq, nmi bool) }

type region struct {
	start, end uint16
	device     Device
}

// Backplane supplies bus decoding, a physical-cycle counter and control lines.
// Its floating-read value defaults to $FF; writes to empty space are ignored.
type Backplane struct {
	regions          []region
	clockSinks       []Ticker
	interruptSources []InterruptSource
	Cycles           uint64
	Floating         byte
	Ready            bool
	IRQ, NMI, Reset  bool
	LastRequest      cpu.Request
	Trace            *BusTrace
}

func NewBackplane() *Backplane { return &Backplane{Floating: 0xff, Ready: true} }

// ConnectClock and ConnectInterrupt attach independent component connections.
// A device mapped into multiple address windows should be connected only once.
func (b *Backplane) ConnectClock(t Ticker) { b.clockSinks = append(b.clockSinks, t) }
func (b *Backplane) ConnectInterrupt(s InterruptSource) {
	b.interruptSources = append(b.interruptSources, s)
}

// Map binds an inclusive address range to a bus target. Overlapping ranges
// are rejected; mirrored or banked mappings can be added as explicit devices.
func (b *Backplane) Map(start, end uint16, device Device) error {
	if device == nil || end < start {
		return errors.New("invalid device range")
	}
	if sized, ok := device.(Sized); ok && uint32(end)-uint32(start)+1 > sized.Size() {
		return errors.New("mapping exceeds device capacity")
	}
	for _, r := range b.regions {
		if start <= r.end && end >= r.start {
			return errors.New("overlapping device range")
		}
	}
	b.regions = append(b.regions, region{start, end, device})
	return nil
}

// Transfer is the only bus interaction exposed to the CPU. Device callbacks
// happen once per Resolve, independent of how many teaching microsteps occur.
func (b *Backplane) Transfer(req cpu.Request) cpu.Response {
	b.LastRequest = req
	in := b.Floating
	for _, r := range b.regions {
		if req.Address >= r.start && req.Address <= r.end {
			offset := req.Address - r.start
			if req.Write {
				r.device.Write(offset, req.DataOut)
			} else {
				in = r.device.Read(offset)
			}
			break
		}
	}
	b.Cycles++
	irq, nmi := b.IRQ, b.NMI
	for _, t := range b.clockSinks {
		t.Tick()
	}
	for _, s := range b.interruptSources {
		i, n := s.Interrupts()
		irq, nmi = irq || i, nmi || n
	}
	res := cpu.Response{DataIn: in, Ready: b.Ready, IRQ: irq, NMI: nmi, Reset: b.Reset}
	b.Trace.Record(b.Cycles, req, res)
	return res
}

// Machine is an assembly of one CPU and its backplane. The host chooses when
// to call Clock; frequency, pacing, and user-interface work stay outside it.
type Machine struct {
	CPU *cpu.CPU
	Bus *Backplane
}

func New(processor *cpu.CPU, bus *Backplane) (*Machine, error) {
	if processor == nil || bus == nil {
		return nil, errors.New("machine requires CPU and backplane")
	}
	return &Machine{CPU: processor, Bus: bus}, nil
}

func (m *Machine) Clock() error                       { return m.CPU.StepCycle(m.Bus) }
func (m *Machine) StepInstruction() error             { return m.CPU.StepInstruction(m.Bus) }
func (m *Machine) StepMicro() (cpu.MicroEvent, error) { return m.CPU.StepMicro(m.Bus) }
