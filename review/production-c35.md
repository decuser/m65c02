# C35: RIOT transition navigator

Adds RIOT transition extraction to the 8,192-cycle bus trace with an API
that reports changed output pin levels, direction masks, and display digit/
segments. The analyzer pauses the shared CPU, lists transitions, and lets
the user navigate to the exact cycle in the waveform and CPU transaction
table. Both RIOT probes use the same CPU cycle counts. First ring sample
is baseline and cannot be classified as a transition. Tests cover two
independent RIOT port streams, direction changes, and same-cycle capture.

Validation: `make validate GO=../.toolchains/go/bin/go` and
`make validate-kim1 GO=../.toolchains/go/bin/go TESTS_DIR=../upload`.
