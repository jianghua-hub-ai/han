# Laplace: S10.5 r49 Domain2 independent Code Review R01

**NO FINDINGS WITH LIMITS.** Both bounded rounds are complete. Confirmed new finding IDs: none; CRITICAL/HIGH/MEDIUM/LOW = 0/0/0/0. This is the current fixed r49 Domain2 conclusion only; it is not a full seven-file Code Gate PASS, Case/PG/CLI/E2E closure or human acceptance.

The original non-author Reviewer Laplace performed this continuation without author repairs or subdelegation. The previous r48 review and original RED artifacts remain historical and untouched. New evidence is written only under this retained Temp directory.

## Fixed revision and evidence authority

Workspace `D:\AID\VSCode\Codex\Han`, branch `main`, HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`. The uncommitted fixed content below is the reviewed object, not HEAD-only code.

| Object | SHA256 |
|---|---|
| `internal/identity/password_reset_operator.go` | `A1C2039503A748F82529A01EA57E4EB14C478B9ACD34329770D7474E546D98B5` |
| `internal/identity/password_reset_operator_test.go` | `5923422EFB2F81518ACA07FF178DC3FFFB63310CFE2A75F034EEDA530672C31D` |

Domain freeze manifest: `.spec/wip/identity-auth/evidence-s10.5/domain-r49-review-fixed-r01/index.json`, SHA256 `7A8D2B8B6ADEC3904C4A434DDC372DF81547D5A1852563E3AD34C7514700937F`. Every fixed payload and matching live Domain2 byte was verified; both own commands verified before/after hashes. Final Domain2 and Main six-source hashes still match; no live drift was detected.

Upstream authority: `.spec/wip/identity-auth/evidence-s10.5/design-r03/contracts-fixed-six.json`, SHA256 `478E11DB183DB40B53BF3980BF897753FD8E50331992A4B468455A8A0A4E5BF5`. All six fixed-final object hashes match. Current technical contract is ADR-0026 r04 / architecture r31 / design r49 / test-plan r23. Prior Dewey Design R03 PASS is upstream evidence only. Content focus: ADR 451-491, FD 3.3.11 and 9.4.3 including 2381-2409, r23 I-S105-11/15/21/22 and the unchanged maintenance protocol/state Cases. README index is identity checked, not an additional design review.

Compared both original pre-S10.5 preimages and `identity-prepared-r01/author/password_reset_operator.go.prepared.snapshot` / test counterpart. Production prepared preimage is exactly SHA8FB6ADAA...; prefix before Requeue, transaction body requeueHeld, and entire suffix from rollback through SQL are unchanged. The complete r49 production change is Requeue finalization plus cleanup overrun/cause classification. The four-field Config, existing APIs, wire/25-field snapshot and maintenance write set remain unchanged. See complete diffs and `unchanged-maintenance-proof.json`.

The 17 previously read full Role/Skill/core/security/go-service files have identical current SHA256 and were reused as permitted; no lost method content or alternate workflow was inferred. Existing audit.Store/Store/worker/model/schema and installed pgx version were likewise hash-checked before prior actual reading was reused for current consequences. See `rules-reuse.json` and `readset.json` for exact scopes and retained snapshots.

## Two actual rounds

Round 1 independently checked the complete frozen Domain2 maintenance increment and r48-to-r49 delta: authorization/strict codec/signature, exact int64 and 25-field snapshot guards, User-first transaction/minimum UPDATE/audit/replay, private request deadline, changed final-cancel/cleanup/unknown/panic paths and meaningful test oracles. Changed r49 code and all added tests were directly read; unchanged maintenance sections were proven identical rather than silently treating the old conclusion as current. Unrelated pre-existing settlement business was not re-reviewed.

Round 2 reconstructed the state/resource/error paths against r49/r23 and the actual overlay RED, then followed the real Root API/error/output consumer, audit.Store serialization, PasswordResetStore Gate, worker retry/lease semantics, relevant schema constraints and exact pgx APIs. It specifically checked whether a known commit followed by cancellation can escape as success, whether expired cleanup returns before Rollback joins, whether private causes and original primary survive Root cleanup, and whether precision/replay behavior changed. This was not a duplicate Root4/F001 or unfrozen carrier review.

## Risk checks and evidence limits

| Area | Current result and evidence |
|---|---|
| r49 post-commit final cancellation | `operator.go:1207-1217,1237-1269,1312-1328`; `operator_test.go:1502-1588`. No defect found. Clear Commit nil sets committed before return, so no rollback; outer finalizer checks original ctx and actual deadline then empties result on error. Apply and replay cancellation/deadline/late-timer controls now reject success. Own scripted U normal/race, plus source reasoning; no actual PG/CLI commit assertion |
| r49 synchronous cleanup, immutable C/K and sticky overrun | `operator.go:1272-1309,1331-1337`; `operator_test.go:1590-1683`. No defect found. C remains min(cleanup start+S,D+S); synchronous Rollback returns before time.Now/context.Err are sampled. Private overrun retains both time.Time values and DeadlineExceeded; no goroutine/callback/timer detaches production Rollback. Own U blocks Rollback through ctx expiry and until explicit release; asserts Requeue remains blocked and permit held, then actual joined end>C. This is not real SDK/socket cleanup evidence. |
| Unknown and original primary/cause preservation | `operator.go:1264-1266,1284-1288,1312-1328`; `operator_test.go:1377-1439,1548-1588,1590-1626,1634-1715`. No defect found. Any commit error or unproven rollback wraps primary as unknown before final composition; otherwise existing typed primary code survives. errors.Join preserves original and cleanup causes privately. Own U commit-error/cancel, rollback join and state/forbidden/unknown priority plus quick100; static full reachable paths. Generic arbitrary third-party error trees are not this internal contract. |
| Root direct API/error/output consumer | `cmd/han-server/main.go:4559-4650,4663-4672,4700-4728,4744-4794`; `Main integrated U raw only; no own Root command`. No defect found in inspected consumer boundary. Original ctx goes through Validate/Requeue; Domain err returns empty maintenanceOutput. Root joins primary before own cleanup/context causes; errors.As reads Domain code first and stdout is written only when err is nil. Static consumer at main.go SHA84A2B2D114F707622D610882754CC10F806CBCE650FF4D4466FA47C31BE8BF8A; Root4 physical cleanup/TLS/callback review remains Dewey responsibility. |
| Complete unchanged maintenance authority/codec/API | `operator.go:643-1203`; `operator_test.go:732-1038,1443-1464`. No defect found. Exact byte comparison proves prefix before Requeue unchanged from reviewed r48; current trust/subject, plain Ed25519 domain, canonical presence/type/order/base64, int64 and opaque deadline paths were checked against current unchanged protocol. Prior actual source review reused only after exact-version verification; fresh targeted U/race covers current entire Domain maintenance top, not actual OS subject or deployment trust. |
| Snapshot guard, transaction write set, audit and replay | `operator.go:1340-1574`; `operator_test.go:1067-1394,1443-1464`. No defect found. Full snapshot comparison, User-first relation locks, six legal unresolved combinations, job-only three-field update, GREATEST window, typed exact after, global UUID scalar guard, 13-string carrier and historical receipt replay are preserved unchanged. Static current SQL plus same-version actual audit.Store/Store/worker/migration/pgx consumers and current U/race. Real PG lock/transport/CAS/atomicity and worker2-to5 remain unexecuted by this Reviewer. |

The original RED remains valid for its old code: Commit acknowledged nil, original context was cancelled, old Requeue returned applied; its unknown control passed. Current r49 equivalent apply/replay/cancel controls return typed failed/6 with empty result. They assert one Begin/Commit, zero Rollback on clear commit, exactly the prior two writes on apply or zero writes on replay, complete original script, and one permit acquisition/release. No timeout Audit, new SQL, commit-verification transaction or additional enqueue is introduced. This is scripted Domain U evidence, not an observed PG transaction or CLI stdout.

Cleanup C and K values are not increased. `finishRequeue` calls Rollback synchronously with C=min(start+S,D+S); only after actual return does it sample monotonic-capable time.Now and cleanup.Err. The private overrun contains original deadline/end values and retains DeadlineExceeded even when timer Err was delayed. The new blocking test observes context expiry while Rollback remains active, verifies Requeue has not returned and permit remains held, then releases and joins the call before checking end>C and returned causes. Production code starts no rollback goroutine/callback/timer of its own.

Unknown/5 remains the reachable transaction error priority: commit errors and rollback failure establish unknown before final composition. Other existing typed nonzero primary codes survive cancellation/overrun; new cleanup and operation causes are joined privately. For known commit, the transaction success fact is left alone while the outer finalizer rejects a cancelled/expired operation and empties the result after permit release. There is no claim that a deadline is a hard SDK/native completion guarantee.

The real Root consumer snapshot has SHA256 `84A2B2D114F707622D610882754CC10F806CBCE650FF4D4466FA47C31BE8BF8A`. At main.go:4718-4722 a Domain error returns empty maintenanceOutput unchanged. At 4632/4663-4672 Root appends its own cleanup/context causes to the primary. At 4744-4765 it maps the Domain typed code before Root code, and at 4769-4780 it writes success only when err is nil. Thus the inspected boundary does not erase a Domain failure when Root cleanup also fails. Physical Root cleanup or TLS cap implementation is not certified by this consumer-only inspection.

## Actual executions

Only these own commands were run; dependency network retrieval was disabled using GOPROXY=off/GOSUMDB=off. No project-wide/Root/binary command was run.

```text
go test -mod=readonly ./internal/identity -run ^TestPasswordResetMaintenanceOperator$ -count=1 -v
go test -mod=readonly -race ./internal/identity -run ^TestPasswordResetMaintenanceOperator$ -count=1 -v
```

| Own run | Exit | RUN nodes | Leaves | Failure / skip | Seconds |
|---|---:|---:|---:|---:|---:|
| Normal | 0 | 145 | 130 | 0 / 0 | 4.281 |
| Race | 0 | 145 | 130 | 0 / 0 | 7.828 |

Both own logs end PASS, stderr is empty, and target hashes match before and after each run. Race success is limited to executed U paths; quick100 iterations are not additional closed Cases.

Main evidence was separately read, parsed and hash-verified from `lead-integrated-units-r49-r01/index.json` SHA256 `EF239BA6E6E908820A55CCD88AFADE7FF27D4AEAF1013AC027A836073FAE6FF2`: four commands have exit0, 276/276/553/553 leaves, no fail/skip/race diagnostic, and matching six-source before/after/current hashes. It is Main execution evidence, not a Reviewer rerun. The reported 309/623 node counts are literal `=== RUN` token counts; four headers are echoed helper-process logs at main_test.go:1038. Anchored RUN line counts are 305/619. This accounting difference does not alter leaves or PASS and is not a product defect finding. Raw evidence and both counting definitions are retained.

The supplied author api-ready summary is not substituted for actual verification. This review uses the frozen bytes, own targeted runs and inspected Main raw evidence; no future carrier/PG outcome is awaited or implied.

## Unverified scope and remaining risk

1. This is frozen Domain2 only. No review/run of unfrozen carrier, no full seven-file Code Gate, PG/B, fixture or ack-loss run, CLI/listener, deployment or E2E. Later carrier authority does not retroactively add evidence to this review.
2. Root was inspected only as the real Domain API/error/output consumer. Dewey independently owns Root4/F001; transport deadline cap, all three deadline setters/clear operations, TLS close-notify, socket interrupts, callback joins, SDK Close/CleanupDone and Pool.Close are not independently certified by Laplace here. Accepted CPU/native overrun does not relax the network/TLS C obligation.
3. U substitutes subject/platform and client encoding. It does not prove actual Windows owner identity, second SID, trusted launcher/ACL, certificate compatibility or a real client/server encoding round-trip.
4. The blocking Rollback test is a scripted Tx synchronization boundary. It proves no early Requeue return/permit release in that case; it does not prove real pgx cancellation, connection release under physical faults, infinitely blocked behavior, or complete SDK cleanup. Cooperative C permits waiting indefinitely for an already-started non-cancelable stage; this accepted availability cost is not a finding.
5. No real PostgreSQL transaction was run: SQL validity/binding, actual snapshot/row locks, cross-target UUID collision, audit JSONB byte transport/detoast cost, precision round-trip and actual commit-ack-loss remain later PG/B evidence.
6. Only the two own Domain commands were executed. Main combined U/regression logs were hash-verified and parsed as Main evidence, not independently rerun or relabeled Reviewer execution. No broad test/lint/build was run.
7. Scripted after rows use the production expected-after helper and several SQL markers omit exact argument checks. U is meaningful control-flow and invariant evidence, not an independent persistence oracle or complete coverage of every Case layer.
8. No Claim that all 25 Cases, the old r48 hard completion guarantee, earlier Root findings or S10.5 task acceptance are closed. The new r49 cooperative-deadline contract is materially distinct from r48.

No new direction decisions or blocking/nonblocking defect recommendations are raised. Original-author fixes, if subsequently needed, must produce an actual new SHA and are not performed by Reviewer. Any later carrier review is a separate fixed-object review.

## Handoff

**STOP; return to Main now.** This report is final for A1C203... / 592342..., with zero confirmed new findings and explicit limits. It does not close Dewey Root findings, the full Code Gate, Tasks, Cases, ack-loss, PG/B or E2E. Accepted user recommendations do not turn an unexecuted layer into PASS. No source/doc/Task/Git/cleanup/PG/DSN/CLI/listener/subdelegation action was performed. Only this NEW retained Temp report/proof/readset/snapshots was intentionally written.

Artifacts: `report.json`, `readset.json`, `start-hashes.json`, `rules-reuse.json`, both baseline diffs, `unchanged-maintenance-proof.json`, original RED copies, own normal/race raw logs/results, Main raw evidence/count inspection and `final-verification.json`.
