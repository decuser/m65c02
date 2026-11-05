# m65c02 Technical Specification 0.1

**Status:** implementation proposal for review, 2026-11-04  
**Governing architecture:** Architecture Baseline 0.3 (frozen)  
**Language:** Go 1.16, standard library first  
**Inputs:** disposable spike c5–c7, especially `review/schema-spike-c6.md`, and the user/Claude review.

This document specifies the **production M1 CPU**, not the disposable spike's source representation. No Apple IIe motherboard, Wails interface, operational logging system, or profiling consumer is required in M1. The existing spike and monitor remain demonstrators until retired; they are not a production implementation to copy wholesale.

## 1. Scope and acceptance

Implement one Apple IIe-compatible **base CMOS 65C02** engine that can be executed by instruction, physical cycle, and explanatory microoperation without alternate interpreters. Supported profile includes the CMOS additions specified in Architecture 0.3; excludes Rockwell RMB/SMB/BBR/BBS and WDC WAI/STP. Documented profile NOPs need correct lengths and cycles.

The production gate is: deterministic local tests; selected SingleStepTests 65C02 vector set (once obtainable, filtering excluded opcodes); Dormann 65C02 functional test; step/snapshot invariants; basic RDY read stalls; and an observable finite program. Missing external vectors are **unavailable**, never passes. The chosen corpus is the default compatibility authority; note deliberate exceptions briefly in opcode notes. The NCR documentation is reference context, not an additional gate.

## 2. Project placement and dependency direction

Suggested eventual paths (create only as needed):

```text
internal/cpu/          CPU state, execution, table definitions and CPU tests
internal/bus/          deterministic RAM bus and bus tests; no CPU dependence required
internal/conformance/ external vector adapter, profile filtering, result summaries
cmd/m65c02/           thin production headless entry point (later M2 integration)
testdata/             compact committed local fixtures; externally supplied corpora tracked separately
```

CPU imports **no** host, terminal, UI, logging, profiler, filesystem or machine package. Host/supervisor owns memory and the physical bus and calls CPU. No forced nine-package decomposition: Architecture 0.3's nine blocks are explanatory, not packages. Existing `internal/spike` and `cmd/m65c02-monitor` remain disposable; remove them in an explicit later cut once replacement demonstrations run.

## 3. Core API sketch

The signatures below define responsibilities and invariants, not immutable spellings or exported Go API. Any adjustment preserving these contracts can be reviewed during M1.

```go
type Request struct {
    Address uint16
    DataOut byte
    Write   bool
    Sync    bool
}
type Response struct {
    DataIn byte
    Ready  bool
    IRQ    bool
    NMI    bool
    Reset  bool
}
type Bus interface {
    Transfer(Request) Response
}

type CPU struct { state State }
func (c *CPU) StepMicro(bus Bus) (MicroEvent, error)
func (c *CPU) StepCycle(bus Bus) error
func (c *CPU) StepInstruction(bus Bus) error
func (c *CPU) Snapshot() State
func (c *CPU) Restore(State) error
func (c *CPU) View() View
```

`StepCycle`, `StepInstruction`, and continuous run **all drive the same microstep sequencer**. The host loops to run/pause, handles time and pacing, and controls externally injected signals; the CPU does not start goroutines, sleep, or own a terminal. A `Clock` alias for a physical cycle is optional. `View` is a read-only value of authoritative state, never a second decoder of pending bus intent. `Bus.Transfer` is the **only** modeled physical bus interaction; a host `Peek` is a separate, noninvasive API, and `Poke` is an explicit administrative mutation. Bus error strategy must be explicit in the implementation (either transfer returns an error or the host reports a fatal bus fault); never silently synthesize a successful transfer.

Bus ownership/DMA arbitration is a host concern deferred beyond initial M1. A request's visible read/write/sync and data-out fields do not themselves increment machine time; a performed transfer does.

## 4. Value-only processor state and cursor

`State` is a plain value with no pointers, slices, maps, closures, retained array views or host object references. It contains at least:

- Architectural `A`, `X`, `Y`, `S`, `P`, `PC`; `IR` (last fetched opcode); sampled interrupt/control state, including NMI-edge bookkeeping where required.
- Explicit scratch latches: `AD` (presented address), `DO` (data out), `DL` (last resolved data), address low/high, base/effective address, **retained operand** for RMW, and PC/status bytes or temporary values needed across BRK/JSR/interrupt cycles. All values needed by future cycles have a named latch.
- Immutable table *identifiers* `TemplateID`, `CycleID` (or indices) and `OperationIndex`, never a template pointer or slice. Instruction/entry variant and conditional decision bits live in state if used by subsequent cycles.
- Phase (`PreBus`, `Resolved`, `PostBus`) and separate explicit validity of current selected cycle and pending request; `Pending` is the fully latched request, `ResolvedResponse` is the stored response with a validity bit. Counts of physical transfers/cycles and completed instructions.

**DL is bus-owned:** each successful `Resolve` may overwrite it, including dummy reads. Retained operands and address bytes must never depend on DL surviving later cycles. Avoid redundant mutable copies of the same bus request that can disagree; if both AD/DO and Pending exist, Pending is built once from those latches and is frozen until the cycle retires.

`Snapshot` can be taken at every microoperation boundary, including after Resolve and during PostBus. `Restore` validates phase, indices, table identifiers and validity flags before accepting state; no transfer on restore. Immutable instruction/template tables are shared code, not snapshot payload. Restored execution must be byte-for-byte/trace-equivalent to a fresh uninterrupted run given equal future external inputs. Snapshot of CPU alone does **not** snapshot RAM or devices: full-machine resume tests must restore the associated bus/device state and input schedule too.

## 5. Bus-cycle state machine

One physical cycle is described by a selected static cycle entry and progresses as follows:

1. **Select:** Choose the entry exactly once for this cycle from the current template/transition state. Store its IDs in `State`. No architectural mutations or bus access. The entry stays selected through all PostBus operations.
2. **PreBus(n):** Execute typed explanatory transfers. These really assign AD and DO and other presentation-only latches, not just labels. They are deterministic/idempotent from committed state and selected entry; no I/O, PC/S/P mutation, or machine-time advance.
3. **Present:** Construct and latch `Pending` from AD, DO and the entry's `Write`/`Sync` controls. Set `PendingValid`; do not call the bus. Do not rebuild Pending from subsequently mutated state. The cycle is now prepared and inspectable.
4. **Resolve:** Invoke `Bus.Transfer(Pending)` **once**. Store complete response and mark it resolved. One actual transfer consumes one physical cycle of machine time even if RDY holds the CPU. For ordinary accepted reads copy `DataIn` to DL. Mid-cycle resume from Resolved/PostBus never calls Transfer a second time.
5. **PostBus(n):** Apply the selected entry's typed commit operations, one at a time, reading saved latches/response. These may alter registers, flags, PC/S and template progress, but **never** redo Resolve or select a different current entry.
6. **Retire:** Evaluate an explicit end/next transition encoded by the cycle entry; clear old pending/response validity, select the next entry when needed. A held read performs a *new* physical cycle with a new Resolve and no normal post-bus architectural effects.

The cursor exposes `PreBus(n)`, `Resolved`, and `PostBus(n)` as in Architecture 0.3; an internal prepared/present condition is distinguishable with validity flags, not a different execution engine. `StepMicro` advances exactly one named operation or Resolve. Selection/presentation and boundary bookkeeping can be internal transitions but must be deterministic and cannot silently perform a transfer. Define step boundaries and test them rather than reporting an unexecuted operation as executed.

### Instruction boundary

After the final committed cycle of an ordinary instruction, prepare **the next opcode fetch** with `SYNC=true`, valid `Pending`, phase at Resolve, and **no** physical transfer of that fetch. `StepInstruction` stops there. That prepared fetch must already reflect the current committed PC; it cannot be stale. Atomic boundary preparation (running the next fetch's idempotent pre-ops without pausing among them) is acceptable in 0.1, but it is declared explicitly and must be reflected accurately in `View` and stepping tests. Interrupt-entry boundaries must follow the implemented profile rather than be mislabeled as ordinary opcode SYNC fetches.

### RDY

On a read response with `Ready=false`, retire that *physical transfer* without committing the normal cycle, and re-present the held operation as a **new** physical cycle. Repeated accesses are observable, including device side effects. Tests distinguish a resumed mid-cycle Resolve (never repeated) from genuine repeated RDY-stall transfers (each resolved once). RDY/write electrical nuances beyond the selected compatibility suite remain deferred; do not assert they are modeled faithfully merely because the interface has `Ready`.

## 6. Static cycle templates and teaching operations

The production schema uses fixed immutable opcode descriptors and cycle-entry tables selected by **addressing mode × access class**, with bespoke paths where needed (stack/interrupt/reset/indirect jumps). Each descriptor contains opcode, mnemonic, addressing mode, length, class, template ID, support classification, and optional short compatibility note. One table owns instruction completion; remove separate `complete()` versus `describe()` length logic.

Each cycle entry provides:

- `PreOps` and `PostOps` as bounded, fixed arrays of **typed** operations with explicit operation count; no runtime slices or string matching.
- Direction/SYNC and a typed address/data-source selection or presentation operations.
- A typed `Next` transition (`NextCycle`, `Conditional`, `EndInstruction`, `EnterInterrupt`, etc.) whose branch condition uses committed state/explicit condition bits; no ad-hoc opcode-switch timing in the main stepper.
- A **separate annotation ID** for user-facing cycle explanation, e.g. opcode fetch, operand read, indexed dummy, RMW read, RMW retry, stack push, vector low. Labels are not executable semantics.

An operation is a small code plus bounded operand selectors (register/latch/status bit/constant). Example op classes: `Transfer(source,dest)`, `Inc(register)`, `Dec(register)`, `ALU(kind,source,dest)`, `SetFlags(mask/source)`, `CombineAddress`, and `SampleControls`. The closed enum can include narrowly scoped special operations; adding one requires a tested semantic reason. `DONE` and parenthetical comments are not operations. `BUS READ/WRITE` is a cycle property and Resolve, not a free-form string microop. Human-readable transfer labels are formatted from the same enum and operands; views may add the cycle annotation without asserting physical transistor internals.

Dynamic conditions are tested with **LDA abs,X** cross/noncross: one path performs a page-cross penalty cycle, the other skips it; both join at exactly the prepared next-instruction boundary. For indexed store and RMW, do **not** reuse load's conditional rule simply because their addressing modes match. Bus address selection, especially CMOS dummy addresses, must follow the selected corpus when available. c6's dummy patterns and INC timing are **provisional** and not promoted to production truths.

The performance target is **zero allocations on the steady-state CPU path** with observation disabled, established by `testing.AllocsPerRun` after correctness tests, not by assuming the spike's allocation count transfers. Template storage and names are static; no per-cycle closures or formatting.

## 7. Interrupts, BRK and reset

Provide a shared **parameterized entry model** where behaviors genuinely coincide, with explicit variation by `BRK`, `IRQ`, `NMI`, `RESET`:

| Parameter | Required distinction |
|---|---|
| Entry cause and sampling | BRK is an opcode; IRQ/NMI are asynchronous; RESET has its own control path and priority |
| Signature / PC advance | BRK consumes its signature byte, others follow their selected-entry bus behavior |
| Pushed P and B representation | BRK pushes B set; IRQ/NMI push B clear; B is a synthesized stack-image distinction, not a freely persistent architectural flag |
| Vector | BRK/IRQ `$FFFE–FFFF`, NMI `$FFFA–FFFB`, RESET `$FFFC–FFFD` |
| Stack accesses | BRK/IRQ/NMI write stack; RESET must not perform those writes merely to share a template |
| Flags | CMOS interrupt entry sets I and clears D; saved P image and exact mutation point follow tested cycle ordering |

Retain the necessary PC bytes and other values across multiple bus cycles. Exact IRQ/NMI sampling edges, RESET bus shape and precedence need direct production tests and the chosen corpus where it covers them. **Do not** treat the spike's BRK sequence as proof of IRQ/NMI or RESET timing. The host's interrupt lines/edges enter through the response/control sample contract, not via hidden asynchronous changes of the CPU state.

## 8. Testing, conformance and visible proof

Required local M1 tests:

1. Opcode descriptors: unique 256-byte map, supported/excluded/NOP profile, lengths and allowed template transitions; no accidental execution of excluded opcodes.
2. Bus traces: exact read/write order, SYNC, values and dummy cycles for small fixtures, distinguishing provisional expectations from corpus-verified ones.
3. Step equivalence: same input schedule produces same registers, RAM and ordered bus trace under micro/cycle/instruction stepping; include conditional-cross and noncross branches.
4. Snapshot: for each reachable phase/microop cursor, compare a restored continuation **against an independently run uninterrupted reference**; count Transfer calls and include read-to-clear side effects to catch duplicate resolves.
5. RDY: hold opcode, operand, indexed/dummy and RMW read accesses; compare the full trace including inserted physical transfers, with device differences permitted for side-effecting reads.
6. BRK/IRQ/NMI/RESET: stack bytes and ordering, PC/S, I/D changes, vector selection, and reset's lack of real stack writes.
7. Performance: `testing.AllocsPerRun == 0` for steady-state stepping; `make bench` records throughput and allocation cost on named local hardware, not a release gate for a specific ns/op.

External adapter (`internal/conformance`) imports corpus files as **test inputs**, not CPU behavior. Pin one 65C02 SingleStepTests revision and report passed, failed, excluded, unavailable, and documented exceptions separately. Dormann is run as a semantic end-to-end test and must be identified by variant/revision. Incomplete downloads or mismatched variants are not silent success. `make test` remains quick; `make validate` runs all available conformance/property checks and prints any missing-corpus status candidly.

**Visible success in M1:** Maintain a finite small program that produces a checkable RAM outcome and a bus/cycle explanation using the *same* CPU state and events. The existing `make run` disposable monitor is a temporary teaching demonstrator, not an interface to freeze. The eventual supervisor/visible computer reads `View`/event records; it never recomputes AD/DO, instruction timing or microoperations from a second model. Terminal history and elaborate UI are M2/later concerns.

## 9. Cut sequence and review points

- **M1-a:** establish `internal/cpu` state, typed microops, immutable tables and deterministic bus tests; demonstrate one opcode and prepared unresolved SYNC. No broad opcode claims.
- **M1-b:** implement the five spike opcodes in production form, both indexed crossing paths, BRK and conditional transitions; prove step/snapshot/bus invariants and zero-allocation stepping. Retire the disposable spike only after the replacement demo works and explicitly report deleted paths.
- **M1-c:** expand coherent instruction/addressing-mode groups, interrupt/reset entry, decimal arithmetic and documented NOPs; continuously test traces and finite demos.
- **M1-d:** integrate SingleStepTests and Dormann once available; document any compatibility exceptions and run the full `make validate` gate. An unavailable corpus may postpone *M1 completion*, not development of its independently testable machinery.

Only after M1 acceptance proceed to **M2** (64KB headless host, nonblocking guest console, supervisor, speed and operational logs) then **M3** (profile/trace consumers). Reassess the concrete type names and table encoding during M1-a review, not the frozen Architecture 0.3 contracts.

## 10. Decisions carried forward and unresolved

**Settled by c5–c7:** one sequencer is feasible; actual AD/DO latches are needed; selected cycle entry must be stable; DL and long-lived RMW operand must be distinct; LDA abs,X requires a conditional cycle; snapshots require value-only positions; pre-resolved SYNC must be visible; post ops cannot dispatch by English labels.

**Not yet proven:** exact CMOS indexed dummy bus addresses and INC abs,X cycle shape against the selected corpus; asynchronous interrupt sampling details; all-opcode conformance; production zero-allocation behavior; full-machine snapshot/replay. These are implementation and conformance questions, not reasons to reconstruct a particular historic chip.
