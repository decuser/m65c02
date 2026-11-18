// Command m65c02-panel renders the original KIM-1 front panel locally.
// It observes and operates the hardware model, never ROM monitor routines.
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"m65c02/internal/machine"
)

//go:embed panel.html analyzer.html visible.html
var panelFiles embed.FS

type panelServer struct {
	mu            sync.Mutex
	kim           *machine.KIM1
	microTrace    *machine.MicroTrace
	microEnabled  bool
	running       bool
	powered       bool
	awaitingReset bool
	clockHz       uint64
	rom002        []byte
	rom003        []byte
	ramMode       string
	ramSeed       int64
	powerCount    int64
}

type panelState struct {
	Cycles        uint64       `json:"cycles"`
	PC            uint16       `json:"pc"`
	Running       bool         `json:"running"`
	Powered       bool         `json:"powered"`
	AwaitingReset bool         `json:"awaitingReset"`
	TTY           bool         `json:"tty"`
	SST           bool         `json:"sst"`
	Active        int          `json:"active"`
	Segments      byte         `json:"segments"`
	LastScan      [6]byte      `json:"lastScan"`
	Dwell         [6][7]uint64 `json:"dwell"`
}

func (p *panelServer) clock(cycles uint64) error {
	for i := uint64(0); i < cycles; i++ {
		if err := p.kim.Clock(); err != nil {
			return err
		}
	}
	return nil
}

func (p *panelServer) powerOn() error {
	k, err := machine.NewKIM1(p.rom002, p.rom003)
	if err != nil {
		return err
	}
	k.Bus.Trace = machine.NewBusTrace(8192)
	p.microTrace = machine.NewMicroTrace(32768)
	if p.microEnabled {
		k.CPU.ObserveMicro = p.microTrace.Record
	}
	k.Bus.Trace.PortSampler = func() machine.BoardPins {
		read := func(chip *machine.RRIOT) machine.RRIOTPins {
			_, da := chip.OutputA()
			_, db := chip.OutputB()
			return machine.RRIOTPins{PA: chip.PinsA(), PB: chip.PinsB(), DDRA: da, DDRB: db}
		}
		return machine.BoardPins{R002: read(k.RRIOT002), R003: read(k.RRIOT003),
			Digit: k.Panel.Current.Digit, Segments: k.Panel.Current.SegmentBits}
	}
	if p.ramMode == "device" {
		// A stable per-device bias with sparse variation on cold starts.
		// Not a transistor-level model of a MOS 6102.
		pattern := rand.New(rand.NewSource(p.ramSeed))
		noise := rand.New(rand.NewSource(p.ramSeed ^ (p.powerCount+1)*0x5deece66d))
		value := func() byte {
			v := byte(pattern.Intn(256))
			for bit := uint(0); bit < 8; bit++ {
				if noise.Intn(64) == 0 {
					v ^= 1 << bit
				}
			}
			return v
		}
		for i := uint16(0); i < 1024; i++ {
			k.RAM.Write(i, value())
		}
		for _, chip := range []*machine.RRIOT{k.RRIOT002, k.RRIOT003} {
			for i := uint16(0); i < 64; i++ {
				chip.RAMDevice().Write(i, value())
			}
		}
	}
	p.powerCount++
	// External connector wiring remains attached across power cycles.
	if p.kim != nil {
		k.TTYSelect = p.kim.TTYSelect
		k.SST = p.kim.SST
	}
	p.kim = k
	p.powered = true
	p.running = false
	p.awaitingReset = true
	return nil
}

func (p *panelServer) run(stop <-chan struct{}) {
	// Execute one simulated second per real second in 10ms batches.
	// Serial and display timing remain in CPU cycles, not wall-clock ticks.
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	var remainder uint64
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			p.mu.Lock()
			if p.powered && p.running {
				remainder += p.clockHz % 100
				cycles := p.clockHz/100 + remainder/100
				remainder %= 100
				if err := p.clock(cycles); err != nil {
					log.Printf("KIM-1 CPU stopped: %v", err)
					p.running = false
				}
			}
			p.mu.Unlock()
		}
	}
}

func (p *panelServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page, err := fs.ReadFile(panelFiles, "panel.html")
		if err != nil {
			http.Error(w, "panel unavailable", 500)
			return
		}
		_, _ = w.Write(page)
	})
	mux.HandleFunc("/analyzer", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET required", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page, err := fs.ReadFile(panelFiles, "analyzer.html")
		if err != nil {
			http.Error(w, "analyzer unavailable", 500)
			return
		}
		_, _ = w.Write(page)
	})
	mux.HandleFunc("/visible", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET required", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		page, err := fs.ReadFile(panelFiles, "visible.html")
		if err != nil {
			http.Error(w, "visible computer unavailable", 500)
			return
		}
		_, _ = w.Write(page)
	})
	mux.HandleFunc("/api/trace", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET required", http.StatusMethodNotAllowed)
			return
		}
		before, err := strconv.ParseUint(r.URL.Query().Get("before"), 10, 64)
		if r.URL.Query().Get("before") == "" {
			before = 0
			err = nil
		}
		if err != nil {
			http.Error(w, "invalid before cycle", 400)
			return
		}
		p.mu.Lock()
		var rows []machine.BusCycle
		if p.kim.Bus.Trace != nil {
			rows = p.kim.Bus.Trace.Window(before, 128)
		}
		result := struct {
			Rows         []machine.BusCycle  `json:"rows"`
			Micro        []machine.MicroStep `json:"micro"`
			MicroEnabled bool                `json:"microEnabled"`
			Cycle        uint64              `json:"cycle"`
			Running      bool                `json:"running"`
			Powered      bool                `json:"powered"`
		}{Rows: rows, MicroEnabled: p.microEnabled, Cycle: p.kim.Bus.Cycles, Running: p.running, Powered: p.powered}
		if r.URL.Query().Get("micro") == "1" && p.microEnabled && len(rows) > 0 {
			result.Micro = p.microTrace.Window(rows[0].Cycle, rows[len(rows)-1].Cycle)
		}
		p.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})
	mux.HandleFunc("/api/trace/micro", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", 405)
			return
		}
		var request struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid JSON", 400)
			return
		}
		p.mu.Lock()
		p.microEnabled = request.Enabled
		if request.Enabled {
			p.microTrace = machine.NewMicroTrace(32768)
			p.kim.CPU.ObserveMicro = p.microTrace.Record
		} else {
			p.kim.CPU.ObserveMicro = nil
		}
		p.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/trace/changes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET required", http.StatusMethodNotAllowed)
			return
		}
		chip := r.URL.Query().Get("probe")
		if chip != "riot002" && chip != "riot003" {
			http.Error(w, "select a RIOT probe", http.StatusBadRequest)
			return
		}
		p.mu.Lock()
		var changes []machine.BusCycle
		if p.kim.Bus.Trace != nil {
			changes = p.kim.Bus.Trace.PortChanges(chip)
		}
		p.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(changes)
	})
	mux.HandleFunc("/api/trace/export", p.exportTrace)
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET required", http.StatusMethodNotAllowed)
			return
		}
		p.mu.Lock()
		s := panelState{
			Cycles: p.kim.Bus.Cycles, PC: p.kim.CPU.View().PC,
			Running: p.running, Powered: p.powered, AwaitingReset: p.awaitingReset, TTY: p.kim.TTYSelect, SST: p.kim.SST,
			Active: p.kim.Panel.Current.Digit, Segments: p.kim.Panel.Current.SegmentBits,
			LastScan: p.kim.Panel.LastScan, Dwell: p.kim.Panel.Dwell,
		}
		p.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s)
	})
	mux.HandleFunc("/api/key", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var q struct {
			Key  string `json:"key"`
			Down bool   `json:"down"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&q); err != nil {
			http.Error(w, "invalid key", 400)
			return
		}
		p.mu.Lock()
		if !p.powered {
			p.mu.Unlock()
			http.Error(w, "power is off", 409)
			return
		}
		err := p.kim.Panel.SetKey(q.Key, q.Down)
		p.mu.Unlock()
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/control", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var q struct {
			Control string `json:"control"`
			Down    bool   `json:"down"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&q); err != nil {
			http.Error(w, "invalid control", 400)
			return
		}
		p.mu.Lock()
		defer p.mu.Unlock()
		if q.Control == "POWER" {
			if q.Down && !p.powered {
				if err := p.powerOn(); err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
			} else if !q.Down {
				p.powered = false
				p.running = false
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !p.powered {
			http.Error(w, "power is off", 409)
			return
		}
		switch q.Control {
		case "RS":
			if q.Down {
				p.running = false
				p.kim.ResetCPU()
				if err := p.clock(8); err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
			} else {
				p.kim.ReleaseReset()
				p.awaitingReset = false
				p.running = true
			}
		case "ST":
			if q.Down {
				p.kim.PressStop()
			} else {
				p.kim.ReleaseStop()
			}
		case "TTY":
			p.kim.TTYSelect = q.Down
		case "SST":
			p.kim.SST = q.Down
		case "PAUSE":
			if p.awaitingReset {
				http.Error(w, "press RS to initialize", 409)
				return
			}
			p.running = !q.Down
		case "STEP":
			if p.awaitingReset {
				http.Error(w, "press RS to initialize", 409)
				return
			}
			if p.running {
				http.Error(w, "pause first", 409)
				return
			}
			if err := p.clock(1); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		case "INSTRUCTION":
			if p.awaitingReset || p.running {
				http.Error(w, "pause an initialized machine first", 409)
				return
			}
			if err := p.kim.StepInstruction(); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		case "TRACE_CLEAR":
			p.kim.Bus.Trace.Clear()
		default:
			http.Error(w, "unknown control", 400)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

func main() {
	config := flag.String("config", "machines/kim1.toml", "machine configuration")
	rom002 := flag.String("rom002", "", "6530-002 ROM path")
	rom003 := flag.String("rom003", "", "6530-003 ROM path")
	address := flag.String("addr", "127.0.0.1:6580", "local HTTP listen address")
	open := flag.Bool("open", true, "open the local panel in the default browser")
	ramMode := flag.String("ram-mode", "device", "cold-start RAM: device or zero")
	ramSeed := flag.Int64("ram-seed", 0x4b494d31, "stable virtual-device SRAM pattern seed")
	flag.Parse()
	if *ramMode != "device" && *ramMode != "zero" {
		log.Fatal("ram-mode must be device or zero")
	}
	board, err := machine.LoadKIM1Config(*config)
	if err != nil {
		log.Fatal(err)
	}
	if *rom002 == "" {
		*rom002 = board.ROM002
	}
	if *rom003 == "" {
		*rom003 = board.ROM003
	}
	image002, err := os.ReadFile(*rom002)
	if err != nil {
		log.Fatal(err)
	}
	image003, err := os.ReadFile(*rom003)
	if err != nil {
		log.Fatal(err)
	}
	p := &panelServer{clockHz: board.ClockHz, rom002: image002, rom003: image003, ramMode: *ramMode, ramSeed: *ramSeed}
	if err := p.powerOn(); err != nil {
		log.Fatal(err)
	}
	stop := make(chan struct{})
	defer close(stop)
	go p.run(stop)
	url := "http://" + *address + "/"
	fmt.Fprintf(os.Stderr, "KIM-1 front panel: %s\n", url)
	if *open {
		go func() { _ = exec.Command("xdg-open", url).Run() }()
	}
	log.Fatal(http.ListenAndServe(*address, p.routes()))
}
