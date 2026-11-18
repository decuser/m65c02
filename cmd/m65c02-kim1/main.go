// Command m65c02-kim1 runs the original KIM-1 ROMs through the cycle-driven
// machine and exposes its physically simulated terminal lines.
package main

import (
	"flag"
	"fmt"
	"os"
	"unicode"

	"m65c02/internal/machine"
)

func main() {
	config := flag.String("config", "machines/kim1.toml", "KIM-1 motherboard configuration")
	rom002 := flag.String("rom002", "", "override 6530-002 ROM image")
	rom003 := flag.String("rom003", "", "override 6530-003 ROM image")
	trace := flag.Bool("trace", false, "print cycle-stamped raw serial edges")
	cycles := flag.Uint64("cycles", 1000000, "maximum machine cycles")
	bitCycles := flag.Uint64("bit-cycles", 9091, "terminal bit duration in CPU cycles (110 baud at 1 MHz)")
	initial := flag.String("input", "", "terminal characters, sent after RUBOUT; may be empty")
	flag.Parse()
	board, err := machine.LoadKIM1Config(*config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	if *rom002 == "" {
		*rom002 = board.ROM002
	}
	if *rom003 == "" {
		*rom003 = board.ROM003
	}
	if *bitCycles == 0 {
		fmt.Fprintln(os.Stderr, "bit-cycles must be positive")
		os.Exit(2)
	}
	k, err := machine.NewKIM1FromFiles(*rom002, *rom003)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	k.TTYSelect = board.TTYSelect
	s := k.ConnectSerial(*bitCycles)
	if err := k.StepInstruction(); err != nil {
		fmt.Fprintln(os.Stderr, "reset:", err)
		os.Exit(1)
	}
	k.ReleaseReset()
	if *trace {
		fmt.Fprintf(os.Stderr, "KIM-1 jumper A-21/A-V: %t (TTY select)\n", k.TTYSelect)
		fmt.Fprintln(os.Stderr, "CYCLE       SIGNAL LEVEL EVENT")
	}
	// The original KIM-1 asks for a RUBOUT to calibrate its software receiver.
	// Injection starts after the ROM's initial board-detection instructions.
	queued := false
	decodedCount := 0
	for k.Bus.Cycles < *cycles {
		if !queued && k.Bus.Cycles >= 2000 {
			s.Queue(append([]byte{0x7f}, []byte(*initial)...))
			queued = true
		}
		if err := k.Clock(); err != nil {
			fmt.Fprintln(os.Stderr, "CPU:", err)
			os.Exit(1)
		}
		for _, edge := range s.DrainEdges() {
			if *trace {
				level := 0
				if edge.High {
					level = 1
				}
				fmt.Fprintf(os.Stderr, "%012d %-6s %d     transition\n", edge.Cycle, edge.Signal, level)
			}
		}
		for _, b := range s.DrainDecoded() {
			decodedCount++
			if unicode.IsPrint(rune(b)) || b == '\r' || b == '\n' {
				fmt.Printf("%c", b)
			}
			if *trace {
				fmt.Fprintf(os.Stderr, "%012d SERIAL       decoded $%02X\n", s.Cycle, b)
			}
		}
	}
	fmt.Fprintf(os.Stderr, "KIM-1 stopped after %d cycles; PC=$%04X; decoded TX characters=%d\n", k.Bus.Cycles, k.CPU.View().PC, decodedCount)
}
