package machine

import (
	"os"
	"testing"
)

func panelFixture(t *testing.T) *KIM1 {
	t.Helper()
	k, err := NewKIM1(make([]byte, 1024), make([]byte, 1024))
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func selectPanelDecoder(k *KIM1, output int) {
	r := k.RRIOT002.RegisterDevice()
	r.Write(3, 0x1e) // PB1..PB4 outputs, PB0 left to TTY
	r.Write(2, byte(output<<1))
}

func TestKIMFrontPanelKeyMatrixAndJumper(t *testing.T) {
	k := panelFixture(t)
	r := k.RRIOT002.RegisterDevice()
	r.Write(1, 0) // PA0..PA6 are inputs during a keyboard scan
	if err := k.Panel.SetKey("A", true); err != nil {
		t.Fatal(err)
	}
	if err := k.Panel.SetKey("GO", true); err != nil {
		t.Fatal(err)
	}
	selectPanelDecoder(k, 1)
	k.Panel.UpdateInputs()
	if got := k.RRIOT002.PinsA(); got != 0xf7 {
		t.Fatalf("row 1, A pressed: PA=%02X, want F7", got)
	}
	selectPanelDecoder(k, 2)
	k.Panel.UpdateInputs()
	if got := k.RRIOT002.PinsA(); got != 0xdf {
		t.Fatalf("row 2, GO pressed: PA=%02X, want DF", got)
	}
	selectPanelDecoder(k, 0)
	k.Panel.UpdateInputs()
	if got := k.RRIOT002.PinsA(); got != 0xff {
		t.Fatalf("unpressed row: PA=%02X", got)
	}
	if err := k.Panel.SetKey("GO", false); err != nil {
		t.Fatal(err)
	}
	selectPanelDecoder(k, 2)
	k.Panel.UpdateInputs()
	if got := k.RRIOT002.PinsA(); got != 0xff {
		t.Fatalf("released GO: PA=%02X", got)
	}
	k.Panel.TTY = true
	selectPanelDecoder(k, 3)
	k.Panel.UpdateInputs()
	if got := k.RRIOT002.PinsA(); got != 0xfe {
		t.Fatalf("TTY-select A-21/A-V jumper: PA=%02X", got)
	}
	k.Panel.TTY = false
	k.Panel.UpdateInputs()
	if got := k.RRIOT002.PinsA(); got != 0xff {
		t.Fatalf("open jumper: PA=%02X", got)
	}
	if err := k.Panel.SetKey("RS", true); err == nil {
		t.Fatal("reset must not be a matrix key")
	}
}

func TestKIMFrontPanelMultiplexAndSerialIsolation(t *testing.T) {
	k := panelFixture(t)
	r := k.RRIOT002.RegisterDevice()
	r.Write(1, 0x7f) // segment drivers PA0..PA6, PA7 serial input
	r.Write(0, 0x55)
	k.RRIOT002.SetInputA(0x7f) // serial RX low on PA7
	for output := 4; output <= 9; output++ {
		selectPanelDecoder(k, output)
		k.Panel.Tick()
		if got := k.Panel.Current; got.Digit != output-4 || got.SegmentBits != 0x55 {
			t.Fatalf("decoder %d: %+v", output, got)
		}
	}
	if got := k.RRIOT002.PinsA(); got&0x80 != 0 {
		t.Fatalf("display must not alter serial PA7: %02X", got)
	}
	for i := 0; i < 6; i++ {
		if k.Panel.LastScan[i] != 0x55 || k.Panel.Dwell[i][0] != 1 || k.Panel.Dwell[i][1] != 0 {
			t.Fatalf("digit %d observation: last=%02X dwell=%v", i, k.Panel.LastScan[i], k.Panel.Dwell[i])
		}
	}
	selectPanelDecoder(k, 3)
	k.Panel.Tick()
	if k.Panel.Current.Digit != -1 {
		t.Fatal("keyboard row must disable display")
	}
	r.Write(3, 0) // inputs cannot drive a 74145 selection
	k.Panel.Tick()
	if k.Panel.Current.Digit != -1 {
		t.Fatal("undriven decoder must disable display")
	}
}

func TestKIMFrontPanelStopAndResetLines(t *testing.T) {
	k := panelFixture(t)
	k.PressStop()
	if !k.Bus.NMI {
		t.Fatal("ST must assert NMI")
	}
	k.ReleaseStop()
	if k.Bus.NMI {
		t.Fatal("releasing ST must deassert NMI")
	}
	if err := k.Panel.SetKey("F", true); err != nil {
		t.Fatal(err)
	}
	k.ResetCPU()
	if !k.Bus.Reset || k.Panel.Current.Digit != -1 {
		t.Fatal("RS must assert reset and blank display")
	}
	k.ReleaseReset()
	if k.Bus.Reset {
		t.Fatal("RS release must deassert reset")
	}
	selectPanelDecoder(k, 2)
	k.Panel.UpdateInputs()
	if got := k.RRIOT002.PinsA(); got != 0xfd {
		t.Fatalf("physical key must remain held over reset, PA=%02X", got)
	}
}

func TestKIMFrontPanelOriginalROMScansDigits(t *testing.T) {
	p002, p003 := os.Getenv("M65C02_KIM_ROM002"), os.Getenv("M65C02_KIM_ROM003")
	if p002 == "" || p003 == "" {
		t.Skip("supply both original KIM-1 ROMs")
	}
	k, err := NewKIM1FromFiles(p002, p003)
	if err != nil {
		t.Fatal(err)
	}
	k.TTYSelect = false
	if err = k.StepInstruction(); err != nil {
		t.Fatal(err)
	}
	k.ReleaseReset()
	for k.Bus.Cycles < 100000 {
		if err = k.Clock(); err != nil {
			t.Fatal(err)
		}
	}
	var active int
	for _, count := range k.Panel.Dwell {
		for _, n := range count {
			if n > 0 {
				active++
				break
			}
		}
	}
	if active != 6 {
		t.Fatalf("original ROM scanned %d of 6 display digits", active)
	}
}
