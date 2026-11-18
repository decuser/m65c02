package cpu

import "testing"

type lineRAM struct {
	RAM
	irq, nmi, reset bool
}

func (m *lineRAM) Transfer(r Request) Response {
	v := m.RAM.Transfer(r)
	v.IRQ, v.NMI, v.Reset = m.irq, m.nmi, m.reset
	return v
}

func TestIRQAndRTI(t *testing.T) {
	m := &lineRAM{}
	m.Data[0x200], m.Data[0x201] = 0xea, 0x40 // NOP followed by RTI location
	m.Data[0x400] = 0x40                      // RTI handler
	m.Data[0xfffe], m.Data[0xffff] = 0x00, 0x04
	c := New(0x200)
	m.irq = true
	if err := c.StepInstruction(m); err != nil {
		t.Fatal(err)
	}
	if c.View().Control != 1 {
		t.Fatalf("IRQ not selected: %+v", c.View())
	}
	m.irq = false
	if err := c.StepInstruction(m); err != nil {
		t.Fatal(err)
	}
	eqAddresses(t, &m.RAM, 0x200, 0x201, 0x201, 0x201, 0x1ff, 0x1fe, 0x1fd, 0xfffe, 0xffff)
	s := c.View()
	if s.PC != 0x400 || s.S != 0xfc || s.P&FlagI == 0 {
		t.Fatalf("IRQ state %+v", s)
	}
	if m.Data[0x1ff] != 2 || m.Data[0x1fe] != 1 || m.Data[0x1fd]&FlagB != 0 {
		t.Fatalf("IRQ stack: %x", m.Data[0x1fd:0x200])
	}
	if err := c.StepInstruction(m); err != nil {
		t.Fatal(err)
	}
	if c.View().PC != 0x201 || c.View().S != 0xff {
		t.Fatalf("RTI state %+v", c.View())
	}
}

func TestNMIEdgeAndPriority(t *testing.T) {
	m := &lineRAM{}
	m.Data[0x200] = 0xea
	m.Data[0xfffa], m.Data[0xfffb] = 0x00, 0x05
	c := New(0x200)
	c.state.P |= FlagI
	m.irq, m.nmi = true, true
	if err := c.StepInstruction(m); err != nil {
		t.Fatal(err)
	}
	if c.View().Control != 2 {
		t.Fatalf("NMI priority %+v", c.View())
	}
	if err := c.StepInstruction(m); err != nil {
		t.Fatal(err)
	}
	if c.View().PC != 0x500 || c.View().NMIPending {
		t.Fatalf("NMI state %+v", c.View())
	}
}

func TestExternalReset(t *testing.T) {
	m := &lineRAM{}
	m.Data[0xfffc], m.Data[0xfffd] = 0x34, 0x12
	c := New(0x200)
	m.reset = true
	c.SetLines(false, false, true)
	if err := c.StepInstruction(m); err != nil {
		t.Fatal(err)
	}
	eqAddresses(t, &m.RAM, 0x200, 0x200, 0x1ff, 0x1fe, 0x1fd, 0xfffc, 0xfffd)
	s := c.View()
	if s.PC != 0x1234 || s.S != 0xfc || s.P&FlagI == 0 || s.P&FlagD != 0 {
		t.Fatalf("RESET %+v", s)
	}
	for _, r := range m.Events {
		if r.Write {
			t.Fatalf("reset wrote bus: %+v", r)
		}
	}
}
