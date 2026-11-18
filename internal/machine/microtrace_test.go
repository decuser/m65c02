package machine

import (
	"m65c02/internal/cpu"
	"testing"
)

func TestMicroTraceCycleAlignmentAndHistory(t *testing.T) {
	trace := NewMicroTrace(4)
	before := cpu.State{Cycles: 5, Phase: cpu.PreBus, PC: 0x200}
	after := before
	after.AD = 0x200
	trace.Record(before, cpu.MicroEvent{Operation: 1}, after)
	before = after
	before.Phase = cpu.Resolve
	trace.Record(before, cpu.MicroEvent{Resolve: true}, after)
	steps := trace.Window(6, 6)
	if len(steps) != 2 || steps[0].Cycle != 6 || steps[0].Phase != "PRE" || steps[1].Phase != "BUS" {
		t.Fatalf("misaligned microsteps: %+v", steps)
	}
	if len(trace.Window(7, 7)) != 0 {
		t.Fatal("events leaked into next cycle")
	}
	for i := 0; i < 5; i++ {
		before.Cycles = uint64(6 + i)
		trace.Record(before, cpu.MicroEvent{Resolve: true}, after)
	}
	steps = trace.Window(0, 20)
	if len(steps) != 4 || steps[0].Cycle != 8 || steps[3].Cycle != 11 {
		t.Fatalf("ring retention: %+v", steps)
	}
}
