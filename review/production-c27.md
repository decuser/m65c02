# C27: KIM-1 front-panel presentation

## Scope

- Introduced `cmd/m65c02-panel`: a standalone, loopback-only HTTP server
  exposing an embedded responsive KIM-1 panel.
- Six seven-segment LED digits render from the C26 cycle-accumulating dwell
  counters, with instantaneous multiplexed digit/segment state in paused
  instrument mode.
- Original 6 × 4 physical key layout with 21 matrix keys, ST, RS, and TTY
  jumper wired to `KIM1` hardware
  methods. The browser never writes memory or calls monitor routines directly.
- Pause, resume and single bus cycle advance; the instrument panel exposes
  PC, cycle count, active digit, segment mask and duty estimates.
- `make run-kim1-panel` uses original ROM binaries in `../tests`, configurable
  with existing Make variables. `PANEL_ADDR` and `PANEL_OPEN` customize startup.

## Validation

- `make validate` succeeds, including new HTTP control tests.
- `make validate-kim1` exercises the actual ROM map and bootstrap using supplied
  6530-002/003 binaries.
- The ROM-backed RS regression checks the steady six-digit `0000 00`
  segment pattern before and after asserting/resetting the CPU. This is
  sampled from accumulated lit cycles, not a transient `LastScan` value.

## Deliberate limits

- This is a local browser UI, not native Wails packaging.
- The panel currently runs the simulated CPU at 1 MHz in 10 ms batches;
  it is not a real-time scheduling guarantee.
- The TTY jumper is a physical wiring selection; Minicom via PTY remains a
  separate runtime target.
- Requires interactive verification of LED/keypad appearance and ergonomics
  on the user's actual desktop.
- The physical SST ON/OFF position is displayed but disabled until its
  single-step input is modeled. It is distinct from the TTY jumper.
- The TTY selection is drawn outside the PCB in an External Connections
  section, labeled A-21 ↔ A-V, to reflect the original connector wiring.
- Corrected the physical keyboard column order to PA6…PA0. A dedicated
  regression checks assertion and release of all 21 original matrix keys.
- Added a separate POWER control. OFF stops the simulated clock and blanks
  LEDs; ON reconstructs the machine with cleared RAM and restarted ROM,
  retaining the external connector wiring. A regression checks RAM loss,
  CPU cycle restart, blocked inputs while off and preserved TTY selection.
- Reduced physical keycaps to two-character width with side spacing;
  SST ON is marked at the left. Removed duplicate ST/RS browser event binding.
