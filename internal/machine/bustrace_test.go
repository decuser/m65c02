package machine

import (
	"m65c02/internal/cpu"
	"testing"
)

func TestBusTraceRingAndPagination(t *testing.T) {
	b := NewBackplane()
	b.Trace = NewBusTrace(3)
	for i := uint16(0); i < 5; i++ {
		b.Transfer(cpu.Request{Address: 0x200 + i, DataOut: byte(i), Write: true, Sync: i == 4})
	}
	last := b.Trace.Window(0, 10)
	if len(last) != 3 || last[0].Cycle != 3 || last[2].Cycle != 5 || last[2].Address != 0x204 || last[2].Data != 4 || !last[2].Sync {
		t.Fatalf("wrong ring contents: %+v", last)
	}
	older := b.Trace.Window(5, 2)
	if len(older) != 2 || older[0].Cycle != 3 || older[1].Cycle != 4 {
		t.Fatalf("wrong older page: %+v", older)
	}
	b.Trace.Clear()
	if got := len(b.Trace.Window(0, 10)); got != 0 {
		t.Fatalf("clear retained %d cycles", got)
	}
}

func TestBusTraceCapturesFetchedInstruction(t *testing.T) {
	b := NewBackplane()
	b.Trace = NewBusTrace(4)
	b.Transfer(cpu.Request{Address: 0x0200, Sync: true}) // floating bus reads $FF
	got := b.Trace.Window(0, 1)
	if len(got) != 1 || got[0].Mnemonic == "" || got[0].Length < 1 || !got[0].Sync {
		t.Fatalf("missing opcode annotation: %+v", got)
	}
}

func TestBusTraceCapturesRRIOTPinsAtSameCycle(t *testing.T) {
	b := NewBackplane()
	b.Trace = NewBusTrace(4)
	b.Trace.PortSampler = func() BoardPins {
		return BoardPins{R002: RRIOTPins{PA: 0x3f, PB: 0x08, DDRA: 0x7f, DDRB: 0x1e},
			R003: RRIOTPins{PA: 0x55, PB: 0xaa, DDRA: 0xff}, Digit: 0, Segments: 0x3f}
	}
	b.Transfer(cpu.Request{Address: 0x1740, Write: true, DataOut: 0x3f})
	rows := b.Trace.Window(0, 1)
	if len(rows) != 1 || rows[0].Pins == nil {
		t.Fatal("RIOT sample missing")
	}
	c := rows[0]
	if c.Cycle != 1 || c.Pins.R002.PA != 0x3f || c.Pins.R003.PB != 0xaa || c.Pins.Digit != 0 || c.Pins.Segments != 0x3f {
		t.Fatalf("wrong RIOT sample: %+v", c)
	}
}

func TestBusTracePortChanges(t *testing.T) {
	b := NewBackplane()
	b.Trace = NewBusTrace(8)
	pins := BoardPins{Digit: -1}
	b.Trace.PortSampler = func() BoardPins { return pins }
	transfer := func() { b.Transfer(cpu.Request{Address: 0x1740}) }
	transfer() // baseline
	transfer() // no change
	pins.R002.PA = 0x71
	pins.Digit = 2
	transfer() // display event
	pins.R003.PB = 0x42
	transfer() // application event
	pins.R002.DDRA = 0x7f
	transfer() // display direction event
	display := b.Trace.PortChanges("riot002")
	application := b.Trace.PortChanges("riot003")
	if len(display) != 2 || display[0].Cycle != 3 || display[1].Cycle != 5 {
		t.Fatalf("display changes: %+v", display)
	}
	if len(application) != 1 || application[0].Cycle != 4 {
		t.Fatalf("application changes: %+v", application)
	}
	if got := b.Trace.PortChanges("cpu"); len(got) != 0 {
		t.Fatalf("unexpected changes for unsupported probe: %+v", got)
	}
}
