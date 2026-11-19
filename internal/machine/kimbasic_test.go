package machine

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"m65c02/internal/cpu"
)

// Exercise the original monitor's autobaud and GO command. KB9 must call
// the monitor's pin-driven serial routines and print its first prompt.
func TestKIM1KB9SerialBoot(t *testing.T) {
	for _, program := range []struct {
		name  string
		file  string
		entry uint16
	}{
		{"v1.1", "kb9-v1.1.ptp", 0x4065},
		{"v1.2", "kb9-v1.2.ptp", 0x3fb1},
		{"v2", "kb9-v2.ptp", 0x2000},
	} {
		for _, bitCycles := range []uint64{9091, 833} {
			t.Run(fmt.Sprintf("%s_bitCycles_%d", program.name, bitCycles), func(t *testing.T) {
				testKB9SerialBoot(t, bitCycles, program.file, program.entry)
			})
		}
	}
}

func testKB9SerialBoot(t *testing.T, bitCycles uint64, file string, entry uint16) {
	rom002, rom003 := os.Getenv("M65C02_KIM_ROM002"), os.Getenv("M65C02_KIM_ROM003")
	if rom002 == "" || rom003 == "" {
		t.Skip("provide original 6530 ROMs")
	}
	k, err := NewKIM1FromFiles(rom002, rom003)
	if err != nil {
		t.Fatal(err)
	}
	k.TTYSelect = true
	s := k.ConnectSerial(bitCycles)
	if err = k.EnableBASICRAM(); err != nil {
		t.Fatal(err)
	}
	if _, err = k.LoadKIMTapeFile(filepath.Join(kimProgramsDir(t), file)); err != nil {
		t.Fatal(err)
	}
	k.RAM.Write(0x00f1, 0)
	k.RAM.Write(0x00f2, 0xff)
	if err = k.StepInstruction(); err != nil {
		t.Fatal(err)
	}
	k.ReleaseReset()
	var output []byte
	pending := []byte(fmt.Sprintf("%04X ", entry))
	entryReply := []byte(fmt.Sprintf("%04X %02X ", entry, k.Bus.Transfer(cpu.Request{Address: entry}).DataIn))
	goSent := false
	var goAt uint64
	monitorReady := false
	var monitorReadyAt uint64
	var nextInputAt uint64
	var memoryResponseAt uint64
	memoryResponded := false
	for k.Bus.Cycles < 35000000 {
		if k.Bus.Cycles == 2000 {
			s.Queue([]byte{0x7f})
		}
		if goAt != 0 && !goSent && k.Bus.Cycles >= goAt {
			pending = append(pending, 'G')
			goSent = true
		}
		if monitorReady && k.Bus.Cycles >= monitorReadyAt && len(pending) != 0 && s.RXIdle() && k.Bus.Cycles >= nextInputAt {
			s.Queue(pending[:1])
			pending = pending[1:]
			nextInputAt = k.Bus.Cycles + 22*bitCycles
		}
		if err = k.Clock(); err != nil {
			t.Fatal(err)
		}
		s.DrainEdges()
		output = append(output, s.DrainDecoded()...)
		if !monitorReady && bytes.Contains(output, []byte("0000 00 ")) {
			monitorReady = true
			monitorReadyAt = k.Bus.Cycles + 120*bitCycles
		}
		if goAt == 0 && bytes.Contains(output, entryReply) {
			goAt = k.Bus.Cycles + 22*bitCycles
		}
		if !memoryResponded && memoryResponseAt == 0 && bytes.Contains(output, []byte("MEMORY SIZE? ")) {
			memoryResponseAt = k.Bus.Cycles + 100000
		}
		if memoryResponseAt != 0 && k.Bus.Cycles >= memoryResponseAt {
			pending = append(pending, '\r')
			memoryResponseAt = 0
			memoryResponded = true
		}
		if bytes.Contains(output, []byte("TERMINAL WIDTH?")) {
			return
		}
	}
	t.Fatalf("KB9 did not reach TERMINAL WIDTH?; GO sent=%v, output=%q, PC=$%04X", goSent, output, k.CPU.View().PC)
}
