package cpu

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// Samples every available opcode. Set M65C02_CORPUS_LIMIT=10000 for a full pass.
// Unsupported/unresolved cases fail, not silently skipped.
func TestSynertekAllOpcodes(t *testing.T) {
	dir := os.Getenv("M65C02_CORPUS")
	if dir == "" {
		t.Skip("set M65C02_CORPUS to Synertek v1")
	}
	limit := 100
	if raw := os.Getenv("M65C02_CORPUS_LIMIT"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 1 {
			t.Fatalf("invalid M65C02_CORPUS_LIMIT=%q", raw)
		}
		limit = v
	}
	for opcode := 0; opcode < 256; opcode++ {
		op := fmt.Sprintf("%02x", opcode)
		t.Run(op, func(t *testing.T) {
			f, e := os.Open(filepath.Join(dir, op+".json"))
			if e != nil {
				t.Fatal(e)
			}
			defer f.Close()
			decoder := json.NewDecoder(f)
			if _, e = decoder.Token(); e != nil {
				t.Fatal(e)
			}
			n := 0
			for decoder.More() && n < limit {
				var v corpusCase
				if e = decoder.Decode(&v); e != nil {
					t.Fatal(e)
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
				if e = c.Restore(s); e != nil {
					t.Fatal(e)
				}
				if e = c.StepInstruction(b); e != nil {
					t.Fatalf("vector %d: %s: %v", n, v.Name, e)
				}
				got := c.View()
				want := v.Final
				if got.PC != want.PC || got.S != want.S || got.A != want.A || got.X != want.X || got.Y != want.Y || got.P != want.P {
					t.Fatalf("vector %d %s: regs got PC=%04x S=%02x A=%02x X=%02x Y=%02x P=%02x want PC=%04x S=%02x A=%02x X=%02x Y=%02x P=%02x", n, v.Name, got.PC, got.S, got.A, got.X, got.Y, got.P, want.PC, want.S, want.A, want.X, want.Y, want.P)
				}
				for _, p := range want.RAM {
					if b.data[p[0]] != byte(p[1]) {
						t.Fatalf("vector %d %s: mem $%04x got %02x want %02x", n, v.Name, p[0], b.data[p[0]], byte(p[1]))
					}
				}
				if len(b.cycles) != len(v.Cycles) {
					t.Fatalf("vector %d %s: cycles got %d want %d; got %+v expected %s", n, v.Name, len(b.cycles), len(v.Cycles), b.cycles, fmt.Sprint(v.Cycles))
				}
				for i, raw := range v.Cycles {
					var addr, val uint16
					var mode string
					if e = json.Unmarshal(raw[0], &addr); e != nil {
						t.Fatal(e)
					}
					if e = json.Unmarshal(raw[1], &val); e != nil {
						t.Fatal(e)
					}
					if e = json.Unmarshal(raw[2], &mode); e != nil {
						t.Fatal(e)
					}
					tr := b.cycles[i]
					if tr.addr != addr || tr.value != byte(val) || tr.write != (mode == "write") {
						t.Fatalf("vector %d %s cycle %d got %+v want %04x %02x %s", n, v.Name, i, tr, addr, val, mode)
					}
				}
				n++
			}
			if n == 0 {
				t.Fatal("empty opcode corpus")
			}
		})
	}
}
