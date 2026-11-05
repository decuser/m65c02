# Production CPU cut c11

Proposed commit: `Implement base CMOS 65C02 instruction set with Synertek conformance`

## Scope

The production `internal/cpu` engine now supports all 256 Synertek v1 opcode
bytes, interpreting x7/xF columns and CB/DB according to the selected corpus
rather than as Rockwell bit extensions or WAI/STP. No second interpreter was
introduced: every instruction uses the same physical-cycle phase cursor,
bus interface, and stepping APIs. The 9 c10 opcodes retain their original
sequencer entries. New instruction families use a static opcode-metadata
table and shared typed read/write cycle entries with address and result
microoperations. They do not dispatch by explanatory label strings.

Added load/store, arithmetic, logic, compare, BIT/TSB/TRB, shifting/rotation,
stack, control-transfer, branches, indirect address modes, accumulator forms,
and reserved NOP profiles. Decimal ADC/SBC timing and flags use the selected
Synertek profile, including its extra decimal-mode bus cycle.

## Conformance evidence

`TestSynertekAllOpcodes` checks initial/final A/X/Y/S/P/PC, final specified
memory, and the **complete bus address/data/direction trace** for every
provided vector. It streams JSON by opcode without storing the corpus in
source control. Set `M65C02_CORPUS=/path/to/synertek65c02/v1`.

The supplied Synertek v1 corpus passed in four groups of 64 opcode files,
10,000 cases per opcode, **2,560,000 / 2,560,000 vectors**. Each group
was executed separately to avoid the runner's process timeout. The existing
c10 selected-opcode test also passed 90,000 vectors. No vectors were
excluded from the 256-file comparison. Full command form, repeated for
ranges [0-3], [4-7], [8-b], [c-f]:

```
M65C02_CORPUS=/path/to/synertek65c02/v1 \
M65C02_CORPUS_LIMIT=10000 \
go test ./internal/cpu -run '^TestSynertekAllOpcodes$/^([0-3][0-9a-f])$' -count=1
```

`make validate` with the corpus installed executes a default **100 vectors
per opcode** in the new test plus the earlier 90,000-vector focused test.
Use `M65C02_CORPUS_LIMIT=10000` and, if needed, the bounded command above
for exhaustive conformance. Without the corpus, the external tests are
explicitly skipped.

`TestProductionLoopWithExtendedInstructions` and `TestExtendedGranularityAndRestore`
exercise a finite program, all three stepping granularities, and snapshot
restoration against an uninterrupted reference trace. A fresh production demo
is available through `go run ./cmd/m65c02-prod -demo loop`, optionally
with `-trace` to show every bus transfer.

## Compatibility decisions

Synertek v1 is the target. The NCR reference lists different cycle counts
for reserved x7/xF opcodes and 5C (8 cycles in its table versus 4 in the
selected corpus). We intentionally follow Synertek. These are compatibility
choices, not claims of identical behavior across physical chips.

## Remaining work

This cut completes instruction dispatch and in-corpus instruction-cycle
conformance, **not the complete M1 milestone**. SingleStepTests does not
exercise asynchronous IRQ/NMI recognition or external RESET; those remain
for c12 with Dormann's interrupt test. RDY edge cases and out-of-profile
hardware timing are not claimed fully verified. Dormann's semantic and
interrupt binaries were not run.

New instruction families use generic typed microoperations for their
common address and commit paths. The visible interface may require finer
explanatory annotations, but those must remain observations of real CPU
state, never a second execution model. Snapshot validation is bounded,
not exhaustive for maliciously malformed snapshots.

Go module target remains 1.16; local compiler was Go 1.23.2.
