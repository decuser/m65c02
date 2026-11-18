package cpu

import (
	"os"
	"testing"
)

// Requires the CMOS-configured Dormann interrupt program described in c14.
// Its $BFFC open-collector feedback register asserts IRQ (bit 0) and NMI
// (bit 1) when written high. The physical CPU inputs themselves are active-low.
func TestDormannInterrupt(t *testing.T) {
	path := os.Getenv("M65C02_DORMANN_INTERRUPT")
	if path == "" {
		t.Skip("configured Dormann interrupt fixture unavailable")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 65526 {
		t.Fatalf("unexpected image size: %d", len(b))
	}
	m := &interruptFeedbackBus{}
	copy(m.data[0x0a:], b)
	c := New(0x400)
	var prev uint16
	var repeat int
	for n := 0; n < 1000000; n++ {
		pc := c.View().PC
		if n > 0 && pc == prev {
			repeat++
		} else {
			repeat = 0
		}
		if repeat >= 3 {
			if pc == 0x0719 {
				t.Logf("passed after %d instructions", n)
				return
			}
			t.Fatalf("interrupt trap at $%04X after %d instructions, feedback=$%02X", pc, n, m.feedback)
		}
		prev = pc
		if err := c.StepInstruction(m); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("interrupt test timed out at $%04X", c.View().PC)
}

type interruptFeedbackBus struct {
	data     [65536]byte
	feedback byte
}

func (m *interruptFeedbackBus) Transfer(r Request) Response {
	var in byte
	if r.Address == 0xbffc {
		if r.Write {
			m.feedback = r.DataOut
		}
		in = m.feedback
	} else if r.Write {
		m.data[r.Address] = r.DataOut
	} else {
		in = m.data[r.Address]
	}
	return Response{DataIn: in, Ready: true, IRQ: m.feedback&1 != 0, NMI: m.feedback&2 != 0}
}
