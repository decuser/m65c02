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

## Visual KIM-1 front panel (c27–c29)

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
The SST slide switch is to the right of GO, ST and RS, with the SST label
at its right position. C29 connects its position to the CPU's opcode-fetch
SYNC and the original KIM-1 ROM inhibit gate: each user instruction's fetch
causes an NMI after that instruction, and monitor ROM fetches do not trigger
SST. Set the NMI vector at `$17FA–$17FB` to `$1C00` (bytes `00 1C`) before
using SST, as required by the original KIM-1 manual. Turn SST on, select the
user program address, and press GO once per instruction.
Instrument mode shows bus
cycle count, CPU PC, active digit, segment lines, and recent dwell.
Pause stops the CPU clock; Step Cycle advances exactly one cycle. When paused,
instrument mode displays the currently selected digit rather than the
persistence-averaged display.

This cut is a local HTML presentation with no third-party GUI dependency.
It does not launch a native Wails window or expose the TTY connection over
the panel page. Use the separate `run-kim1-pty` targets for Minicom.

### Logic analyzer (c30–c31)

Click **OPEN LOGIC ANALYZER** in the KIM-1 panel to open a separate browser
window on `/analyzer`. The original panel remains usable. The analyzer shows
all 16 address bits, all 8 data bits, R/W, opcode-fetch SYNC, IRQ, NMI,
RESET, RDY and the sampled active Φ2 phase. A transaction table below the
waveforms shows cycle, address, data and control lines. Click a cycle to
inspect its resolved bus transaction.

The machine holds a bounded 8,192-cycle ring. The analyzer displays 128
cycles at a time; **OLDER** pauses the machine if it is running, then pages
backward through retained history. **NEWER** returns one page toward the most
recent capture, **LIVE** follows the latest edge again without restarting the
CPU, and **CLEAR TRACE** discards recorded samples. Pausing before navigating
is important: at 1 MHz, an 8,192-cycle ring otherwise wraps in about 8 ms.
**PAUSE**, **STEP CYCLE**, and **STEP INSTRUCTION** operate the shared machine.
After the initial automatic pause, history paging changes only the view, not
CPU or peripheral state. Reverse machine execution is not yet implemented. Because every
record is an active bus transfer, Φ2 is shown as the sampled high phase, not
a synthesized continuous clock waveform. Capture restarts on cold power-on.

C31 adds amber SYNC boundaries over the waveforms and instruction headers in
the transaction table. Each opcode is decoded using the CPU's existing opcode
table; the operand bytes are taken from subsequent recorded bus reads, not
from a separate memory inspection. Incomplete operands at capture-page edges
are shown as `??`. Selecting a bus cycle also shows the nearest captured
instruction context. Page boundaries may begin partway through an instruction.

C32 defaults to **grouped hex buses** instead of individual address/data bit
lanes. The ADDRESS and DATA lanes draw bus-style crossing transitions and
hex-value labels for every recorded cycle; selectable cycle widths keep the
values legible. **Expand address/data bits** restores the individual A15–A0
and D7–D0 waveforms. The control signals, disassembly, and history controls
are unchanged. The grouped display remains a read-only presentation of
captured bus values.

C33 adds a **FOLLOW** switch: scrolling or selecting a historical cycle turns
automatic right-edge following off while retaining LIVE data updates. FOLLOW
turns it back on. Selecting a transaction row centers its waveform column;
selecting a waveform column scrolls the transaction table to its row. The
waveform now has cycle-number ticks, and Φ2 shows the ideal alternating
half-cycle high/low clock aligned to each sampled bus transaction.

C34 adds a **Probe** selector to the same logic-analyzer window: CPU BUS,
6530-002 / DISPLAY, and 6530-003 / APPLICATION. Each CPU bus-cycle record
now also captures both RIOT chips' externally driven PA/PB pin values and
their direction registers. The display probe additionally shows the currently
selected LED digit and segment pattern. Each probe displays the **same cycle
numbers**; switching probes does not rerun or advance the machine. Use
**Expand bus/port bits** to inspect PA7–PA0 and PB7–PB0 individually, or leave
the ports grouped as hex bytes. The table and selected-cycle detail show the
corresponding values. Φ2 is an ideal, drawn half-cycle clock; the port values
are sampled after each corresponding emulated bus transfer.

C35 adds **CAPTURE RIOT CHANGES** to the selected 6530 probe. It pauses
execution, searches the 8,192 retained bus cycles for transitions on PA/PB,
direction registers, or (for the display chip) the digit/segment outputs,
and lists the exact cycle of each change. **◀ CHANGE** and **CHANGE ▶**
jump between events, fetching the matching 128-cycle waveform and transaction
window and centering the event. The event selector shows PA/PB and the LED
digit/segment values. The transaction table and selected-cycle details retain
the associated CPU instruction context. This capture uses existing bus trace
data; it does not add a separate independent clock or trace timeline. A full
six-digit scan is visible when its transitions fit within the retained ring;
at higher emulated speeds, pause to capture before history wraps.

C36 adds **EXPORT CSV** and **EXPORT TEXT** in the logic-analyzer toolbar.
Either download automatically pauses the CPU and copies the full retained
8,192-cycle trace (or fewer cycles since reset/clear) as a consistent snapshot.
Both formats include CPU address, data, read/write and control signals, the
two RIOTs' PA/PB and direction masks, and the front-panel digit and segments.
CSV uses one row per cycle with an instruction-context column; TEXT groups
cycles under decoded instruction headers. Unknown operand bytes at capture
boundaries are marked `??`. These exports contain only the retained ring,
not every cycle since power-on.

C37 adds an original-ROM end-to-end timing regression for `SCANDS` at `$1F1F`.
After allowing the genuine monitor to boot, a RAM-resident `JSR $1F1F`
invokes the monitor's real routine. The test observes opcode fetches at entry
and on return, captures RIOT pin transitions on the same bus-cycle timeline,
and checks that all six LED digits are scanned in order. With the original
ROM images, SCANDS takes exactly **4,231 cycles** from entry to return, or
**4,237 cycles including the six-cycle JSR**, matching the independent timing
reported in Kimulator's hardware notes. This is a timing assertion, not a
timing adjustment to the emulator. Run it with `make validate-kim1` when
the original ROM binaries are installed.

C38 replaces the exclusive waveform probe selector with **independent signal
selection**. CPU, 6530-002 and 6530-003 groups each have ALL/NONE controls;
individual buses, control lines, RIOT ports, direction registers and LED
signals can be enabled together on a single cycle axis. The **CPU, Display,
Keypad, Serial, All, None** presets offer useful starting points. **Expand
buses into bits** additionally exposes individual address, data and port bits
with their own checkboxes. Selection changes redraw existing captures only:
they do not touch CPU execution or the trace ring. The transaction table
and selected-cycle details identify both RIOTs, independent of waveform
selection. **Change source** still selects which RIOT's transitions the C35
change navigator searches; it does not restrict visible waveform signals.

C39 adds opt-in **MICRO TRACE** instrumentation. With capture enabled, the CPU
reports each executed pre-bus operation, the bus resolution, and each post-bus
operation together with before/after CPU state. The selected bus
cycle and internal steps use the same cycle number. Internal events are kept in
a separate bounded 32,768-step ring, so very old CPU cycles may outlive their
microstep detail. Toggling capture on starts a fresh internal history; toggling
off removes the observation callback. This is observation, not reversible
execution. The existing CSV/TEXT exports remain physical bus-cycle exports.

C40 separates internal observability from external machine signals. The KIM-1
logic analyzer now contains only CPU bus/control and RIOT probes. The front
panel's **VISIBLE COMPUTER** button opens `/visible`, a dedicated CPU-internals
view. Start micro capture there, then pause/step and select a cycle to see its
actual executed PRE/BUS/POST operations and before/after register/latch changes.
Both windows share the running machine and cycle numbering. Internal capture
is opt-in; the logic analyzer no longer requests or renders internal details.
The Visible Computer page is an initial observational interface, not yet a
reversible microstep debugger or a full internal waveform view.


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
