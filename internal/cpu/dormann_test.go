package cpu

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// These are the upstream as65 default binaries, not generated fixtures.
// See Klaus2m5/6502_65C02_functional_tests, commit 7954e2dbb49c.
func TestDormann(t *testing.T) {
	dir := os.Getenv("M65C02_DORMANN")
	if dir == "" {
		t.Skip("Dormann binary directory unavailable (set M65C02_DORMANN)")
	}
	for _, tc := range []struct {
		name, file string
		success    uint16
	}{
		{"6502 documented", "6502_functional_test.bin", 0x3469},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join(dir, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			if len(b) != 65536 {
				t.Fatalf("%s has %d bytes, want 65536", tc.file, len(b))
			}
			m := &dormannRAM{}
			copy(m.data[:], b)
			c := New(0x400)
			var lastPC uint16
			var repeated int
			for n := 0; n < 100000000; n++ {
				pc := c.View().PC
				if n > 0 && pc == lastPC {
					repeated++
				} else {
					repeated = 0
				}
				if repeated == 3 {
					if pc == tc.success {
						t.Logf("passed after %d instructions (%d cycles)", n, c.View().Cycles)
						return
					}
					t.Fatalf("trapped at $%04X after %d instructions; expected $%04X", pc, n, tc.success)
				}
				lastPC = pc
				if err := c.StepInstruction(m); err != nil {
					t.Fatal(err)
				}
			}
			t.Fatal(fmt.Sprintf("did not finish within 100 million instructions; PC=$%04X", c.View().PC))
		})
	}
}

type dormannRAM struct{ data [65536]byte }

func (m *dormannRAM) Transfer(r Request) Response {
	if r.Write {
		m.data[r.Address] = r.DataOut
		return Response{Ready: true}
	}
	return Response{DataIn: m.data[r.Address], Ready: true}
}
