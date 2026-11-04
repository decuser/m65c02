# Apple 65C02 Processor Laboratory
## Architecture Baseline 0.3 — FROZEN

**Frozen:** 2026-11-04  
**Implementation language:** Go  
**Target:** Apple IIe-compatible 65C02 (base CMOS instruction set)  
**Initial host:** SIMH-inspired console machine  
**Later, out of initial scope:** Visible Computer-inspired Wails application; enhanced Apple IIe system

This document incorporates the user-mediated architecture review with Claude, including the revised compatibility target and spike decisions. It specifies contracts and boundaries, not a Go representation. Technical Specification 0.1 follows a disposable schema spike.

### 1. Purpose and nonnegotiable principles

Make a **cycle-accurate** 65C02 emulator useful as a teaching and experimentation instrument, then as the CPU of a full Apple IIe host. One authoritative execution model supports running, instruction stepping, cycle stepping, and **pausing between explanatory microoperations inside a physical cycle**. The CPU is independent of memory mappings, terminal, files, GUI, logger and profiler. No separate fast interpreter or teaching CPU. Profiling/tracing observe execution without controlling it. State is explicit and snapshotable; no hidden goroutine, closure, or call-stack continuation carries sequencer progress.

### 2. CPU compatibility profile and conformance

Target the Apple IIe-compatible **base CMOS 65C02** instruction set. Include BRA, STZ, TRB/TSB, PHX/PHY/PLX/PLY, accumulator INC/DEC, CMOS BIT and zero-page indirect modes, and JMP (abs,X). Exclude Rockwell RMB/SMB/BBR/BBS and WDC WAI/STP. Do not add a runtime CPU-variant selector in M1. Documented NOP forms in the selected profile are implemented, including their operand lengths, cycles and bus behavior where tested.

A declarative opcode record carries opcode, mnemonic, length, addressing mode, access class, cycle template, conditional cycle rules, supported/NOP/excluded classification and **optional notes** documenting specific compatibility decisions. No mandatory per-opcode provenance/evidence matrix.

**Conformance rule:** Use one selected SingleStepTests 65C02 test set, excluding opcodes outside the profile, as the practical instruction/cycle oracle. Use Dormann's 65C02 functional test as complementary semantic verification. Where tests and the intended profile disagree, follow the selected corpus unless a concrete reason for a deliberate exception is written down. Do not modify the core merely to satisfy a test for an excluded opcode. Preserve actual bus transaction ordering and dummy accesses, not only final registers or total cycles. Record any observed corpus discrepancy and the choice in an opcode note or concise exception list; do not invent expected traces when the corpus is unavailable.

For `INC abs,X` and `DEC abs,X`, use the selected corpus's timing, including the crossing and noncrossing cases; record a discrepancy if one arises rather than undertake chip-provenance research. RDY remains a supported stall input in the contract; detailed write-stall electrical timing and unusual interrupt edge cases may be deferred beyond the initial M1 gate. Ordinary program-visible interrupt behavior remains in M1.

### 3. Conceptual processor blocks

Nine conceptual hardware blocks, **not** nine mandated packages: (1) A/X/Y/S registers, (2) PC, (3) P/status, (4) ALU, (5) opcode register/decode, (6) cycle timing/control, (7) operand/address/data latches, (8) bus buffers/interface, (9) reset/IRQ/NMI control. The core's plain-data state includes architectural registers, opcode, sequencer/template position, named microoperation cursor, temporary latches, sampled/pending interrupts, relevant bus signals, and stored resolved-cycle response. This is an explanatory behavioral datapath, not a claim of transistor-level chip netlist identity. Label internal AD/DL/etc. explanatory where not independently documented.

### 4. Execution hierarchy and instruction boundary

**Machine time → physical CPU-cycle phases → teaching microoperations.** The host owns machine time and decides when the CPU participates; an eventual DMA cycle may grant the bus to another master. The headless host is initially simple.

- `StepMicro`: one named teaching transfer or operation.
- `StepCycle`: one complete emulated CPU clock cycle using the same sequencer.
- `StepInstruction`: complete one instruction and stop at **Present of the next cycle with SYNC asserted, before Resolve**. Interrupt-entry exceptions are determined by the selected compatibility behavior, not assumed to assert SYNC like a normal opcode fetch.
- `Run`: repeatedly advance the same sequencer, paced by host or unlimited.

**Software convention:** architectural register and flag results are represented as completing inside the modeled instruction's final cycle, even where physical silicon overlaps work with the next opcode fetch. This is an explicitly declared emulation convention, not a silicon claim, and must preserve externally visible bus ordering/timing.

Example teaching vocabulary: `T: PC -> AD`, `READ: MEM -> DL`, `T: DL -> IR`, `INC: PC`. Bus-facing correctness is authoritative; explanatory transfers may divide a physical cycle into inspectable substeps. Pausing does not advance machine time, perform I/O again, or complete another physical cycle.

### 5. Phase cursor and resolve-once bus protocol

Every in-progress CPU cycle carries explicit phase cursor **`PreBus(n)`, `Resolved`, `PostBus(n)`** (with an idle/ready boundary as needed). The exact Go encoding is intentionally deferred. Result of Resolve (incoming data and sampled RDY/IRQ/NMI/RESET) is retained in CPU state. Resuming from `Resolved` or `PostBus(n)` **never calls the bus again for that resolution**.

1. **PreBus/Present:** Run individual pre-bus microoperations that form/present address, direction, SYNC and data-out. Pre-bus ops must be idempotent and derived from committed state; they may assign explanatory latches, but must not irreversibly modify architectural registers or emit I/O. Replaying them after a stall is harmless.
2. **Resolve:** Machine processes **one actual bus interaction for this machine cycle** and samples external signals. Resolve is called exactly once for a particular transaction; a real repeated stalled cycle is a *new* machine cycle and a *new* Resolve, recorded in traces and permitted to repeat device side effects. The bus may return a host-defined value for undriven/floating reads.
3. **PostBus/Commit:** Consume the stored result and advance approved post-bus microoperations. Architectural mutations (PC/S increments, ALU results, P flag changes) occur post-bus, not as replayable PreBus operations. For a stall, the sequencer holds, skips normal state mutation and restarts its bus cycle on a subsequent machine tick. If a particular real sequence cannot obey this rule, justify and document a **local** staging exception, not a global shadow-CPU layer.

**Time:** One real machine-cycle transaction consumes machine time **even while CPU remains stalled**. A teaching pause partway through a cycle consumes **no additional machine time** and advances **no devices**. Future host timebase is not required to tick while the user inspects a paused mid-cycle state. Clock stretching is a machine scheduler duration, not an additional CPU microoperation.

**Ordering caution:** Exact electrical sampling order, RDY write behavior, bus ownership/DMA and unusual interrupt edge cases are not prerequisites for M1 unless exercised by the compatibility tests; the interface retains capacity for them. A convenient `Clock()` completes the phases for the headless harness, but **the phases remain separately invocable** for microstepping and tests.

Supervisor `Peek` must be noninvasive and must not synthesize a CPU bus transaction; `Poke`/deposit is an explicit administrative mutation, may be unsupported on devices, and is not an emulated write. The CPU holds no pointer into machine RAM.

### 6. Cycle sequencer

Use reusable cycle **templates keyed by addressing mode × access class** (read, write, RMW, stack, control-flow, implied/register), with explicit conditional transitions for page crossings, branches, decimal timing and interrupts. An operation slot invokes opcode-specific register/ALU semantics. Exceptional sequences (BRK, RTI/RTS, JSR, indirect JMP, reset, interrupt entry) may be bespoke. The desired absence of opcode-specific control-flow `if`s outside operation slots is a testable design goal, **not** a reason to distort real hardware behavior.

Templates list named pre-bus ops, one Resolve point, and named post-bus ops. Template phase positions are true modeled bus-cycle positions; latch labels and transfers are pedagogical descriptions. Do not equate BRK with IRQ/NMI merely because the sequences share steps; use tested compatibility behavior.

### 7. Host machine and supervisor

Headless M2 host: 64KB bus address space backed by file-loaded memory with a simple mapped nonblocking status/data console. Guest input can be made available at cycle-stamped positions. CPU never blocks for terminal input; guest may poll. Supervisor console and guest console are logically distinct and cannot accidentally consume one another's input. Runner may idle the host thread without changing simulated cycles; host sinks must eventually be suppressible during replay.

SIMH-inspired supervisor controls load/save, examine/deposit, registers, reset, run, stop, step cycle/instruction (and teaching microstep if exposed), breakpoint, status, speed, profile, trace and log configuration. Clear help, abbreviations/completion and command history are desirable. Supervisor commands do not drive a second interpreter and are not guest I/O. An eventual Wails UI issues equivalent engine requests rather than parsing CLI text; UI update suspension during run and synchronous single-step engine calls are future adapter details.

Host speed choices: 1×, 2×, 4×, 8×, 16×, unlimited, using nominal ~1.023 MHz for the simple headless host. Real Apple IIe average clock is slightly slower due to video-related stretching. Host pacing does not enter CPU semantics.

### 8. Four distinct observational/recording concerns

- **Operational logging (Harpy-inspired):** structured, low volume, persistent optional record of **operator commands and outcomes, configuration/load/save/run/reset, warnings and errors**. Not execution trace. Logging failure must not affect emulator results. Do not routinely log CPU cycles or memory accesses.
- **Profiling (Aiki/DTrace-inspired, no Aiki dependency):** bus-cycle stream is the mandatory push observation boundary; optional observer guard with negligible off-cost and no per-cycle allocations in counting mode. Collect opcode/mode/cycle histograms, per-PC attribution, branches, page-cross behavior, bus activity. Semantic simulated work and measured host cost remain distinct. Internal teaching state and current named microop are obtained by pull snapshots, not independently reconstructed by profiler logic. Add internal push probes only for a proven gap.
- **Execution trace:** optional filtered ordered cycle/instruction event record, with bounded/explicit retention; not syslog.
- **History/reversal:** future state-preserving facility, separate from trace/log/profile. CPU and all consequential device state snapshotable or declared limited, external input cycle-stamped, replay suppresses irreversible host sinks. Choice of snapshot/replay vs journal vs hybrid intentionally deferred.

The M1 CPU/bus interface must establish the observation boundary but need not implement profiling consumers. Observers must not change execution semantics.

### 9. Tests and invariants

M1 acceptance includes:
- All supported base CMOS instructions and documented NOP forms, with SingleStepTests 65C02 as the selected cycle/transaction corpus and Dormann as a functional semantic test. Explicitly exclude Rockwell bit operations and WAI/STP. Report the selected corpus revision and any deliberate, documented exceptions.
- Decimal arithmetic, stack, branches, CMOS differences, reset/BRK/IRQ/NMI, SYNC, dummy bus accesses, RMW sequences, page crossings and conditional cycles as exercised by the suite.
- **Mid-cycle snapshot round-trip:** snapshot at every phase/microop cursor, restore, resume and match uninterrupted execution with **no extra Resolve**.
- **Stepping equivalence:** microsteps, whole cycles and instructions reach identical results and bus traces under equivalent inputs.
- **Basic RDY read-stall tests:** the same architectural result after inserted read stalls in deterministic RAM, preserving actual repeated bus interactions. Defer exhaustive write-stall electrical tests and unusual interrupt sampling edge cases as needed; do not remove the stall input.
- After profiling arrives, observer-on/off results match CPU and relevant device state, with measured overhead.

### 10. Milestone order and gates

**Before Technical Specification 0.1: disposable 4-template schema spike**, separately described in `schema-spike-mandate.md`. Its code is discarded; test results and successful contracts become specification inputs.

**M1: CPU + bus + headless conformance harness.** Clock, Present/Resolve/Commit, microoperation table schema, snapshot state, opcode table, baseline bus observation contract, SingleStepTests and Dormann harnesses. Gate: selected-corpus conformance with documented exceptions, mid-cycle snapshot and stepping equivalence, plus basic RDY read-stall coverage.

**M2: operable machine.** 64KB file-backed memory, nonblocking guest console, SIMH-style supervisor, speed control, structured operational logging. Gate: deterministic host behavior, accurate control/inspection and device separation, load/save, correct operator logs.

**M3: profile and trace.** DTrace-inspired bus-stream consumer, fixed-count aggregates, trace, observer invariance, overhead benchmarking. Gate: attribution correctness, same execution with observers on/off, measured overhead.

After M3 (not prerequisites): reversibility, Wails Visible Computer front end, complete Apple IIe ROM/video/audio/disk/peripherals.

### 11. Reference roles and compatibility decisions

**Behavioral oracle:** one selected SingleStepTests 65C02 set for cycle/bus behavior, complemented by Dormann's 65C02 functional test for semantic correctness. Explicitly filter out Rockwell RMB/SMB/BBR/BBS and WDC WAI/STP. The selected corpus wins by default; a deliberate exception needs a short, concrete explanation. Neither an archival provenance hierarchy nor a classification-aware transfer matrix is required.

**Context:** Apple's *Apple IIe Technical Reference Manual* and Sather's *Understanding the Apple IIe* explain machine behavior and IIe constraints; the NCR appendix is an optional contemporary reference, not the target chip specification. AppleWin is a useful implementation comparison, not automatically a bus-cycle oracle. *The Visible Computer: 6502* informs teaching vocabulary without overriding bus behavior. Aiki, Harpy, V6/PDP-11 and Nezha inform software structure only.

### 12. Freeze and deferred choices

**Frozen:** goals, compatibility profile, selected-corpus-first rule, one sequencer, microstep pause, three-phase bus cycle with retained Resolve result, phase cursor/idempotent PreBus constraint, host-owned time, side-effect separation, nonblocking console, four observation categories and M1→M2→M3 order.

**Not frozen:** concrete Go struct layout; detailed template encoding; undocumented or untested bus edge cases; profiling report syntax; console address assignment; snapshot/history strategy; UI visual layout.

Prototype the schema, report contradictions and bring genuine architectural obstacles back for review before changing this baseline.
