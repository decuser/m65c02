# Apple 65C02 Processor Laboratory
## Architecture Baseline 0.2 — AMENDED AND FROZEN

**Frozen:** 2026-11-04  
**Implementation language:** Go  
**Historical target:** Apple-documented 65C02 profile, NCR reference processor  
**Initial host:** SIMH-inspired console machine  
**Later, out of initial scope:** Visible Computer-inspired Wails application; enhanced Apple IIe system

This document incorporates the user-mediated architecture review with Claude, including the final authority and spike decisions. It specifies contracts and boundaries, not a Go representation. Technical Specification 0.1 follows a disposable schema spike.

### 1. Purpose and nonnegotiable principles

Make a historically grounded **cycle-accurate** 65C02 emulator useful as a teaching and experimentation instrument, then as the CPU of a full Apple IIe host. One authoritative execution model supports running, instruction stepping, cycle stepping, and **pausing between explanatory microoperations inside a physical cycle**. The CPU is independent of memory mappings, terminal, files, GUI, logger and profiler. No separate fast interpreter or teaching CPU. Profiling/tracing observe execution without controlling it. State is explicit and snapshotable; no hidden goroutine, closure, or call-stack continuation carries sequencer progress.

### 2. Historical CPU identity and behavior

Target the Apple-supported CMOS instruction profile for the enhanced IIe, using the NCR 65C02 as behavioral reference, **not** modern W65C02S semantics by default. Include documented common CMOS instructions such as BRA, STZ, TRB/TSB, PHX/PHY/PLX/PLY, INC/DEC accumulator, CMOS BIT and zero-page indirect modes, and JMP (abs,X). Rockwell RMB/SMB/BBR/BBS and WDC WAI/STP are not in the target supported profile. Do not infer a single physical supplier across all enhanced IIe machines. There is no runtime CPU-variant selector in M1.

A declarative opcode record carries value, mnemonic, length, mode, access class, cycle template, conditional cycle rules, supported/undefined classification, **source citation/provenance**, evidence category and notes. Documented NCR invalid-opcode NOP behavior is eligible for implementation as documented. Truly unknown opcodes stop with explicit `UnresolvedOpcode` instead of fabricated behavior.

**Physical-behavior precedence:** The target is the **physical behavior of the reference processor**. Apple documentation defines the supported instruction profile and is primary historical evidence; NCR documentation gives processor-specific semantics/signals/timing. Verified **target-variant hardware** measurements override conflicting printed behavior, with the discrepancy recorded as an erratum. Transferred evidence from WDC, Rockwell, NMOS, or other CMOS variants does not override target-specific documentation automatically. If transfer evidence conflicts and no direct target-chip proof exists, behavior remains **Unresolved**; use the documented interpretation **provisionally**, with its classification visible. An Apple-authored comparison and an NCR datasheet reproduced within the same Apple manual are not two independent physical measurements: preserve their source provenance and relationship.

The `INC abs,X` and `DEC abs,X` cycle issue is **Unresolved**. Apple's Table A-1 reports six cycles for CMOS, whereas reports of seven fixed cycles concern other CMOS variants. Six plus one on page crossing is a *hypothesis*, not an established NCR fact. The first documentary check is the NCR appendix's own timing entry and notes, with/without page crossing; only then compare transferable corpus traces. Never alter the core to make a WDC-only vector pass.

RDY on write cycles is documented for NCR as stopping the CPU, but the exact bus presentation/data during stalled writes still requires evidence. Interrupt edge timing and all undefined-opcode details are classified per evidence rather than assumed.

### 3. Conceptual processor blocks

Nine conceptual hardware blocks, **not** nine mandated packages: (1) A/X/Y/S registers, (2) PC, (3) P/status, (4) ALU, (5) opcode register/decode, (6) cycle timing/control, (7) operand/address/data latches, (8) bus buffers/interface, (9) reset/IRQ/NMI control. The core's plain-data state includes architectural registers, opcode, sequencer/template position, named microoperation cursor, temporary latches, sampled/pending interrupts, relevant bus signals, and stored resolved-cycle response. This is an explanatory behavioral datapath, not a claim of transistor-level NCR netlist identity. Label internal AD/DL/etc. explanatory where not independently documented.

### 4. Execution hierarchy and instruction boundary

**Machine time → physical CPU-cycle phases → teaching microoperations.** The host owns machine time and decides when the CPU participates; an eventual DMA cycle may grant the bus to another master. The headless host is initially simple.

- `StepMicro`: one named teaching transfer or operation.
- `StepCycle`: one complete emulated CPU clock cycle using the same sequencer.
- `StepInstruction`: complete one instruction and stop at **Present of the next cycle with SYNC asserted, before Resolve**. Interrupt-entry exceptions are evidence-backed rather than assumed to assert SYNC like a normal opcode fetch.
- `Run`: repeatedly advance the same sequencer, paced by host or unlimited.

**Software convention:** architectural register and flag results are represented as completing inside the modeled instruction's final cycle, even where physical silicon overlaps work with the next opcode fetch. This is an explicitly declared emulation convention, not a silicon claim, and must preserve externally visible bus ordering/timing.

Example teaching vocabulary: `T: PC -> AD`, `READ: MEM -> DL`, `T: DL -> IR`, `INC: PC`. Bus-facing correctness is authoritative; explanatory transfers may divide a physical cycle into inspectable substeps. Pausing does not advance machine time, perform I/O again, or complete another physical cycle.

### 5. Phase cursor and resolve-once bus protocol

Every in-progress CPU cycle carries explicit phase cursor **`PreBus(n)`, `Resolved`, `PostBus(n)`** (with an idle/ready boundary as needed). The exact Go encoding is intentionally deferred. Result of Resolve (incoming data and sampled RDY/IRQ/NMI/RESET) is retained in CPU state. Resuming from `Resolved` or `PostBus(n)` **never calls the bus again for that resolution**.

1. **PreBus/Present:** Run individual pre-bus microoperations that form/present address, direction, SYNC and data-out. Pre-bus ops must be idempotent and derived from committed state; they may assign explanatory latches, but must not irreversibly modify architectural registers or emit I/O. Replaying them after a stall is harmless.
2. **Resolve:** Machine processes **one actual bus interaction for this machine cycle** and samples external signals. Resolve is called exactly once for a particular transaction; a real repeated stalled cycle is a *new* machine cycle and a *new* Resolve, recorded in traces and permitted to repeat device side effects. The bus may return a host-defined value for undriven/floating reads.
3. **PostBus/Commit:** Consume the stored result and advance approved post-bus microoperations. Architectural mutations (PC/S increments, ALU results, P flag changes) occur post-bus, not as replayable PreBus operations. For a stall, the sequencer holds, skips normal state mutation and restarts its bus cycle on a subsequent machine tick. If a particular real sequence cannot obey this rule, justify and document a **local** staging exception, not a global shadow-CPU layer.

**Time:** One real machine-cycle transaction consumes machine time **even while CPU remains stalled**. A teaching pause partway through a cycle consumes **no additional machine time** and advances **no devices**. Future host timebase is not required to tick while the user inspects a paused mid-cycle state. Clock stretching is a machine scheduler duration, not an additional CPU microoperation.

**Ordering caution:** Exact electrical sampling order, RDY write behavior, bus ownership/DMA and interrupt recognition are per-target evidence items; this baseline specifies interface capacity without claiming the unknown waveform. A convenient `Clock()` completes the phases for the headless harness, but **the phases remain separately invocable** for microstepping and tests.

Supervisor `Peek` must be noninvasive and must not synthesize a CPU bus transaction; `Poke`/deposit is an explicit administrative mutation, may be unsupported on devices, and is not an emulated write. The CPU holds no pointer into machine RAM.

### 6. Cycle sequencer

Use reusable cycle **templates keyed by addressing mode × access class** (read, write, RMW, stack, control-flow, implied/register), with explicit conditional transitions for page crossings, branches, decimal timing and interrupts. An operation slot invokes opcode-specific register/ALU semantics. Exceptional sequences (BRK, RTI/RTS, JSR, indirect JMP, reset, interrupt entry) may be bespoke. The desired absence of opcode-specific control-flow `if`s outside operation slots is a testable design goal, **not** a reason to distort real hardware behavior.

Templates list named pre-bus ops, one Resolve point, and named post-bus ops. Template phase positions are true modeled bus-cycle positions; latch labels and transfers are pedagogical descriptions. Verify rather than assert BRK/IRQ equivalence and IRQ sampling points for the Apple/NCR target.

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

### 9. Test evidence and invariants

Classify individual opcode/cycle/edge cases as **Authoritative, Transferred, Excluded, Unresolved** with actual provenance, specific tested variant, conflicting evidence and provisional behavior. A passing transferred test is not proof of physical NCR identity. A known classified divergence from a WDC corpus must be reported as **classified divergence**, not failing target conformance and not silently edited into WDC behavior.

M1 acceptance includes:
- Documented opcode semantics, decimal flags, stack, branches, CMOS differences, reset/BRK/IRQ/NMI and applicable bus cycles.
- SYNC, intermediate/dummy access, RMW bus read/read/write, page-cross sequence, decimal extra cycles, target-backed interrupt sampling timing.
- **Stall property:** inject RDY low at arbitrary **read** cycles in deterministic RAM with fixed async inputs. Final architectural state must match unstalled execution and trace must equal baseline trace with repeated stalled cycles at their positions. Extend to write stalls **only after establishing physical write-bus behavior**; independently test a read-side-effect device, where device state may intentionally diverge.
- **Mid-cycle snapshot round-trip:** snapshot at every phase/microop cursor, restore, resume, match uninterrupted execution exactly, **no extra Resolve**.
- **Stepping equivalence:** microsteps, whole cycles and instructions must reach identical results and bus traces under equivalent inputs.
- After profiling arrives, observer-on/off results must match complete relevant CPU + machine/device state at chosen points; benchmark overhead.

### 10. Milestone order and gates

**Before Technical Specification 0.1: disposable 4-template schema spike**, separately described in `schema-spike-mandate.md`. Its code is discarded; the evidence and successful contracts become specification inputs.

**M1: CPU + bus + headless conformance harness.** Clock, Present/Resolve/Commit, microoperation table schema, snapshot state, opcode evidence table, baseline bus observation contract, functional and transferred cycle tests. Gate: classified conformance, RDY and mid-cycle snapshot properties, stepping equivalence, documented unresolved exceptions.

**M2: operable machine.** 64KB file-backed memory, nonblocking guest console, SIMH-style supervisor, speed control, structured operational logging. Gate: deterministic host behavior, accurate control/inspection and device separation, load/save, correct operator logs.

**M3: profile and trace.** DTrace-inspired bus-stream consumer, fixed-count aggregates, trace, observer invariance, overhead benchmarking. Gate: attribution correctness, same execution with observers on/off, measured overhead.

After M3 (not prerequisites): reversibility, Wails Visible Computer front end, complete Apple IIe ROM/video/audio/disk/peripherals.

### 11. Authority by domain (single definitive hierarchy)

For **the machine's intended configuration and Apple-supported CPU profile**: Apple *Apple IIe Technical Reference Manual* (1985) is normative. For **processor-specific behavioral facts**: NCR 65C02 manufacturer's documentation, including its reproduction in Apple's appendix, is primary evidence; contemporary Apple discussion and Jim Sather's *Understanding the Apple IIe* provide machine/hardware context. **Direct, verified measurements of the target reference chip** supersede conflicting printed timing; record the documentary erratum. For **per-cycle transaction details not established by the target documents**, variant-identified independent hardware traces and SingleStepTests/65x02 are candidates subject to a transfer map. AppleWin is a semantic/implementation comparison, **not** assumed a bus-cycle oracle. For teaching **interaction and vocabulary**, *The Visible Computer: 6502* manual is primary, subordinate to authentic bus behavior. For software architecture only, Aiki (profiling), Harpy (operational log), the V6/PDP-11 simulator experiments (supervisor), and Nezha (later Wails) are inspirations, not source dependencies.

Never count the Apple comparison table and the NCR sheet reproduced in the same volume as independent hardware measurements. Record exact edition, page/table, chip variant, whether evidence is original/reproduced/derived, and any conflict. When evidence is lacking, maintain **Unresolved**, implement documented behavior provisionally where possible, and do not erase divergence to appease a foreign variant's test.

### 12. Freeze and deferred choices

**Frozen:** goals, target profile, one sequencer, microstep pause, three-phase bus cycle with retained Resolve result, phase cursor/idempotent PreBus constraint, host-owned time, side-effect separation, nonblocking console, four observation categories, authority/provenance rules, evidence-aware M1 and M1→M2→M3 order.

**Not frozen:** concrete Go struct layout; detailed template encoding; uncertain NCR electrical/bus behavior; exact undefined opcode cases; physical-vs-documentary open discrepancies; profiling report syntax; console address assignment; snapshot/history strategy; UI visual layout.

The next implementer must **not** treat open evidence classifications as permission to invent behavior. Prototype the schema, report contradictions, and bring any genuine architectural obstacle back for review before changing this baseline.
