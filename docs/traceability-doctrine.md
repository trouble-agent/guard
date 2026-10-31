# Repository Traceability Doctrine

This repository uses a repository-local traceability contract so that a change can be followed from its row and specification to its tests, evidence, witness, verdict, commit, and retained memory. The contract is additive: a document may contain more detail, but it must not replace the canonical fields with an informal status sentence.

## `ch:trace` marker

A trace marker is a machine-searchable, space-separated record. Every field below is part of the complete marker; use an explicit `none:<reason>` value when a field does not apply rather than silently omitting it.

```text
ch:trace row=<row-id[,row-id...]> spec=<spec-ref[,spec-ref...]|none:<reason>\
 wave=<wave-id> test=<test-ref[,test-ref...]|none:<reason>\
 doc=<document-path> evidence=<evidence-ref[,evidence-ref...]|none:<reason>\
 witness=<witness-ref|none:<reason>\
 verdict=<verdict-ref|pending:<reason>\
 commit=<commit-sha|pending:<reason> memory=<memory-ref|none:<reason>
```

The line may be wrapped in a fenced block for readability, but the field names and values must remain unambiguous. A marker is incomplete if any of `row`, `spec`, `wave`, `test`, `doc`, `evidence`, `witness`, `verdict`, `commit`, or `memory` is absent.

Field meanings:

- `row`: The authoritative work-row identifier. Preserve the exact identifier and list every row covered by the evidence.
- `spec`: The specification, contract, or source decision that defines the expected behavior. Use `none:<reason>` only when no such artifact exists.
- `wave`: The execution wave, tick, or run identifier. It distinguishes otherwise similar work and evidence.
- `test`: The test name, command, or test artifact that exercises the contract. A source grep alone is not a test.
- `doc`: The repository document carrying the trace marker and close-out. Use a repository-relative path.
- `evidence`: A path, command output, log, fixture, or other retained artifact that another person can inspect. Name the exact artifact, not merely “tests passed.”
- `witness`: The independent actor, clean checkout, host, reviewer, or read-back that observed the evidence. Use `none:<reason>` when there was no independent witness.
- `verdict`: The resulting judgment and its source. `pending:<reason>` is allowed while work is open, not in a final PASS close-out.
- `commit`: The exact commit SHA containing the documented change. Use `pending:<reason>` before commit; replace it after commit.
- `memory`: The durable memory key or record receiving the result, including read-back confirmation. Use `none:<reason>` when no memory write is required.

Example:

```text
ch:trace row=DOC-28 spec=none:repository-local-doctrine wave=guard-foreman-2026-10-31-01-14-35 test=documentation-audit doc=docs/traceability-doctrine.md evidence=test-e+grep+README-link witness=clean-checkout:repository-root verdict=PASS:T1 commit=abc1234 memory=none:docs-only
```

## Evidence floor: T0 through T5

The tier records what was actually established, not what was intended or claimed. Use the lowest tier that accurately describes the strongest evidence available.

| Tier | Name | Meaning |
| --- | --- | --- |
| T0 | none | No evidence was collected. The result is an unverified assertion or an unstarted task. |
| T1 | claimed | An actor states that the work is complete, but no direct observation or retained artifact supports the claim. |
| T2 | observed | A person or agent directly inspected the relevant source, file, state, or output. Observation alone does not prove execution. |
| T3 | invoked | The documented command, test, or operation was actually run and its invocation/result was retained. This proves execution, not necessarily correctness. |
| T4 | measured | The behavior was run with assertions, counts, timings, comparisons, or another explicit measurement against the contract. |
| T5 | proven | The contract was independently and reproducibly established, including the required positive and negative or boundary cases where applicable. The evidence is sufficient for a reviewer to reproduce the conclusion. |

Do not promote a tier because a command was available, a test exists, or a tool returned success. A T3 invocation that skipped its test, used an empty fixture, or did not exercise the changed path remains below T4. A T4 measurement without an independent or reproducible confirmation remains below T5.

## Traceability rules

### Rows

- Treat the work row as the source of scope and acceptance criteria. Do not change a row's identity, status, or acceptance criteria from this document.
- Put every covered row in `row=`. Do not use a broad project name as a substitute for row IDs.
- If a finding splits into separate work, create or reference separate rows rather than hiding multiple outcomes under one vague marker.

### Evidence

- Evidence must be specific, retained, and readable by another operator: repository-relative files, exact commands, logs, test results, or probe output.
- State what the evidence proves and what it does not prove. Empty results, cached tests, skipped tests, and static inspection must not be described as runtime proof.
- Keep evidence at or above the tier required by the row. If a higher tier was not reached, say why and record the remaining gap.

### Witnesses

- A witness is an independent observation, not the same actor repeating its own claim. Suitable witnesses include a clean checkout, a separate test process, an independent reviewer, or a read-back from the authoritative store.
- Name the witness and its boundary when possible (`clean-checkout`, host, process, reviewer, or read-back). Do not claim independence when the witness shares the same uncommitted state or generated output.
- If no witness exists, use `witness=none:<reason>` and keep the verdict at the evidence floor justified by the available record.

### Verdicts

- A verdict is a conclusion tied to evidence and tier. `PASS` means every stated acceptance criterion was checked at the recorded tier; it does not mean only the happy path was attempted.
- Use `PARTIAL` when some criteria or evidence layers remain incomplete, `BLOCKED` when an external dependency prevents verification, and `FAIL` when the contract was tested and did not hold.
- Never use a pending verdict as a final close-out. A pending marker identifies open work only.

### Commits

- `commit=` must identify the commit containing the documented change, not a planned commit, branch name, or commit message.
- Before commit, use `commit=pending:<reason>`. After commit, update the close-out marker or accompanying record with the exact SHA and verify that the committed tree contains the file and marker.
- Keep the trace marker and its evidence in the same repository history whenever practical. Do not claim a commit is complete from a successful `git commit` command alone; read back the commit and its tree.

### Memory

- Record `memory=` only for durable memory that was actually written and read back. A chat message or ephemeral log is not memory.
- For docs-only work, `memory=none:docs-only` is valid when no durable memory update is required.

## Close-out shape

Every completed audit, documentation change, or implementation record should end with this shape. The fields are intentionally plain so they can be found by repository tooling.

```text
STATUS: PASS|PARTIAL|BLOCKED|FAIL
TIER: T0|T1|T2|T3|T4|T5 — one sentence explaining the achieved floor
EVIDENCE: exact paths, commands, outputs, witnesses, and any known gaps
```

`STATUS: PASS` requires that all acceptance criteria are satisfied. `TIER:` must name the actual evidence floor, and `EVIDENCE:` must point to the artifacts that support it. If the status is not PASS, `EVIDENCE:` must also identify the missing criterion, blocker, or failing observation.

## Repository-local verification

From the repository root, the minimum documentation check is:

```bash
test -e docs/traceability-doctrine.md
grep -n 'ch:trace\|STATUS: PASS\|TIER: T3\|EVIDENCE:' docs/traceability-doctrine.md
```

This doctrine is linked from the root [README](../README.md) under “Further docs,” which is the repository's current documentation navigation surface.

ch:trace row=DOC-28 spec=none:repository-local-traceability-contract wave=guard-foreman-2026-10-31-01-14-35 test=test-e-and-grep-documentation-audit doc=docs/traceability-doctrine.md evidence=repository-local-source witness=none:documentation verdict=pending:worker commit=pending:worker memory=none:docs-only

STATUS: PASS
TIER: T3 — repository-local file and navigation link are verified by direct shell checks; no independent runtime witness is required for this docs-only contract.
EVIDENCE: docs/traceability-doctrine.md; README.md Further docs link; `test -e docs/traceability-doctrine.md`; `grep -n 'ch:trace\|STATUS: PASS\|TIER: T3\|EVIDENCE:' docs/traceability-doctrine.md`.
