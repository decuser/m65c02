package machine

import (
	"testing"

	"m65c02/internal/cpu"
)

type testMemory struct {
	data                 [65536]byte
	reads, writes, ticks int
	irq                  bool
}

func (m *testMemory) Read(addr uint16) byte     { m.reads++; return m.data[addr] }
func (m *testMemory) Write(addr uint16, v byte) { m.writes++; m.data[addr] = v }
func (m *testMemory) Tick()                     { m.ticks++ }
func (m *testMemory) Interrupts() (bool, bool)  { return m.irq, false }

func TestBackplaneAddressDecoding(t *testing.T) {
	b := NewBackplane()
	a, other := &testMemory{}, &testMemory{}
	a.data[2], other.data[2] = 0x55, 0xaa
	if err := b.Map(0x1000, 0x10ff, a); err != nil {
		t.Fatal(err)
	}
	if err := b.Map(0x2000, 0x20ff, other); err != nil {
		t.Fatal(err)
	}
	if err := b.Map(0x10ff, 0x1100, other); err == nil {
		t.Fatal("accepted overlapping map")
	}
	if err := b.Map(0x3000, 0x2fff, a); err == nil {
		t.Fatal("accepted inverted map")
	}
	if r := b.Transfer(cpu.Request{Address: 0x1002}); r.DataIn != 0x55 || !r.Ready {
		t.Fatalf("first device: %+v", r)
	}
	if r := b.Transfer(cpu.Request{Address: 0x2002}); r.DataIn != 0xaa {
		t.Fatalf("second device: %+v", r)
	}
	b.Transfer(cpu.Request{Address: 0x2002, Write: true, DataOut: 0x42})
	if other.data[2] != 0x42 || a.data[2] != 0x55 {
		t.Fatal("wrong target selected")
	}
	if r := b.Transfer(cpu.Request{Address: 0x3000}); r.DataIn != 0xff {
		t.Fatalf("floating read: %+v", r)
	}
	if b.Cycles != 4 {
		t.Fatalf("cycles=%d", b.Cycles)
	}
}

func TestCPUExecutesThroughBackplane(t *testing.T) {
	ram := &testMemory{}
	copy(ram.data[0x200:], []byte{0xa9, 0x42, 0x8d, 0x00, 0x03, 0xea})
	bus := NewBackplane()
	if err := bus.Map(0, 0xffff, ram); err != nil {
		t.Fatal(err)
	}
	bus.ConnectClock(ram)
	m, err := New(cpu.New(0x200), bus)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.StepInstruction(); err != nil {
		t.Fatal(err)
	}
	if err := m.StepInstruction(); err != nil {
		t.Fatal(err)
	}
	if ram.data[0x300] != 0x42 || m.CPU.View().PC != 0x205 {
		t.Fatalf("execution: %+v, RAM=%02x", m.CPU.View(), ram.data[0x300])
	}
	if bus.Cycles != 6 || ram.ticks != 6 || m.CPU.View().Cycles != 6 {
		t.Fatalf("clock drift: backplane=%d device=%d CPU=%d", bus.Cycles, ram.ticks, m.CPU.View().Cycles)
	}
	if ram.writes != 1 || !m.CPU.AtBoundary() {
		t.Fatal("incorrect write or instruction boundary")
	}
	if err := m.Clock(); err != nil {
		t.Fatal(err)
	}
	if bus.Cycles != 7 || ram.ticks != 7 {
		t.Fatal("Clock did not advance one machine cycle")
	}
}

func TestMicrostepPauseAndRDY(t *testing.T) {
	ram := &testMemory{}
	ram.data[0x200] = 0xea
	b := NewBackplane()
	if err := b.Map(0, 0xffff, ram); err != nil {
		t.Fatal(err)
	}
	b.ConnectClock(ram)
	b.ConnectInterrupt(ram)
	m, err := New(cpu.New(0x200), b)
	if err != nil {
		t.Fatal(err)
	}
	b.Ready = false
	if err := m.Clock(); err != nil {
		t.Fatal(err)
	}
	if b.Cycles != 1 || ram.reads != 1 || ram.ticks != 1 {
		t.Fatal("stalled physical cycle was lost")
	}
	b.Ready = true
	if _, err := m.StepMicro(); err != nil {
		t.Fatal(err)
	} // re-present only
	if b.Cycles != 1 {
		t.Fatal("microstep advanced machine time before Resolve")
	}
	if err := m.Clock(); err != nil {
		t.Fatal(err)
	}
	if b.Cycles != 2 || ram.reads != 2 || ram.ticks != 2 {
		t.Fatal("RDY retry did not repeat one physical read")
	}
	ram.irq = true
	if r := b.Transfer(cpu.Request{Address: 0x200}); !r.IRQ {
		t.Fatal("device IRQ not wired")
	}
}
