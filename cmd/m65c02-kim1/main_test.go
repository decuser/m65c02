package main

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
	"time"

	"m65c02/internal/machine"
)

func TestLocalTeleprinterEcho(t *testing.T) {
	var b bytes.Buffer
	for _, c := range []byte("1234 ") {
		localEcho(&b, c)
	}
	localEcho(&b, '\r')
	if got := b.String(); got != "1234 \r\n" {
		t.Fatalf("local echo %q", got)
	}
}

func TestBootstrapKey(t *testing.T) {
	for _, tc := range []struct {
		name string
		input []byte
		want int
	}{
		{"rubout", []byte{0x7f}, 1},
		{"return", []byte{'\r'}, 1},
		{"modern-delete", []byte{0x1b, '[', '3', '~'}, 1},
		{"ordinary-address", []byte("1234 "), 0},
		{"broken-escape", []byte{0x1b, '[', '4', '~'}, 0},
		{"escape-followed-by-return", []byte{0x1b, '\r'}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var k bootstrapKey
			got := 0
			for _, b := range tc.input {
				if k.accept(b) {
					got++
				}
			}
			if got != tc.want {
				t.Fatalf("bootstrap keys recognized = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPTYClientConnectionAndInput(t *testing.T) {
	p, err := machine.NewPTY()
	if err != nil {
		t.Skipf("PTY unavailable: %v", err)
	}
	defer p.Close()
	input := make(chan byte, 8)
	connected := make(chan struct{}, 1)
	stop := make(chan struct{})
	defer close(stop)
	go pollPTY(p, input, connected, stop)
	slave, err := os.OpenFile(p.SlavePath, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	stty := exec.Command("stty", "raw", "-echo")
	stty.Stdin = slave
	if err = stty.Run(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-connected:
	case <-time.After(time.Second):
		t.Fatal("PTY connection not detected")
	}
	if _, err = slave.Write([]byte{0x7f}); err != nil {
		t.Fatal(err)
	}
	select {
	case b := <-input:
		if b != 0x7f {
			t.Fatalf("PTY byte %02x", b)
		}
	case <-time.After(time.Second):
		t.Fatal("PTY input not forwarded")
	}
}
