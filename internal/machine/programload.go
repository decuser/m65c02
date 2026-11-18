package machine

import (
	"fmt"
	"os"
)

// Segment represents bytes with an absolute load address.
type Segment struct {
	Address uint16
	Data    []byte
}

// ParseKIMTape reads the checksummed MOS paper-tape format used by the
// original KIM-1 monitor. NUL leader bytes between records are accepted.
func ParseKIMTape(data []byte) ([]Segment, error) {
	var segments []Segment
	pos, records := 0, 0
	read := func(sum *uint32) (byte, error) {
		if pos+2 > len(data) {
			return 0, fmt.Errorf("truncated hex byte at offset %d", pos)
		}
		var v byte
		for i := 0; i < 2; i++ {
			c := data[pos+i]
			var n byte
			switch {
			case c >= '0' && c <= '9':
				n = c - '0'
			case c >= 'A' && c <= 'F':
				n = c - 'A' + 10
			case c >= 'a' && c <= 'f':
				n = c - 'a' + 10
			default:
				return 0, fmt.Errorf("invalid hex at offset %d", pos+i)
			}
			v = v<<4 | n
		}
		pos += 2
		*sum += uint32(v)
		return v, nil
	}
	for {
		for pos < len(data) && data[pos] != ';' {
			pos++
		}
		if pos == len(data) {
			return nil, fmt.Errorf("KIM tape has no terminal record")
		}
		pos++
		var sum uint32
		count, err := read(&sum)
		if err != nil {
			return nil, err
		}
		hi, err := read(&sum)
		if err != nil {
			return nil, err
		}
		lo, err := read(&sum)
		if err != nil {
			return nil, err
		}
		address := uint16(hi)<<8 | uint16(lo)
		bytes := make([]byte, int(count))
		for i := range bytes {
			bytes[i], err = read(&sum)
			if err != nil {
				return nil, err
			}
		}
		var ignored uint32
		checkHi, err := read(&ignored)
		if err != nil {
			return nil, err
		}
		checkLo, err := read(&ignored)
		if err != nil {
			return nil, err
		}
		checksum := uint16(checkHi)<<8 | uint16(checkLo)
		// KIM's terminating ;00 record puts the data-record count in
		// the address. Historical tapes vary in the trailing field:
		// some repeat that count, others use a distinct value.
		if count != 0 && uint16(sum) != checksum {
			return nil, fmt.Errorf("KIM tape checksum at record %d: got %04X, want %04X", records+1, uint16(checkHi)<<8|uint16(checkLo), uint16(sum))
		}
		if count == 0 {
			if int(address) != records {
				return nil, fmt.Errorf("KIM tape trailer reports %d records, read %d", address, records)
			}
			return segments, nil
		}
		if uint32(address)+uint32(count) > 0x10000 {
			return nil, fmt.Errorf("KIM tape record crosses $FFFF")
		}
		segments = append(segments, Segment{Address: address, Data: bytes})
		records++
	}
}

// LoadKIMTape deposits a tape into expanded RAM without generating CPU bus
// cycles. Validate all addresses before writing to avoid partial loads.
func (k *KIM1) LoadKIMTape(data []byte) (int, error) {
	segments, err := ParseKIMTape(data)
	if err != nil {
		return 0, err
	}
	if k.Expansion == nil {
		return 0, fmt.Errorf("KIM tape loading requires expansion RAM")
	}
	for _, seg := range segments {
		start := uint32(seg.Address)
		if start < 0x2000 || start+uint32(len(seg.Data)) > 0x6000 {
			return 0, fmt.Errorf("KIM tape record $%04X of %d bytes outside BASIC RAM", seg.Address, len(seg.Data))
		}
	}
	total := 0
	for _, seg := range segments {
		if err := k.Expansion.Load(seg.Address-0x2000, seg.Data); err != nil {
			return 0, err
		}
		total += len(seg.Data)
	}
	return total, nil
}

func (k *KIM1) LoadKIMTapeFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return k.LoadKIMTape(data)
}
