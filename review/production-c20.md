# Production cut 20: KIM-1 motherboard

## Goal

Assemble the 6502-compatible CPU, original discrete RAM and two MOS 6530 RRIOT chips into a machine with the KIM-1 motherboard address decode. Keep device models reusable.

## Implemented

- `NewKIM1(rom002, rom003)` and `NewKIM1FromFiles(path002, path003)` require exactly 1024 bytes per original ROM.
- 1 KB discrete RAM at `$0000-$03FF`.
- 6530-003 I/O at `$1700-$170F`, RAM at `$1780-$17BF`, ROM at `$1800-$1BFF`.
- 6530-002 I/O at `$1740-$174F`, RAM at `$17C0-$17FF`, ROM at `$1C00-$1FFF`.
- 8 KB motherboard selection pattern repeated across the 16-bit address space, making the 6530-002 reset vector at `$FFFC/$FFFD` visible.
- 6530 timers connected to the machine clock once each. PB7 IRQ/NMI wiring is not guessed.
- CPU RESET entry through real vector bus transactions, plus explicit release of RESET.

## Validation

- `make validate GO=../.toolchains/go/bin/go`: pass.
- Tests exercise ROM protection, independent chip RAM and I/O decode, high address mirrors, and CPU reset-vector fetch.
- Actual user-provided `6530-002.bin` and `6530-003.bin`: integration test passed. Reset vector resolved to `$1C22`, and the first 20 instructions executed without an emulator error.
- To reproduce with real ROM files: `M65C02_KIM_ROM002=/absolute/path/6530-002.bin M65C02_KIM_ROM003=/absolute/path/6530-003.bin go test ./internal/machine -run TestKIM1RealMonitorROM -count=1 -v`.
- Full external CPU conformance corpus was not rerun in c20.

## Deliberate limits

- The documented KIM-1 I/O chip-select regions are 64 bytes wide. This cut exposes only the established 16-byte timer/port subregions, and leaves the remaining aliases floating; the ROM test does not establish the behavior of undocumented mirrors.
- There is no front-panel LED/keypad electrical model, serial current-loop circuit, or cassette connection yet, so a usable interactive monitor has **not** been demonstrated.
- The production CPU is a 65C02 implementation, not a separate NMOS 6502 silicon profile. Initial monitor execution does not prove complete KIM-1 software compatibility.
- The KIM motherboard maps address and chip selects at logical-bus granularity; analog power, electrical contention, and SYNC-based hardware single-stepping remain out of scope.
- The original ROM binaries are not included in this overlay.

## Next cut

c21: model KIM-1 keypad and six multiplexed seven-segment displays through 6530 pin wiring; then add the serial console path. Validate boot-to-monitor behavior with real ROMs rather than merely counting executed instructions.

## Recommended commit

`Assemble KIM-1 motherboard with dual 6530s and ROM boot tests`
