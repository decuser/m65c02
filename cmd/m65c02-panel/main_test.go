package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"m65c02/internal/machine"
)

func testPanel(t *testing.T) *panelServer {
	t.Helper()
	rom002, rom003 := make([]byte, 1024), make([]byte, 1024)
	k, err := machine.NewKIM1(rom002, rom003)
	if err != nil {
		t.Fatal(err)
	}
	return &panelServer{kim: k, powered: true, rom002: rom002, rom003: rom003, clockHz: 1000000}
}

func TestPanelPowerCycleClearsVolatileRAM(t *testing.T) {
	p := testPanel(t)
	p.ramMode = "zero"
	p.kim.RAM.Write(0x200, 0xa9)
	postPanel(t, p, "/api/control", `{"control":"TTY","down":true}`, 204)
	postPanel(t, p, "/api/control", `{"control":"SST","down":true}`, 204)
	postPanel(t, p, "/api/control", `{"control":"POWER","down":false}`, 204)
	if p.powered || p.running {
		t.Fatal("power off must stop machine")
	}
	postPanel(t, p, "/api/key", `{"key":"A","down":true}`, 409)
	postPanel(t, p, "/api/control", `{"control":"STEP","down":true}`, 409)
	postPanel(t, p, "/api/control", `{"control":"POWER","down":true}`, 204)
	if !p.powered || p.running || !p.awaitingReset || !p.kim.TTYSelect || !p.kim.SST {
		t.Fatal("power on failed or lost external wiring")
	}
	if got := p.kim.RAM.Read(0x200); got != 0 {
		t.Fatalf("RAM survived unplug/replug: %02X", got)
	}
	if p.kim.Bus.Cycles != 0 {
		t.Fatalf("CPU cycle count did not restart: %d", p.kim.Bus.Cycles)
	}
	postPanel(t, p, "/api/control", `{"control":"STEP","down":true}`, 409)
	postPanel(t, p, "/api/control", `{"control":"RS","down":true}`, 204)
	postPanel(t, p, "/api/control", `{"control":"RS","down":false}`, 204)
	if !p.running || p.awaitingReset {
		t.Fatal("RS must start the machine after cold power on")
	}
}

func TestPanelDeviceRAMPowerPattern(t *testing.T) {
	p := testPanel(t)
	p.ramMode = "device"
	p.ramSeed = 0x4b494d31
	postPanel(t, p, "/api/control", `{"control":"POWER","down":false}`, 204)
	postPanel(t, p, "/api/control", `{"control":"POWER","down":true}`, 204)
	first := make([]byte, 1024)
	for i := range first {
		first[i] = p.kim.RAM.Read(uint16(i))
	}
	if first[0] == 0 && first[1] == 0 && first[2] == 0 {
		t.Fatal("device RAM resembles a zero-filled image")
	}
	for _, chip := range []*machine.RRIOT{p.kim.RRIOT002, p.kim.RRIOT003} {
		var nonzero bool
		for i := uint16(0); i < 64; i++ {
			if chip.RAMDevice().Read(i) != 0 {
				nonzero = true
				break
			}
		}
		if !nonzero {
			t.Fatal("6530 RAM was not initialized")
		}
	}
	postPanel(t, p, "/api/control", `{"control":"POWER","down":false}`, 204)
	postPanel(t, p, "/api/control", `{"control":"POWER","down":true}`, 204)
	var stable, changed int
	for i, v := range first {
		if p.kim.RAM.Read(uint16(i)) == v {
			stable++
		} else {
			changed++
		}
	}
	if stable < 500 || changed < 1 {
		t.Fatalf("expected stable device bias with cold-start variation: stable=%d changed=%d", stable, changed)
	}
}

func postPanel(t *testing.T, p *panelServer, path, data string, code int) {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(data))
	w := httptest.NewRecorder()
	p.routes().ServeHTTP(w, r)
	if w.Code != code {
		t.Fatalf("%s %s: got HTTP %d, want %d: %s", path, data, w.Code, code, w.Body.String())
	}
}

func TestPanelRoutesAndHardwareControls(t *testing.T) {
	p := testPanel(t)
	w := httptest.NewRecorder()
	p.routes().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "KIM-1") {
		t.Fatalf("missing panel HTML: %d", w.Code)
	}
	w = httptest.NewRecorder()
	p.routes().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/analyzer", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Logic Analyzer") {
		t.Fatalf("missing analyzer page: %d", w.Code)
	}
	w = httptest.NewRecorder()
	p.routes().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/state", nil))
	var s panelState
	if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.Running || s.Cycles != 0 || s.TTY {
		t.Fatalf("wrong initial state: %+v", s)
	}
	postPanel(t, p, "/api/key", `{"key":"A","down":true}`, 204)
	postPanel(t, p, "/api/key", `{"key":"A","down":false}`, 204)
	postPanel(t, p, "/api/key", `{"key":"NOT-A-KEY","down":true}`, 400)
	postPanel(t, p, "/api/control", `{"control":"TTY","down":true}`, 204)
	postPanel(t, p, "/api/control", `{"control":"SST","down":true}`, 204)
	if !p.kim.TTYSelect {
		t.Fatal("TTY jumper not applied")
	}
	if !p.kim.SST {
		t.Fatal("SST switch not applied")
	}
	postPanel(t, p, "/api/control", `{"control":"ST","down":true}`, 204)
	if !p.kim.Bus.NMI {
		t.Fatal("ST did not assert NMI")
	}
	postPanel(t, p, "/api/control", `{"control":"ST","down":false}`, 204)
	if p.kim.Bus.NMI {
		t.Fatal("ST did not release NMI")
	}
	postPanel(t, p, "/api/control", `{"control":"RS","down":true}`, 204)
	if !p.kim.Bus.Reset {
		t.Fatal("RS did not assert reset")
	}
	postPanel(t, p, "/api/control", `{"control":"RS","down":false}`, 204)
	if p.kim.Bus.Reset {
		t.Fatal("RS did not release reset")
	}
	postPanel(t, p, "/api/control", `{"control":"PAUSE","down":true}`, 204)
	before := p.kim.Bus.Cycles
	postPanel(t, p, "/api/control", `{"control":"STEP","down":true}`, 204)
	if p.kim.Bus.Cycles != before+1 {
		t.Fatal("STEP did not clock one cycle")
	}
	p.kim.Bus.Trace = machine.NewBusTrace(16)
	postPanel(t, p, "/api/control", `{"control":"INSTRUCTION","down":true}`, 204)
	w = httptest.NewRecorder()
	p.routes().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/trace", nil))
	var capture struct {
		Rows []machine.BusCycle `json:"rows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &capture); err != nil {
		t.Fatal(err)
	}
	if len(capture.Rows) == 0 {
		t.Fatal("instruction did not produce bus capture")
	}
	postPanel(t, p, "/api/control", `{"control":"TRACE_CLEAR","down":true}`, 204)
	if got := len(p.kim.Bus.Trace.Window(0, 16)); got != 0 {
		t.Fatalf("trace clear retained %d samples", got)
	}
	postPanel(t, p, "/api/control", `{"control":"PAUSE","down":false}`, 204)
	postPanel(t, p, "/api/control", `{"control":"STEP","down":true}`, 409)
}
