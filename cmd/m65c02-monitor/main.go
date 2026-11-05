// Command m65c02-monitor provides a disposable, inspectable spike monitor.
// It is not the production supervisor or a general-purpose 65C02 emulator.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"m65c02/internal/spike"
)

type observation struct {
	req   spike.Request
	reply spike.Result
}

type monitor struct {
	cpu      *spike.CPU
	mem      *spike.RAM
	trace    bool
	goal     uint64
	observed []observation
}

func (m *monitor) demo() {
	m.cpu = spike.New(0x0200)
	m.mem = &spike.RAM{}
	// Start at $F6. Ten INC operations wrap the byte to $00;
	// BNE takes the back edge nine times, then falls through to BRK.
	m.mem.Bytes[0x0300] = 0xf6
	m.mem.Bytes[0x0200] = 0xfe // INC $0300,X
	m.mem.Bytes[0x0201] = 0x00
	m.mem.Bytes[0x0202] = 0x03
	m.mem.Bytes[0x0203] = 0xd0 // BNE $0200 (-5)
	m.mem.Bytes[0x0204] = 0xfb
	m.mem.Bytes[0x0205] = 0x00 // BRK
	m.mem.Bytes[0x0206] = 0x00
	m.mem.Bytes[0xfffe] = 0x00
	m.mem.Bytes[0xffff] = 0x04
	m.goal = 21
	m.observed = nil
}

// figureEight is an intentionally endless pair of jumps for cycle exploration.
// It has no run-to-completion goal; use cycle N or step instead.
func (m *monitor) figureEight() {
	m.cpu = spike.New(0x0200)
	m.mem = &spike.RAM{}
	m.mem.Bytes[0x0200] = 0x4c // JMP $0210
	m.mem.Bytes[0x0201] = 0x10
	m.mem.Bytes[0x0202] = 0x02
	m.mem.Bytes[0x0210] = 0x4c // JMP $0200
	m.mem.Bytes[0x0211] = 0x00
	m.mem.Bytes[0x0212] = 0x02
	m.goal = 0
	m.observed = nil
}
func (m *monitor) status(w io.Writer) {
	s := m.cpu.Snapshot()
	fmt.Fprintf(w, "PC=$%04X  A=%02X X=%02X Y=%02X  SP=%02X P=%02X  cycles=%d instructions=%d\n", s.PC, s.A, s.X, s.Y, s.S, s.P, s.Cycles, s.Instructions)
	if s.PendingValid {
		fmt.Fprintf(w, "Prepared: $%04X %s SYNC=%v (not yet resolved)\n", s.Pending.Addr, kind(s.Pending), s.Pending.Sync)
	}
}
func kind(r spike.Request) string {
	if r.Write {
		return "WRITE"
	}
	return "READ"
}

// Transfer is the trace tap. It records the actual Resolve response, rather
// than rereading memory during inspection (which could yield a later value).
func (m *monitor) Transfer(r spike.Request) spike.Result {
	reply := m.mem.Transfer(r)
	m.observed = append(m.observed, observation{r, reply})
	return reply
}
func (m *monitor) displayEvents(w io.Writer, from int) {
	for i := from; i < len(m.observed); i++ {
		e := m.observed[i]
		description := "Data read"
		if e.req.Write {
			description = "Data write"
		} else if e.req.Sync {
			description = "Instruction fetch"
		}
		data := e.reply.Data
		if e.req.Write {
			data = e.req.Data
		}
		fmt.Fprintf(w, "%5d  $%04X  %-5s $%02X  %s\n", i+1, e.req.Addr, kind(e.req), data, description)
	}
}
func (m *monitor) cycle(w io.Writer, n int) error {
	if n < 1 || n > 10000 {
		return fmt.Errorf("cycle count must be 1..10000")
	}
	from := len(m.observed)
	for i := 0; i < n; i++ {
		if err := m.cpu.StepCycle(m); err != nil {
			return err
		}
	}
	fmt.Fprintf(w, "Executed %d physical cycles.\n", n)
	if m.trace {
		m.displayEvents(w, from)
	}
	m.status(w)
	return nil
}
func (m *monitor) step(w io.Writer) error {
	from := len(m.observed)
	before := m.cpu.Snapshot()
	if err := m.cpu.StepInstruction(m); err != nil {
		return err
	}
	fmt.Fprintf(w, "Instruction at $%04X: %d cycles, A %02X -> %02X, PC $%04X -> $%04X\n", before.PC, m.cpu.State.Cycles-before.Cycles, before.A, m.cpu.State.A, before.PC, m.cpu.State.PC)
	if m.trace {
		m.displayEvents(w, from)
	}
	m.status(w)
	return nil
}
func (m *monitor) run(w io.Writer) error {
	if m.goal == 0 {
		return fmt.Errorf("no demo loaded")
	}
	for m.cpu.State.Instructions < m.goal {
		if err := m.cpu.StepInstruction(m); err != nil {
			return err
		}
	}
	fmt.Fprintf(w, "Demo completed. Counter at $0300: $%02X (expected $00 after 10 increments from $F6)\n", m.mem.Bytes[0x0300])
	m.status(w)
	return nil
}
func (m *monitor) command(w io.Writer, line string) bool {
	args := strings.Fields(strings.ToLower(strings.TrimSpace(line)))
	if len(args) == 0 {
		return true
	}
	switch args[0] {
	case "quit", "exit", "q":
		return false
	case "help", "?":
		fmt.Fprintln(w, "Commands: demo counter|figure8 | cycle N | step | run | regs | mem HEXADDR [N] | trace on/off | micro | help | quit")
	case "demo":
		if len(args) != 2 || (args[1] != "counter" && args[1] != "figure8") {
			fmt.Fprintln(w, "usage: demo counter|figure8")
			break
		}
		if args[1] == "figure8" {
			m.figureEight()
			fmt.Fprintln(w, "Loaded Figure Eight: JMP $0210 at $0200 and JMP $0200 at $0210. Use step or cycle N.")
		} else {
			m.demo()
			fmt.Fprintln(w, "Loaded INC $0300,X / BNE loop followed by BRK at $0200. Counter starts at $F6 and wraps to $00 after 10 increments.")
		}
		m.status(w)
	case "cycle":
		n := 1
		if len(args) > 2 {
			fmt.Fprintln(w, "usage: cycle [N]")
			break
		}
		if len(args) == 2 {
			v, e := strconv.Atoi(args[1])
			if e != nil {
				fmt.Fprintln(w, "invalid count")
				break
			}
			n = v
		}
		if err := m.cycle(w, n); err != nil {
			fmt.Fprintln(w, "error:", err)
		}
	case "step":
		if err := m.step(w); err != nil {
			fmt.Fprintln(w, "error:", err)
		}
	case "run":
		if err := m.run(w); err != nil {
			fmt.Fprintln(w, "error:", err)
		}
	case "regs", "status":
		m.status(w)
	case "micro":
		from := len(m.observed)
		label, err := m.cpu.StepMicro(m)
		if err != nil {
			fmt.Fprintln(w, "error:", err)
			break
		}
		fmt.Fprintln(w, "Microstep:", label)
		if m.trace {
			m.displayEvents(w, from)
		}
		m.status(w)
	case "mem":
		if len(args) < 2 || len(args) > 3 {
			fmt.Fprintln(w, "usage: mem HEXADDR [N]")
			break
		}
		addr, err := strconv.ParseUint(strings.TrimPrefix(args[1], "$"), 16, 16)
		if err != nil {
			fmt.Fprintln(w, "invalid hex address")
			break
		}
		n := 16
		if len(args) == 3 {
			n, err = strconv.Atoi(args[2])
			if err != nil || n < 1 || n > 256 {
				fmt.Fprintln(w, "length must be 1..256")
				break
			}
		}
		for i := 0; i < n; i++ {
			if i%16 == 0 {
				fmt.Fprintf(w, "\n$%04X: ", uint16(addr+uint64(i)))
			}
			fmt.Fprintf(w, "%02X ", m.mem.Bytes[uint16(addr+uint64(i))])
		}
		fmt.Fprintln(w)
	case "trace":
		if len(args) != 2 || (args[1] != "on" && args[1] != "off") {
			fmt.Fprintln(w, "usage: trace on|off")
			break
		}
		m.trace = args[1] == "on"
		fmt.Fprintf(w, "Trace %s.\n", args[1])
	default:
		fmt.Fprintln(w, "unknown command; type help")
	}
	return true
}
func interact(r io.Reader, w io.Writer) error {
	m := &monitor{}
	m.demo()
	fmt.Fprintln(w, "m65c02 disposable spike monitor (not a full CPU)")
	fmt.Fprintln(w, "Demo counter loaded. Type help. Use cycle 1, cycle 10, step, run, mem 0300.")
	m.status(w)
	scanner := bufio.NewScanner(r)
	for {
		fmt.Fprint(w, "> ")
		if !scanner.Scan() {
			return scanner.Err()
		}
		if !m.command(w, scanner.Text()) {
			return nil
		}
	}
}
func main() {
	if err := interact(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
