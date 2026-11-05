package cpu

import (
	"reflect"
	"testing"
)

type RAM struct {
	Data   [65536]byte
	Events []Request
}

func (m *RAM) Transfer(r Request) Response {
	m.Events = append(m.Events, r)
	if r.Write {
		m.Data[r.Address] = r.DataOut
		return Response{Ready: true}
	}
	return Response{DataIn: m.Data[r.Address], Ready: true}
}
func memory(code ...byte) *RAM {
	m := &RAM{}
	copy(m.Data[0x200:], code)
	m.Data[0xfffe] = 0x00
	m.Data[0xffff] = 0x04
	return m
}
func requests(m *RAM) []Request { return append([]Request(nil), m.Events...) }
func eqAddresses(t *testing.T, m *RAM, expected ...uint16) {
	t.Helper()
	if len(m.Events) != len(expected) {
		t.Fatalf("got %d cycles, want %d: %v", len(m.Events), len(expected), m.Events)
	}
	for i, w := range expected {
		if m.Events[i].Address != w {
			t.Fatalf("cycle %d address %04x, want %04x", i, m.Events[i].Address, w)
		}
	}
}
func TestInstructions(t *testing.T) {
	tests := []struct {
		name           string
		code           []byte
		x              byte
		cycles         int
		addr           []uint16
		wantA, wantMem byte
		wantPC         uint16
	}{
		{"LDA immediate", []byte{0xa9, 0x80}, 0, 2, []uint16{0x200, 0x201}, 0x80, 0, 0x202},
		{"LDA indexed no cross", []byte{0xbd, 0x00, 0x30}, 1, 4, []uint16{0x200, 0x201, 0x202, 0x3001}, 0x09, 0x09, 0x203},
		{"LDA indexed cross", []byte{0xbd, 0xff, 0x30}, 1, 5, []uint16{0x200, 0x201, 0x202, 0x202, 0x3100}, 0x09, 0x09, 0x203},
		{"STA indexed", []byte{0x9d, 0xff, 0x30}, 1, 5, []uint16{0x200, 0x201, 0x202, 0x202, 0x3100}, 0, 0, 0x203},
		{"INC indexed no cross", []byte{0xfe, 0x00, 0x30}, 1, 7, []uint16{0x200, 0x201, 0x202, 0x202, 0x3001, 0x3001, 0x3001}, 0, 0x0a, 0x203},
		{"INC indexed cross", []byte{0xfe, 0xff, 0x30}, 1, 7, []uint16{0x200, 0x201, 0x202, 0x202, 0x3100, 0x3100, 0x3100}, 0, 0x0a, 0x203},
		{"JMP", []byte{0x4c, 0x00, 0x04}, 0, 3, []uint16{0x200, 0x201, 0x202}, 0, 0, 0x400},
		{"NOP", []byte{0xea, 0x00}, 0, 2, []uint16{0x200, 0x201}, 0, 0, 0x201},
		{"one-cycle NOP", []byte{0x03, 0x03}, 0, 1, []uint16{0x200}, 0, 0, 0x201},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := memory(tc.code...)
			c := New(0x200)
			c.state.X = tc.x
			c.state.A = 0
			m.Data[0x3001] = 9
			m.Data[0x3100] = 9
			if tc.code[0] == 0x9d {
				c.state.A = 0
			}
			if err := c.StepInstruction(m); err != nil {
				t.Fatal(err)
			}
			eqAddresses(t, m, tc.addr...)
			s := c.View()
			if s.PC != tc.wantPC {
				t.Fatalf("PC=%04x want %04x", s.PC, tc.wantPC)
			}
			if s.Cycles != uint64(tc.cycles) {
				t.Fatalf("cycles=%d want %d", s.Cycles, tc.cycles)
			}
			if tc.code[0] == 0xbd || tc.code[0] == 0xa9 {
				if s.A != tc.wantA {
					t.Fatalf("A=%02x want %02x", s.A, tc.wantA)
				}
			}
			if tc.code[0] == 0xfe {
				if m.Data[s.EA] != tc.wantMem {
					t.Fatalf("INC data=%02x want %02x", m.Data[s.EA], tc.wantMem)
				}
			}
			if !c.AtBoundary() {
				t.Fatal("missing prepared next SYNC")
			}
		})
	}
}
func TestBackToBackSYNC(t *testing.T) {
	m := memory(0x03, 0x03, 0xea)
	c := New(0x200)
	for i := 0; i < 2; i++ {
		if err := c.StepInstruction(m); err != nil {
			t.Fatal(err)
		}
		s := c.View()
		if !c.AtBoundary() || s.PC != 0x201+uint16(i) || s.Cycles != uint64(i+1) || s.Instructions != uint64(i+1) {
			t.Fatalf("boundary %d: %+v", i, s)
		}
	}
	eqAddresses(t, m, 0x200, 0x201)
	for _, v := range m.Events {
		if !v.Sync {
			t.Fatal("non SYNC")
		}
	}
}
func TestBRK(t *testing.T) {
	m := memory(0x00, 0x99)
	c := New(0x200)
	c.state.P |= FlagD
	if err := c.StepInstruction(m); err != nil {
		t.Fatal(err)
	}
	eqAddresses(t, m, 0x200, 0x201, 0x1ff, 0x1fe, 0x1fd, 0xfffe, 0xffff)
	s := c.View()
	if s.PC != 0x400 || s.S != 0xfc || s.P&FlagD != 0 || s.P&FlagI == 0 {
		t.Fatalf("BRK %+v", s)
	}
	if m.Data[0x1ff] != 0x02 || m.Data[0x1fe] != 0x02 || m.Data[0x1fd]&FlagB == 0 || m.Data[0x1fd]&FlagD == 0 {
		t.Fatalf("stack: %02x %02x %02x", m.Data[0x1ff], m.Data[0x1fe], m.Data[0x1fd])
	}
}
func TestBNE(t *testing.T) {
	for _, tc := range []struct {
		name string
		pc   uint16
		zero bool
		addr []uint16
		end  uint16
	}{{"untaken", 0x200, true, []uint16{0x200, 0x201}, 0x202}, {"taken", 0x200, false, []uint16{0x200, 0x201, 0x202}, 0x204}, {"cross", 0x2fc, false, []uint16{0x2fc, 0x2fd, 0x2fe, 0x200}, 0x300}} {
		t.Run(tc.name, func(t *testing.T) {
			m := &RAM{}
			m.Data[tc.pc] = 0xd0
			m.Data[tc.pc+1] = 0x02
			c := New(tc.pc)
			if tc.zero {
				c.state.P |= FlagZ
			}
			if err := c.StepInstruction(m); err != nil {
				t.Fatal(err)
			}
			eqAddresses(t, m, tc.addr...)
			if c.View().PC != tc.end {
				t.Fatalf("PC %04x != %04x", c.View().PC, tc.end)
			}
		})
	}
}
func TestAllStepModesAndRestore(t *testing.T) {
	program := []byte{0xfe, 0xff, 0x30, 0xd0, 0xfb}
	referenceBus := memory(program...)
	referenceBus.Data[0x3100] = 7
	reference := New(0x200)
	reference.state.X = 1
	for i := 0; i < 2; i++ {
		if err := reference.StepInstruction(referenceBus); err != nil {
			t.Fatal(err)
		}
	}
	wantState, wantBus, wantEvents := reference.View(), referenceBus.Data, requests(referenceBus)
	for _, mode := range []string{"cycle", "micro"} {
		m := memory(program...)
		m.Data[0x3100] = 7
		c := New(0x200)
		c.state.X = 1
		for c.View().Instructions < 2 {
			var err error
			if mode == "cycle" {
				err = c.StepCycle(m)
			} else {
				_, err = c.StepMicro(m)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		if !reflect.DeepEqual(c.View(), wantState) || m.Data != wantBus || !reflect.DeepEqual(requests(m), wantEvents) {
			t.Fatalf("%s differs from instruction reference", mode)
		}
	}
	for cut := 0; cut < 28; cut++ {
		m := memory(program...)
		m.Data[0x3100] = 7
		c := New(0x200)
		c.state.X = 1
		for i := 0; i < cut && c.View().Instructions < 2; i++ {
			if _, err := c.StepMicro(m); err != nil {
				t.Fatal(err)
			}
		}
		snap := c.Snapshot()
		copyBus := *m
		copyBus.Events = append([]Request(nil), m.Events...)
		restored := New(0x111)
		if err := restored.Restore(snap); err != nil {
			t.Fatalf("cut %d restore: %v", cut, err)
		}
		for restored.View().Instructions < 2 {
			if _, err := restored.StepMicro(&copyBus); err != nil {
				t.Fatal(err)
			}
		}
		if !reflect.DeepEqual(restored.View(), wantState) || copyBus.Data != wantBus || !reflect.DeepEqual(requests(&copyBus), wantEvents) {
			t.Fatalf("snapshot cut %d differs", cut)
		}
	}
}
func TestRDYReadStall(t *testing.T) {
	m := memory(0xa9, 0x55)
	bus := &stallBus{ram: m, remaining: 2}
	c := New(0x200)
	if err := c.StepInstruction(bus); err != nil {
		t.Fatal(err)
	}
	eqAddresses(t, m, 0x200, 0x200, 0x200, 0x201)
	if c.View().Cycles != 4 || c.View().A != 0x55 {
		t.Fatalf("unexpected: %+v", c.View())
	}
}

type stallBus struct {
	ram       *RAM
	remaining int
}

func (b *stallBus) Transfer(r Request) Response {
	out := b.ram.Transfer(r)
	if b.remaining > 0 {
		b.remaining--
		out.Ready = false
	}
	return out
}
func TestNoAllocationsSteadyState(t *testing.T) {
	m := &noAllocBus{}
	m.data[0x200] = 0x03
	c := New(0x200)
	alloc := testing.AllocsPerRun(500, func() {
		c.state.PC = 0x200
		c.state.Stage = 0
		c.state.Phase = PreBus
		c.selectEntry()
		if err := c.StepInstruction(m); err != nil {
			panic(err)
		}
	})
	if alloc != 0 {
		t.Fatalf("CPU execution allocs %f", alloc)
	}
}

type noAllocBus struct{ data [65536]byte }

func (b *noAllocBus) Transfer(r Request) Response {
	if r.Write {
		b.data[r.Address] = r.DataOut
		return Response{Ready: true}
	}
	return Response{DataIn: b.data[r.Address], Ready: true}
}
func BenchmarkProductionNOP(b *testing.B) {
	bus := &noAllocBus{}
	bus.data[0x200] = 0x03
	c := New(0x200)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c.state.PC = 0x200
		c.state.Stage = 0
		c.state.Phase = PreBus
		c.selectEntry()
		if err := c.StepInstruction(bus); err != nil {
			b.Fatal(err)
		}
	}
}

// Pausing after Resolve must not repeat a side-effecting bus read.
func TestResolvedSnapshotDoesNotReadTwice(t *testing.T) {
	m := memory(0xa9, 0x15)
	c := New(0x200)
	if _, err := c.StepMicro(m); err != nil {
		t.Fatal(err)
	} // opcode fetch resolves
	if c.View().Phase != PostBus {
		t.Fatal("not after Resolve")
	}
	snapshot := c.Snapshot()
	copied := *m
	copied.Events = append([]Request(nil), m.Events...)
	other := New(0x500)
	if err := other.Restore(snapshot); err != nil {
		t.Fatal(err)
	}
	if err := other.StepInstruction(&copied); err != nil {
		t.Fatal(err)
	}
	eqAddresses(t, &copied, 0x200, 0x201)
	if copied.Data[0x200] != 0xa9 || other.View().A != 0x15 {
		t.Fatal("restore changed execution")
	}
}
func TestRDYInRMW(t *testing.T) {
	m := memory(0xfe, 0x00, 0x30)
	m.Data[0x3000] = 0x0f
	bus := &stallOnAddress{ram: m, address: 0x3000, remaining: 2}
	c := New(0x200)
	if err := c.StepInstruction(bus); err != nil {
		t.Fatal(err)
	}
	eqAddresses(t, m, 0x200, 0x201, 0x202, 0x202, 0x3000, 0x3000, 0x3000, 0x3000, 0x3000)
	if m.Data[0x3000] != 0x10 || c.View().Cycles != 9 {
		t.Fatalf("RMW stall: %+v memory=%02x", c.View(), m.Data[0x3000])
	}
}

type stallOnAddress struct {
	ram       *RAM
	address   uint16
	remaining int
}

func (b *stallOnAddress) Transfer(r Request) Response {
	reply := b.ram.Transfer(r)
	if !r.Write && r.Address == b.address && b.remaining > 0 {
		b.remaining--
		reply.Ready = false
	}
	return reply
}
