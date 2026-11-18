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

## KIM-1 ROM verification

Place `6530-002.bin` and `6530-003.bin` directly in `~/forge/dev/tests`,
alongside the other external test directories. From `~/forge/dev/m65c02`:

```
make validate-kim1
```

The target requires both files, verifies all 2,048 bytes through the emulated
bus (including low/high ROM mirrors), performs the CPU reset-vector fetch, and
executes the first 20 instructions of the original monitor. This is a ROM
mapping and initial execution test, **not** an interactive KIM-1 boot session;
the keypad/display and serial wiring are still pending. Use `TESTS_DIR` or
`KIM_ROM002`/`KIM_ROM003` to override the default paths.

## KIM-1 serial pin trace (c21)

With the ROMs in `../tests`, run:

```
make run-kim1-trace
make run-kim1
```

Both targets launch a separate emulator process. The trace version prints
cycle-stamped 6530-002 PB0 (TX) and PA7 (RX) transitions, while the ordinary
version suppresses the raw pin log. They are **bounded headless runs**, not
interactive terminals yet. A RUBOUT autobaud character is injected shortly
after CPU reset. Use `KIM_CYCLES=3000000`, `KIM_BIT_CYCLES=9091`, or
`KIM_INPUT=...` to experiment.

The `machines/kim1.toml` configuration selects the physical KIM-1 TTY jumper.
When `tty_select = true`, A-21 (6530-002 PA0) connects to A-V (74145 row 3);
the jumper changes the port input only while row 3 is selected. With the
original ROMs the serial decoder now receives the monitor's `KIM` greeting
and address prompt. The regression test `make validate-kim1` checks both
jumper positions, and verifies that the greeting appears only when closed.

The serial component watches real port pins and does not intercept monitor
code. The runner still consumes a fixed `KIM_INPUT` string and stops after a
chosen number of cycles; live keyboard input is a separate next step. Some
initial nonprinting bytes appear in the raw decoder output and have not yet
been explained.

## Disposable interactive spike monitor

Run `make run`, then `help`, `trace on`, `cycle 1`, `cycle 10`, `step`,
`micro`, `regs`, `mem 0300`, or `run`. `demo counter` resets a finite demo
with ten `INC $0300,X` instructions and BRK. The counter ends at `$0A`.
This is a teaching/validation adapter, not a complete 65C02 or M2 supervisor.
