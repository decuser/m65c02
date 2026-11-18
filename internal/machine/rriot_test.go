package machine

import (
	"reflect"
	"testing"

	"m65c02/internal/cpu"
)

func newTestRRIOT(t *testing.T) *RRIOT {
	t.Helper()
	image := make([]byte, 1024)
	image[0], image[1023] = 0xa9, 0x42
	r, err := NewRRIOT(image)
	if err != nil {
		t.Fatal(err)
	}
	image[0] = 0 // ROM must own a copy
	return r
}

func TestRRIOTMemoryAndPorts(t *testing.T) {
	r := newTestRRIOT(t)
	b := NewBackplane()
	if err := b.Map(0x1800, 0x1bff, r.ROMDevice()); err != nil {
		t.Fatal(err)
	}
	if err := b.Map(0x1780, 0x17bf, r.RAMDevice()); err != nil {
		t.Fatal(err)
	}
	if err := b.Map(0x1700, 0x170f, r.RegisterDevice()); err != nil {
		t.Fatal(err)
	}
	if got := b.Transfer(cpu.Request{Address: 0x1800}).DataIn; got != 0xa9 {
		t.Fatalf("ROM: %02x", got)
	}
	b.Transfer(cpu.Request{Address: 0x1800, Write: true, DataOut: 0xff})
	if got := b.Transfer(cpu.Request{Address: 0x1800}).DataIn; got != 0xa9 {
		t.Fatal("ROM write changed data")
	}
	b.Transfer(cpu.Request{Address: 0x1782, Write: true, DataOut: 0x66})
	if got := b.Transfer(cpu.Request{Address: 0x1782}).DataIn; got != 0x66 {
		t.Fatal("RAM write missing")
	}
	r.SetInputA(0x55)
	b.Transfer(cpu.Request{Address: 0x1701, Write: true, DataOut: 0xf0})
	b.Transfer(cpu.Request{Address: 0x1700, Write: true, DataOut: 0xa0})
	if got := b.Transfer(cpu.Request{Address: 0x1700}).DataIn; got != 0xa5 {
		t.Fatalf("mixed output/input pins: %02x", got)
	}
	if value, mask := r.OutputA(); value != 0xa0 || mask != 0xf0 {
		t.Fatal("output pins not observable")
	}
	r.Reset()
	if r.PinsA() != 0xff || b.Transfer(cpu.Request{Address: 0x1782}).DataIn != 0x66 {
		t.Fatal("reset did not preserve RAM and clear DDR")
	}
}

func TestRRIOTTimerDividers(t *testing.T) {
	for _, tc := range []struct {
		register uint16
		divider  uint32
	}{
		{4, 1}, {5, 8}, {6, 64}, {7, 1024},
	} {
		r := newTestRRIOT(t)
		io := r.RegisterDevice()
		io.Write(tc.register, 2)
		r.Tick() // the write cycle is not a countdown tick
		for i := uint32(0); i < 2*tc.divider; i++ {
			r.Tick()
		}
		if got := io.Read(7); got != 0 {
			t.Fatalf("divider %d expired early: %02x", tc.divider, got)
		}
		if got := io.Read(6); got != 0 {
			t.Fatalf("divider %d count=%d want 0", tc.divider, got)
		}
		r.Tick()
		if got := io.Read(7); got != 0x80 {
			t.Fatalf("divider %d did not expire: %02x", tc.divider, got)
		}
		if got := io.Read(6); got != 0xff {
			t.Fatalf("divider %d post-underflow count=%02x", tc.divider, got)
		}
	}
}

func TestRRIOTTimerReferenceExampleAndIRQ(t *testing.T) {
	r := newTestRRIOT(t)
	io := r.RegisterDevice()
	io.Write(5, 52) // manual example: 52*8+1 clocks until underflow
	r.Tick()
	for i := 0; i < 213; i++ {
		r.Tick()
	}
	if got := io.Read(6); got != 25 {
		t.Fatalf("reference count at t=213: %d, want 25", got)
	}
	for i := 213; i < 416; i++ {
		r.Tick()
	}
	if got := io.Read(6); got != 0 {
		t.Fatalf("reference count at t=416: %d", got)
	}
	r.Tick()
	if got := io.Read(7); got != 0x80 {
		t.Fatalf("underflow status: %02x", got)
	}
	io.Write(0x0c, 0) // /1 with interrupt enabled
	r.Tick()
	if r.TimerIRQ() {
		t.Fatal("IRQ asserted on load cycle")
	}
	r.Tick()
	if !r.TimerIRQ() || r.PinsB()&0x80 != 0 {
		t.Fatal("PB7 timer interrupt not asserted")
	}
	io.Read(6)
	if r.TimerIRQ() {
		t.Fatal("timer read did not acknowledge IRQ")
	}
	io.Write(0x0c, 0)
	r.Tick()
	r.Tick()
	io.Write(3, 0x80) // PB7 configured as ordinary output
	if r.TimerIRQ() {
		t.Fatal("timer drove output-configured PB7")
	}
}

func TestRRIOTRestore(t *testing.T) {
	r := newTestRRIOT(t)
	r.RAMDevice().Write(3, 0x71)
	r.RegisterDevice().Write(0x0f, 8)
	r.Tick()
	for i := 0; i < 43; i++ {
		r.Tick()
	}
	s := r.Snapshot()
	clone := newTestRRIOT(t)
	if err := clone.Restore(s); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 101; i++ {
		r.Tick()
		clone.Tick()
	}
	if !reflect.DeepEqual(r.Snapshot(), clone.Snapshot()) {
		t.Fatal("timer diverged after restore")
	}
	s.Divider = 3
	if err := clone.Restore(s); err == nil {
		t.Fatal("invalid prescaler accepted")
	}
}
