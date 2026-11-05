// m65c02-prod demonstrates the production sequencer with a finite program.
// It is deliberately not the eventual supervisor or visible computer.
package main

import (
	"flag"
	"fmt"
	"os"

	"m65c02/internal/cpu"
)

type observed struct {
	request cpu.Request
	data    byte
}
type machine struct {
	ram   [65536]byte
	trace []observed
}

func (m *machine) Transfer(r cpu.Request) cpu.Response {
	if r.Write {
		m.ram[r.Address] = r.DataOut
		m.trace = append(m.trace, observed{r, r.DataOut})
		return cpu.Response{Ready: true}
	}
	v := m.ram[r.Address]
	m.trace = append(m.trace, observed{r, v})
	return cpu.Response{DataIn: v, Ready: true}
}
func demo(cycleLimit int) error {
	m := &machine{}
	// INC $0300,X ; BNE -5 ; BRK. X starts at 0, RAM starts at F6.
	copy(m.ram[0x200:], []byte{0xfe, 0x00, 0x03, 0xd0, 0xfb, 0x00, 0x00})
	m.ram[0x300] = 0xf6
	m.ram[0xfffe] = 0
	m.ram[0xffff] = 4
	c := cpu.New(0x200)
	for m.ram[0x300] != 0 || c.View().Instructions == 0 {
		if int(c.View().Cycles) >= cycleLimit {
			return fmt.Errorf("cycle limit reached at %d", c.View().Cycles)
		}
		if err := c.StepCycle(m); err != nil {
			return err
		}
	}
	// Execute BRK once the last INC and untaken BNE have completed.
	for c.View().PC != 0x400 {
		if int(c.View().Cycles) >= cycleLimit {
			return fmt.Errorf("cycle limit reached at %d", c.View().Cycles)
		}
		if err := c.StepCycle(m); err != nil {
			return err
		}
	}
	for i, entry := range m.trace {
		dir := "READ"
		if entry.request.Write {
			dir = "WRITE"
		}
		sync := ""
		if entry.request.Sync {
			sync = " SYNC"
		}
		fmt.Printf("%3d  $%04X  %-5s $%02X%s\n", i+1, entry.request.Address, dir, entry.data, sync)
	}
	s := c.View()
	fmt.Printf("Counter $0300: $%02X (expected $00); PC=$%04X; cycles=%d; instructions=%d\n", m.ram[0x300], s.PC, s.Cycles, s.Instructions)
	if s.Cycles != 106 || s.Instructions != 21 || m.ram[0x300] != 0 {
		return fmt.Errorf("unexpected demo result")
	}
	return nil
}
func main() {
	limit := flag.Int("limit", 1000, "maximum physical cycles")
	flag.Parse()
	if err := demo(*limit); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
