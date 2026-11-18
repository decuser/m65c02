# C36: complete retained-trace export

The analyzer provides CSV and plain-text downloads of up to 8,192 retained
cycles. Each request atomically pauses the producer under the panel lock and
snapshots the trace before writing the response. Both formats include CPU
and both 6530 pins, directions, and LED state, alongside decoded instruction
context from recorded opcode/operand reads. CSV is row-oriented; text adds
instruction boundaries. No fetches from current machine memory are used to
retroactively fill incomplete captures.

Validation: general Go tests, KIM-1 ROM tests, export CSV/text unit tests,
analyzer JavaScript syntax.
