package machine

import (
	"testing"

	"m65c02/internal/cpu"
)

func TestKIMSingleStepSyncAndROMInhibit(t *testing.T) {
	k, err := NewKIM1(make([]byte, 1024), make([]byte, 1024))
	if err != nil {
		t.Fatal(err)
	}
	k.SST = true
	for _, tc := range []struct {
		address    uint16
		sync, want bool
	}{
		{0x0200, true, true},
		{0x0201, false, false},
		{0x1800, true, false},
		{0x1c00, true, false},
		{0x3800, true, false},
		{0xfc00, true, false},
		{0x0202, true, true},
	} {
		r := k.Bus.Transfer(cpu.Request{Address: tc.address, Sync: tc.sync})
		if r.NMI != tc.want {
			t.Fatalf("SST NMI at %04X sync=%v: got %v want %v", tc.address, tc.sync, r.NMI, tc.want)
		}
	}
	k.SST = false
	if k.Bus.Transfer(cpu.Request{Address: 0x0200, Sync: true}).NMI {
		t.Fatal("SST off asserted NMI")
	}
	k.PressStop()
	if !k.Bus.Transfer(cpu.Request{Address: 0x1c00, Sync: true}).NMI {
		t.Fatal("ST must work independently of SST")
	}
}

func TestKIMSingleStepInterruptsAfterUserInstruction(t *testing.T) {
	rom002, rom003 := make([]byte, 1024), make([]byte, 1024)
	// Reset vector -> $0200; NMI vector -> $1C00.
	rom002[0x3fc], rom002[0x3fd] = 0x00, 0x02
	rom002[0x3fa], rom002[0x3fb] = 0x00, 0x1c
	// The ROM handler loops without producing another SST interrupt.
	rom002[0], rom002[1], rom002[2] = 0x4c, 0x00, 0x1c
	k, err := NewKIM1(rom002, rom003)
	if err != nil {
		t.Fatal(err)
	}
	k.RAM.Write(0x0200, 0xa9) // LDA #$42
	k.RAM.Write(0x0201, 0x42)
	k.RAM.Write(0x0202, 0xea) // NOP: must not execute before NMI
	k.SST = true
	if err := k.Clock(); err != nil {
		t.Fatal(err)
	}
	k.ReleaseReset()
	for i := 0; i < 60; i++ {
		if err := k.Clock(); err != nil {
			t.Fatal(err)
		}
	}
	if got := k.CPU.View().A; got != 0x42 {
		t.Fatalf("instruction did not execute before NMI: A=%02X", got)
	}
	if pc := k.CPU.View().PC; pc < 0x1c00 || pc > 0x1c03 {
		t.Fatalf("SST did not branch to ROM NMI handler: PC=%04X", pc)
	}
}
