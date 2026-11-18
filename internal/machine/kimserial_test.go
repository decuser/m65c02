package machine

import "testing"

func TestKIMSerialPinTraceAndFrames(t *testing.T) {
	r := newTestRRIOT(t)
	s := NewKIMSerial(r, 8)
	io := r.RegisterDevice()
	io.Write(3, 1) // PB0 output
	io.Write(2, 1) // idle mark
	s.Tick()
	io.Write(2, 0) // start
	for i := 0; i < 8; i++ {
		s.Tick()
	}
	// $41 = eight data bits, LSB first
	for i := 0; i < 8; i++ {
		if 0x41&(1<<i) != 0 {
			io.Write(2, 1)
		} else {
			io.Write(2, 0)
		}
		for j := 0; j < 8; j++ {
			s.Tick()
		}
	}
	io.Write(2, 1)
	for i := 0; i < 16; i++ {
		s.Tick()
	}
	if got := string(s.DrainDecoded()); got != "A" {
		t.Fatalf("TX decode %q", got)
	}
	edges := s.DrainEdges()
	if len(edges) < 4 || edges[0].Signal != "TX" || edges[0].High {
		t.Fatalf("TX transitions: %+v", edges)
	}
	s.Queue([]byte{0x7f})
	var low bool
	for i := 0; i < 90; i++ {
		s.Tick()
		if r.PinsA()&0x80 == 0 {
			low = true
		}
	}
	if !low || r.PinsA()&0x80 == 0 {
		t.Fatal("RX did not drive start/data bits and restore idle")
	}
}
