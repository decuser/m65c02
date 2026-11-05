# Production CPU cut c10

Proposed commit: `Introduce production 65C02 sequencer and corpus-backed instruction tests`

## Scope

First production M1 implementation, separate from the disposable spike. `internal/cpu`
contains a value-only state, frozen cycle-entry IDs, typed microoperations, the
Present/Resolve/PostBus cursor, selected-entry stability, per-cycle and
per-instruction stepping, RDY read-stall repetition, snapshots, and prepared
unresolved SYNC at boundaries. CPU imports only `errors` from the standard
library and knows nothing about the host or RAM.

Currently implemented opcodes: `03`, `EA`, `A9`, `BD`, `9D`, `FE`, `4C`, `D0`,
`00`. This is **not** the complete production 65C02 instruction set.

`make run-prod` runs the production engine's finite counter program. The old
`make run` remains the disposable spike monitor until its replacement exists.

## Conformance

Set `M65C02_CORPUS` to a directory containing Synertek v1 opcode files
(e.g. `/path/to/65x02/synertek65c02/v1`), then run `make validate`.
The adapter streams **10,000 vectors each** for all nine supported opcodes,
checking complete register state, specified final memory, and the ordered bus
address/data/read-write triplets. The corpus is not committed to m65c02.

The local reference run passed all **90,000** vectors with no exclusions among
these nine opcodes. Other 247 opcodes, IRQ/NMI/RESET, and Dormann tests have not
been implemented or passed. `make validate` reports unavailable corpus when
`M65C02_CORPUS` is unset.

The production NOP benchmark is zero allocations/op on the local Go toolchain,
but throughput is not comparable to the earlier spike benchmark (different
instruction, different work and hardware). The `go.mod` language target remains
Go 1.16; the available test compiler was Go 1.23.2, not an installed Go 1.16.

## Findings and limitations

- Post-bus microoperations use the selected immutable cycle-entry ID.
- Indexed CMOS dummy reads use the last operand address (PC-1 after high-byte
  fetch); verified against all selected Synertek vectors.
- The common fetch cycle handles a single-cycle `03` NOP and consecutive SYNC.
- No dynamic allocation occurs in the core NOP execution path when the host bus
  itself does not allocate.
- The initial `next` sequencing still uses a bounded opcode/stage switch;
  moving conditional transitions wholly into table records is outstanding.
- Snapshot validation is conservative but not yet exhaustive for malformed
  cursor combinations; external devices must be snapshotted separately.
- Runtime bus faults, asynchronous interrupts, external RESET, and other
  unsupported opcodes are not yet production-complete.
