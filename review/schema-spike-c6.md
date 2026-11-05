# c6: Disposable schema spike follow-up

The spike remains disposable. Architecture 0.3 is not changed.

- The selected cycle entry is retained across the entire physical cycle, including PostBus mutations of PC or S.
- AD and DO are stored in CPU State, and pre-bus transfers populate the named destination latches. Pending is latched from those values, not recalculated in PostBus.
- At instruction boundaries, the next SYNC request is prepared, valid, and unresolved. Boundary preparation is atomic in this spike, not independently micro-steppable. That tradeoff requires explicit specification before production.
- The conditional LDA abs,X template has four- and five-cycle paths and reconverges at a prepared SYNC boundary.
- CMOS BRK clears D. Retained RMW Operand survives intervening resolved bus values.
- Snapshot tests compare restored execution against a separate uninterrupted reference trace; RDY reads perform distinct physical cycles.

## Remaining design questions

- String-dispatched teaching operations and dynamic slices are not a production schema. Use typed fixed entries, operands, annotations, and zero-allocation execution in M1.
- Selected cycle entries are snapshotted with immutable slice references in the spike. Production snapshots require a value-only representation.
- The precise dummy addresses and INC abs,X timing are unverified against SingleStepTests; provisional spike bus patterns are not compatibility assertions.
- Interrupt entry and RESET variants remain for Technical Specification 0.1; the four-template spike does not prove those paths.
