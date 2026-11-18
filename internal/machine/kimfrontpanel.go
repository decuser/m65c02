package machine

import "fmt"

// KIMFrontPanel is the original KIM-1's matrix and multiplexed display.
// PB1..PB4 drive a 74145 decoder: outputs 0..2 scan the three key rows,
// output 3 is the TTY-select jumper row, and outputs 4..9 enable six LEDs.
// PA0..PA6 are both keyboard columns A..G and display segments A..G.
// PA7 belongs to the separate serial receive circuit.
type KIMFrontPanel struct {
	chip     *RRIOT
	keys     [3][7]bool
	TTY      bool
	Current  LEDScan
	Dwell    [6][7]uint64
	LastScan [6]byte
}

// LEDScan describes the actual electrical selection in one CPU cycle.
// Digit is -1 when no display is selected; SegmentBits are active-high.
type LEDScan struct {
	Digit       int
	SegmentBits byte
}

var kimKeyNames = [3][7]string{
	{"0", "1", "2", "3", "4", "5", "6"},
	{"7", "8", "9", "A", "B", "C", "D"},
	{"E", "F", "AD", "DA", "+", "GO", "PC"},
}

func NewKIMFrontPanel(chip *RRIOT) *KIMFrontPanel {
	return &KIMFrontPanel{chip: chip, Current: LEDScan{Digit: -1}}
}

// SetKey changes one physical switch contact. RS and ST are dedicated
// hardware lines and are intentionally not members of this matrix.
func (p *KIMFrontPanel) SetKey(name string, pressed bool) error {
	for row := range kimKeyNames {
		for col, key := range kimKeyNames[row] {
			if name == key {
				p.keys[row][col] = pressed
				return nil
			}
		}
	}
	return fmt.Errorf("unknown KIM-1 keypad key %q", name)
}

func (p *KIMFrontPanel) selectedRow() int {
	value, mask := p.chip.OutputB()
	if mask&0x1e != 0x1e {
		return -1 // decoder address inputs not all driven
	}
	return int((value >> 1) & 0x0f)
}

// UpdateInputs resolves closed switches against the selected active-low
// row. Only PA0..PA6 are affected; serial RX on PA7 remains untouched.
func (p *KIMFrontPanel) UpdateInputs() {
	input := p.chip.input[0] | 0x7f // inactive columns have pull-ups
	row := p.selectedRow()
	if row >= 0 && row <= 2 {
		for col, pressed := range p.keys[row] {
			if pressed {
				// The physical KIM keyboard connects matrix columns in
				// descending PA order: leftmost is PA6, rightmost PA0.
				input &^= 1 << uint(6-col)
			}
		}
	} else if row == 3 && p.TTY {
		input &^= 1 // A-21 to A-V jumper is column A / row 3
	}
	p.chip.SetInputA(input)
}

// Tick samples the actual 6530 output pins once per emulated bus cycle.
// Dwell is cumulative lit cycles per segment, not a simulated brightness.
func (p *KIMFrontPanel) Tick() {
	p.Current = LEDScan{Digit: -1}
	row := p.selectedRow()
	if row < 4 || row > 9 {
		return
	}
	value, mask := p.chip.OutputA()
	segments := value & mask & 0x7f
	digit := row - 4
	p.Current = LEDScan{Digit: digit, SegmentBits: segments}
	p.LastScan[digit] = segments
	for bit := 0; bit < 7; bit++ {
		if segments&(1<<uint(bit)) != 0 {
			p.Dwell[digit][bit]++
		}
	}
}
