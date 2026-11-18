package cpu

// SetLines supplies asynchronous input levels between machine cycles. The
// transfer response may also supply these levels for bus-driven hosts.
// Control entry is selected at an instruction boundary; mid-instruction RESET
// preemption and electrical pin sampling are intentionally not modeled here.
func (c *CPU) SetLines(irq, nmi, reset bool) {
	s := &c.state
	s.IRQ, s.Reset = irq, reset
	if nmi && !s.NMISeen {
		s.NMIPending = true
	}
	s.NMISeen, s.NMI = nmi, nmi
	if c.AtBoundary() && (reset || s.NMIPending || (irq && s.P&FlagI == 0)) {
		if c.beginControl() {
			_ = c.selectEntry()
		}
	}
}

func (c *CPU) sampleLines(r Response) {
	s := &c.state
	// A transfer response represents the sampled input levels. Hosts that use
	// SetLines should return the same levels from Transfer.
	s.IRQ, s.Reset = r.IRQ, r.Reset
	if r.NMI && !s.NMISeen {
		s.NMIPending = true
	}
	s.NMISeen, s.NMI = r.NMI, r.NMI
}

func (c *CPU) beginControl() bool {
	s := &c.state
	switch {
	case s.Reset:
		s.Control = 3
	case s.NMIPending:
		s.Control = 2
		s.NMIPending = false
	case s.IRQ && s.P&FlagI == 0:
		s.Control = 1
	default:
		return false
	}
	s.Stage = 0
	return true
}

func (c *CPU) controlPrepare() {
	s := &c.state
	s.DO = 0
	switch s.Stage {
	case 0, 1:
		s.AD = s.PC
	case 2, 3, 4:
		s.AD = 0x100 | uint16(s.S)
		if s.Control != 3 {
			switch s.Stage {
			case 2:
				s.DO = byte(s.PC >> 8)
			case 3:
				s.DO = byte(s.PC)
			case 4:
				s.DO = (s.P &^ FlagB) | FlagU
			}
		}
	case 5, 6:
		vector := uint16(0xfffe)
		if s.Control == 2 {
			vector = 0xfffa
		}
		if s.Control == 3 {
			vector = 0xfffc
		}
		s.AD = vector + uint16(s.Stage-5)
	}
}

func (c *CPU) controlCommit() {
	s := &c.state
	switch s.Stage {
	case 2, 3:
		s.S--
	case 4:
		s.S--
		s.P = (s.P | FlagI | FlagU) &^ (FlagD | FlagB)
	case 5:
		s.Lo = s.DL
	case 6:
		s.Hi = s.DL
		s.PC = uint16(s.Hi)<<8 | uint16(s.Lo)
	}
}
