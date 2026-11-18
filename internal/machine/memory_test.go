package machine

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"m65c02/internal/cpu"
)

func TestRAMAndROMMapping(t *testing.T) {
	ram, err := NewRAM(0x1000)
	if err != nil {
		t.Fatal(err)
	}
	romImage := make([]byte, 0x1000)
	copy(romImage, []byte{0xa9, 0xab, 0x8d, 0x00, 0x02, 0xea})
	rom, err := NewROM(romImage)
	if err != nil {
		t.Fatal(err)
	}
	romImage[0] = 0x00 // demonstrate ownership of the image
	b := NewBackplane()
	if err := b.Map(0, 0x0fff, ram); err != nil {
		t.Fatal(err)
	}
	if err := b.Map(0xf000, 0xffff, rom); err != nil {
		t.Fatal(err)
	}
	if err := b.Map(0x2000, 0x3000, ram); err == nil {
		t.Fatal("accepted undersized RAM mapping")
	}
	if err := b.Map(0x4000, 0x5000, rom); err == nil {
		t.Fatal("accepted undersized ROM mapping")
	}
	m, err := New(cpu.New(0xf000), b)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.StepInstruction(); err != nil {
		t.Fatal(err)
	}
	if err := m.StepInstruction(); err != nil {
		t.Fatal(err)
	}
	if ram.Read(0x200) != 0xab || m.CPU.View().PC != 0xf005 || b.Cycles != 6 {
		t.Fatalf("ROM program failed: PC=%04X RAM=%02X cycles=%d", m.CPU.View().PC, ram.Read(0x200), b.Cycles)
	}
	b.Transfer(cpu.Request{Address: 0xf000, Write: true, DataOut: 0x00})
	if got := b.Transfer(cpu.Request{Address: 0xf000}).DataIn; got != 0xa9 {
		t.Fatalf("ROM modified by write: %02x", got)
	}
	if rom.Image()[0] != 0xa9 {
		t.Fatal("ROM source was mutated")
	}
}

func TestRAMLoadSaveAndSnapshot(t *testing.T) {
	ram, err := NewRAM(256)
	if err != nil {
		t.Fatal(err)
	}
	if err := ram.Load(0xfe, []byte{0x12, 0x34}); err != nil {
		t.Fatal(err)
	}
	if err := ram.Load(0xff, []byte{1, 2}); err == nil {
		t.Fatal("accepted overflowing image")
	}
	before := ram.Snapshot()
	ram.Write(0xfe, 0x99)
	if before[0xfe] != 0x12 {
		t.Fatal("snapshot aliases RAM")
	}
	if err := ram.Restore(before); err != nil {
		t.Fatal(err)
	}
	if err := ram.Restore(before[:10]); err == nil {
		t.Fatal("accepted wrong snapshot size")
	}
	if ram.Read(0xfe) != 0x12 {
		t.Fatal("restore failed")
	}
	path := filepath.Join(t.TempDir(), "ram.bin")
	if err := ram.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	readback, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(readback, before) {
		t.Fatal("saved RAM differs")
	}
	second, err := NewRAM(256)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.LoadFile(0, path); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(second.Snapshot(), before) {
		t.Fatal("RAM file load differs")
	}
}

func TestROMFileAndConstructorLimits(t *testing.T) {
	if _, err := NewRAM(0); err == nil {
		t.Fatal("accepted empty RAM")
	}
	if _, err := NewRAM(65537); err == nil {
		t.Fatal("accepted oversized RAM")
	}
	if _, err := NewROM(nil); err == nil {
		t.Fatal("accepted empty ROM")
	}
	path := filepath.Join(t.TempDir(), "firmware.bin")
	if err := os.WriteFile(path, []byte{0x4c, 0, 0xf0}, 0600); err != nil {
		t.Fatal(err)
	}
	rom, err := ROMFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if rom.Size() != 3 || rom.Read(0) != 0x4c {
		t.Fatal("ROM file load failed")
	}
	copyImage := rom.Image()
	copyImage[0] = 0xff
	if rom.Read(0) != 0x4c {
		t.Fatal("ROM image exposed mutable storage")
	}
	if _, err := ROMFromFile(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing ROM accepted")
	}
}
