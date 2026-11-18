package machine

import "m65c02/internal/cpu"

// RRIOTPins captures resolved electrical port levels and direction masks.
// A direction bit of 1 means the 6530 drives the corresponding pin.
type RRIOTPins struct {
	PA   byte `json:"pa"`
	PB   byte `json:"pb"`
	DDRA byte `json:"ddra"`
	DDRB byte `json:"ddrb"`
}

// BoardPins stores both 6530 pin banks at the same bus-cycle instant.
type BoardPins struct {
	R002     RRIOTPins `json:"riot002"`
	R003     RRIOTPins `json:"riot003"`
	Digit    int       `json:"digit"`
	Segments byte      `json:"segments"`
}

// BusCycle is one resolved physical bus transfer, including sampled control
// lines. Phi2 is the active phase on which Transfer resolves each cycle.
type BusCycle struct {
	Cycle    uint64     `json:"cycle"`
	Address  uint16     `json:"address"`
	Data     byte       `json:"data"`
	Write    bool       `json:"write"`
	Sync     bool       `json:"sync"`
	IRQ      bool       `json:"irq"`
	NMI      bool       `json:"nmi"`
	Reset    bool       `json:"reset"`
	Ready    bool       `json:"ready"`
	Mnemonic string     `json:"mnemonic,omitempty"`
	Mode     string     `json:"mode,omitempty"`
	Length   int        `json:"length,omitempty"`
	Pins     *BoardPins `json:"pins,omitempty"`
}

// BusTrace bounds storage even when the original ROM runs indefinitely.
// Like Backplane, it is owned by its machine's clocking goroutine.
type BusTrace struct {
	rows        []BusCycle
	next        int
	count       int
	PortSampler func() BoardPins
}

func NewBusTrace(capacity int) *BusTrace {
	if capacity < 1 {
		capacity = 1
	}
	return &BusTrace{rows: make([]BusCycle, capacity)}
}

func (t *BusTrace) Record(cycle uint64, req cpu.Request, res cpu.Response) {
	if t == nil {
		return
	}
	data := res.DataIn
	if req.Write {
		data = req.DataOut
	}
	c := BusCycle{Cycle: cycle, Address: req.Address, Data: data, Write: req.Write,
		Sync: req.Sync, IRQ: res.IRQ, NMI: res.NMI, Reset: res.Reset, Ready: res.Ready}
	if req.Sync {
		c.Mnemonic, c.Mode, c.Length = cpu.DecodeOpcode(data)
	}
	if t.PortSampler != nil {
		pins := t.PortSampler()
		c.Pins = &pins
	}
	t.rows[t.next] = c
	t.next = (t.next + 1) % len(t.rows)
	if t.count < len(t.rows) {
		t.count++
	}
}

// Window returns up to limit entries strictly before before (zero = newest).
func (t *BusTrace) Window(before uint64, limit int) []BusCycle {
	if t == nil {
		return nil
	}
	if limit < 1 {
		limit = 1
	}
	if limit > t.count {
		limit = t.count
	}
	out := make([]BusCycle, 0, limit)
	for i := 0; i < t.count && len(out) < limit; i++ {
		at := (t.next - 1 - i + len(t.rows)) % len(t.rows)
		c := t.rows[at]
		if before == 0 || c.Cycle < before {
			out = append(out, c)
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// PortChanges returns cycles on which the selected RIOT's observable pins
// changed. The first captured sample is a baseline, not an event.
func (t *BusTrace) PortChanges(chip string) []BusCycle {
	rows := t.Window(0, len(t.rows))
	out := make([]BusCycle, 0)
	for i := 1; i < len(rows); i++ {
		old, now := rows[i-1].Pins, rows[i].Pins
		if old == nil || now == nil {
			continue
		}
		var a, b RRIOTPins
		switch chip {
		case "riot002":
			a, b = old.R002, now.R002
		case "riot003":
			a, b = old.R003, now.R003
		default:
			return nil
		}
		if a != b || chip == "riot002" && (old.Digit != now.Digit || old.Segments != now.Segments) {
			out = append(out, rows[i])
		}
	}
	return out
}

func (t *BusTrace) Clear() {
	if t != nil {
		t.next, t.count = 0, 0
	}
}
