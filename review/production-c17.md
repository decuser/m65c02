# c17: M2 backplane and machine clock

Proposed commit: `Introduce machine backplane with mapped bus and clock connections`

## Objective

Establish the first reusable M2 machine layer without changing the frozen M1
CPU implementation or introducing KIM-1-specific motherboard behavior.

## Changed files

- `internal/machine/backplane.go`: one-CPU machine, mapped bus targets,
  machine-owned physical-cycle counter, floating reads, RDY and control
  signals, separate clock/interrupt connections.
- `internal/machine/backplane_test.go`: address decoding, overlap rejection,
  CPU program through mapped RAM, cycle counts, RDY retry, microstep pause and
  device IRQ propagation.
- `review/production-c17.md`: this report.

## Validation

Ran `gofmt` and `make validate GO=../.toolchains/go/bin/go` with Go 1.24.10:
**pass** for build, vet, all local CPU/spike/command tests and new machine tests.
External Synertek and Dormann fixtures were not installed in this workspace;
their tests were skipped, not counted as passes. M1 CPU code is untouched.

## Design limits

- One CPU master and one transfer per machine cycle; DMA/arbitration deferred.
- Address windows are inclusive and nonoverlapping. Devices receive relative
  addresses. Memory chips, mirrors and file-backed ROM/RAM belong to c18.
- Clock and interrupt wiring are explicit: `Map` does not implicitly connect
  a device's timer or interrupt pin. Connect each physical device once even if
  it is mapped into multiple address windows.
- Unmapped reads return `$FF`; unmapped writes are ignored. There is no
  electrical contention or analog power model.
- The CPU's existing bus response has no error channel, so mapping changes
  are validated before run; a missing device is not a runtime bus fault.
- Device snapshots, console pacing and operational logging are later M2 work.

Next cut: c18 memory devices, ROM loading, write protection and map tests.
