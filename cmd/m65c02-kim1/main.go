// Command m65c02-kim1 connects a host terminal to original KIM-1 ROM code.
// Serial traffic travels through 6530 pins, not monitor subroutine hooks.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode"

	"m65c02/internal/machine"
)

// enableRawTerminal uses stty for interactive stdin, leaving pipes alone.
func enableRawTerminal() (func(), error) {
	get := exec.Command("stty", "-g")
	get.Stdin = os.Stdin
	saved, err := get.Output()
	if err != nil {
		return func() {}, nil
	}
	set := exec.Command("stty", "raw", "-echo")
	set.Stdin = os.Stdin
	if err := set.Run(); err != nil {
		return nil, err
	}
	return func() {
		restore := exec.Command("stty", strings.TrimSpace(string(saved)))
		restore.Stdin = os.Stdin
		_ = restore.Run()
	}, nil
}

func readKeys(out chan<- byte, stop chan<- struct{}) {
	r := bufio.NewReader(os.Stdin)
	for {
		b, err := r.ReadByte()
		if err != nil {
			if err != io.EOF {
				fmt.Fprintln(os.Stderr, "terminal input:", err)
			}
			close(stop)
			return
		}
		if b == 3 || b == 4 {
			close(stop)
			return
		} // Ctrl-C / Ctrl-D in raw mode
		if b == '\n' {
			b = '\r'
		}
		localEcho(os.Stdout, b)
		out <- b
	}
}

// Full-duplex teleprinters print their own keys. The monitor prints replies.
func localEcho(w io.Writer, b byte) {
	switch b {
	case '\r':
		fmt.Fprint(w, "\r\n")
	case 8, 127:
		fmt.Fprint(w, "\b \b")
	default:
		if b >= 32 && b < 127 {
			fmt.Fprintf(w, "%c", b)
		}
	}
}

// bootstrapKey accepts traditional RUBOUT, Return (PAL-1 terminal habit),
// or a modern terminal's ESC [ 3 ~ Delete sequence. It is active only
// before the original KIM-1 ROM has completed its first baud measurement.
// All recognized keys become the same physical $7F frame on PA7.
type bootstrapKey struct{ escape uint8 }

func (k *bootstrapKey) accept(b byte) bool {
	if k.escape != 0 {
		want := []byte{'[', '3', '~'}
		if b == want[k.escape-1] {
			k.escape++
			if k.escape == 4 {
				k.escape = 0
				return true
			}
			return false
		}
		k.escape = 0
	}
	if b == 0x1b {
		k.escape = 1
		return false
	}
	return b == 0x7f || b == '\r' || b == '\n'
}

// pollPTY watches the master until a client opens the slave, then delivers
// incoming bytes. EIO means the slave is not open; EAGAIN means it is open
// but has no data. Neither condition pauses the emulated CPU.
func pollPTY(port *machine.PTY, input chan<- byte, connected chan<- struct{}, stop <-chan struct{}) {
	buf := make([]byte, 128)
	fd := int(port.Master.Fd())
	_ = syscall.SetNonblock(fd, true)
	announced := false
	for {
		select {
		case <-stop:
			return
		default:
		}
		n, err := syscall.Read(fd, buf)
		if n > 0 || err == syscall.EAGAIN {
			if !announced {
				select {
				case connected <- struct{}{}:
				default:
				}
				announced = true
			}
		}
		if n > 0 {
			for _, b := range buf[:n] {
				select {
				case input <- b:
				case <-stop:
					return
				}
			}
		}
		if err == syscall.EIO {
			announced = false
		}
		if n == 0 || err != nil {
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func startTraceWindow() (io.Writer, func(), error) {
	f, err := os.CreateTemp("", "m65c02-serial-*.log")
	if err != nil {
		return nil, nil, err
	}
	args := []string{"-e", "tail", "--pid=" + fmt.Sprint(os.Getpid()), "-n", "+1", "-f", f.Name()}
	cmd := exec.Command("x-terminal-emulator", args...)
	if err = cmd.Start(); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, nil, err
	}
	go func() { _ = cmd.Wait() }()
	return f, func() { f.Close(); os.Remove(f.Name()) }, nil
}

func run() error {
	config := flag.String("config", "machines/kim1.toml", "motherboard configuration")
	rom002 := flag.String("rom002", "", "override 6530-002 ROM")
	rom003 := flag.String("rom003", "", "override 6530-003 ROM")
	trace := flag.Bool("trace", false, "cycle-stamped serial transitions")
	traceWindow := flag.Bool("trace-window", false, "open only a separate serial-trace terminal")
	usePTY := flag.Bool("pty", false, "expose the serial port as a Linux /dev/pts device")
	live := flag.Bool("live", false, "read host keyboard until Ctrl-C or Ctrl-D")
	cycles := flag.Uint64("cycles", 0, "maximum cycles; zero means unbounded with -live")
	bitCycles := flag.Uint64("bit-cycles", 9091, "CPU cycles per serial bit")
	initial := flag.String("input", "", "command text to send after monitor greeting")
	flag.Parse()
	board, err := machine.LoadKIM1Config(*config)
	if err != nil {
		return err
	}
	if *rom002 == "" {
		*rom002 = board.ROM002
	}
	if *rom003 == "" {
		*rom003 = board.ROM003
	}
	if *bitCycles == 0 {
		return fmt.Errorf("bit-cycles must be positive")
	}
	if !*live && !*usePTY && *cycles == 0 {
		return fmt.Errorf("specify -live or a finite -cycles limit")
	}
	k, err := machine.NewKIM1FromFiles(*rom002, *rom003)
	if err != nil {
		return err
	}
	k.TTYSelect = board.TTYSelect
	s := k.ConnectSerial(*bitCycles)
	if err = k.StepInstruction(); err != nil {
		return err
	}
	k.ReleaseReset()
	traceOut := io.Writer(os.Stderr)
	if *traceWindow {
		var cleanup func()
		traceOut, cleanup, err = startTraceWindow()
		if err != nil {
			return fmt.Errorf("trace window: %w", err)
		}
		defer cleanup()
		*trace = true
	}
	if *trace {
		fmt.Fprintf(traceOut, "KIM-1 TTY jumper: %t\nCYCLE       SIGNAL LEVEL EVENT\n", k.TTYSelect)
	}
	input := make(chan byte, 4096)
	stopKeys := make(chan struct{})
	var port *machine.PTY
	connected := make(chan struct{}, 1)
	stopPTY := make(chan struct{})
	if *usePTY {
		port, err = machine.NewPTY()
		if err != nil {
			return err
		}
		defer port.Close()
		defer close(stopPTY)
		go pollPTY(port, input, connected, stopPTY)
		fmt.Fprintf(os.Stderr, "KIM-1 serial device: %s\nConnect in another terminal:\n  minicom -D %s -b 1200 -8\nEnable local echo with Ctrl-A, E; press Return to start the monitor.\n", port.SlavePath, port.SlavePath)
	}
	if *live {
		restore, err := enableRawTerminal()
		if err != nil {
			return err
		}
		defer restore()
		go readKeys(input, stopKeys)
		fmt.Fprintln(os.Stderr, "KIM-1 terminal: Ctrl-C or Ctrl-D to stop")
	}
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)
	var pending []byte
	ready := false
	autobaudQueued := false
	var bootstrap bootstrapKey
	readyCycle := uint64(3000000)
	decodedCount := 0
	start := time.Now()
	for *cycles == 0 || k.Bus.Cycles < *cycles {
		select {
		case <-interrupt:
			return nil
		default:
		}
		if *live {
			select {
			case <-stopKeys:
				return nil
			default:
			}
		}
		if *usePTY && !autobaudQueued {
			select {
			case b := <-input:
				if *trace {
					fmt.Fprintf(traceOut, "%012d HOST         received $%02X\n", k.Bus.Cycles, b)
				}
				if bootstrap.accept(b) {
					baud, readErr := port.BaudRate()
					if readErr != nil {
						fmt.Fprintf(os.Stderr, "KIM-1 autobaud: %v; check Minicom speed and press Return again\n", readErr)
					} else if baud > board.ClockHz || baud == 0 {
						fmt.Fprintf(os.Stderr, "KIM-1 autobaud: invalid PTY speed %d; check Minicom speed and press Return again\n", baud)
					} else {
						s.BitCycles = (board.ClockHz + baud/2) / baud
						s.Queue([]byte{0x7f})
						autobaudQueued = true
						// The 110-baud prompt takes roughly 3M CPU cycles.
						// Scale this conservative input holdoff to the client rate.
						readyCycle = k.Bus.Cycles + 400*board.ClockHz/baud
						fmt.Fprintf(os.Stderr, "KIM-1 autobaud: PTY %d baud, %d cycles/bit; sending RUBOUT ($7F) on PA7\n", baud, s.BitCycles)
					}
				}
			default:
			}
		} else if !*usePTY && k.Bus.Cycles == 2000 {
			s.Queue([]byte{0x7f})
		}
		// Original monitor greeting and first address prompt take about 3M
		// cycles at 110 baud. Defer host input until after that interval.
		if !ready && (!*usePTY || autobaudQueued) && k.Bus.Cycles >= readyCycle {
			ready = true
			pending = append(pending, []byte(*initial)...)
		}
		if ready {
			select {
			case b := <-input:
				if *trace {
					fmt.Fprintf(traceOut, "%012d HOST         received $%02X\n", k.Bus.Cycles, b)
				}
				pending = append(pending, b)
			default:
			}
			if len(pending) > 0 && s.RXIdle() {
				s.Queue(pending[:1])
				pending = pending[1:]
			}
		}
		if err = k.Clock(); err != nil {
			return err
		}
		for _, edge := range s.DrainEdges() {
			if *trace {
				level := 0
				if edge.High {
					level = 1
				}
				fmt.Fprintf(traceOut, "%012d %-6s %d     transition\n", edge.Cycle, edge.Signal, level)
			}
		}
		for _, b := range s.DrainDecoded() {
			decodedCount++
			if port != nil {
				_, _ = port.Master.Write([]byte{b}) // disconnected slave has no receiver
			} else if unicode.IsPrint(rune(b)) || b == '\r' || b == '\n' {
				fmt.Printf("%c", b)
			}
			if *trace {
				fmt.Fprintf(traceOut, "%012d SERIAL       decoded $%02X\n", s.Cycle, b)
			}
		}
		if (*live || *usePTY) && k.Bus.Cycles%1000 == 0 {
			// Real-time pacing changes host scheduling, never simulated cycles.
			elapsed := time.Duration(k.Bus.Cycles) * time.Second / time.Duration(board.ClockHz)
			if delay := time.Until(start.Add(elapsed)); delay > 0 {
				time.Sleep(delay)
			}
		}
	}
	fmt.Fprintf(os.Stderr, "KIM-1 stopped after %d cycles; PC=$%04X; decoded TX characters=%d\n", k.Bus.Cycles, k.CPU.View().PC, decodedCount)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "KIM-1:", err)
		os.Exit(1)
	}
}
