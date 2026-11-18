# C37: original KIM-1 SCANDS timing

The original 6530 ROM monitor boots normally and then a 4-byte RAM harness
invokes `JSR $1F1F`. The test records the first SCANDS opcode fetch, its
return to the RAM caller, and the RIOT port transitions that drive the six
LED digits. Exact results at 1 MHz:

- SCANDS entry fetch through return fetch: 4,231 cycles.
- JSR fetch through return fetch: 4,237 cycles.
- All six digits have positive LED dwell and a complete 0,1,2,3,4,5
  transition sequence exists in the recorded RIOT trace.

The six-cycle difference is the JSR itself; the measured total agrees with
Hans Van Slooten's Kimulator hardware note reporting 4,237 cycles. The test
is conditional on the two original external ROM files and does not modify CPU
timing or substitute host-side rendering for ROM execution.

Validation: general Go tests and `make validate-kim1` with original ROMs.
