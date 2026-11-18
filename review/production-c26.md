# Production c26: KIM-1 front-panel electrical model

## Scope

Implement the original keyboard/display circuitry as a separate clocked
component. No graphics, terminal controls, fabricated ROM behavior, or
register-level bypasses.

## Wiring contract

MOS KIM-1 User's Manual (August 1976), Figure 3.5, printed page 28:

- 6530-002 PB1..PB4 select one of 16 outputs of the 74145 decoder.
- Decoder rows 0, 1, 2 select the three keypad rows.
- Decoder row 3 selects the TTY jumper at PA0 (A-21 to A-V).
- Decoder outputs 4..9 enable the six seven-segment digits, left to right.
- 6530-002 PA0..PA6 represent keyboard columns A..G and segments A..G.
- PA7 remains the separate serial RX input. PB0 remains serial TX.
- ST drives the CPU NMI control line; RS drives reset, independently of keys.

## Delivered

- `KIMFrontPanel` with contact closures for the 21 standard matrix keys.
- Matrix input sensing on register reads, preserving serial PA7.
- Per-cycle live digit/segment selection and cumulative segment dwell counts.
- `PressStop`/`ReleaseStop` for ST; existing `ResetCPU`/`ReleaseReset` for RS.
- The existing TTY-select jumper now uses the same matrix input resolver.
- Key state survives CPU reset (the key is a physical contact).
- The component ticks once per physical bus cycle and works without a UI.

## Validation

- `make validate`: pass.
- `make validate-kim1` with original ROMs: pass, including serial/PTY tests.
- Original ROM in keypad mode illuminates all six decoder positions.
- Matrix, TTY jumper, PA7 isolation, dwell and ST/RS line tests pass.

## Deferred

- UI/front-panel drawing and host input bindings (c27).
- End-to-end keypad program entry, Reset interaction and mode switching (c28).
- Broader backplane/pin trace visualization (c29).
