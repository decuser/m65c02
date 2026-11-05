package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestInteractiveCounter(t *testing.T) {
	var out bytes.Buffer
	script := "trace on\ncycle 7\nmem 0300 1\nrun\nmem 0300 1\nquit\n"
	if err := interact(strings.NewReader(script), &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Prepared: $0200 READ SYNC=true", "$0300  WRITE $F7", "Counter at $0300: $00", "$0300: 00"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in output: %s", want, out.String())
		}
	}
}
func TestMonitorInspectionDoesNotAdvance(t *testing.T) {
	m := &monitor{}
	m.demo()
	before := m.cpu.Snapshot()
	var output bytes.Buffer
	m.command(&output, "regs")
	m.command(&output, "mem 0200 16")
	if len(m.observed) != 0 || before.Cycles != m.cpu.State.Cycles || before.PC != m.cpu.State.PC {
		t.Fatal("inspection advanced the emulator")
	}
}

func TestFigureEight(t *testing.T) {
	m := &monitor{}
	m.figureEight()
	var out bytes.Buffer
	if err := m.cycle(&out, 6); err != nil {
		t.Fatal(err)
	}
	if m.cpu.State.PC != 0x0200 || m.cpu.State.Instructions != 2 {
		t.Fatalf("figure eight did not return: %+v", m.cpu.State)
	}
	if len(m.observed) != 6 || !m.observed[0].req.Sync || !m.observed[3].req.Sync {
		t.Fatal("expected two JMP opcode fetches")
	}
}
