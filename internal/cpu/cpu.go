// Package cpu implements the production, bus-cycle-driven 65C02 core.
// The instruction table is immutable; snapshots contain values only.
package cpu

import "errors"

const (
	FlagN byte = 0x80
	FlagV byte = 0x40
	FlagU byte = 0x20
	FlagB byte = 0x10
	FlagD byte = 0x08
	FlagI byte = 0x04
	FlagZ byte = 0x02
	FlagC byte = 0x01
)

type Phase uint8

const (
	PreBus Phase = iota
	Resolve
	PostBus
)

type Request struct {
	Address     uint16
	DataOut     byte
	Write, Sync bool
}
type Response struct {
	DataIn byte
	Ready  bool
}
type Bus interface{ Transfer(Request) Response }

type State struct {
	A, X, Y, S, P, IR               byte
	PC                              uint16
	AD                              uint16
	DO, DL, Lo, Hi, Operand, Offset byte
	Base, EA                        uint16
	Taken                           bool
	Phase                           Phase
	Stage, Entry, OpIndex           uint8
	Pending                         Request
	PendingValid                    bool
	Reply                           Response
	ReplyValid                      bool
	Cycles, Instructions            uint64
}
type CPU struct{ state State }

func New(pc uint16) *CPU {
	c := &CPU{state: State{PC: pc, S: 0xff, P: FlagU, Phase: PreBus}}
	c.selectEntry()
	return c
}
func (c *CPU) Snapshot() State { return c.state }
func (c *CPU) Restore(s State) error {
	if s.Phase > PostBus || s.Stage > 7 || s.Entry >= entryCount {
		return errors.New("invalid cursor")
	}
	if s.Phase == Resolve && !s.PendingValid {
		return errors.New("resolve without pending request")
	}
	if s.Phase == PostBus && (!s.PendingValid || !s.ReplyValid) {
		return errors.New("postbus without resolved request")
	}
	if s.Phase == PreBus && s.PendingValid {
		return errors.New("prebus has pending request")
	}
	if s.OpIndex > maxOps {
		return errors.New("invalid operation index")
	}
	if s.Stage == 0 && s.Entry != 0 {
		return errors.New("fetch stage with non-fetch entry")
	}
	c.state = s
	return nil
}
func (c *CPU) View() State { return c.state }
func (c *CPU) AtBoundary() bool {
	s := &c.state
	return s.Stage == 0 && s.Phase == Resolve && s.PendingValid && s.Pending.Sync
}

type opCode uint8

const (
	opNone opCode = iota
	opADPC
	opADEA
	opADStack
	opADVectorLo
	opADVectorHi
	opADBranchDummy
	opADPCMinusOne
	opDLA
	opDLIR
	opDLLo
	opDLHi
	opDLOperand
	opDLOffset
	opINCPC
	opDECS
	opSetNZ_A
	opSetNZOperand
	opCalcIndexed
	opINCOperand
	opCalcBranch
	opBranchPCIfSame
	opBranchPC
	opPCFromHiLo
	opSetIClrD
	opDOPCH
	opDOPCL
	opDOP
	opDOA
	opDOOperand
	opExtPrepare
	opExtCommit
)
const maxOps = 5

type micro struct{ code opCode }
type entry struct {
	pre         [maxOps]micro
	post        [maxOps]micro
	npre, npost uint8
	write, sync bool
	annotation  string
}

func ops(a ...opCode) ([maxOps]micro, uint8) {
	var r [maxOps]micro
	for i, v := range a {
		if i >= maxOps {
			panic("too many microops")
		}
		r[i] = micro{v}
	}
	return r, uint8(len(a))
}
func cycle(pre []opCode, post []opCode, write, sync bool, note string) entry {
	var e entry
	e.pre, e.npre = ops(pre...)
	e.post, e.npost = ops(post...)
	e.write, e.sync, e.annotation = write, sync, note
	return e
}

// A cycle's selected descriptor is identified by Entry, not a pointer in State.
// All variable-stage transitions are centralized in next(), not in execution.
var fetch = cycle([]opCode{opADPC}, []opCode{opDLIR, opINCPC}, false, true, "Opcode fetch")
var immediate = cycle([]opCode{opADPC}, []opCode{opDLA, opSetNZ_A, opINCPC}, false, false, "Immediate operand")
var operandLow = cycle([]opCode{opADPC}, []opCode{opDLLo, opINCPC}, false, false, "Address low")
var indexedHigh = cycle([]opCode{opADPC}, []opCode{opDLHi, opCalcIndexed, opINCPC}, false, false, "Address high / index")
var indexedDummy = cycle([]opCode{opADPCMinusOne}, nil, false, false, "Indexed CMOS dummy read (last operand)")
var indexedRead = cycle([]opCode{opADEA}, []opCode{opDLA, opSetNZ_A}, false, false, "Indexed data read")
var indexedWrite = cycle([]opCode{opADEA, opDOA}, nil, true, false, "Indexed data write")
var rmwRead = cycle([]opCode{opADEA}, []opCode{opDLOperand}, false, false, "RMW operand read")
var rmwDummy = cycle([]opCode{opADEA}, []opCode{opINCOperand, opSetNZOperand}, false, false, "RMW dummy read")
var rmwWrite = cycle([]opCode{opADEA, opDOOperand}, nil, true, false, "RMW result write")
var nopDummy = cycle([]opCode{opADPC}, nil, false, false, "NOP dummy read")
var jumpHi = cycle([]opCode{opADPC}, []opCode{opDLHi, opPCFromHiLo}, false, false, "Jump high")
var branchOffset = cycle([]opCode{opADPC}, []opCode{opDLOffset, opINCPC, opCalcBranch}, false, false, "Branch offset")
var branchDummy = cycle([]opCode{opADPC}, []opCode{opBranchPCIfSame}, false, false, "Taken branch dummy")
var branchCross = cycle([]opCode{opADBranchDummy}, []opCode{opBranchPC}, false, false, "Branch page-cross dummy")
var brkSignature = cycle([]opCode{opADPC}, []opCode{opINCPC}, false, false, "BRK signature")
var brkPushHi = cycle([]opCode{opADStack, opDOPCH}, []opCode{opDECS}, true, false, "BRK stack PC high")
var brkPushLo = cycle([]opCode{opADStack, opDOPCL}, []opCode{opDECS}, true, false, "BRK stack PC low")
var brkPushP = cycle([]opCode{opADStack, opDOP}, []opCode{opDECS, opSetIClrD}, true, false, "BRK stack flags")
var brkVectorLo = cycle([]opCode{opADVectorLo}, []opCode{opDLLo}, false, false, "IRQ vector low")
var brkVectorHi = cycle([]opCode{opADVectorHi}, []opCode{opDLHi, opPCFromHiLo}, false, false, "IRQ vector high")

// Each ID names a static immutable cycle entry. Selection is frozen for the
// entire physical cycle, including all post-bus microoperations.
const (
	entryFetch uint8 = iota
	entryImmediate
	entryLow
	entryIndexedHigh
	entryIndexedDummy
	entryIndexedRead
	entryIndexedWrite
	entryRMWRead
	entryRMWDummy
	entryRMWWrite
	entryNOPDummy
	entryJumpHi
	entryBranchOffset
	entryBranchDummy
	entryBranchCross
	entryBRKSignature
	entryBRKPushHi
	entryBRKPushLo
	entryBRKPushP
	entryBRKVectorLo
	entryBRKVectorHi
	entryExtendedRead
	entryExtendedWrite
	entryCount
)

var entries = [entryCount]entry{
	fetch, immediate, operandLow, indexedHigh, indexedDummy, indexedRead,
	indexedWrite, rmwRead, rmwDummy, rmwWrite, nopDummy, jumpHi,
	branchOffset, branchDummy, branchCross, brkSignature, brkPushHi,
	brkPushLo, brkPushP, brkVectorLo, brkVectorHi, extendedRead, extendedWrite,
}

func (c *CPU) chooseEntry() (uint8, error) {
	s := &c.state
	if s.Stage == 0 {
		return entryFetch, nil
	}
	switch s.IR {
	case 0xa9:
		if s.Stage == 1 {
			return entryImmediate, nil
		}
	case 0x9d, 0xfe, 0xbd:
		switch s.Stage {
		case 1:
			return entryLow, nil
		case 2:
			return entryIndexedHigh, nil
		case 3:
			if s.IR == 0xbd && s.EA&0xff00 == s.Base&0xff00 {
				return entryIndexedRead, nil
			}
			return entryIndexedDummy, nil
		case 4:
			if s.IR == 0xbd {
				return entryIndexedRead, nil
			}
			if s.IR == 0x9d {
				return entryIndexedWrite, nil
			}
			return entryRMWRead, nil
		case 5:
			if s.IR == 0xfe {
				return entryRMWDummy, nil
			}
		case 6:
			if s.IR == 0xfe {
				return entryRMWWrite, nil
			}
		}
	case 0xea:
		if s.Stage == 1 {
			return entryNOPDummy, nil
		}
	case 0x4c:
		if s.Stage == 1 {
			return entryLow, nil
		}
		if s.Stage == 2 {
			return entryJumpHi, nil
		}
	case 0xd0:
		switch s.Stage {
		case 1:
			return entryBranchOffset, nil
		case 2:
			return entryBranchDummy, nil
		case 3:
			return entryBranchCross, nil
		}
	case 0x00:
		switch s.Stage {
		case 1:
			return entryBRKSignature, nil
		case 2:
			return entryBRKPushHi, nil
		case 3:
			return entryBRKPushLo, nil
		case 4:
			return entryBRKPushP, nil
		case 5:
			return entryBRKVectorLo, nil
		case 6:
			return entryBRKVectorHi, nil
		}
	}
	if c.extWriteCycle() {
		return entryExtendedWrite, nil
	}
	return entryExtendedRead, nil
}

func (c *CPU) operate(op opCode) {
	s := &c.state
	switch op {
	case opADPC:
		s.AD = s.PC
	case opADPCMinusOne:
		s.AD = s.PC - 1
	case opADEA:
		s.AD = s.EA
	case opADStack:
		s.AD = 0x100 | uint16(s.S)
	case opADVectorLo:
		s.AD = 0xfffe
	case opADVectorHi:
		s.AD = 0xffff
	case opADBranchDummy:
		s.AD = (s.Base & 0xff00) | (s.EA & 0x00ff)
	case opDLA:
		s.A = s.DL
	case opDLIR:
		s.IR = s.DL
	case opDLLo:
		s.Lo = s.DL
	case opDLHi:
		s.Hi = s.DL
	case opDLOperand:
		s.Operand = s.DL
	case opDLOffset:
		s.Offset = s.DL
	case opINCPC:
		s.PC++
	case opDECS:
		s.S--
	case opSetNZ_A:
		c.setNZ(s.A)
	case opSetNZOperand:
		c.setNZ(s.Operand)
	case opCalcIndexed:
		s.Base = uint16(s.Hi)<<8 | uint16(s.Lo)
		s.EA = s.Base + uint16(s.X)
	case opINCOperand:
		s.Operand++
	case opCalcBranch:
		s.Taken = s.P&FlagZ == 0
		s.Base = s.PC
		s.EA = uint16(int32(s.PC) + int32(int8(s.Offset)))
	case opBranchPCIfSame:
		if s.Base&0xff00 == s.EA&0xff00 {
			s.PC = s.EA
		}
	case opBranchPC:
		s.PC = s.EA
	case opPCFromHiLo:
		s.PC = uint16(s.Hi)<<8 | uint16(s.Lo)
	case opSetIClrD:
		s.P |= FlagI
		s.P &^= FlagD
	case opDOPCH:
		s.DO = byte(s.PC >> 8)
	case opDOPCL:
		s.DO = byte(s.PC)
	case opDOP:
		s.DO = s.P | FlagB | FlagU
	case opDOA:
		s.DO = s.A
	case opDOOperand:
		s.DO = s.Operand
	case opExtPrepare:
		c.extPrepare()
	case opExtCommit:
		c.extCommit()
	}
}
func (c *CPU) setNZ(v byte) {
	s := &c.state
	s.P &^= FlagN | FlagZ
	if v == 0 {
		s.P |= FlagZ
	}
	if v&0x80 != 0 {
		s.P |= FlagN
	}
}

// selectEntry prepares a cycle without resolving or mutating architectural state.
// Used for the initial and subsequent instruction SYNC boundaries.
func (c *CPU) selectEntry() error {
	s := &c.state
	id, err := c.chooseEntry()
	if err != nil {
		return err
	}
	s.Entry = id
	e := &entries[id]
	s.AD = 0
	s.DO = 0
	for i := uint8(0); i < e.npre; i++ {
		c.operate(e.pre[i].code)
	}
	s.Pending = Request{Address: s.AD, DataOut: s.DO, Write: e.write, Sync: e.sync}
	s.PendingValid = true
	s.ReplyValid = false
	s.Phase = Resolve
	s.OpIndex = 0
	return nil
}

func (c *CPU) current() *entry { return &entries[c.state.Entry] }
func (c *CPU) endCycle() error {
	s := &c.state
	done := false
	switch s.IR {
	case 0x03:
		done = s.Stage == 0
	case 0xea, 0xa9:
		done = s.Stage == 1
	case 0x4c:
		done = s.Stage == 2
	case 0x9d:
		done = s.Stage == 4
	case 0xfe, 0x00:
		done = s.Stage == 6
	case 0xbd:
		done = (s.Stage == 3 && s.EA&0xff00 == s.Base&0xff00) || s.Stage == 4
	case 0xd0:
		done = (s.Stage == 1 && !s.Taken) || (s.Stage == 2 && s.Base&0xff00 == s.EA&0xff00) || s.Stage == 3
	default:
		done = c.extDone()
	}
	s.Cycles++
	if done {
		s.Instructions++
		s.Stage = 0
	} else {
		s.Stage++
	}
	s.Pending = Request{}
	s.PendingValid = false
	s.ReplyValid = false
	s.OpIndex = 0
	s.Phase = PreBus
	if done {
		return c.selectEntry()
	}
	id, err := c.chooseEntry()
	if err != nil {
		return err
	}
	s.Entry = id
	s.AD = 0
	s.DO = 0
	return nil
}

// MicroEvent is a value. The name is static table text, not executable semantics.
type MicroEvent struct {
	Operation     uint8
	Resolve       bool
	CycleComplete bool
}

func (c *CPU) StepMicro(b Bus) (MicroEvent, error) {
	s := &c.state
	switch s.Phase {
	case PreBus:
		e := c.current()
		if e == nil {
			return MicroEvent{}, errors.New("unsupported opcode")
		}
		if s.OpIndex < e.npre {
			i := s.OpIndex
			c.operate(e.pre[i].code)
			s.OpIndex++
			if s.OpIndex == e.npre {
				s.Pending = Request{Address: s.AD, DataOut: s.DO, Write: e.write, Sync: e.sync}
				s.PendingValid = true
				s.Phase = Resolve
				s.OpIndex = 0
			}
			return MicroEvent{Operation: uint8(e.pre[i].code)}, nil
		}
		return MicroEvent{}, errors.New("invalid prebus index")
	case Resolve:
		if !s.PendingValid {
			return MicroEvent{}, errors.New("no pending bus transaction")
		}
		s.Reply = b.Transfer(s.Pending)
		s.ReplyValid = true
		if !s.Reply.Ready && !s.Pending.Write {
			s.Cycles++
			s.PendingValid = false
			s.ReplyValid = false
			s.Phase = PreBus
			s.OpIndex = 0
			return MicroEvent{Resolve: true, CycleComplete: true}, nil
		}
		s.DL = s.Reply.DataIn
		s.Phase = PostBus
		s.OpIndex = 0
		return MicroEvent{Resolve: true}, nil
	case PostBus:
		e := c.current()
		if e == nil {
			return MicroEvent{}, errors.New("unsupported opcode")
		}
		if s.OpIndex < e.npost {
			i := s.OpIndex
			c.operate(e.post[i].code)
			s.OpIndex++
			if s.OpIndex == e.npost {
				return MicroEvent{Operation: uint8(e.post[i].code), CycleComplete: true}, c.endCycle()
			}
			return MicroEvent{Operation: uint8(e.post[i].code)}, nil
		}
		// Zero-post-op cycles still need an explicit retire step.
		if e.npost == 0 {
			return MicroEvent{CycleComplete: true}, c.endCycle()
		}
		return MicroEvent{}, errors.New("invalid postbus index")
	}
	return MicroEvent{}, errors.New("invalid phase")
}
func (c *CPU) StepCycle(b Bus) error {
	n := c.state.Cycles
	for c.state.Cycles == n {
		if _, err := c.StepMicro(b); err != nil {
			return err
		}
	}
	return nil
}
func (c *CPU) StepInstruction(b Bus) error {
	n := c.state.Instructions
	for c.state.Instructions == n {
		if _, err := c.StepMicro(b); err != nil {
			return err
		}
	}
	return nil
}
func (c *CPU) Annotation() string {
	return entries[c.state.Entry].annotation
}
