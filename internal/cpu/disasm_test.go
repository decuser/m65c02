package cpu

import "testing"

func TestDecodeOpcodeMetadata(t *testing.T) {
	for _, tc := range []struct {
		op             byte
		mnemonic, mode string
		length         int
	}{
		{0xa9, "LDA", "imm", 2},
		{0x18, "CLC", "impl", 1},
		{0x69, "ADC", "imm", 2},
		{0x85, "STA", "zp", 2},
		{0x4c, "JMP", "abs", 3},
		{0x20, "JSR", "abs", 3},
		{0x00, "BRK", "impl", 2},
		{0x7c, "JMP", "inx", 3},
	} {
		mnemonic, mode, length := DecodeOpcode(tc.op)
		if mnemonic != tc.mnemonic || mode != tc.mode || length != tc.length {
			t.Fatalf("%02X: got %s %s %d", tc.op, mnemonic, mode, length)
		}
	}
}
