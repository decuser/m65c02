# Dormann conformance fixtures (c14)

Upstream: `Klaus2m5/6502_65C02_functional_tests`, commit
`7954e2dbb49c469ea286070bf46cdd71aeb29e4b`.

Do not commit external binaries to the source tree. `make validate` runs each
test only when its corresponding environment variable points to a fixture.
All paths supplied below should be absolute, because `go test` runs in its
package directory.

## Three runnable fixtures

1. `M65C02_DORMANN=/absolute/path/to/upstream/bin_files`
   uses the upstream `6502_functional_test.bin` without modifications. Expected
   success PC `$3469`.
2. `M65C02_DORMANN_EXTENDED=/absolute/path/to/65C02_profile.bin`
   uses a rebuild of `65C02_extended_opcodes_test.a65c` with
   `rkwl_wdc_op = 2` (do not test Rockwell opcodes), `wdc_op = 1`
   (do not test WAI/STP), and `skip_nop = 0`. Expected success PC `$1C3C`.
3. `M65C02_DORMANN_INTERRUPT=/absolute/path/to/6502_interrupt_cmos.bin`
   uses a rebuild of `6502_interrupt_test.a65` with `D_clear = 1`
   (CMOS behavior). The test's feedback register is `$BFFC`, with bit 0
   asserting IRQ and bit 1 asserting NMI when written **high**. Expected
   success PC `$0719`; the later WAI/STP tests are manual and not executed.

Rebuild the last two with the upstream `as65` assembler after copying and
changing only the specified constants in the two source files:

```
as65 -x -qm -l65C02_profile.lst -o65C02_profile.bin 65C02_profile.a65c
as65 -x -qm -l6502_interrupt_cmos.lst -o6502_interrupt_cmos.bin 6502_interrupt_cmos.a65
```

The upstream package's `as65` is a **32-bit executable**, so a suitable 32-bit
runtime or user-mode emulator may be needed on modern Linux. The resulting
binaries are 65,526 bytes and begin at address `$000A`; the test adapter maps
their first byte at `$000A`, not `$0000`. Check the listing success addresses
after rebuilding; a change in assembler source/configuration requires updating
the test adapter's expected locations before relying on results.

The upstream prebuilt extended `.bin` does **not** match this profile: it tests
Rockwell BBR/BBS. Do not pass it as the extended fixture. The original interrupt
source defaults to `D_clear = 0` (NMOS behavior) and likewise needs a rebuild.

Run `make validate` with all three variables set to execute the independent
functional and interrupt checks. Full Synertek cycle conformance remains an
additional, separate test input controlled by `M65C02_CORPUS`.
