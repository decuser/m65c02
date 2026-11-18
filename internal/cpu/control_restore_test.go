package cpu

import (
	"reflect"
	"testing"
)

func TestInterruptSnapshotEveryMicrostep(t *testing.T) {
	for pos := 0; pos < 22; pos++ {
		base := &lineRAM{}
		base.Data[0x200] = 0xea
		base.Data[0xfffa], base.Data[0xfffb] = 0x00, 0x04
		base.nmi = true
		c := New(0x200)
		if err := c.StepInstruction(base); err != nil {
			t.Fatal(err)
		}
		if c.View().Control != 2 {
			t.Fatal("NMI not queued")
		}
		for i := 0; i < pos; i++ {
			if c.View().Control == 0 {
				return
			}
			if _, err := c.StepMicro(base); err != nil {
				t.Fatal(err)
			}
		}
		state := c.Snapshot()
		cloneBus := *base
		cloneBus.Events = append([]Request(nil), base.Events...)
		clone := New(0)
		if err := clone.Restore(state); err != nil {
			t.Fatalf("position %d: %v", pos, err)
		}
		for clone.View().Control != 0 {
			if _, err := clone.StepMicro(&cloneBus); err != nil {
				t.Fatal(err)
			}
		}
		for c.View().Control != 0 {
			if _, err := c.StepMicro(base); err != nil {
				t.Fatal(err)
			}
		}
		if !reflect.DeepEqual(clone.Snapshot(), c.Snapshot()) || !reflect.DeepEqual(cloneBus, *base) {
			t.Fatalf("restored interrupt diverged at microstep %d", pos)
		}
	}
}
