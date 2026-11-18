package machine

import (
	"os"
	"testing"

	"m65c02/internal/cpu"
)

// This executes the monitor's actual SCANDS subroutine from a RAM JSR,
// then observes the first opcode fetch after its RTS. No host-side display
// emulation or ROM routine shortcut is involved.
func TestKIM1SCANDSTiming(t *testing.T) {
	p002, p003 := os.Getenv("M65C02_KIM_ROM002"), os.Getenv("M65C02_KIM_ROM003")
	if p002 == "" || p003 == "" {
		t.Skip("supply both original KIM-1 ROMs")
	}
	k, err := NewKIM1FromFiles(p002, p003)
	if err != nil {
		t.Fatal(err)
	}
	k.TTYSelect = false
	if err := k.StepInstruction(); err != nil {
		t.Fatal(err)
	}
	k.ReleaseReset()
	for k.Bus.Cycles < 100000 {
		if err := k.Clock(); err != nil {
			t.Fatal(err)
		}
	}
	// RAM harness: JSR SCANDS; NOP (return marker).
	for i, b := range []byte{0x20, 0x1f, 0x1f, 0xea} {
		k.RAM.Write(uint16(0x0200+i), b)
	}
	if err := k.CPU.Restore(cpu.New(0x0200).Snapshot()); err != nil {
		t.Fatal(err)
	}
	k.Bus.Trace = NewBusTrace(5000)
	k.Bus.Trace.PortSampler = func() BoardPins {
		_, ddra := k.RRIOT002.OutputA()
		_, ddrb := k.RRIOT002.OutputB()
		return BoardPins{R002: RRIOTPins{PA: k.RRIOT002.PinsA(), PB: k.RRIOT002.PinsB(), DDRA: ddra, DDRB: ddrb},
			Digit: k.Panel.Current.Digit, Segments: k.Panel.Current.SegmentBits}
	}
	var callStart, start, end uint64
	var initial, final [6][7]uint64
	for n := 0; n < 20000; n++ {
		if err := k.Clock(); err != nil {
			t.Fatal(err)
		}
		req := k.Bus.LastRequest
		if !req.Sync {
			continue
		}
		if req.Address == 0x0200 && callStart == 0 {
			callStart = k.Bus.Cycles
		}
		if req.Address == 0x1f1f && start == 0 {
			start = k.Bus.Cycles
			initial = k.Panel.Dwell
		}
		if req.Address == 0x0203 && start != 0 {
			end = k.Bus.Cycles
			final = k.Panel.Dwell
			break
		}
	}
	if callStart == 0 || start == 0 || end == 0 {
		t.Fatalf("SCANDS did not complete: call=%d start=%d end=%d PC=%04X", callStart, start, end, k.CPU.View().PC)
	}
	for digit := 0; digit < 6; digit++ {
		var active bool
		for seg := 0; seg < 7; seg++ {
			active = active || final[digit][seg] > initial[digit][seg]
		}
		if !active {
			t.Fatalf("SCANDS skipped display digit %d", digit)
		}
	}
	var scanOrder []int
	for _, event := range k.Bus.Trace.PortChanges("riot002") {
		digit := event.Pins.Digit
		if digit >= 0 && (len(scanOrder) == 0 || scanOrder[len(scanOrder)-1] != digit) {
			scanOrder = append(scanOrder, digit)
		}
	}
	if len(scanOrder) < 6 {
		t.Fatalf("too few selected digits in trace: %v", scanOrder)
	}
	// The initial digit may be the previous scan's latched selection.
	complete := false
	for i := 0; i+6 <= len(scanOrder); i++ {
		ordered := true
		for digit := 0; digit < 6; digit++ {
			if scanOrder[i+digit] != digit {
				ordered = false
				break
			}
		}
		if ordered {
			complete = true
			break
		}
	}
	if !complete {
		t.Fatalf("no complete left-to-right refresh: %v", scanOrder)
	}
	if got := end - start; got != 4231 {
		t.Errorf("SCANDS entry-to-return = %d cycles, want 4231", got)
	}
	if got := end - callStart; got != 4237 {
		t.Errorf("JSR SCANDS through return = %d cycles, want 4237", got)
	}
	t.Logf("SCANDS entry-to-return: %d cycles; including JSR: %d cycles (Hans reports 4237); digits: %v", end-start, end-callStart, scanOrder)
}
