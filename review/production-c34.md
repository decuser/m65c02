# C34: KIM-1 RIOT logic-analyzer probes

The existing CPU bus capture now records an electrical snapshot of both
6530 RIOT chips at each bus cycle, including external PA and PB pin levels,
DDRA, DDRB, and the decoded front-panel digit and segment bits. A probe
selector in the existing analyzer switches between CPU, 6530-002 (display),
and 6530-003 (application) without changing the cycle window or machine state.
The RIOT ports can be viewed as grouped hex bytes or individual bit waveforms.
The same cycle selection, history paging, and FOLLOW behavior applies to all
three probes.

This samples the emulator's driven pin levels at the CPU-cycle boundary. It
does not imply analogue electrical simulation or independently sampled clock
edges. The Φ2 waveform remains a synthetic ideal half-cycle representation.

Validation: `make validate GO=../.toolchains/go/bin/go` and
`make validate-kim1 GO=../.toolchains/go/bin/go TESTS_DIR=../upload`.
