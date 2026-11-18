# Production c21: serial pin model and trace runner

## Scope

- External KIM-1 TTY connection at 6530-002 PA7 receive and PB0 transmit pins.
- A per-clock serial component that generates framed RX input, observes actual PB0 output, records timestamped transitions, and attempts seven-bit start/data/stop decoding.
- A KIM-1 executable launched as a child OS process through `make run-kim1` or `make run-kim1-trace`.
- Fixed-cycle, bounded execution; cycle, bit timing, and initial input configurable from Make.
- Unit test for RX input, TX bit sampling, and transition logging.

## Validation

- `make validate GO=../.toolchains/go/bin/go`: passed.
- `make run-kim1-trace GO=../.toolchains/go/bin/go TESTS_DIR=../upload KIM_CYCLES=300000`: passed process execution, recorded TX transitions at cycles 37 and 43 and injected RX start/stop at 2001 and 11092. No decoded TX characters.
- `make run-kim1` with the same inputs: passed bounded execution, no decoded TX characters.
- The original 6530 ROM files are separately required and not redistributed in this overlay.

## Critical limitation

**This does not yet deliver an interactive original monitor.** The authentic monitor did not emit its `KIM` prompt in this test. KIM-1 TTY selection requires a physical jumper between connector pins A-21 and A-V, which is not yet modeled. The receiving electrical inversion and accurate autobaud handshake also require confirmation. The 110 baud/1 MHz default is a configurable starting parameter, not a proven ROM-compatible setting. Input is a command-line string preceded by RUBOUT, not live stdin.

The trace logs physical line transitions rather than synthesizing output by intercepting ROM code. All original CPU and 6530 components remain reusable.

## Next cut

c22: model the KIM-1 TTY selection jumper and true board-level electrical polarity, then verify the ROM's `KIM` prompt with an automated integration test before calling the console interactive.

Suggested commit: `Add cycle-stamped KIM-1 serial pin trace and headless runners`.
