# Production CPU cut c13: Dormann functional harness

Proposed commit: `Add Dormann functional conformance and interrupt restore tests`

## Scope

- Added `TestDormann`, an opt-in streaming-independent harness loading the
  published 64 KiB full-memory functional binary and recognizing upstream's
  success self-loop at `$3469` versus other trap loops.
- Added microstep-position snapshot/restore regression for NMI entry. The
  independently continued original must match the restored CPU, RAM and full
  ordered bus request stream without duplicate resolved transfers.
- `make validate` now distinguishes unavailable Dormann binaries from pass/fail.

## Reproduction

Reference: Klaus Dormann, `Klaus2m5/6502_65C02_functional_tests`, commit
`7954e2dbb49c469ea286070bf46cdd71aeb29e4b` (upstream `bin_files/`).
No upstream binaries are distributed in this overlay.

Set `M65C02_DORMANN` to an **absolute** directory containing
`6502_functional_test.bin` and run `make validate`. The test starts at `$0400`,
uses plain 64 KiB RAM and a bounded 100-million-instruction limit.

Local run with Go 1.24.10:

- `make validate` with `M65C02_DORMANN` set: **pass** (build, vet, unit tests).
- Dormann documented 6502 functional test: **pass**, 30,646,179 instructions,
  96,561,330 cycles.
- NMI mid-microstep snapshot/restore equivalence: **pass**.
- Synertek external vectors: unavailable in this workspace, not rerun.
- Dormann interrupt test: source exists upstream; **not assembled or run**.

## Profile boundaries and remaining gate

The upstream *prebuilt* `65C02_extended_opcodes_test.bin` was tried and
trapped at `$0730`, where its listing shows a Rockwell `BBR`/`BBS` test.
Its source defaults to `rkwl_wdc_op = 1`; that is outside Architecture 0.3's
base-CMOS 65C02 profile. **No core change was made to satisfy this fixture.**
The extended test requires a profile-correct rebuild (Rockwell extensions
disabled, WAI/STP disabled) and a separately established success address.

c13 supplies substantial independent functional confidence, but **M1 is not
yet fully accepted**. The Dormann interrupt program requires a feedback I/O
register and a compiled fixture. Precise interrupt recognition timing and
external full-corpus reruns remain explicitly open.
