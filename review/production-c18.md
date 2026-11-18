# c18: RAM, ROM and file-backed images

Proposed commit: `Add mapped RAM and ROM components with image persistence`

## Scope

- `internal/machine/memory.go`: bounded RAM with administrative image loading,
  file load/save and independent snapshot/restore; immutable ROM from memory or
  a file, ignoring writes from the guest bus.
- `internal/machine/backplane.go`: optional `Sized` check to prevent attaching
  more address space than the device can decode.
- `internal/machine/memory_test.go`: decoding and execution of actual ROM code
  that writes to RAM, ROM immutability, load/save, snapshot alias prevention,
  invalid sizes and mapping overflows.

## Validation

Ran `gofmt` and `make validate GO=../.toolchains/go/bin/go` with workspace
Go 1.24.10: **pass**, including build, vet, all local regressions and machine
tests. No M1 CPU files changed. External Synertek and Dormann fixtures were
not configured in this workspace and were not rerun.

## Boundaries

- File load/save is explicit administration; RAM does not automatically write
  to a host file on every guest write. A host supervisor will own persistence.
- `ROMFromFile` accepts a byte image; mapping decides its location. ROM bus
  writes are ignored, though actual EPROM/programmer behavior is out of scope.
- The default $FF floating read remains a backplane policy, not RAM behavior.
- No ROM mirroring/banking, address-line aliasing, or KIM-1-specific layout yet.
- Machine/device snapshot coordination remains a supervisor responsibility.

Next: c19 6530 RRIOT design and implementation following chip-specific
memory/register/timer documentation and KIM-1 address mapping.
