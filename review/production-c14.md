# Production CPU cut c14: profile-correct external conformance

Proposed commit: `Validate base CMOS 65C02 and interrupt behavior with Dormann`

## Scope

- Extended the opt-in Dormann runner to accept a *profile-correct* 65C02
  binary; never run the unmodified Rockwell-enabled upstream image as a gate.
- Added a feedback-bus adapter and end-to-end run of Dormann's interrupt test
  built for CMOS decimal-flag handling.
- Documented pinned source, exact build switches, memory layout, bus feedback
  polarity, expected success traps and opt-in validation commands.
- Added explicit status lines to `make validate` for the two new fixtures.

## Evidence

Source revision: Klaus Dormann `7954e2dbb49c469ea286070bf46cdd71aeb29e4b`.
The upstream AS65 1.42 assembler ran under a workspace-local i386 user-mode
emulator. No binaries or assembler runtime were added to the project overlay.

- Profile-correct extended 65C02: **pass**, 21,977,668 instructions and
  66,871,399 cycles; terminal success `$1C3C`.
- CMOS-configured interrupt/feedback test: **pass**, 1,070 stepping units;
  terminal success `$0719`.
- Existing Dormann documented 6502 functional test: **pass** (from c13).
- Full `make validate` with all three Dormann variables: **pass** on local Go
  1.24.10, including build/vet/unit suites.
- Synertek SingleStepTests external corpus: **not available in this workspace;
  not rerun for c14**. c11's previous 2.56-million-vector result remains
  historical evidence, not a newly executed pass.

## Boundaries

The Dormann interrupt program checks ordinary IRQ/NMI/BRK behaviors using
its feedback register, but cannot establish every physical pin sampling edge,
write-stall detail, RESET electrical timing or exotic concurrent interrupts.
Those remain governed by the frozen M1 compatibility scope and explicitly
documented exceptions. The profile-correct extended test deliberately omits
Rockwell and WDC-only instructions, not ordinary documented base-CMOS opcodes.

Next: review M1 acceptance with the user and, if desired, rerun the pinned
Synertek corpus against c14 before moving to M2. No architecture change.
