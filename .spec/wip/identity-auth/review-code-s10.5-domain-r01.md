# S10.5 domain2 bounded independent code review R01

**Conclusion: NO FINDINGS WITH LIMITS.** Two review rounds are complete for the exact two frozen domain2 files. Confirmed defect finding IDs: none; CRITICAL/HIGH/MEDIUM/LOW = 0/0/0/0. This is a partial-domain result, not a full S10.5 Code Gate PASS, Case closure, E2E conclusion or risk acceptance.

The executing role is the assigned non-author Reviewer. No reviewed code was authored or repaired in this execution; no subdelegation was performed and no new Agent/Session ID is asserted. The work remains in development-delivery review and returns to Lead.

## Fixed object and source contract

Workspace: `D:\AID\VSCode\Codex\Han`; branch `main`; HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`. The object is the uncommitted frozen domain2 content, not HEAD-only code.

| Object | SHA256 |
|---|---|
| `password_reset_operator.go` | `8FB6ADAA6F32F0054E6C09D13157AAFE549D615DD89CCC5DDE14701F0791E1E9` |
| `password_reset_operator_test.go` | `CA1A05BA8C2737E50B8ACB6693C6022D9E12BBC20DE0FB7838CBFFC068336BF6` |

Frozen source location: `.spec/wip/identity-auth/evidence-s10.5/domain-review-input-r01/fixed/internal/identity/`. Matching live hashes were checked at start, before and after each own test run, and at report handoff; no domain2 drift was detected. `go.mod` and `go.sum` also match the Reviewer start snapshots. Shared-worktree changes by other owners were preserved; repository cleanliness is not claimed.

Original preimages were found at `.spec/wip/identity-auth/evidence-s10.5/design-r01/source-before/internal/identity/`: operator `4DC973ECB2272E07083884224A898D0C7C38E4CFEE8B03E44DA14E9D9F951F42`, test `9463D7B8B098A21BA3F68BDB190CFCF51C6FA47038BA0BD8A7E7505ADD210E02`. Complete unified diffs are retained. Existing executable settlement/Execute bodies and old test bodies have no changed logic; the production delta is imports, the private maintenance field and appended maintenance implementation. This review does not re-approve the entire pre-existing settlement classifier.

Both supplied manifests were verified entry by entry; see `initial-hashes.json`. Immutable six-object contract input is `design-r02/inputs-final.json` and its `fixed-final/` files. Contract content consumed: full ADR-0026 r03 (identity.md:336-495), FD r48 maintenance part of 3.3.11 (design.md:2286-2406) and 9.4.3 (3770-3797), relevant architecture maintenance boundaries, S10/maintenance Story, and E431 I-S105-01-23 with the applicable detailed matrices. The decisions README was hash-checked, not independently reviewed for content.

The live test plan hashes to `86B8C4026643F4B3B7876077510FDC29103D12952E7CF5FEB850D7437D5035C4`. The comparison confirms four changed lines in 9.5 mapping only; frozen E431 remains the review source contract. Diff retained as `test-plan-frozen-to-live.diff`. Prior design PASS is not used as evidence of code correctness.

## Two actual rounds

Round 1 reviewed the complete two-file delta, all appended production maintenance code (operator.go:643-1529), added tests (operator_test.go:732-1461), affected existing constructor/permit/Execute boundary and shared SQL-test harness. It checked canonical codec, signature/domain/subject/time binding, private request ownership/deadline, states, SQL/transaction exits, audit preservation, and test oracles.

Round 2 revisited state, authority, transaction and replay behavior against the immutable upstream contracts, then followed actual consumers: `internal/audit/store.go` Append/Get/decodeMetadata; `PasswordResetStore.CheckLockedUser` and its unresolved-operation Gate; worker Reconcile/Claim/Renew/Fail and current retry configuration; model enums; migrations 0001/0008; installed pgx v5.11.0 Tx/pgxpool/QueryRow and format-option implementation. The allowed dependency review is bounded to these consequences; Root4 and the evolving integration carrier were not reviewed.

Methods/rules: full add-reviewer Role, add-code-review, established add-delivery-workflow, supplemental add-research, code-review-checklist, abstraction, coding, no-hardcode, task-execution, multi-agent, observability, security redlines and relevant enabled go-service rules. No unrelated profile or other workflow was applied. `readset.json` names exact scopes, hashes and retained snapshots, including hash-only entries.

## Evidence and checks by risk

Positions below refer to the frozen files. Numbered copies are retained as `password_reset_operator.go.numbered.txt` and `password_reset_operator_test.go.numbered.txt`.

| Risk | Actual check and evidence boundary |
|---|---|
| Scope/API compatibility | `password_reset_operator.go:118-154,643-1529`; tests/evidence `preimage unified diffs; existing constructor/Execute/settlement bodies and old tests unchanged`. static diff and direct worker consumer. |
| Trust, real-owner boundary, independent subject | `password_reset_operator.go:819-858,926-997`; tests/evidence `password_reset_operator_test.go:822-885,928-974`. static production resolver; runtime private platform/SID seams, not actual owner verification. |
| Canonical precision and bounded signed input | `password_reset_operator.go:692-764,1024-1203`; tests/evidence `password_reset_operator_test.go:887-974,1439-1460`. static full codec; runtime malformed encoding, real fixture Ed25519, int64 > 2^53 and quick100. |
| Opaque deadline and fresh authorization | `password_reset_operator.go:862-942,1257-1263,1300-1302`; tests/evidence `password_reset_operator_test.go:976-1034,1392-1435`. runtime controlled Clock/context and byte mutation; static production deadline ownership. |
| Snapshot guard, lock order, minimum write set | `password_reset_operator.go:1233-1234,1295-1355,1374-1420,1500-1520`; tests/evidence `password_reset_operator_test.go:1063-1241`. static SQL/schema/worker; runtime scripted lock order, six eligible combinations, 25-field stale-before rejection, past/equal/future next_at; not real PG. |
| Audit scalar guard and text-format carrier | `password_reset_operator.go:855-858,1358-1371,1400-1406,1427-1466,1522-1529`; tests/evidence `password_reset_operator_test.go:1112-1154,1262-1370`. runtime scripted scalar/sequence/text option; static actual audit.Store and pgx v5.11.0; not PG transport or detoast measurement. |
| Replay and one enqueue | `password_reset_operator.go:1303-1313,1423-1497`; tests/evidence `password_reset_operator_test.go:1243-1297`. runtime exact historical receipt/after, corrupt/extra/missing/float/padded/oversize rejection, zero Exec; static worker 2-to-5 policy, not real concurrent execution. |
| Unknown, panic, rollback and permit | `password_reset_operator.go:1207-1292`; tests/evidence `password_reset_operator_test.go:1097-1109,1334-1435`. runtime synthetic commit errors, rollback error, panic, expiry, permit counters and rollback context deadline; static pgxpool return semantics; physical resource cleanup excluded. |

Static transaction conclusions: User-first lock order and operation/User relation are explicit; before is compared as the complete typed 25-field snapshot. Eligibility matches the six unresolved/pending combinations from migration 0008. The only maintenance business UPDATE is the selected recovery job status/next_attempt_at/updated_at; GREATEST preserves the protection window and count/failure/created values are not reset. UPDATE row-count, re-read after, Audit Append and scalar guard errors leave the transaction on the rollback path. These are source conclusions, not observed PostgreSQL commits.

Static audit/replay conclusions: scalar guard queries the global UUID and returns only size/binding scalars; it precedes Get, and no-row does not fall through to Get. Actual append-only statement triggers were inspected as the basis for no intervening growth. Replay uses existing audit.Store.Get via the private JSONB text adapter and accepts only metadata exactly equal to the reconstructed 13-string contract. This compares exact original P/signature and SHA fields without recovering int64 from the generic float64 map. Historical after/receipt is returned independently of current job progress; no UPDATE/Append occurs on the replay branch. Actual query transport/server cost is unverified.

Static worker/security conclusions: maintenance never calls Execute/settlement, creates a lease, clears a hold or changes the operation safety authority. The unchanged Store Gate still blocks unresolved operation state. Worker Claim increments count and worker Fail compares the new lease count against that running worker's MaxRetries: the 2-to-5 scenario can legally have several automatic attempts after one enqueue. This consequence was traced in real source; it was not exercised against a database.

Static cleanup boundary: the domain holds its permit around Requeue and transaction finalization; rollback gets min(cleanup start + ShutdownBudget, original D + ShutdownBudget) via a non-cancelled bounded context. Any commit error remains unknown; no second classification transaction is opened. Actual pgxpool.Tx Commit/Rollback returns the borrowed pool connection. Physical SDK/network cleanup and pool Close remain Root responsibilities and are not certified here.

## Own execution and author proof separation

Only the following two authorized own commands were run, under Go 1.27.0 with installed pgx v5.11.0. `GOPROXY=off` and `GOSUMDB=off` prevented dependency network retrieval; `-mod=readonly` was retained.

```text
go test -mod=readonly ./internal/identity -run ^TestPasswordResetMaintenanceOperator$ -count=1 -v
go test -mod=readonly -race ./internal/identity -run ^TestPasswordResetMaintenanceOperator$ -count=1 -v
```

| Own run | Exit | Run nodes | Leaf nodes | Failure / skip | Wall seconds |
|---|---:|---:|---:|---:|---:|
| Normal | 0 | 126 | 114 | 0 / 0 | 3.891 |
| Race | 0 | 126 | 114 | 0 / 0 | 10.422 |

Raw own stdout/stderr and command/hash/result JSON are retained for each run. The 100 quick-check iterations are one test leaf, not 100 separately closed Cases. There is no coverage-percentage or exhaustive-interleaving claim.

Author evidence was independently hash-checked and inspected, not counted as own execution. Frozen normal/race logs each show 126 nodes, 114 leaves, zero failure/skip; frozen reset/resend regression shows 385 nodes, 340 leaves, zero failure/skip, terminal PASS, and its result JSON records exit 0. The raw author logs are UTF-16; normalized retained copies and parsed counts are provided. Author freeze explicitly records rootReady=false, combinedCLIExecuted=false and PGExecuted=false.

Meaningful U evidence includes six eligible combinations, stale rejection for every snapshot field, exact-int64 fixture 9007199254740993, real fixture Ed25519, several canonical-input rejections, opaque/cross-instance/cancel checks, replay corruption and no-write assertions, guard rejection before Get, post-Append guard failure, commit-error unknown mapping, rollback error, panic and expiry, plus permit counters. It does not close the missing PG/B layers.

## Unverified risks and suggestions

1. Partial domain2 review only. Root4, native stdin adapters, new integration carrier, composition-root pool/SDK cleanup, product binary, server compatibility runtime and the seven-file final code gate were not reviewed/run here.
2. No PostgreSQL connection, DSN, DDL or fixture operation. SQL syntax/binds, actual lock/constraint behavior, cross-user UUID collision atomicity, text transport upper bound, metadata JSONB rendering cost, microsecond round-trip and commit acknowledgement loss lack independent real-PG evidence in this review.
3. The production os/user.Current owner-SID path was inspected statically; the executed unit suite uses a private platform/subject seam and a private client_encoding seam. It is not proof of actual owner identity, second Windows subject, trust deployment/launcher isolation or server encoding behavior.
4. Rollback tests use scripted pgx.Tx and record a finite cleanup context; they do not block a real Rollback, prove socket/SDK CleanupDone, or establish constructor/pool quiescence. Those belong to the separately assigned Root review and later integration verification.
5. Own tests ran only TestPasswordResetMaintenanceOperator normal and race. Author reset/resend regression was inspected as author evidence, not independently rerun. Race success is limited to executed paths; no claim of concurrent real requeue coverage.
6. Unit SQL scripts match markers and some arguments; apply snapshots are supplied using the production expected-after helper. They establish meaningful control-flow/invariant behavior but are not an independent SQL execution or full persisted-after oracle.
7. Case I-S105-01-23 are a coverage framework, not 23 independently closed Cases. Partial U evidence is mapped below; real PG/B layers, full configuration/server/OS and all Case/E2E closure remain outside this result.
8. Only the two relevant original preimages were diffed; no claim that all 185 original sources were independently inventoried or preserved by this Reviewer.
9. No whole-repository lint/build/test, product-binary build, dependency update, deployment or E2E was authorized/performed. No Root test or subdelegation was performed.

Pending direction decisions raised by this review: none. Nonblocking test suggestion N1: original author may strengthen the scripted tests at operator_test.go:1077-1084/1121-1144 with exact lock/update argument assertions and independently constructed after/audit expectations. This is separate from defect findings and is not a request for Reviewer repair.

## Handoff and re-review

No defect findings are emitted, so there are no finding IDs to close in this report. Lead must separately reconcile this frozen two-file result with the other Reviewer result, actual final seven-file hashes, author/Lead evidence, integration/PG/binary checks and unresolved delivery gates. This report neither accepts missing evidence nor authorizes the next action.

Stop here and await Lead. Any author repair or drift requires an actual new SHA256 and an explicit fixed object for re-review. Recheck the changed logic, relevant regression and any original finding IDs if findings are subsequently raised; preserve this report and its old evidence. No repository source/doc/Task edits, Git mutation, PG/DSN/DDL, key export, mail, cleanup or child delegation were performed. Reports, proofs and snapshots were intentionally written only under this newly created retained Temp directory.

Machine-readable result: `report.json`. Evidence inventory: `readset.json`, `initial-hashes.json`, `final-hashes.json`, both unified diffs, own logs/results, author inspection/decoded logs, final Git status snapshot.
