# Production c23: live original KIM-1 terminal

## Objective

Operate the original KIM-1 monitor from a host terminal using actual PA7/PB0 serial input and output, with continuous paced CPU execution and clean shutdown.

## Changes

- `make run-kim1` and `make run-kim1-trace` default to unbounded, real-time paced execution, in a child Go process.
- Host keyboard input is read asynchronously, delivered to the emulator thread through a bounded channel, and serialized through the 6530-002 PA7 input. CPU execution never blocks for a keystroke.
- A Linux terminal is placed in raw, no-echo mode using `stty`; the original terminal mode is restored on Ctrl-C/Ctrl-D exit.
- Host keys are displayed with local echo, matching the original manual's requirement for a full-duplex teleprinter. The KIM monitor only transmits its responses.
- ROM startup is still triggered with RUBOUT; host input is buffered until the initial monitor greeting and prompt have had time to finish.
- `KIM_CYCLES` now defaults to 0 (interactive indefinitely). A nonzero value enables finite, unpaced regression runs.
- Test for full serial command handling: entering `1234 ` (including the trailing space) produces the original monitor's `1234 FF` response on PB0.

## Verified results

- `make validate`: passed.
- `make validate-kim1 GO=../.toolchains/go/bin/go TESTS_DIR=../upload`: all three ROM integration tests passed, including the two-way command test.
- Manually tested `make run-kim1` in a pseudo-terminal, typed `1234 ` after the greeting, received `1234 FF`, and exited cleanly using Ctrl-C.
- Full Synertek and Dormann corpora were not rerun for this terminal-focused cut.

## Limitations

- `stty` expects a Unix-like host with a terminal, and live host input is presently byte-oriented rather than a complete ANSI terminal abstraction.
- The host's wall-clock pacing is approximate (1,000-cycle checkpoints) and does not drop emulated cycles.
- The initial input enable time is fixed at 3 million cycles for 110 baud/1 MHz; auto-detecting the complete original monitor prompt is a future refinement.
- Some initial nonprinting decoded bytes remain unexplained, and the guest remains a 65C02 rather than an NMOS 6502 profile.
- No seven-segment panel, keypad, cassette, or whole-machine reverse stepping yet.

## Next cut

c24: add the six multiplexed seven-segment displays, preserving 6530 port/decoder wiring and cycle visibility.

Suggested commit: `Run interactive KIM-1 monitor with local echo and paced serial input`.
