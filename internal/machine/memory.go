package machine

import (
	"errors"
	"os"
)

// Sized lets the backplane reject address windows larger than a device.
type Sized interface{ Size() uint32 }

// RAM is a byte-addressable device. Administrative loads do not issue bus
// transactions or advance machine time.
type RAM struct{ data []byte }

func NewRAM(size int) (*RAM, error) {
	if size < 1 || size > 65536 {
		return nil, errors.New("RAM size must be 1..65536 bytes")
	}
	return &RAM{data: make([]byte, size)}, nil
}
func (m *RAM) Size() uint32 { return uint32(len(m.data)) }
func (m *RAM) Read(offset uint16) byte {
	if int(offset) >= len(m.data) {
		return 0xff
	}
	return m.data[offset]
}
func (m *RAM) Write(offset uint16, value byte) {
	if int(offset) < len(m.data) {
		m.data[offset] = value
	}
}
func (m *RAM) Load(offset uint16, image []byte) error {
	if int(offset) > len(m.data) || len(image) > len(m.data)-int(offset) {
		return errors.New("RAM image exceeds device capacity")
	}
	copy(m.data[int(offset):], image)
	return nil
}
func (m *RAM) LoadFile(offset uint16, path string) error {
	image, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return m.Load(offset, image)
}

// Snapshot returns an independent copy. Restore rejects mismatched sizes.
func (m *RAM) Snapshot() []byte { return append([]byte(nil), m.data...) }
func (m *RAM) Restore(image []byte) error {
	if len(image) != len(m.data) {
		return errors.New("RAM snapshot size mismatch")
	}
	copy(m.data, image)
	return nil
}
func (m *RAM) SaveFile(path string) error { return os.WriteFile(path, m.data, 0600) }

// ROM owns a private copy of its initial image. Guest writes have no effect.
// Creating new ROM represents replacing a chip, not rewriting it on the bus.
type ROM struct{ data []byte }

func NewROM(image []byte) (*ROM, error) {
	if len(image) < 1 || len(image) > 65536 {
		return nil, errors.New("ROM image must be 1..65536 bytes")
	}
	return &ROM{data: append([]byte(nil), image...)}, nil
}
func ROMFromFile(path string) (*ROM, error) {
	image, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewROM(image)
}
func (m *ROM) Size() uint32 { return uint32(len(m.data)) }
func (m *ROM) Read(offset uint16) byte {
	if int(offset) >= len(m.data) {
		return 0xff
	}
	return m.data[offset]
}
func (m *ROM) Write(offset uint16, value byte) {} // electrical write ignored
func (m *ROM) Image() []byte                   { return append([]byte(nil), m.data...) }
