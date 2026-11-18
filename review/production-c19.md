# Production cut 19: MOS 6530 RRIOT

## Objective

Add a reusable MOS 6530 chip to the modular machine framework, without embedding KIM-1 motherboard decisions in the device.

## Delivered

- One 1024-byte mask ROM, one 64-byte RAM, and two 8-bit I/O ports with data-direction registers, observable pin state, and external input injection.
- Programmable timer with /1, /8, /64, and /1024 prescalers, underflow status, post-underflow /1 counting, and PB7 interrupt output when that pin is an input.
- Independent ROM, RAM, and register bus targets for mapping by the host backplane.
- Reset and snapshot/restore support for reversible execution.
- Tests for memory, pin direction, timer intervals, timer interrupt behavior, and restoration.

## Evidence and validation

- Consulted *KIM-1 User's Manual* (August 1976), especially Appendix H, MOS 6530 RAM-I/O-Timer description and timer example.
- Consulted MOS MCS6500 hardware manual and supplied 6530 datasheet.
- Verified supplied `6530-002.bin` and `6530-003.bin` are each 1024 bytes. They are input fixtures for a subsequent full KIM-1 machine cut, not redistributed here.
- `make validate GO=../.toolchains/go/bin/go`: passed all local build, vet, and unit tests, including `internal/machine`.
- External Synertek/Dormann corpus runs were **not** repeated for this peripheral-only cut.

## Deliberate boundaries

- Chip-select decode, the actual KIM-1 address map, PB6 chip-selection wiring, display/keyboard multiplexing, and serial current-loop circuitry belong to the motherboard and peripheral wiring in c20.
- This cut exposes a 16-byte register target. The KIM-1 decodes wider `$1700-$173F` and `$1740-$177F` I/O windows; c20 must explicitly handle addressing/mirroring according to the board's decoding, without assuming that all undocumented register aliases are valid.
- PB7 is a chip output. Whether it reaches CPU IRQ or NMI is a separate wiring decision.
- Timer tests cover documented software-visible cases; they do not establish transistor-level electrical or physical-chip waveform equivalence.

## Next cut

c20: assemble a KIM-1 motherboard with both provided ROMs, two 6530s, discrete RAM, chip selects, and monitor bootstrap; add tests for the actual address map and reset vector.

## Suggested commit

`Implement MOS 6530 RRIOT component with timer and port tests`
