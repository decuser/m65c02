# C41: Cycle-accurate 65C02 release candidate

**Candidate:** `m65c02` at developer baseline `dev-v0.1.0`, commit `f0a1946` (2026-11-18).  
**Architectural contract:** `docs/architecture-baseline-0.3.md` remains the
specification. This cut makes no CPU execution changes and does not supersede
the frozen architectural decisions.

## CPU scope

- Apple IIe-compatible base CMOS 65C02 execution profile, including BRA,
  STZ, TRB/TSB, PHX/PHY/PLX/PLY, accumulator INC/DEC, extended BIT,
  zero-page indirect, and JMP (abs,X).
- Deliberately excludes Rockwell RMB/SMB/BBR/BBS and WDC WAI/STP behavior.
  Opcode bytes outside the chosen profile use the selected corpus's NOP
  interpretations; this is not a promise of full NMOS illegal-opcode behavior.
- Explicit PRE/BUS/POST sequencer with public microstep, cycle, and instruction
  stepping. A resolved bus response is retained across a teaching pause.
- Bus interface is `cpu.Bus.Transfer(cpu.Request) cpu.Response`. The CPU does
  not own or know the host's RAM, devices, scheduler, or visual interface.
- Value snapshots via `Snapshot`, checked `Restore`, and a nil-by-default
  microstep observation callback. Restoring a CPU snapshot alone does not undo
  RAM writes or peripheral activity.
- This is a behavioral, cycle-accurate model, not a transistor-level netlist.
  Named internal latches and teaching operations are explanatory.

## Verification

| Check | Status | Evidence |
| --- | --- | --- |
| `make validate` | Passed in audit workspace | Go build, vet, local unit/integration tests |
| `make validate-kim1` | Passed in audit workspace | Supplied original 6530 ROMs, serial, LED scan and SCANDS checks |
| `make validate-full` | **Developer-reported passed** | Run on development machine with external test corpus; full stdout was not supplied with this baseline |
| Synertek 65C02 | Covered by full target | Four 640,000-vector groups (2,560,000 vectors total) |
| Dormann | Covered by full target | NMOS functional, custom CMOS 65C02 profile and CMOS interrupt fixtures |
| CPU snapshot/step equivalence | Passed locally | Unit tests include all-step modes, mid-cycle restore and interrupts |
| Micro-observation callback | Passed locally | On/off execution test; callback is disabled by default |

`make validate` by itself is **not** the full corpus gate. Those corpus tests
skip when their environment variables are absent. `make validate-full` checks
for required external binaries/JSON before running them, and fails when they
are missing. No third-party corpus is included in the baseline source archive.

## Reproducible acceptance

On the development machine, from the repository root:

```sh
make validate
make validate-full
make validate-kim1
git status --short
make baseline
```

For independent release evidence, retain the passing command output and the
exact revisions or hashes of external test datasets. The uploaded baseline
archive omitted a tracked `m65c02-dormann-profile-fixtures.tar.gz` because the
old `make baseline` excluded every root-level `*.tar.gz`. C41 removes that
broad exclusion and fails archival creation if Git reports missing tracked
files. An archive built from the developer's complete working tree should
contain the tracked fixture; this audit copy cannot reconstruct that file
from the tarball itself (the Git blob exists in its embedded repository).

## Deferred work, not release blockers

- No full physical-silicon or analog timing representation.
- The Apple IIe host, complete video/audio/disk peripherals, and machine-wide
  reversible execution are separate projects/milestones.
- Extraordinary electrical RDY write timing, bus ownership/DMA, and unusual
  untested interrupt-edge details remain outside the stated profile guarantee.
- The KIM-1 logic analyzer and Visible Computer UI are clients of the same
  execution core but do not define CPU semantics.

## Release decision

The candidate is appropriate to freeze **conditional on the recorded full
corpus pass and a complete clean baseline archive**. This document does not
assert a new release tag was created or pushed.
