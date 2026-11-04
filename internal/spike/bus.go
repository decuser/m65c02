package spike

// RAM is a deliberately simple deterministic 64K bus for spike tests.
// Events includes dummy accesses and writes. It is not part of CPU state.
type RAM struct {
	Bytes  [65536]byte
	Events []Request
	Reads  uint64
	Writes uint64
}

func (m *RAM) Transfer(r Request) Result {
	m.Events = append(m.Events, r)
	if r.Write {
		m.Bytes[r.Addr] = r.Data
		m.Writes++
		return Result{Ready: true}
	}
	m.Reads++
	return Result{Data: m.Bytes[r.Addr], Ready: true}
}
