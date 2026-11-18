package machine

import (
	"os"
	"testing"

	"m65c02/internal/cpu"
)

func testKIM1(t *testing.T) *KIM1 {
	t.Helper()
	a, b := make([]byte, 1024), make([]byte, 1024)
	a[0x3fc], a[0x3fd] = 0x22, 0x1c
	a[0x22] = 0xea // NOP at reset vector
	b[0] = 0xa9
	k, err := NewKIM1(a, b)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestKIM1DecodeAndMirrors(t *testing.T) {
	k := testKIM1(t)
	read := func(address uint16) byte { return k.Bus.Transfer(cpu.Request{Address: address}).DataIn }
	write := func(address uint16, data byte) {
		k.Bus.Transfer(cpu.Request{Address: address, Write: true, DataOut: data})
	}
	if read(0x1800) != 0xa9 || read(0x3800) != 0xa9 || read(0xf800) != 0xa9 {
		t.Fatal("6530-003 ROM decode")
	}
	if read(0xfffc) != 0x22 || read(0xfffd) != 0x1c {
		t.Fatal("reset vector mirror")
	}
	write(0x0003, 0x56)
	if read(0x2003) != 0x56 {
		t.Fatal("discrete RAM mirror")
	}
	write(0x17c2, 0x66)
	write(0x1782, 0x77)
	if read(0x37c2) != 0x66 || read(0x3782) != 0x77 {
		t.Fatal("6530 RAM decode")
	}
	write(0x1741, 0xff)
	write(0x1740, 0x12)
	if read(0x1740) != 0x12 || read(0x1700) != 0xff {
		t.Fatal("6530 ports shorted")
	}
	write(0x1c22, 0xff)
	if read(0x1c22) != 0xea {
		t.Fatal("mask ROM is writable")
	}
	if read(0x1600) != 0xff {
		t.Fatal("undecoded area did not float high")
	}
}

func TestKIM1ResetFetch(t *testing.T) {
	k := testKIM1(t)
	if err := k.StepInstruction(); err != nil {
		t.Fatal(err)
	}
	if k.CPU.View().PC != 0x1c22 {
		t.Fatalf("reset vector: %04x", k.CPU.View().PC)
	}
	k.ReleaseReset()
	if err := k.StepInstruction(); err != nil {
		t.Fatal(err)
	}
	if k.CPU.View().PC != 0x1c23 {
		t.Fatalf("ROM NOP not executed: %04x", k.CPU.View().PC)
	}
}

func TestKIM1RealMonitorROM(t *testing.T) {
	p002, p003 := os.Getenv("M65C02_KIM_ROM002"), os.Getenv("M65C02_KIM_ROM003")
	if p002 == "" || p003 == "" {
		t.Skip("supply M65C02_KIM_ROM002 and M65C02_KIM_ROM003")
	}
	rom002, err := os.ReadFile(p002)
	if err != nil {
		t.Fatal(err)
	}
	rom003, err := os.ReadFile(p003)
	if err != nil {
		t.Fatal(err)
	}
	k, err := NewKIM1FromFiles(p002, p003)
	if err != nil {
		t.Fatal(err)
	}
	for _, rom := range []struct {
		name  string
		base  uint16
		image []byte
	}{
		{"6530-003", 0x1800, rom003},
		{"6530-002", 0x1c00, rom002},
	} {
		for offset, want := range rom.image {
			address := rom.base + uint16(offset)
			for _, mirror := range []uint16{0, 0x2000, 0xe000} {
				mapped := address + mirror
				if got := k.Bus.Transfer(cpu.Request{Address: mapped}).DataIn; got != want {
					t.Fatalf("%s ROM %04x: got %02x, want %02x", rom.name, mapped, got, want)
				}
			}
		}
	}
	t.Logf("verified all %d ROM bytes at base, +$2000, and high mirrors", len(rom002)+len(rom003))
	if err := k.StepInstruction(); err != nil {
		t.Fatal(err)
	}
	if got := k.CPU.View().PC; got != 0x1c22 {
		t.Fatalf("original ROM reset vector: %04x", got)
	}
	k.ReleaseReset()
	for i := 0; i < 20; i++ {
		if err := k.StepInstruction(); err != nil {
			t.Fatalf("monitor instruction %d: %v", i, err)
		}
	}
	if k.Bus.Cycles < 20 {
		t.Fatalf("monitor advanced only %d cycles", k.Bus.Cycles)
	}
}
