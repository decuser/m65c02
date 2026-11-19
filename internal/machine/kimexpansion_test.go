package machine

import (
	"testing"

	"m65c02/internal/cpu"
)

func TestKIMBASICRAMWithoutExternalFixtures(t *testing.T) {
	k := testKIM1(t)
	if err := k.EnableBASICRAM(); err != nil {
		t.Fatal(err)
	}
	for _, addr := range []uint16{0x2000, 0x4fff, 0x5000, 0x5fff} {
		k.Bus.Transfer(cpu.Request{Address: addr, DataOut: 0x5a, Write: true})
		if got := k.Bus.Transfer(cpu.Request{Address: addr}).DataIn; got != 0x5a {
			t.Fatalf("expanded RAM $%04X: got %02X", addr, got)
		}
	}
	k.Bus.Transfer(cpu.Request{Address: 0x0000, DataOut: 0x53, Write: true})
	k.Bus.Transfer(cpu.Request{Address: 0x6000, DataOut: 0xa7, Write: true})
	if got := k.Bus.Transfer(cpu.Request{Address: 0x6000}).DataIn; got != 0xff {
		t.Fatalf("unpopulated RAM $6000: got %02X", got)
	}
	if got := k.Bus.Transfer(cpu.Request{Address: 0x0000}).DataIn; got != 0x53 {
		t.Fatalf("$6000 aliased zero page: got %02X", got)
	}
}
