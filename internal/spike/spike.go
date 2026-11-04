// Package spike contains a disposable execution-schema experiment.
// It is not the production 65C02 implementation.
package spike

import "fmt"

type Phase uint8

const (
	PreBus Phase = iota
	Resolve
	PostBus
)

const (
	FlagZ byte = 0x02
	FlagI byte = 0x04
	FlagB byte = 0x10
	FlagU byte = 0x20
	FlagN byte = 0x80
)

type Request struct {
	Addr  uint16
	Data  byte
	Write bool
	Sync  bool
}

type Result struct {
	Data  byte
	Ready bool
}

type Bus interface{ Transfer(Request) Result }

// State is value-only, including the pending request and its resolved reply.
// Copying it at any phase reproduces the sequencer cursor exactly.
type State struct {
	A, X, Y, S, P, IR   byte
	PC, Addr, Base      uint16
	Lo, Hi, DL, Operand byte
	Step                uint8
	Phase               Phase
	Index               uint8
	Pending             Request
	Resolved            Result
	Cycles              uint64
	Instructions        uint64
}

type CPU struct{ State State }

func New(pc uint16) *CPU {
	return &CPU{State: State{PC: pc, S: 0xff, P: FlagU, Phase: PreBus}}
}

func (c *CPU) Snapshot() State { return c.State }
func (c *CPU) Restore(s State) { c.State = s }
func (c *CPU) AtInstructionBoundary() bool {
	return c.State.Step == 0 && c.State.Phase == PreBus && c.State.Index == 0
}

// Each bus cycle is described by named explanatory operations around a single
// resolved transaction. These labels are not a transistor-level claim.
type cycle struct {
	addr        uint16
	data        byte
	write, sync bool
	pre         []string
	post        []string
}

func (c *CPU) describe() (cycle, error) {
	s := &c.State
	if s.Step == 0 {
		return cycle{addr: s.PC, sync: true, pre: []string{"T: PC -> AD"}, post: []string{"T: DL -> IR", "INC: PC"}}, nil
	}
	switch s.IR {
	case 0xa9: // LDA #
		if s.Step == 1 {
			return cycle{addr: s.PC, pre: []string{"T: PC -> AD"}, post: []string{"T: DL -> A", "SET: NZ(A)", "INC: PC"}}, nil
		}
	case 0x9d, 0xfe: // STA abs,X; INC abs,X
		switch s.Step {
		case 1:
			return cycle{addr: s.PC, pre: []string{"T: PC -> AD"}, post: []string{"T: DL -> LO", "INC: PC"}}, nil
		case 2:
			return cycle{addr: s.PC, pre: []string{"T: PC -> AD"}, post: []string{"T: DL -> HI", "CALC: BASE + X", "INC: PC"}}, nil
		case 3:
			return cycle{addr: (s.Base & 0xff00) | ((uint16(s.Lo) + uint16(s.X)) & 0xff), pre: []string{"T: INDEXED DUMMY -> AD"}, post: []string{"T: DUMMY -> DISCARD"}}, nil
		case 4:
			if s.IR == 0x9d {
				return cycle{addr: s.Addr, data: s.A, write: true, pre: []string{"T: EA -> AD", "T: A -> DO"}, post: []string{"DONE: STA"}}, nil
			}
			return cycle{addr: s.Addr, pre: []string{"T: EA -> AD"}, post: []string{"T: DL -> OPERAND"}}, nil
		case 5:
			if s.IR == 0xfe {
				return cycle{addr: s.Addr, pre: []string{"T: EA -> AD (RMW DUMMY)"}, post: []string{"INC: OPERAND", "SET: NZ(OPERAND)"}}, nil
			}
		case 6:
			if s.IR == 0xfe {
				return cycle{addr: s.Addr, data: s.Operand, write: true, pre: []string{"T: EA -> AD", "T: OPERAND -> DO"}, post: []string{"DONE: INC"}}, nil
			}
		}
	case 0x00: // BRK
		switch s.Step {
		case 1:
			return cycle{addr: s.PC, pre: []string{"T: PC -> AD (SIGNATURE)"}, post: []string{"INC: PC"}}, nil
		case 2:
			return cycle{addr: 0x100 | uint16(s.S), data: byte(s.PC >> 8), write: true, pre: []string{"T: S -> AD", "T: PCH -> DO"}, post: []string{"DEC: S"}}, nil
		case 3:
			return cycle{addr: 0x100 | uint16(s.S), data: byte(s.PC), write: true, pre: []string{"T: S -> AD", "T: PCL -> DO"}, post: []string{"DEC: S"}}, nil
		case 4:
			return cycle{addr: 0x100 | uint16(s.S), data: s.P | FlagB | FlagU, write: true, pre: []string{"T: S -> AD", "T: P -> DO"}, post: []string{"DEC: S", "SET: I"}}, nil
		case 5:
			return cycle{addr: 0xfffe, pre: []string{"T: IRQ VECTOR LOW -> AD"}, post: []string{"T: DL -> LO"}}, nil
		case 6:
			return cycle{addr: 0xffff, pre: []string{"T: IRQ VECTOR HIGH -> AD"}, post: []string{"T: DL -> HI", "T: VECTOR -> PC"}}, nil
		}
	}
	return cycle{}, fmt.Errorf("unsupported instruction %02X at cycle %d", s.IR, s.Step)
}

func (c *CPU) setNZ(v byte) {
	c.State.P &^= FlagN | FlagZ
	if v == 0 {
		c.State.P |= FlagZ
	}
	if v&0x80 != 0 {
		c.State.P |= FlagN
	}
}

func (c *CPU) commit(name string) error {
	s := &c.State
	switch name {
	case "T: DL -> IR":
		s.IR = s.DL
		if s.IR != 0xa9 && s.IR != 0x9d && s.IR != 0xfe && s.IR != 0x00 {
			return fmt.Errorf("unsupported opcode %02X", s.IR)
		}
	case "INC: PC":
		s.PC++
	case "T: DL -> A":
		s.A = s.DL
	case "SET: NZ(A)":
		c.setNZ(s.A)
	case "T: DL -> LO":
		s.Lo = s.DL
	case "T: DL -> HI":
		s.Hi = s.DL
	case "CALC: BASE + X":
		s.Base = uint16(s.Hi)<<8 | uint16(s.Lo)
		s.Addr = s.Base + uint16(s.X)
	case "T: DL -> OPERAND":
		s.Operand = s.Resolved.Data
	case "INC: OPERAND":
		s.Operand++
	case "SET: NZ(OPERAND)":
		c.setNZ(s.Operand)
	case "DEC: S":
		s.S--
	case "SET: I":
		s.P |= FlagI
	case "T: VECTOR -> PC":
		s.PC = uint16(s.Hi)<<8 | uint16(s.Lo)
	case "T: DUMMY -> DISCARD", "DONE: STA", "DONE: INC":
	default:
		return fmt.Errorf("unknown microoperation %q", name)
	}
	return nil
}

// StepMicro executes exactly one named operation or one bus Resolve.
// A resolved transaction is never replayed when stepping resumes.
func (c *CPU) StepMicro(b Bus) (string, error) {
	s := &c.State
	d, err := c.describe()
	if err != nil {
		return "", err
	}
	switch s.Phase {
	case PreBus:
		name := d.pre[s.Index]
		s.Index++
		if int(s.Index) == len(d.pre) {
			s.Pending = Request{Addr: d.addr, Data: d.data, Write: d.write, Sync: d.sync}
			s.Phase = Resolve
			s.Index = 0
		}
		return name, nil
	case Resolve:
		s.Resolved = b.Transfer(s.Pending)
		if !s.Resolved.Ready && !s.Pending.Write {
			// A stalled read is a completed physical bus cycle, not a
			// teaching pause. Reissue it as a new bus cycle; commit none
			// of this cycle's architectural post-bus operations.
			s.Cycles++
			s.Phase = PreBus
			s.Index = 0
			return "BUS: Resolve (RDY stall)", nil
		}
		s.DL = s.Resolved.Data
		s.Phase = PostBus
		return "BUS: Resolve", nil
	case PostBus:
		name := d.post[s.Index]
		if err := c.commit(name); err != nil {
			return "", err
		}
		s.Index++
		if int(s.Index) == len(d.post) {
			if s.Resolved.Ready || s.Pending.Write {
				s.Step++
				if complete(s.IR, s.Step) {
					s.Step = 0
					s.Instructions++
				}
			}
			s.Cycles++
			s.Index = 0
			s.Phase = PreBus
		}
		return name, nil
	default:
		return "", fmt.Errorf("invalid phase %d", s.Phase)
	}
}

func complete(op byte, step uint8) bool {
	switch op {
	case 0xa9:
		return step == 2
	case 0x9d:
		return step == 5
	case 0xfe, 0x00:
		return step == 7
	}
	return false
}

// Clock, StepCycle and StepInstruction all use the same StepMicro path.
func (c *CPU) Clock(b Bus) error { return c.StepCycle(b) }
func (c *CPU) StepCycle(b Bus) error {
	start := c.State.Cycles
	for c.State.Cycles == start {
		if _, err := c.StepMicro(b); err != nil {
			return err
		}
	}
	return nil
}
func (c *CPU) StepInstruction(b Bus) error {
	start := c.State.Instructions
	for c.State.Instructions == start {
		if _, err := c.StepMicro(b); err != nil {
			return err
		}
	}
	return nil
}
