package machine

import (
	"bytes"
	"os"
	"testing"

	"m65c02/internal/cpu"
)

func TestKIMBASICPaperTape(t *testing.T) {
	data, err := os.ReadFile("../../programs/kim1/kb9.kim")
	if err != nil {
		t.Fatal(err)
	}
	segments, err := ParseKIMTape(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(segments) != 368 {
		t.Fatalf("got %d records", len(segments))
	}
	k := testKIM1(t)
	if _, err := k.LoadKIMTape(data); err == nil {
		t.Fatal("accepted tape without expansion")
	}
	if err := k.EnableBASICRAM(); err != nil {
		t.Fatal(err)
	}
	for _, addr := range []uint16{0x2000, 0x4fff, 0x5000, 0x5fff} {
		k.Bus.Transfer(cpu.Request{Address: addr, DataOut: byte(addr ^ 0x5a), Write: true})
		if got := k.Bus.Transfer(cpu.Request{Address: addr}).DataIn; got != byte(addr^0x5a) {
			t.Fatalf("expanded RAM $%04X: %02X", addr, got)
		}
	}
	// Probing beyond expansion must neither find writable RAM nor alias
	// the original zero page, while the high monitor ROM remains visible.
	k.Bus.Transfer(cpu.Request{Address: 0x0000, DataOut: 0x53, Write: true})
	k.Bus.Transfer(cpu.Request{Address: 0x6000, DataOut: 0xa7, Write: true})
	if got := k.Bus.Transfer(cpu.Request{Address: 0x0000}).DataIn; got != 0x53 {
		t.Fatalf("$6000 write aliased zero page: %02X", got)
	}
	if got := k.Bus.Transfer(cpu.Request{Address: 0x6000}).DataIn; got != 0xff {
		t.Fatalf("expected open bus at $6000, got %02X", got)
	}
	n, err := k.LoadKIMTape(data)
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("empty program")
	}
	for _, seg := range segments {
		for i, want := range seg.Data {
			addr := seg.Address + uint16(i)
			if got := k.Bus.Transfer(cpu.Request{Address: addr}).DataIn; got != want {
				t.Fatalf("$%04X: %02X != %02X", addr, got, want)
			}
		}
	}
	if got := k.Bus.Transfer(cpu.Request{Address: 0x1c22}).DataIn; got != 0xea {
		t.Fatalf("ROM changed: %02X", got)
	}
}

func TestKIMBASICVersions(t *testing.T) {
	var original []byte
	for _, version := range []string{"kb9.kim", "kb9-v1.1.ptp", "kb9-v1.2.ptp", "kb9-v2.ptp"} {
		data, err := os.ReadFile("../../programs/kim1/" + version)
		if err != nil {
			t.Fatal(err)
		}
		segments, err := ParseKIMTape(data)
		if err != nil {
			t.Fatalf("%s: %v", version, err)
		}
		image := make([]byte, 0x4000)
		for _, seg := range segments {
			copy(image[int(seg.Address)-0x2000:], seg.Data)
		}
		if version == "kb9.kim" {
			original = image
		} else if version == "kb9-v1.1.ptp" && !bytes.Equal(original, image) {
			t.Fatal("Hans's V1.1 differs from the existing KB9 image")
		}
	}
}

func TestKIMTapeRejectsCorruption(t *testing.T) {
	data, err := os.ReadFile("../../programs/kim1/kb9.kim")
	if err != nil {
		t.Fatal(err)
	}
	data[12] = '0'
	if _, err := ParseKIMTape(data); err == nil {
		t.Fatal("accepted bad checksum")
	}
}
