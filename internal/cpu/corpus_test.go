package cpu

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type corpusRegisters struct {
	PC            uint16 `json:"pc"`
	S, A, X, Y, P byte
	RAM           [][2]uint16 `json:"ram"`
}
type corpusCase struct {
	Name    string              `json:"name"`
	Initial corpusRegisters     `json:"initial"`
	Final   corpusRegisters     `json:"final"`
	Cycles  [][]json.RawMessage `json:"cycles"`
}

// A trace captures the actual data transferred, not post-execution RAM peeks.
type traceBus struct {
	data   [65536]byte
	cycles []traceCycle
}
type traceCycle struct {
	addr  uint16
	value byte
	write bool
}

func (b *traceBus) Transfer(r Request) Response {
	if r.Write {
		b.data[r.Address] = r.DataOut
		b.cycles = append(b.cycles, traceCycle{r.Address, r.DataOut, true})
		return Response{Ready: true}
	}
	v := b.data[r.Address]
	b.cycles = append(b.cycles, traceCycle{r.Address, v, false})
	return Response{DataIn: v, Ready: true}
}
func TestSynertekV1Selected(t *testing.T) {
	dir := os.Getenv("M65C02_CORPUS")
	if dir == "" {
		t.Skip("Synertek v1 corpus not configured (set M65C02_CORPUS)")
	}
	// Only opcodes implemented in this initial production cut.
	for _, op := range []string{"03", "ea", "a9", "bd", "9d", "fe", "4c", "d0", "00"} {
		t.Run(op, func(t *testing.T) {
			file, err := os.Open(filepath.Join(dir, op+".json"))
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			d := json.NewDecoder(file)
			if _, err = d.Token(); err != nil {
				t.Fatal(err)
			}
			n := 0
			for d.More() {
				var v corpusCase
				if err := d.Decode(&v); err != nil {
					t.Fatal(err)
				}
				b := &traceBus{}
				for _, p := range v.Initial.RAM {
					b.data[p[0]] = byte(p[1])
				}
				c := New(v.Initial.PC)
				s := c.Snapshot()
				s.A = v.Initial.A
				s.X = v.Initial.X
				s.Y = v.Initial.Y
				s.S = v.Initial.S
				s.P = v.Initial.P
				if err := c.Restore(s); err != nil {
					t.Fatal(err)
				}
				if err := c.StepInstruction(b); err != nil {
					t.Fatalf("vector %d %s: %v", n, v.Name, err)
				}
				final := c.View()
				if final.PC != v.Final.PC || final.S != v.Final.S || final.A != v.Final.A || final.X != v.Final.X || final.Y != v.Final.Y || final.P != v.Final.P {
					t.Fatalf("vector %d %s: got PC=%04x S=%02x A=%02x X=%02x Y=%02x P=%02x, want PC=%04x S=%02x A=%02x X=%02x Y=%02x P=%02x", n, v.Name, final.PC, final.S, final.A, final.X, final.Y, final.P, v.Final.PC, v.Final.S, v.Final.A, v.Final.X, v.Final.Y, v.Final.P)
				}
				for _, p := range v.Final.RAM {
					if b.data[p[0]] != byte(p[1]) {
						t.Fatalf("vector %d %s: memory %04x got %02x want %02x", n, v.Name, p[0], b.data[p[0]], byte(p[1]))
					}
				}
				if len(b.cycles) != len(v.Cycles) {
					t.Fatalf("vector %d %s: cycles got %d want %d", n, v.Name, len(b.cycles), len(v.Cycles))
				}
				for i, raw := range v.Cycles {
					if len(raw) != 3 {
						t.Fatalf("vector %d unexpected cycle fields", n)
					}
					var addr, val uint16
					var mode string
					if err := json.Unmarshal(raw[0], &addr); err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(raw[1], &val); err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(raw[2], &mode); err != nil {
						t.Fatal(err)
					}
					got := b.cycles[i]
					if got.addr != addr || got.value != byte(val) || got.write != (mode == "write") {
						t.Fatal(fmt.Sprintf("vector %d %s cycle %d got %+v want %04x %02x %s", n, v.Name, i, got, addr, val, mode))
					}
				}
				n++
			}
			if n == 0 {
				t.Fatal("empty opcode corpus")
			}
			t.Logf("%s: %d vectors passed", op, n)
		})
	}
}
