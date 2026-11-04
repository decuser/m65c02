package spike

import (
	"reflect"
	"testing"
)

type example struct {
	name      string
	program   []byte
	x         byte
	effective uint16
	initial   byte
	cycles    uint64
	wantA     byte
	wantMem   byte
	wantPC    uint16
}

var examples = []example{
	{name: "lda", program: []byte{0xa9, 0x80}, cycles: 2, wantA: 0x80, wantPC: 0x202},
	{name: "sta-nocross", program: []byte{0x9d, 0x00, 0x30}, x: 2, effective: 0x3002, cycles: 5, wantMem: 0x5a, wantPC: 0x203},
	{name: "sta-cross", program: []byte{0x9d, 0xff, 0x30}, x: 2, effective: 0x3101, cycles: 5, wantMem: 0x5a, wantPC: 0x203},
	{name: "inc-nocross", program: []byte{0xfe, 0x00, 0x30}, x: 2, effective: 0x3002, initial: 0x7f, cycles: 7, wantMem: 0x80, wantPC: 0x203},
	{name: "inc-cross", program: []byte{0xfe, 0xff, 0x30}, x: 2, effective: 0x3101, initial: 0x7f, cycles: 7, wantMem: 0x80, wantPC: 0x203},
	{name: "brk", program: []byte{0x00, 0xea}, cycles: 7, wantPC: 0x5678},
}

func setup(e example) (*CPU, *RAM) {
	c := New(0x200)
	c.State.X = e.x
	c.State.A = 0x5a
	m := &RAM{}
	for i, v := range e.program {
		m.Bytes[0x200+uint16(i)] = v
	}
	if e.effective != 0 {
		m.Bytes[e.effective] = e.initial
	}
	m.Bytes[0xfffe] = 0x78
	m.Bytes[0xffff] = 0x56
	return c, m
}

func check(t *testing.T, e example, c *CPU, m *RAM) {
	t.Helper()
	if c.State.Cycles != e.cycles {
		t.Errorf("cycles: got %d want %d", c.State.Cycles, e.cycles)
	}
	if uint64(len(m.Events)) != e.cycles {
		t.Errorf("events: %d want %d", len(m.Events), e.cycles)
	}
	if !c.AtInstructionBoundary() {
		t.Error("not at next SYNC boundary")
	}
	if c.State.PC != e.wantPC {
		t.Errorf("PC %04x != %04x", c.State.PC, e.wantPC)
	}
	if e.name == "lda" && (c.State.A != e.wantA || c.State.P&FlagN == 0) {
		t.Errorf("LDA A/P %02x %02x", c.State.A, c.State.P)
	}
	if e.effective != 0 && m.Bytes[e.effective] != e.wantMem {
		t.Errorf("memory %04x =%02x wanted %02x", e.effective, m.Bytes[e.effective], e.wantMem)
	}
	if e.name == "brk" {
		if c.State.S != 0xfc {
			t.Errorf("S=%02x", c.State.S)
		}
		if m.Bytes[0x1ff] != 0x02 || m.Bytes[0x1fe] != 0x02 || m.Bytes[0x1fd]&FlagB == 0 {
			t.Errorf("stack: %02x %02x %02x", m.Bytes[0x1ff], m.Bytes[0x1fe], m.Bytes[0x1fd])
		}
	}
}

func TestTemplatesAndStepEquivalence(t *testing.T) {
	for _, e := range examples {
		t.Run(e.name, func(t *testing.T) {
			a, ma := setup(e)
			if err := a.StepInstruction(ma); err != nil {
				t.Fatal(err)
			}
			check(t, e, a, ma)
			b, mb := setup(e)
			for b.State.Instructions == 0 {
				if err := b.StepCycle(mb); err != nil {
					t.Fatal(err)
				}
			}
			c, mc := setup(e)
			for c.State.Instructions == 0 {
				if _, err := c.StepMicro(mc); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(a.State, b.State) || !reflect.DeepEqual(a.State, c.State) {
				t.Fatalf("stepping states differ: instruction=%+v cycle=%+v micro=%+v", a.State, b.State, c.State)
			}
			if !reflect.DeepEqual(ma.Events, mb.Events) || !reflect.DeepEqual(ma.Events, mc.Events) {
				t.Fatal("stepping bus traces differ")
			}
		})
	}
}

func TestEveryMicrostepSnapshotAndResolveOnce(t *testing.T) {
	for _, e := range examples {
		t.Run(e.name, func(t *testing.T) {
			// Exercise every microoperation boundary from fresh initial state.
			for n := 0; ; n++ {
				p, m := setup(e)
				for k := 0; k < n && p.State.Instructions == 0; k++ {
					if _, err := p.StepMicro(m); err != nil {
						t.Fatal(err)
					}
				}
				if p.State.Instructions != 0 {
					break
				}
				state := p.Snapshot()
				originalBus := *m
				originalBus.Events = append([]Request(nil), m.Events...)
				restored := &CPU{}
				restored.Restore(state)
				if err := p.StepInstruction(m); err != nil {
					t.Fatal(err)
				}
				if err := restored.StepInstruction(&originalBus); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(p.State, restored.State) || !reflect.DeepEqual(m.Events, originalBus.Events) || m.Bytes != originalBus.Bytes {
					t.Fatalf("snapshot failed at microstep %d: %+v", n, state)
				}
				if n > 100 {
					t.Fatal("unexpectedly long instruction")
				}
			}
		})
	}
}

func TestBusAddressesAndDummyAccess(t *testing.T) {
	for _, e := range examples {
		t.Run(e.name, func(t *testing.T) {
			c, m := setup(e)
			if err := c.StepInstruction(m); err != nil {
				t.Fatal(err)
			}
			if !m.Events[0].Sync || m.Events[0].Addr != 0x200 {
				t.Fatal("opcode fetch SYNC missing")
			}
			for _, event := range m.Events[1:] {
				if event.Sync {
					t.Error("unexpected SYNC")
				}
			}
			if e.effective != 0 {
				dummy := uint16(e.program[2])<<8 | uint16(byte(uint16(e.program[1])+uint16(e.x)))
				if m.Events[3].Addr != dummy || m.Events[3].Write {
					t.Errorf("indexed dummy: %+v wanted %04x read", m.Events[3], dummy)
				}
				if m.Events[4].Addr != e.effective {
					t.Errorf("effective: %+v", m.Events[4])
				}
				if e.name[:3] == "inc" && (m.Events[5].Write || !m.Events[6].Write) {
					t.Error("CMOS RMW dummy read/final write")
				}
			}
			if e.name == "brk" && (m.Events[2].Addr != 0x1ff || m.Events[4].Addr != 0x1fd || m.Events[5].Addr != 0xfffe || m.Events[6].Addr != 0xffff) {
				t.Errorf("BRK trace: %+v", m.Events)
			}
		})
	}
}

func TestInspectNoBusSideEffects(t *testing.T) {
	c, m := setup(examples[0])
	_, err := c.StepMicro(m)
	if err != nil {
		t.Fatal(err)
	}
	before := c.Snapshot()
	events := len(m.Events)
	for i := 0; i < 100; i++ {
		_ = c.Snapshot()
		_ = c.AtInstructionBoundary()
	}
	if len(m.Events) != events || c.State != before {
		t.Fatal("inspection changed state or device")
	}
}

func BenchmarkSpikeInstruction(b *testing.B) {
	c, m := setup(examples[0])
	for i := 0; i < b.N; i++ {
		c = New(0x200)
		if err := c.StepInstruction(m); err != nil {
			b.Fatal(err)
		}
	}
}

type stallBus struct {
	RAM
	remaining int
}

func (m *stallBus) Transfer(r Request) Result {
	result := m.RAM.Transfer(r)
	if !r.Write && m.remaining > 0 {
		m.remaining--
		result.Ready = false
	}
	return result
}

func TestReadStallsRepeatRealTransactions(t *testing.T) {
	e := examples[0]
	reference, normal := setup(e)
	if err := reference.StepInstruction(normal); err != nil {
		t.Fatal(err)
	}
	stalled, base := setup(e)
	m := &stallBus{RAM: *base, remaining: 3}
	if err := stalled.StepInstruction(m); err != nil {
		t.Fatal(err)
	}
	if stalled.State.Cycles != reference.State.Cycles+3 {
		t.Errorf("stalled cycles %d", stalled.State.Cycles)
	}
	if stalled.State.A != reference.State.A || stalled.State.PC != reference.State.PC {
		t.Fatal("stalls changed result")
	}
	if len(m.Events) != len(normal.Events)+3 {
		t.Fatal("repeated reads absent")
	}
	for i := 0; i < 4; i++ {
		if m.Events[i].Addr != 0x200 || m.Events[i].Write {
			t.Fatal("held read is not reissued")
		}
	}
}

type clearOnRead struct {
	RAM
	address uint16
	reads   int
}

func (m *clearOnRead) Transfer(r Request) Result {
	result := m.RAM.Transfer(r)
	if !r.Write && r.Addr == m.address {
		m.reads++
		m.Bytes[r.Addr] = 0
	}
	return result
}
func TestStallWithSideEffectBus(t *testing.T) {
	c, r := setup(examples[0])
	m := &clearOnRead{RAM: *r, address: 0x200}
	// This bus test asserts that distinct physical reads are visible. There is
	// intentionally no stall here; RDY scripting is exercised separately.
	if err := c.StepInstruction(m); err != nil {
		t.Fatal(err)
	}
	if m.reads != 1 || m.Bytes[0x200] != 0 {
		t.Fatal("side-effect read not performed")
	}
	if c.State.A != 0x80 {
		t.Fatal("side-effect read corrupted completed opcode")
	}
}

type stalledClearBus struct {
	clearOnRead
	stalls int
}

func (m *stalledClearBus) Transfer(r Request) Result {
	result := m.clearOnRead.Transfer(r)
	if !r.Write && m.stalls > 0 {
		m.stalls--
		result.Ready = false
	}
	return result
}
func TestStalledSideEffectReadOccursAgain(t *testing.T) {
	c, r := setup(examples[0])
	m := &stalledClearBus{clearOnRead: clearOnRead{RAM: *r, address: 0x200}, stalls: 2}
	// A read-to-clear location is intentionally not invariant under RDY stalls.
	// Two stalled fetches clear the opcode, so the next read sees BRK ($00).
	if err := c.StepInstruction(m); err != nil {
		t.Fatal(err)
	}
	if c.State.PC != 0x5678 || c.State.Cycles != 9 {
		t.Fatalf("repeated read-to-clear did not change fetched opcode: %+v", c.State)
	}
	if m.reads != 3 {
		t.Fatalf("side effect occurred %d times, want 3", m.reads)
	}
	if len(m.Events) != 9 {
		t.Fatalf("expected nine physical transfers: %d", len(m.Events))
	}
}
