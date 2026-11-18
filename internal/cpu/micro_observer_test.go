package cpu

import "testing"

type observerBus struct{}

func (observerBus) Transfer(req Request) Response { return Response{DataIn: 0xea, Ready: true} }

func TestMicroObserverSeesExecutedPhases(t *testing.T) {
	c := New(0x200)
	var phases []Phase
	var events []MicroEvent
	c.ObserveMicro = func(before State, event MicroEvent, after State) {
		phases = append(phases, before.Phase)
		events = append(events, event)
	}
	if err := c.StepCycle(observerBus{}); err != nil {
		t.Fatal(err)
	}
	if len(events) < 2 || phases[0] != Resolve {
		t.Fatalf("missing observed microsteps: %v", phases)
	}
	seenBus := false
	for i, phase := range phases {
		if phase == Resolve && events[i].Resolve {
			seenBus = true
		}
	}
	if !seenBus {
		t.Fatalf("missing resolved bus phase: %v", phases)
	}
	c.ObserveMicro = nil
	n := len(events)
	if err := c.StepCycle(observerBus{}); err != nil {
		t.Fatal(err)
	}
	if len(events) != n {
		t.Fatal("disabled observer was invoked")
	}
}
