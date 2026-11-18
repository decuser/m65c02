package machine

// SerialEdge records a physical serial pin transition at a machine cycle.
type SerialEdge struct {
	Cycle  uint64
	Signal string
	High   bool
}

// KIMSerial connects a virtual current-loop terminal to the KIM-1's 6530-002
// PA7 receive and PB0 transmit pins. It does not intercept monitor routines.
// One Tick is one emulated CPU bus cycle; the host controls pacing.
type KIMSerial struct {
	chip           *RRIOT
	Cycle          uint64
	BitCycles      uint64
	DataBits       uint8
	TXInvert       bool
	RXInvert       bool
	RXHigh, TXHigh bool
	edges          []SerialEdge
	pending        []byte
	rxFrames       []bool
	rxPos          uint64
	txStart        uint64
	txSampling     bool
	txStartChecked bool
	txIndex        uint8
	txValue        byte
	decoded        []byte
}

func NewKIMSerial(chip *RRIOT, bitCycles uint64) *KIMSerial {
	if bitCycles == 0 {
		bitCycles = 1
	}
	s := &KIMSerial{chip: chip, BitCycles: bitCycles, DataBits: 8, RXHigh: true, TXHigh: true}
	chip.SetInputA(chip.input[0] | 0x80)
	return s
}

// Queue appends bytes to the incoming eight-data-bit, start/stop framed stream.
// KIM-1 autobaud begins with RUBOUT ($7F), so hosts normally queue it first.
func (s *KIMSerial) Queue(data []byte) { s.pending = append(s.pending, data...) }

// RXIdle reports that the current host byte has completed its stop bits.
func (s *KIMSerial) RXIdle() bool { return len(s.pending) == 0 && len(s.rxFrames) == 0 }

// DrainEdges and DrainDecoded return independent events since the last drain.
func (s *KIMSerial) DrainEdges() []SerialEdge {
	out := append([]SerialEdge(nil), s.edges...)
	s.edges = nil
	return out
}
func (s *KIMSerial) DrainDecoded() []byte {
	out := append([]byte(nil), s.decoded...)
	s.decoded = nil
	return out
}

func (s *KIMSerial) driveRX(high bool) {
	if s.RXHigh != high {
		s.RXHigh = high
		s.edges = append(s.edges, SerialEdge{s.Cycle, "RX", high})
	}
	level := high
	if s.RXInvert {
		level = !level
	}
	v := s.chip.input[0]
	if level {
		v |= 0x80
	} else {
		v &^= 0x80
	}
	s.chip.SetInputA(v)
}

func (s *KIMSerial) nextRXFrame() {
	if len(s.pending) == 0 {
		return
	}
	v := s.pending[0]
	s.pending = s.pending[1:]
	s.rxFrames = append(s.rxFrames, false)
	for i := uint8(0); i < s.DataBits; i++ {
		s.rxFrames = append(s.rxFrames, v&(1<<i) != 0)
	}
	s.rxFrames = append(s.rxFrames, true, true)
	s.rxPos = 0
}

func (s *KIMSerial) Tick() {
	s.Cycle++
	// PB0 is the serial output only while its data-direction bit is set.
	tx := true
	if value, mask := s.chip.OutputB(); mask&1 != 0 {
		tx = value&1 != 0
		if s.TXInvert {
			tx = !tx
		}
	}
	if tx != s.TXHigh {
		s.TXHigh = tx
		s.edges = append(s.edges, SerialEdge{s.Cycle, "TX", tx})
		if !tx && !s.txSampling {
			s.txSampling = true
			s.txStartChecked = false
			s.txStart = s.Cycle
			s.txIndex = 0
			s.txValue = 0
		}
	}
	if s.txSampling {
		if !s.txStartChecked && s.Cycle >= s.txStart+s.BitCycles/2 {
			s.txStartChecked = true
			if s.TXHigh {
				s.txSampling = false
			} // reject a brief low glitch
		}
		// Sample at centers of the data bits, following the start bit.
		for s.txSampling && s.txIndex < s.DataBits && s.Cycle >= s.txStart+(uint64(s.txIndex)+1)*s.BitCycles+s.BitCycles/2 {
			if s.TXHigh {
				s.txValue |= 1 << s.txIndex
			}
			s.txIndex++
		}
		if s.txSampling && s.txIndex == s.DataBits && s.Cycle >= s.txStart+(uint64(s.DataBits)+1)*s.BitCycles+s.BitCycles/2 {
			if s.TXHigh {
				s.decoded = append(s.decoded, s.txValue)
			}
			s.txSampling = false
		}
	}
	if len(s.rxFrames) == 0 {
		s.nextRXFrame()
	}
	if len(s.rxFrames) > 0 {
		index := int(s.rxPos / s.BitCycles)
		if index < len(s.rxFrames) {
			s.driveRX(s.rxFrames[index])
		}
		s.rxPos++
		if s.rxPos >= uint64(len(s.rxFrames))*s.BitCycles {
			s.rxFrames = nil
			s.driveRX(true)
		}
	} else {
		s.driveRX(true)
	}
}
