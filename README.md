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
version suppresses the raw pin log. Since c23 they run continuously and
accept keyboard input by default. A RUBOUT autobaud character is injected
shortly after CPU reset. Use `KIM_CYCLES=3000000`, `KIM_BIT_CYCLES=9091`, or
`KIM_INPUT=...` to experiment.

The `machines/kim1.toml` configuration selects the physical KIM-1 TTY jumper.
When `tty_select = true`, A-21 (6530-002 PA0) connects to A-V (74145 row 3);
the jumper changes the port input only while row 3 is selected. With the
original ROMs the serial decoder now receives the monitor's `KIM` greeting
and address prompt. The regression test `make validate-kim1` checks both
jumper positions, and verifies that the greeting appears only when closed.

The serial component watches real port pins and does not intercept monitor
code. Some initial nonprinting bytes appear in the raw decoder output and
have not yet been explained.

## Live KIM-1 terminal (c23)

Run `make run-kim1` in a real terminal. It boots the original ROMs, performs
the RUBOUT baud-rate handshake, and continues running until Ctrl-C or Ctrl-D.
The keyboard is put in raw mode and restored on exit. The terminal displays
typed characters locally because the original KIM-1 expects a full-duplex
teleprinter, then displays responses decoded from PB0. For example, type
`1234 ` (including the trailing space) to select an address and see `1234 FF`.

`make run-kim1-trace` also prints cycle-stamped RX/TX transitions to stderr.
`KIM_CYCLES=6000000` selects a finite, unpaced run useful for automated
checks, rather than the default paced interactive operation.

The configured `clock_hz` determines wall-clock pacing; every simulated bus
cycle still executes. Host input is buffered and injected as serial frames
after the initial monitor greeting. The keyboard currently uses `stty` on
Linux, and the first input window is calibrated for 110 baud at 1 MHz.

## KIM-1 front-panel electrical model (c26)

The original keypad/display circuitry is now an independent clocked
`KIMFrontPanel`, attached to the same 6530-002 pins that the ROM accesses.
Its 74145 decoder uses PB1..PB4 to select keypad rows 0..2, the TTY jumper
on row 3, or LED digits 0..5 on decoder outputs 4..9. PA0..PA6 carry the
seven shared columns/segments; serial PA7 and PB0 remain independent.
The 21 matrix keys can be held/released with `Panel.SetKey(name, pressed)`;
ST maps to `PressStop`/`ReleaseStop` on NMI and RS uses
`ResetCPU`/`ReleaseReset`.
The physical key matrix is wired with descending PA columns (left to right
PA6 through PA0), rather than ascending bit order. A complete 21-key test
checks each asserted pin and its release; this corrects cases where pressing
`1` registered as `5` and pressing `B` registered as `9`.

The `Current` electrical digit/segment selection, `LastScan` per digit,
and cycle-accumulating `Dwell` counters are observable without any GUI.
The original ROM is verified to scan all six digits in keypad mode.
This cut does not yet include clickable controls or an LED renderer.

The ROM-backed reset-display regression samples full LED-on dwell windows
before and after pressing RS. With the test fixture's **zero-filled RAM**,
the original ROM produces `0000 00` and returns to the same display after
a subsequent reset. Real cold-start RAM need not be zero-filled; this is
not a claim that original KIM-1 hardware always displays `0000 00`. The per-cycle
`LastScan` sample is not suitable as a stable rendered character because
the 6530 pins are multiplexed; use the dwell observations instead.

## Virtual serial port for Minicom (c24)

## Visual KIM-1 front panel (c27–c28)

The front panel now has an interactive local browser view backed by the
cycle-driven C26 hardware model. Put both original `6530-002.bin` and
`6530-003.bin` in `../tests` (or set `KIM_ROM002` and `KIM_ROM003`), then:

```
make run-kim1-panel
```

The command starts the Go KIM-1 machine on `127.0.0.1:6580` and opens the
display in your default browser. If automatic opening fails, visit
`http://127.0.0.1:6580/` manually. Use `PANEL_ADDR=127.0.0.1:6581` for a
different port, or `PANEL_OPEN=0` to disable automatic browser launch.

The original ROM drives the six LED digits via the 6530 port lines. The
display uses measured on-cycle dwell for LED persistence. The keypad follows
the original six-row, four-column physical arrangement; its 21 matrix keys are
actual row/column contacts, not calls into the monitor. The SST ON/OFF switch
is shown but disabled until its electrical single-step behavior is modeled.
Keyboard shortcuts
for `0`–`9` and `A`–`F` are also available. ST asserts NMI and RS asserts
reset. The TTY toggle is in a separate External Connections section because
the original used an optional jumper or switch between application connector
pins A-21 and A-V, not a switch built into the KIM-1 keypad. OPEN selects
the keypad and LEDs; CLOSED selects the TTY. The panel starts in keypad mode.
The POWER control sits above the LED bank. OFF stops the CPU and blanks
the virtual display. ON constructs a fresh KIM-1 but **does not clock the CPU**:
the LEDs remain dark until you press and release RS, just as when powering
a physical board and then pressing reset. RS preserves RAM, while OFF/ON
reinitializes all volatile main and 6530 RAM. The external TTY jumper
position persists across power cycles. The default cold-start `device` RAM
policy generates a stable, per-device synthetic power-up pattern with small
changes between power cycles; this is not a measured physical SRAM model.
Use `PANEL_RAM_MODE=zero` for deterministic zero-filled RAM, and optionally
`PANEL_RAM_SEED=<integer>` to choose a different synthetic device pattern.
The six LEDs are scaled to the compact keypad and maintain the 4+2 spacing.
The SST slide switch is to the right of GO, ST and RS, with OFF at the left
and ON at the right, but remains disabled until electrically modeled.
Instrument mode shows bus
cycle count, CPU PC, active digit, segment lines, and recent dwell.
Pause stops the CPU clock; Step Cycle advances exactly one cycle. When paused,
instrument mode displays the currently selected digit rather than the
persistence-averaged display.

This cut is a local HTML presentation with no third-party GUI dependency.
It does not launch a native Wails window or expose the TTY connection over
the panel page. Use the separate `run-kim1-pty` targets for Minicom.


On Linux, start the emulator in the existing terminal:

```
make run-kim1-pty
```

It prints the exact connection command, using the allocated device (for
example `/dev/pts/7`). In a **separate** terminal, run that command:

```
minicom -D /dev/pts/7 -b 1200 -8
```

Configure Minicom for 8N1 and local echo (`Ctrl-A`, then `E` if needed).
Once Minicom is ready, press Return to start the original monitor's
baud-detection sequence. The host adapter translates that initial Return
(or ASCII DEL `$7F`, or the modern Delete key's `ESC [ 3 ~`) into the RUBOUT
frame that the **original KIM-1 ROM requires**. This translation happens only
before startup completes. Opening the device does not inject RUBOUT.
The PTY carries terminal *bytes*, but retains Minicom's termios baud setting.
The adapter reads that setting when the startup key arrives and converts it
to timed serial frames on the 6530 pins. The original ROM then measures the
RUBOUT frame to negotiate the rate itself. No separate `KIM_BIT_CYCLES`
configuration is needed for the PTY targets. The printed Minicom command
suggests 1200 baud, but 110 or 300 can be selected in Minicom instead.
Changing Minicom's speed **after** the ROM has negotiated does not reset
the KIM-1 or re-run baud detection. Reboot the emulator to test another speed
until the virtual Reset control is implemented. The startup message reports
the PTY's actual termios rate. For example, Linux termios encoding `0x2`
means 75 baud, not 110, and may indicate a Minicom configuration mismatch.

With a trace window, `HOST received` shows the terminal's actual bytes;
`RX` transitions show the resulting 6530 input pins. The original MOS
KIM-1 User's Manual (August 1976, page 18 and monitor description)
explicitly requires RUBOUT for the ROM's initial baud measurement. A PAL-1
may instead initialize its terminal using Return, as documented in the
2022 PAL-1 serial programming tutorial.

`make run-kim1-pty-trace` opens only an **additional trace window** using
`x-terminal-emulator`. The emulator remains in the current terminal and
Minicom remains in its own terminal. Likewise `make run-kim1-trace` opens a
separate trace window without creating a second KIM-1 emulator. Closing the
trace window does not stop the CPU.

The PTY integration test opens a real Linux pseudoterminal, communicates with
the original ROM, and verifies the `1234 FF` response. A separately installed
Minicom GUI was not exercised in the development container. Reconnection to
an already-running emulator has not been qualified. Ctrl-C interrupts the
Make target; an interrupted Make invocation can return nonzero without
indicating an emulator failure.

## Disposable interactive spike monitor

Run `make run`, then `help`, `trace on`, `cycle 1`, `cycle 10`, `step`,
`micro`, `regs`, `mem 0300`, or `run`. `demo counter` resets a finite demo
with ten `INC $0300,X` instructions and BRK. The counter ends at `$0A`.
This is a teaching/validation adapter, not a complete 65C02 or M2 supervisor.
