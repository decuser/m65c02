# Apple 65C02 Processor Laboratory

**Architecture Baseline:** 0.3 (frozen 2026-11-04)  
**Status:** production CPU through c14; M1 final corpus gate pending.  
**Language:** Go 1.16. **Target:** Apple IIe-compatible base CMOS 65C02 (no Rockwell bit operations or WAI/STP).

Read in this order:
1. `docs/architecture-baseline-0.3.md` — frozen contracts and compatibility target.
2. `docs/working-agreement.md` — development conventions, cuts, tests and reviews.
3. `docs/schema-spike-mandate.md` — disposable four-template spike.
4. `docs/reference-collection.md` — test corpora, reference texts and inspirations.

The production engine lives in `internal/cpu`. All 256 Synertek-profile opcode
bytes, cycle/microstep/instruction stepping, snapshots, and IRQ/NMI/RESET paths
are implemented. See `review/production-c10.md` through `production-c14.md`
for actual conformance evidence and remaining limitations.

## Local conformance corpuses

The two upstream repositories are kept **outside** this source tree, at the
following locations on the development machine:

```
~/forge/dev/tests/65x02/
~/forge/dev/tests/6502_65C02_functional_tests/
```

From `~/forge/dev/m65c02`, run:

```
make validate             # quick local tests; external tests if configured
make validate-full        # 2,560,000 Synertek vectors + Dormann functional
```

`validate-full` uses the sibling `../tests` directory by default. It runs
Synertek v1 in four groups of 640,000 vectors, then Dormann's published 6502
functional binary. A missing required repository is a **failure**, not a skip.
Override `TESTS_DIR=/other/directory` if needed. `validate-full` also requires
`../tests/dormann-profile/65C02_profile.bin` and
`../tests/dormann-profile/6502_interrupt_cmos.bin`, the independently rebuilt
65C02 and CMOS interrupt fixtures. Their setup is documented in
`docs/dormann-conformance.md`. They must not be confused with the upstream
Rockwell-enabled 65C02 binary.

The selected Synertek test set remains the default bus/cycle oracle and
Dormann complements it with semantic and interrupt testing. Deliberate
compatibility deviations must be documented rather than silently introduced.

The eventual Visible Computer/Wails UI and complete Apple IIe host are not part of the initial headless CPU milestone.

## Disposable interactive spike monitor

Run `make run`, then `help`, `trace on`, `cycle 1`, `cycle 10`, `step`,
`micro`, `regs`, `mem 0300`, or `run`. `demo counter` resets a finite demo
with ten `INC $0300,X` instructions and BRK. The counter ends at `$0A`.
This is a teaching/validation adapter, not a complete 65C02 or M2 supervisor.
