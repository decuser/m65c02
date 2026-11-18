package machine

import "m65c02/internal/cpu"

// MicroStep captures an actual CPU transition; Cycle is the bus-cycle number
// that the pre/resolve/post steps belong to, not a separate time base.
type MicroStep struct {
	Cycle     uint64    `json:"cycle"`
	Phase     string    `json:"phase"`
	Operation string    `json:"operation"`
	Before    cpu.State `json:"before"`
	After     cpu.State `json:"after"`
}

type MicroTrace struct {
	rows  []MicroStep
	next  int
	count int
}

func NewMicroTrace(capacity int) *MicroTrace {
	if capacity < 1 {
		capacity = 1
	}
	return &MicroTrace{rows: make([]MicroStep, capacity)}
}

func (t *MicroTrace) Record(before cpu.State, event cpu.MicroEvent, after cpu.State) {
	phase := "PRE"
	if event.Resolve {
		phase = "BUS"
	} else if before.Phase == cpu.PostBus {
		phase = "POST"
	}
	op := cpu.MicroName(event.Operation)
	if event.Resolve {
		op = "RESOLVE"
	} else if before.Phase == cpu.PostBus && event.Operation == 0 {
		op = "RETIRE"
	}
	t.rows[t.next] = MicroStep{Cycle: before.Cycles + 1, Phase: phase, Operation: op, Before: before, After: after}
	t.next = (t.next + 1) % len(t.rows)
	if t.count < len(t.rows) {
		t.count++
	}
}

// Window returns captured microsteps in chronological order for an inclusive
// range of bus cycles. It never mutates the running machine.
func (t *MicroTrace) Window(first, last uint64) []MicroStep {
	out := make([]MicroStep, 0)
	if t == nil || last < first {
		return out
	}
	for i := 0; i < t.count; i++ {
		step := t.rows[(t.next-t.count+i+len(t.rows))%len(t.rows)]
		if step.Cycle >= first && step.Cycle <= last {
			out = append(out, step)
		}
	}
	return out
}
