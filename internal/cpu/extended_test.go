package cpu

import (
	"reflect"
	"testing"
)

// A finite program uses new instruction families and produces a visible result.
func TestProductionLoopWithExtendedInstructions(t *testing.T) {
	b := &RAM{}
	code := []byte{
		0xa2, 0x00, // LDX #$00
		0x64, 0x20, // STZ $20
		0xe8,       // loop: INX
		0x86, 0x21, // STX $21
		0xe0, 0x0a, // CPX #$0A
		0xd0, 0xf9, // BNE loop
		0xa5, 0x21, // LDA $21
		0x8d, 0x00, 0x03, // STA $0300
		0x00, 0x00, // BRK
	}
	copy(b.Data[0x200:], code)
	b.Data[0xfffe] = 0x00
	b.Data[0xffff] = 0x04
	c := New(0x200)
	for i := 0; i < 45; i++ {
		if err := c.StepInstruction(b); err != nil {
			t.Fatal(err)
		}
	}
	s := c.View()
	if b.Data[0x21] != 10 || b.Data[0x300] != 10 || s.A != 10 || s.X != 10 || s.PC != 0x0400 || s.Instructions != 45 || s.Cycles != 118 {
		t.Fatalf("loop not correct: RAM21=%02x RAM0300=%02x state=%+v", b.Data[0x21], b.Data[0x300], s)
	}
}

func extendedMemory() *RAM {
	b := &RAM{}
	copy(b.Data[0x200:], []byte{
		0xa0, 0x01, // LDY #1
		0xb1, 0x30, // LDA ($30),Y, page crossing
		0xf8,       // SED
		0x69, 0x19, // ADC #$19, extra decimal cycle
		0x8d, 0x00, 0x03, // STA $0300
		0xea, // NOP
	})
	b.Data[0x30] = 0xff
	b.Data[0x31] = 0x30
	b.Data[0x3100] = 0x22
	b.Data[0x56] = 0xa5
	return b
}

func TestExtendedGranularityAndRestore(t *testing.T) {
	baseline := New(0x200)
	refBus := extendedMemory()
	for i := 0; i < 6; i++ {
		if err := baseline.StepInstruction(refBus); err != nil {
			t.Fatal(err)
		}
	}
	wantState := baseline.View()
	wantEvents := requests(refBus)
	// Physical cycles and microoperations must produce the same bus trace.
	cyc := New(0x200)
	cycBus := extendedMemory()
	for cyc.View().Instructions < 6 {
		if err := cyc.StepCycle(cycBus); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(wantEvents, requests(cycBus)) || !reflect.DeepEqual(wantState, cyc.View()) {
		t.Fatal("cycle stepping diverges")
	}
	micro := New(0x200)
	microBus := extendedMemory()
	for micro.View().Instructions < 6 {
		if _, err := micro.StepMicro(microBus); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(wantEvents, requests(microBus)) || !reflect.DeepEqual(wantState, micro.View()) {
		t.Fatal("micro stepping diverges")
	}
	// Pause immediately after Resolve, with the bus transaction already applied.
	paused := New(0x200)
	pausedBus := extendedMemory()
	for i := 0; i < 2000; i++ {
		s := paused.View()
		if s.Phase == PostBus && s.ReplyValid && s.Instructions == 1 {
			break
		}
		if _, err := paused.StepMicro(pausedBus); err != nil {
			t.Fatal(err)
		}
	}
	save := paused.Snapshot()
	if save.Phase != PostBus || !save.ReplyValid {
		t.Fatal("failed to pause after Resolve")
	}
	resumed := New(0x100)
	if err := resumed.Restore(save); err != nil {
		t.Fatal(err)
	}
	for resumed.View().Instructions < 6 {
		if err := resumed.StepInstruction(pausedBus); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(wantState, resumed.View()) || !reflect.DeepEqual(wantEvents, requests(pausedBus)) {
		t.Fatal("midcycle restored execution diverges from uninterrupted bus trace")
	}
}

// The instruction-family dispatcher must not allocate in steady-state execution.
func TestExtendedSteadyStateAllocations(t *testing.T) {
	b := &noAllocBus{}
	b.data[0x200] = 0xa5
	b.data[0x201] = 0x20
	b.data[0x20] = 0x42
	c := New(0x200)
	got := testing.AllocsPerRun(1000, func() {
		c.state.PC = 0x200
		c.state.Stage = 0
		c.state.Phase = PreBus
		if err := c.selectEntry(); err != nil {
			panic(err)
		}
		if err := c.StepInstruction(b); err != nil {
			panic(err)
		}
	})
	if got != 0 {
		t.Fatalf("extended LDA zp allocated %g times", got)
	}
}
func BenchmarkExtendedLDAZeroPage(b *testing.B) {
	memory := &noAllocBus{}
	memory.data[0x200] = 0xa5
	memory.data[0x201] = 0x20
	memory.data[0x20] = 0x42
	c := New(0x200)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c.state.PC = 0x200
		c.state.Stage = 0
		c.state.Phase = PreBus
		if err := c.selectEntry(); err != nil {
			b.Fatal(err)
		}
		if err := c.StepInstruction(memory); err != nil {
			b.Fatal(err)
		}
	}
}
