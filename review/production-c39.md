# C39: Executed CPU micro-operations on the bus-cycle timeline

- CPU microstep instrumentation is opt-in through a nil-by-default callback. Each event includes the state before and after the executed operation, its phase, and the operation name.
- A bounded 32,768-event ring stores real pre-bus, resolve, and post-bus transitions. Events are labeled with their corresponding physical cycle number.
- The KIM-1 analyzer adds a MICRO TRACE toggle. The selected-cycle panel lists actual events and changed CPU registers/latches in execution order.
- Enable/disable operations are protected by the existing panel server mutex. Starting a capture clears prior microstep history. Cold power-on reconnects observation when enabled.
- CPU execution, bus transaction capture, RIOT capture, and existing exports retain their established semantics. Reverse execution and microsteps as waveform lanes remain future work.

## Validation

- Embedded analyzer JavaScript syntax checked with Node.js.
- `make validate` passed.
- Micro trace ring test checks same-cycle phase alignment and bounded retention.
- `make validate-kim1` passed with the original ROMs.
