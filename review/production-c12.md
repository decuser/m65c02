# Production CPU cut c12 (provisional)

Proposed commit: `Add IRQ, NMI, and RESET cycle sequences to production CPU`

## Scope

- Added sampled IRQ, NMI and RESET line signals to the bus response, with an
  explicit `SetLines` API for host-driven input changes.
- Added snapshot-resident interrupt type, NMI edge latch and sequencer position.
- Implemented seven-cycle IRQ/NMI entry with separate vector selection,
  stack pushes, CMOS D clearing and I setting; seven-cycle RESET with dummy
  stack reads and reset-vector fetch; shared the existing phase/step engine.
- Added focused tests for IRQ/RTI, NMI priority and RESET bus traces.

## Validation status

Go 1.24.10 was installed in a workspace-local toolchain because `/usr/bin/go`
is a board-game executable, not the Go compiler. `gofmt` was run on changed Go
files. `make validate GO=../.toolchains/go/bin/go` **passed** its build,
`go vet`, and all local `go test ./...` packages, including new control tests.
The first run exposed an incorrect new test expectation (omitted one NOP bus
cycle), which was fixed before the passing run.

The external Synertek corpus was unavailable (`M65C02_CORPUS` unset), so its
vectors were skipped. Dormann semantic and interrupt binaries were also not
available or executed. This is **local validation only**, not M1 acceptance.

## Known limitations and review questions

- Ordinary IRQ/NMI entry is modeled at an instruction boundary. Exact
  penultimate-cycle IRQ recognition, CLI/SEI/PLP latency, interrupt hijacking,
  and other electrical edge cases require independent oracle review.
- RESET aborts on a bus-cycle boundary, rather than resolving analog timing.
  RESET held high will cause repeated reset entry on subsequent boundaries.
- The seven-cycle control sequence's first two reads are nonsync dummy reads.
  These addresses and SYNC behavior require validation against an actual
  chosen target before timing compatibility is claimed.
- `SetLines` hosts must mirror their current input levels in `Transfer`
  responses. Mixing contradictory sources will overwrite sampled levels.
- Interrupt entry increments the instruction counter as a stepping unit.
  Supervisor accounting may eventually distinguish retired instructions
  from interrupt entry sequences.

Next: validate and correct c12 against reference behavior, then run Dormann
functional and interrupt validation before declaring M1 complete.
