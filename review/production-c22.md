# Production cut 22: KIM-1 TTY jumper and original monitor greeting

## Objective

Model the KIM-1 TTY-select jumper as physical board wiring, loaded from a reproducible machine configuration file. Verify the original ROM's terminal behavior through the existing serial pins.

## Source evidence

- KIM-1 User's Manual (August 1976), Figure 2.4 (printed page 18): TTY uses a jumper from connector A-21 to A-V.
- Figure 3.5 (printed page 28): connector A-21 is column A on PA0; A-V is keyboard decoder row 3. A closed jumper is a key-matrix contact, not a hardwired low bit.
- Figure 3.9 (printed page 32): connector mapping corroborates A-21 and A-V.

## Delivered

- `machines/kim1.toml` specifies ROM files, nominal clock, and `jumpers.tty_select`.
- Strict small TOML-subset parser with diagnostics for unknown and malformed settings; no new dependency.
- Board decoder combines the 74145 PB1-PB4 row selection and PA0 input when the jumper is closed.
- Runner accepts `-config`, with Make preserving its existing ROM overrides.
- Adjusted serial framing to eight data bits, based on real PB0 waveform and original ROM output.
- `TestKIM1TTYBoot` runs the original ROMs with both jumper states, injects RUBOUT into PA7, and requires that the `KIM` greeting appears only with the TTY jumper closed.

## Validation

- `make validate`: passed all quick tests.
- `make validate-kim1 GO=../.toolchains/go/bin/go TESTS_DIR=../upload`: passed byte-for-byte ROM verification and TTY boot tests with the actual binaries.
- `make run-kim1 GO=../.toolchains/go/bin/go TESTS_DIR=../upload KIM_CYCLES=3000000`: produced a `KIM` greeting and an address prompt from original ROM pin output.
- Original ROM output captured by the 1.5 million-cycle test included `KIM` with CR/LF; some preceding NUL-like decoded bytes remain unexplained.

## Limits and next cut

- The monitor's output is real, but this is not yet a live, bidirectional terminal application. Input is presently pre-supplied and execution is bounded.
- The CLI's nominal clock does not rate-limit actual wall-clock execution.
- Serial framing/polarity at electrical-edge fidelity and the nonprinting decoded bytes still deserve inspection.
- c23: provide interactive host input and a long-running terminal session, with safe execution pacing and optional logic trace.

Suggested commit: `Wire KIM-1 TTY jumper from configuration and validate ROM greeting`.
