// Package spike contains a disposable execution-schema experiment.
// It is not the production 65C02 implementation.
package spike

import (
	"fmt"
	"strings"
)

type Phase uint8

const (
	PreBus Phase = iota
	Resolve
	PostBus
)

const (
	FlagZ byte = 0x02
	FlagD byte = 0x08
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
	Offset              byte
	BranchTaken         bool
	AD                  uint16
	DO                  byte
	PendingValid        bool
	Selected            cycle
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
	c := &CPU{State: State{PC: pc, S: 0xff, P: FlagU, Phase: PreBus}}
	_ = c.prepareBoundary()
	return c
}

func (c *CPU) Snapshot() State { return c.State }
func (c *CPU) Restore(s State) { c.State = s }
func (c *CPU) AtInstructionBoundary() bool {
	return c.State.Step == 0 && c.State.Phase == Resolve && c.State.PendingValid && c.State.Pending.Sync
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
	case 0x9d, 0xfe, 0xbd: // STA abs,X; INC abs,X; LDA abs,X
		switch s.Step {
		case 1:
			return cycle{addr: s.PC, pre: []string{"T: PC -> AD"}, post: []string{"T: DL -> LO", "INC: PC"}}, nil
		case 2:
			return cycle{addr: s.PC, pre: []string{"T: PC -> AD"}, post: []string{"T: DL -> HI", "CALC: BASE + X", "INC: PC"}}, nil
		case 3:
			if s.IR == 0xbd && s.Addr&0xff00 == s.Base&0xff00 {
				return cycle{addr: s.Addr, pre: []string{"T: EA -> AD"}, post: []string{"T: DL -> A", "SET: NZ(A)"}}, nil
			}
			return cycle{addr: s.PC - 1, pre: []string{"T: INDEXED DUMMY -> AD"}, post: []string{"T: DUMMY -> DISCARD"}}, nil
		case 4:
			if s.IR == 0xbd {
				return cycle{addr: s.Addr, pre: []string{"T: EA -> AD"}, post: []string{"T: DL -> A", "SET: NZ(A)"}}, nil
			}
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
	case 0x03: // Synertek single-cycle NOP, handled in fetch completion
	case 0xea: // ordinary two-cycle NOP
		if s.Step == 1 {
			return cycle{addr: s.PC, pre: []string{"T: PC -> AD"}, post: []string{"T: DUMMY -> DISCARD"}}, nil
		}
	case 0x4c: // JMP abs
		switch s.Step {
		case 1:
			return cycle{addr: s.PC, pre: []string{"T: PC -> AD"}, post: []string{"T: DL -> LO", "INC: PC"}}, nil
		case 2:
			return cycle{addr: s.PC, pre: []string{"T: PC -> AD"}, post: []string{"T: DL -> HI", "T: VECTOR -> PC"}}, nil
		}
	case 0xd0: // BNE (branch if Z clear)
		switch s.Step {
		case 1:
			return cycle{addr: s.PC, pre: []string{"T: PC -> AD"}, post: []string{"T: DL -> OFFSET", "INC: PC", "CALC: BRANCH"}}, nil
		case 2:
			return cycle{addr: s.PC, pre: []string{"T: PC -> AD (BRANCH DUMMY)"}, post: []string{"T: DUMMY -> DISCARD", "T: BRANCH -> PC (IF SAME PAGE)"}}, nil
		case 3:
			return cycle{addr: (s.PC & 0xff00) | (s.Addr & 0xff), pre: []string{"T: BRANCH CROSS DUMMY -> AD"}, post: []string{"T: DUMMY -> DISCARD", "T: BRANCH -> PC"}}, nil
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
		if s.IR != 0xa9 && s.IR != 0x9d && s.IR != 0xfe && s.IR != 0xbd && s.IR != 0x00 && s.IR != 0x03 && s.IR != 0xea && s.IR != 0x4c && s.IR != 0xd0 {
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
	case "T: DL -> OFFSET":
		s.Offset = s.DL
	case "CALC: BRANCH":
		s.BranchTaken = s.P&FlagZ == 0
		s.Base = s.PC
		s.Addr = uint16(int32(s.PC) + int32(int8(s.Offset)))
	case "T: BRANCH -> PC (IF SAME PAGE)":
		if s.Addr&0xff00 == s.Base&0xff00 {
			s.PC = s.Addr
		}
	case "T: BRANCH -> PC":
		s.PC = s.Addr
	case "T: DL -> OPERAND":
		s.Operand = s.DL
	case "INC: OPERAND":
		s.Operand++
	case "SET: NZ(OPERAND)":
		c.setNZ(s.Operand)
	case "DEC: S":
		s.S--
	case "SET: I":
		s.P |= FlagI
		s.P &^= FlagD
	case "T: VECTOR -> PC":
		s.PC = uint16(s.Hi)<<8 | uint16(s.Lo)
	case "T: DUMMY -> DISCARD", "DONE: STA", "DONE: INC":
	default:
		return fmt.Errorf("unknown microoperation %q", name)
	}
	return nil
}

// selectCycle is invoked once per physical bus cycle. The selected entry is
// retained across all post-bus operations, even when PC or S changes.
func (c *CPU) selectCycle() error {
	d, err := c.describe()
	if err != nil {
		return err
	}
	c.State.Selected = d
	c.State.AD = 0
	c.State.DO = 0
	return nil
}

// prepare executes a real pre-bus transfer into AD/DO and latches the request.
// The example's transfer vocabulary is intentionally still string-coded;
// production will replace it with an operation enum.
func (c *CPU) prepare(name string) {
	s := &c.State
	// The descriptive transfer source was frozen into Selected at cycle entry.
	// Each pre-bus operation updates only the latch named by its destination.
	if strings.Contains(name, "-> AD") {
		s.AD = s.Selected.addr
	}
	if strings.Contains(name, "-> DO") {
		s.DO = s.Selected.data
	}
}
func (c *CPU) present() {
	s := &c.State
	d := s.Selected
	s.Pending = Request{Addr: s.AD, Data: s.DO, Write: d.write, Sync: d.sync}
	s.PendingValid = true
	s.Phase = Resolve
	s.Index = 0
}
func (c *CPU) prepareBoundary() error {
	if err := c.selectCycle(); err != nil {
		return err
	}
	for _, op := range c.State.Selected.pre {
		c.prepare(op)
	}
	c.present()
	return nil
}

// StepMicro executes one teaching operation or one physical bus Resolve.
// The next SYNC fetch is prepared without resolving it at instruction boundary.
func (c *CPU) StepMicro(b Bus) (string, error) {
	s := &c.State
	switch s.Phase {
	case PreBus:
		if s.Index == 0 {
			if err := c.selectCycle(); err != nil {
				return "", err
			}
		}
		d := s.Selected
		if int(s.Index) >= len(d.pre) {
			return "", fmt.Errorf("pre index out of range")
		}
		name := d.pre[s.Index]
		c.prepare(name)
		s.Index++
		if int(s.Index) == len(d.pre) {
			c.present()
		}
		return name, nil
	case Resolve:
		if !s.PendingValid {
			return "", fmt.Errorf("no presented bus request")
		}
		s.Resolved = b.Transfer(s.Pending)
		if !s.Resolved.Ready && !s.Pending.Write {
			s.Cycles++
			// This was a real physical read. The next attempt is a new cycle,
			// with no architectural post-bus operations applied.
			s.PendingValid = false
			s.Phase = PreBus
			s.Index = 0
			return "BUS: Resolve (RDY stall)", nil
		}
		s.DL = s.Resolved.Data
		s.Phase = PostBus
		s.Index = 0
		return "BUS: Resolve", nil
	case PostBus:
		d := s.Selected
		if int(s.Index) >= len(d.post) {
			return "", fmt.Errorf("post index out of range")
		}
		name := d.post[s.Index]
		if err := c.commit(name); err != nil {
			return "", err
		}
		s.Index++
		if int(s.Index) == len(d.post) {
			s.Step++
			if completeState(s) {
				s.Step = 0
				s.Instructions++
				s.Cycles++
				s.Index = 0
				s.Phase = PreBus
				s.PendingValid = false
				s.Pending = Request{}
				if err := c.prepareBoundary(); err != nil {
					return "", err
				}
				return name, nil
			}
			s.Cycles++
			s.Index = 0
			s.Phase = PreBus
			s.PendingValid = false
			s.Pending = Request{}
		}
		return name, nil
	default:
		return "", fmt.Errorf("invalid phase %d", s.Phase)
	}
}

func completeState(s *State) bool {
	if s.IR == 0xd0 {
		if !s.BranchTaken {
			return s.Step == 2
		}
		if s.Addr&0xff00 == s.Base&0xff00 {
			return s.Step == 3
		}
		return s.Step == 4
	}
	if s.IR == 0xbd {
		return s.Step == 5 || (s.Step == 4 && s.Addr&0xff00 == s.Base&0xff00)
	}
	return complete(s.IR, s.Step)
}
func complete(op byte, step uint8) bool {
	switch op {
	case 0x03:
		return step == 1
	case 0xea:
		return step == 2
	case 0x4c:
		return step == 3
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
