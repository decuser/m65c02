package cpu

// The extended instruction set shares the CPU's physical-cycle cursor and
// Present/Resolve/Commit path. No instruction executes outside StepMicro.
var extendedRead = cycle([]opCode{opExtPrepare}, []opCode{opExtCommit}, false, false, "Instruction bus read")
var extendedWrite = cycle([]opCode{opExtPrepare}, []opCode{opExtCommit}, true, false, "Instruction bus write")

func isStore(k instrKind) bool { return k == insSTA || k == insSTX || k == insSTY || k == insSTZ }
func isRMW(k instrKind) bool {
	switch k {
	case insASL, insLSR, insROL, insROR, insINC, insDEC, insTSB, insTRB:
		return true
	}
	return false
}
func (c *CPU) extWriteCycle() bool {
	s := &c.state
	d := instructionTable[s.IR]
	st := s.Stage
	if d.kind == insJSR {
		return st == 3 || st == 4
	}
	switch d.mode {
	case modeZp:
		return (isStore(d.kind) && st == 2) || (isRMW(d.kind) && st == 4)
	case modeAbs:
		return (isStore(d.kind) && st == 3) || (isRMW(d.kind) && st == 5)
	case modeZpx, modeZpy:
		return isStore(d.kind) && st == 3 || isRMW(d.kind) && st == 5
	case modeAbx, modeAby:
		return isStore(d.kind) && st == 4 || isRMW(d.kind) && st == c.rmwFinalStage()
	case modeIx:
		return isStore(d.kind) && st == 5
	case modeIy:
		return isStore(d.kind) && st == 5
	case modeIz:
		return isStore(d.kind) && st == 4
	case modeImpl:
		return (d.kind == insPHA || d.kind == insPHP || d.kind == insPHX || d.kind == insPHY) && st == 2
	}
	switch d.kind {
	case insJSR:
		return st == 3 || st == 4
	}
	return false
}
func (c *CPU) extPrepare() {
	s := &c.state
	d := instructionTable[s.IR]
	st := s.Stage
	s.AD = s.PC
	if c.decimalExtra() && d.mode == modeImm && st == 2 {
		if d.kind == insADC {
			s.AD = 0x56
		} else {
			s.AD = 0
		}
	}
	s.DO = 0
	switch d.mode {
	case modeZp:
		if st >= 2 {
			s.AD = s.EA
		}
	case modeZpx, modeZpy:
		if st == 2 {
			s.AD = uint16(s.Offset)
		}
		if st >= 3 {
			s.AD = s.EA
		}
	case modeAbs:
		if st >= 3 {
			s.AD = s.EA
		}
	case modeAbx, modeAby:
		if st == 3 && (isStore(d.kind) || c.rmwPreDummy() || s.Base&0xff00 != s.EA&0xff00) {
			s.AD = s.PC - 1
		} else if st >= 3 {
			s.AD = s.EA
		}
	case modeIx:
		if st == 6 {
			s.AD = s.EA
		}
		switch st {
		case 2:
			s.AD = uint16(s.Offset)
		case 3:
			s.AD = uint16(byte(s.Offset + s.X))
		case 4:
			s.AD = uint16(byte(s.Offset + s.X + 1))
		case 5:
			s.AD = s.EA
		}
	case modeIy, modeIz:
		if st >= 5 {
			s.AD = s.EA
		}
		switch st {
		case 2:
			s.AD = uint16(s.Offset)
		case 3:
			s.AD = uint16(byte(s.Offset + 1))
		case 4:
			if d.mode == modeIy && (isStore(d.kind) || s.Base&0xff00 != s.EA&0xff00) {
				s.AD = s.PC - 1
			} else {
				s.AD = s.EA
			}
		case 5:
			s.AD = s.EA
		}
	case modeRel:
		if st == 3 {
			s.AD = (s.Base & 0xff00) | (s.EA & 0xff)
		}
	case modeAbsxno:
		if st == 3 {
			s.AD = s.PC - 1
		}
	case modeInd:
		if st == 3 {
			s.AD = s.EA
		}
		if st == 4 {
			s.AD = (s.EA & 0xff00) | uint16(byte(s.EA+1))
		}
		if st == 5 {
			s.AD = s.EA + 1
		}
	case modeInx:
		if st == 3 {
			s.AD = s.PC - 2
		}
		if st == 4 {
			s.AD = s.EA
		}
		if st == 5 {
			s.AD = s.EA + 1
		}
	case modeImpl:
		if st >= 2 {
			s.AD = 0x100 | uint16(s.S)
		}
		if st == 3 {
			s.AD = 0x100 | uint16(s.S+1)
		}
	}
	switch d.kind {
	case insJSR:
		if st == 5 {
			s.AD = s.PC
		}
		if st == 2 || st == 3 || st == 4 {
			s.AD = 0x100 | uint16(s.S)
		}
		if st == 3 {
			s.DO = byte(s.PC >> 8)
		}
		if st == 4 {
			s.DO = byte(s.PC)
		}
	case insRTS, insRTI:
		if st == 2 {
			s.AD = 0x100 | uint16(s.S)
		}
		if st >= 3 && st <= 4 {
			s.AD = 0x100 | uint16(s.S+1)
		}
		if d.kind == insRTI && st == 5 {
			s.AD = 0x100 | uint16(s.S+1)
		}
		if d.kind == insRTS && st == 5 {
			s.AD = s.PC
		}
	}
	if c.extWriteCycle() {
		switch d.kind {
		case insJSR: // already prepared
		case insPHA:
			s.DO = s.A
		case insPHP:
			s.DO = s.P | FlagB | FlagU
		case insPHX:
			s.DO = s.X
		case insPHY:
			s.DO = s.Y
		case insSTA:
			s.DO = s.A
		case insSTX:
			s.DO = s.X
		case insSTY:
			s.DO = s.Y
		case insSTZ:
			s.DO = 0
		default:
			s.DO = s.Operand
		}
	}
}
func (c *CPU) extCommit() {
	s := &c.state
	d := instructionTable[s.IR]
	st := s.Stage
	switch d.kind {
	case insJSR:
		switch st {
		case 1:
			s.Lo = s.DL
			s.PC++
		case 3, 4:
			s.S--
		case 5:
			s.Hi = s.DL
			s.PC = uint16(s.Hi)<<8 | uint16(s.Lo)
		}
		return
	case insRTS:
		switch st {
		case 3:
			s.S++
			s.Lo = s.DL
		case 4:
			s.S++
			s.Hi = s.DL
			s.PC = uint16(s.Hi)<<8 | uint16(s.Lo)
		case 5:
			s.PC++
		}
		return
	case insRTI:
		switch st {
		case 3:
			s.S++
			s.P = (s.DL &^ FlagB) | FlagU
		case 4:
			s.S++
			s.Lo = s.DL
		case 5:
			s.S++
			s.Hi = s.DL
			s.PC = uint16(s.Hi)<<8 | uint16(s.Lo)
		}
		return
	}
	if d.mode == modeImpl {
		switch d.kind {
		case insPHA, insPHP, insPHX, insPHY:
			if st == 2 {
				s.S--
			}
		case insPLA, insPLP, insPLX, insPLY:
			if st == 3 {
				s.S++
				switch d.kind {
				case insPLA:
					s.A = s.DL
					c.setNZ(s.A)
				case insPLP:
					s.P = (s.DL &^ FlagB) | FlagU
				case insPLX:
					s.X = s.DL
					c.setNZ(s.X)
				case insPLY:
					s.Y = s.DL
					c.setNZ(s.Y)
				}
			}
		default:
			if st == 1 {
				c.implied(d.kind)
			}
		}
		return
	}
	switch d.mode {
	case modeOne:
		return
	case modeImm, modeTwo:
		if st == 1 {
			if d.mode == modeImm {
				c.readEffect(d.kind, s.DL)
			}
			s.PC++
		}
	case modeAcc:
		if st == 1 {
			s.A = c.modify(d.kind, s.A)
		}
	case modeZp:
		if st == 1 {
			s.Offset = s.DL
			s.EA = uint16(s.Offset)
			s.PC++
		} else if st >= 2 {
			c.memoryCycle(d.kind, st, 2)
		}
	case modeZpx, modeZpy:
		if st == 1 {
			s.Offset = s.DL
			s.PC++
		} else if st == 2 {
			idx := s.X
			if d.mode == modeZpy {
				idx = s.Y
			}
			s.EA = uint16(byte(s.Offset + idx))
		} else if st >= 3 {
			c.memoryCycle(d.kind, st, 3)
		}
	case modeAbs, modeAbx, modeAby:
		if st == 1 {
			s.Lo = s.DL
			s.PC++
		} else if st == 2 {
			s.Hi = s.DL
			s.PC++
			s.Base = uint16(s.Hi)<<8 | uint16(s.Lo)
			s.EA = s.Base
			if d.mode == modeAbx {
				s.EA += uint16(s.X)
			}
			if d.mode == modeAby {
				s.EA += uint16(s.Y)
			}
		} else if st >= 3 {
			first := uint8(3)
			if d.mode != modeAbs && (isStore(d.kind) || c.rmwPreDummy() || s.Base&0xff00 != s.EA&0xff00) {
				first = 4
			}
			if st >= first {
				c.memoryCycle(d.kind, st, first)
			}
		}
	case modeIx:
		switch st {
		case 1:
			s.Offset = s.DL
			s.PC++
		case 3:
			s.Lo = s.DL
		case 4:
			s.Hi = s.DL
			s.EA = uint16(s.Hi)<<8 | uint16(s.Lo)
		case 5:
			c.memoryCycle(d.kind, st, 5)
		}
	case modeIy, modeIz:
		switch st {
		case 1:
			s.Offset = s.DL
			s.PC++
		case 2:
			s.Lo = s.DL
		case 3:
			s.Hi = s.DL
			s.Base = uint16(s.Hi)<<8 | uint16(s.Lo)
			s.EA = s.Base
			if d.mode == modeIy {
				s.EA += uint16(s.Y)
			}
		case 4, 5:
			first := uint8(4)
			if d.mode == modeIy && (isStore(d.kind) || s.Base&0xff00 != s.EA&0xff00) {
				first = 5
			}
			if st >= first {
				c.memoryCycle(d.kind, st, first)
			}
		}
	case modeRel:
		switch st {
		case 1:
			s.Offset = s.DL
			s.PC++
			s.Base = s.PC
			s.EA = uint16(int32(s.PC) + int32(int8(s.Offset)))
			s.Taken = c.branchTaken(d.kind)
		case 2:
			if s.Base&0xff00 == s.EA&0xff00 {
				s.PC = s.EA
			}
		case 3:
			s.PC = s.EA
		}
	case modeAbsnop, modeAbsxno:
		if st == 1 || st == 2 {
			s.PC++
		}
	case modeInd, modeInx:
		switch st {
		case 1:
			s.Lo = s.DL
			s.PC++
		case 2:
			s.Hi = s.DL
			s.PC++
			s.EA = uint16(s.Hi)<<8 | uint16(s.Lo)
			if d.mode == modeInx {
				s.EA += uint16(s.X)
			}
		case 3:
			if d.mode == modeInd {
				s.Lo = s.DL
			}
		case 4:
			if d.mode == modeInd {
				s.Hi = s.DL
			}
			if d.mode == modeInx {
				s.Lo = s.DL
			}
		case 5:
			if d.mode == modeInd {
				s.Hi = s.DL
				s.PC = uint16(s.Hi)<<8 | uint16(s.Lo)
			} else {
				s.Hi = s.DL
				s.PC = uint16(s.Hi)<<8 | uint16(s.Lo)
			}
		}
	}
}
func (c *CPU) memoryCycle(k instrKind, stage, first uint8) {
	s := &c.state
	if isStore(k) {
		return
	}
	if isRMW(k) {
		if stage == first {
			s.Operand = s.DL
		} else if stage == first+1 {
			s.Operand = c.modify(k, s.Operand)
		}
		return
	}
	if stage == first {
		c.readEffect(k, s.DL)
	}
}
func (c *CPU) extDone() bool {
	s := &c.state
	d := instructionTable[s.IR]
	st := s.Stage
	switch d.kind {
	case insJSR, insRTS, insRTI:
		return st == 5
	}
	switch d.mode {
	case modeOne:
		return st == 0
	case modeImpl:
		if d.kind == insPHA || d.kind == insPHP || d.kind == insPHX || d.kind == insPHY {
			return st == 2
		}
		if d.kind == insPLA || d.kind == insPLP || d.kind == insPLX || d.kind == insPLY {
			return st == 3
		}
		return st == 1
	case modeImm, modeTwo, modeAcc:
		if c.decimalExtra() {
			return st == 2
		}
		return st == 1
	case modeZp:
		if c.decimalExtra() {
			return st == 3
		}
		return st == 2+boolNum(isRMW(d.kind))*2
	case modeZpx, modeZpy:
		if c.decimalExtra() {
			return st == 4
		}
		return st == 3+boolNum(isRMW(d.kind))*2
	case modeAbs:
		if c.decimalExtra() {
			return st == 4
		}
		return st == 3+boolNum(isRMW(d.kind))*2
	case modeAbx, modeAby:
		if isRMW(d.kind) {
			return st == c.rmwFinalStage()
		}
		if isStore(d.kind) || s.Base&0xff00 != s.EA&0xff00 {
			if c.decimalExtra() {
				return st == 5
			}
			return st == 4
		}
		if c.decimalExtra() {
			return st == 4
		}
		return st == 3
	case modeIx:
		if c.decimalExtra() {
			return st == 6
		}
		return st == 5
	case modeIy:
		if isStore(d.kind) || s.Base&0xff00 != s.EA&0xff00 {
			if c.decimalExtra() {
				return st == 6
			}
			return st == 5
		}
		if c.decimalExtra() {
			return st == 5
		}
		return st == 4
	case modeIz:
		if c.decimalExtra() {
			return st == 5
		}
		return st == 4
	case modeRel:
		if st == 1 && !s.Taken {
			return true
		}
		if st == 2 && s.Base&0xff00 == s.EA&0xff00 {
			return true
		}
		return st == 3
	case modeAbsnop:
		return st == 2
	case modeAbsxno:
		return st == 3
	case modeInd, modeInx:
		return st == 5
	}
	return false
}
func boolNum(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}
func (c *CPU) branchTaken(k instrKind) bool {
	p := c.state.P
	switch k {
	case insBRA:
		return true
	case insBNE:
		return p&FlagZ == 0
	case insBEQ:
		return p&FlagZ != 0
	case insBCC:
		return p&FlagC == 0
	case insBCS:
		return p&FlagC != 0
	case insBMI:
		return p&FlagN != 0
	case insBPL:
		return p&FlagN == 0
	case insBVC:
		return p&FlagV == 0
	case insBVS:
		return p&FlagV != 0
	}
	return false
}
func (c *CPU) implied(k instrKind) {
	s := &c.state
	switch k {
	case insCLC:
		s.P &^= FlagC
	case insSEC:
		s.P |= FlagC
	case insCLI:
		s.P &^= FlagI
	case insSEI:
		s.P |= FlagI
	case insCLD:
		s.P &^= FlagD
	case insSED:
		s.P |= FlagD
	case insCLV:
		s.P &^= FlagV
	case insTAX:
		s.X = s.A
		c.setNZ(s.X)
	case insTAY:
		s.Y = s.A
		c.setNZ(s.Y)
	case insTXA:
		s.A = s.X
		c.setNZ(s.A)
	case insTYA:
		s.A = s.Y
		c.setNZ(s.A)
	case insTSX:
		s.X = s.S
		c.setNZ(s.X)
	case insTXS:
		s.S = s.X
	case insINX:
		s.X++
		c.setNZ(s.X)
	case insINY:
		s.Y++
		c.setNZ(s.Y)
	case insDEX:
		s.X--
		c.setNZ(s.X)
	case insDEY:
		s.Y--
		c.setNZ(s.Y)
	}
}
func (c *CPU) readEffect(k instrKind, v byte) {
	s := &c.state
	switch k {
	case insLDA:
		s.A = v
		c.setNZ(v)
	case insLDX:
		s.X = v
		c.setNZ(v)
	case insLDY:
		s.Y = v
		c.setNZ(v)
	case insORA:
		s.A |= v
		c.setNZ(s.A)
	case insAND:
		s.A &= v
		c.setNZ(s.A)
	case insEOR:
		s.A ^= v
		c.setNZ(s.A)
	case insADC:
		c.add(v, false)
	case insSBC:
		c.add(v, true)
	case insCMP:
		c.compare(s.A, v)
	case insCPX:
		c.compare(s.X, v)
	case insCPY:
		c.compare(s.Y, v)
	case insBIT:
		s.P &^= FlagZ
		if s.A&v == 0 {
			s.P |= FlagZ
		}
		if instructionTable[s.IR].mode != modeImm {
			s.P = (s.P &^ (FlagN | FlagV)) | (v & (FlagN | FlagV))
		}
	}
}
func (c *CPU) compare(a, b byte) {
	s := &c.state
	d := a - b
	c.setNZ(d)
	s.P &^= FlagC
	if a >= b {
		s.P |= FlagC
	}
}
func (c *CPU) modify(k instrKind, v byte) byte {
	s := &c.state
	switch k {
	case insINC:
		v++
		c.setNZ(v)
	case insDEC:
		v--
		c.setNZ(v)
	case insASL:
		s.P &^= FlagC
		if v&0x80 != 0 {
			s.P |= FlagC
		}
		v <<= 1
		c.setNZ(v)
	case insLSR:
		s.P &^= FlagC
		if v&1 != 0 {
			s.P |= FlagC
		}
		v >>= 1
		c.setNZ(v)
	case insROL:
		carry := s.P & FlagC
		s.P &^= FlagC
		if v&0x80 != 0 {
			s.P |= FlagC
		}
		v = v<<1 | carry
		c.setNZ(v)
	case insROR:
		carry := s.P & FlagC
		s.P &^= FlagC
		if v&1 != 0 {
			s.P |= FlagC
		}
		v = v>>1 | carry<<7
		c.setNZ(v)
	case insTSB:
		s.P &^= FlagZ
		if s.A&v == 0 {
			s.P |= FlagZ
		}
		v |= s.A
	case insTRB:
		s.P &^= FlagZ
		if s.A&v == 0 {
			s.P |= FlagZ
		}
		v &^= s.A
	}
	return v
}
func (c *CPU) add(v byte, subtract bool) {
	s := &c.state
	a := s.A
	carry := int(s.P & FlagC)
	b := int(v)
	var result byte
	var overflow byte
	var raw int
	if subtract {
		raw = int(a) - b - (1 - carry)
		overflow = byte(raw)
		if s.P&FlagD != 0 {
			negative := raw < 0
			if int(a&15)-int(v&15)-(1-carry) < 0 {
				raw -= 6
			}
			if negative {
				raw -= 0x60
			}
		}
		result = byte(raw)
		s.P &^= FlagV | FlagC
		if int(a)-b-(1-carry) >= 0 {
			s.P |= FlagC
		}
		if ((a ^ v) & (a ^ overflow) & 0x80) != 0 {
			s.P |= FlagV
		}
	} else {
		raw = int(a) + b + carry
		overflow = byte(raw)
		if s.P&FlagD != 0 {
			lo := int(a&15) + int(v&15) + carry
			hi := int(a>>4) + int(v>>4)
			if lo > 9 {
				lo += 6
				hi++
			}
			overflow = byte((hi << 4) | (lo & 15))
			if hi > 9 {
				hi += 6
			}
			result = byte((hi << 4) | (lo & 15))
			if hi > 15 {
				raw = 256
			} else {
				raw = 0
			}
		} else {
			result = byte(raw)
		}
		s.P &^= FlagV | FlagC
		if raw > 255 {
			s.P |= FlagC
		}
		if ((a ^ overflow) & (v ^ overflow) & 0x80) != 0 {
			s.P |= FlagV
		}
	}
	s.A = result
	c.setNZ(result)
}
func (c *CPU) decimalExtra() bool {
	s := &c.state
	k := instructionTable[s.IR].kind
	return s.P&FlagD != 0 && (k == insADC || k == insSBC)
}
func (c *CPU) rmwPreDummy() bool {
	s := &c.state
	k := instructionTable[s.IR].kind
	if k == insASL || k == insROL || k == insLSR || k == insROR {
		return s.Base&0xff00 != s.EA&0xff00
	}
	return isRMW(k)
}
func (c *CPU) rmwFinalStage() uint8 {
	if c.rmwPreDummy() {
		return 6
	}
	return 5
}
