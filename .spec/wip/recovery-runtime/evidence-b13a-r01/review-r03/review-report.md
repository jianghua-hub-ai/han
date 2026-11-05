# Ptolemy independent Code R03

Result: PASS for the bounded code re-review. RECOVERY-B13A-CR-R02-F001 and RECOVERY-B13A-CR-R02-F002 are CLOSED on the exact object below. New findings: 0. No remaining component blocker was identified in the combined R01 full scope, R02 repair review and this targeted R03 scope. STOP/frozen; Main owns final Task/batch reception.

Reviewer: original non-author Ptolemy/add-reviewer, 01a18492-af9e-7ea2-9cd5-e4dd368d7bee. Implementation belongs to original Zeno/add-backend-dev, 01a18486-3e11-7852-9723-ad380e0bb392. Fixed workspace D:/AID/VSCode/Codex/Han, main, HEAD 6da8abbceb0fa197364810a92a8fa2d97b38ddca. No shared source/document/Git write or delegation occurred. All report/probe output is in this personal Temp.

## Frozen object and provenance

Author index: .spec/wip/recovery-runtime/evidence-b13a-r01/author-r03/index.json, SHA256 7EF2616BCD599DA9169FD0DF6F6E440ABF920DC3FD56E13DFD1387E7138C0F50. All 118 indexed retained files were independently byte/hash checked. Current live source equals final-*.go.frozen at entry and completion; preliminary runtime_test.go.frozen is not the final review object. Exact R02 before-images were checked against author-f001-r02. Independent delta: +677/-22 across original five, journal unchanged; no source change before runtime.go:960 except unchanged context. No sixth source/dependency was added by this reviewer. Main's whole protected-file/staged guard remains Main evidence, not a Reviewer rerun.

| Source in internal/recovery | SHA256, same before/after review |
|---|---|
| runtime.go | 284886C23EA1038B685F1EA5BD1A6B1BC423077042939687C6EF062392594384 |
| admission.go | 0581C8CD4D03B20FB78245904E1E72DB0B5E93734DCEF2B4CF45565055A5D7CC |
| runtime_test.go | 991EA06139EF4F4C2E2845C2EC5EA08B5D35AB718FD4869BF85C4BEDCF95D5B6 |
| admission_test.go | 7DC50B6F4BF577D0C1FEC37EC81531CEC91ED07646982A981C3AB61E41BA1682 |
| journal.go | C2C1806A592299A534D826A7B7DFC5DE5CA18D192AFDA1D1F53E87205C2E218E |

## Two actual review rounds

Round 1: full 860-line R02-to-R03 diff; current controlled runtime.go:873-1352, admission.go:604-799, new runtime_test.go:1183-1609 and test registration, added admission_test.go:50-167 and current fixture. Followed Begin -> restore -> independent verification -> bind -> identity -> Consume -> Exit -> startup. Reviewed immutable namespace snapshots, exact expected Journal append, nil-current-row history, explicit private Store effect, read-only revision fencing, error/cancellation propagation and absence of external calls under Runtime mutex. Stable original R01 mechanisms and four CLOSED findings were reused, not re-audited as new work.

Round 2: compared changed boundaries to frozen r52 current binding/purpose/source contracts and existing Coordinator. Re-read Coordinator production state progression/decoder path and relevant article publication boundaries. All seven original direct consumer files were byte-hashed unchanged against R02; necessary earlier consumer slices were reused. No full-repository scan or future-method expansion. Production Coordinator plus production Runtime/decoder were exercised together with synthetic Journal/Store/article ports; no future implementation was assumed to make a positive case pass.

Contract distinction: ordinary recovery replay reads/reconciles persisted article tombstones and publishes DB evidence; it does not authorize fresh article Journal append. Unchanged article namespace during this call is therefore consistent with the current boundary. Fresh recovery Begin and successful Consume may explain only their exact recovery Journal successors. Private Store effects allow the approved restore counter replacement and owned identity/consumer/session transactions; a pure verifier cannot reset the Store or namespace baseline. These effects expose no arbitrary purpose or SetReady API.

## Finding dispositions

### RECOVERY-B13A-CR-R02-F001: CLOSED

Original HIGH: unexplained same-head/new-revision change during VerifyRestored was adopted as a new baseline and reached BindRestoredRun/open. Original independent failure report/raw remains in review-r02 and original R02 Temp; it is not overwritten.

Current fix: admission.go:621/626 copies complete entry/prefix collections; :779-799 defines stable complete observations and exact append. runtime.go:988-1009 freezes pre-Begin history and accepts only the owned exact Begin successor; :1116-1153 compares both namespaces across calls and phases and permits exact consumed extension only with a fresh matching Consumer result; :1094-1107 separates read-only/restore/owned-transaction Store observations. :1046 and :1248/1259 check read-only Prepare/candidate/verifier intervals. :1303 confirms Exit-owned Store progression before final closure proof. Full expected history and installation binding are still checked.

Independent evidence: TestR03ReboundRestoreRevisionABARejectsBeforeBind now PASS in normal/race, with zero bind calls. Independent cross-call negatives at prepare/replace/verify/finalize/exit for BOTH namespaces all reject and keep startup/work/grants closed; read-only Store drift and in-place same-token history rewrite reject before bind. Positives accept restore counter 100 -> 2 followed by legitimate owned writes and exact Begin/Consume; Exit-owned session transaction counter advance also succeeds. Real production Coordinator legal progress from identity_ready/article_deletion_ready/ready_to_consume/journal_consumed succeeds under the new fence. This is neutral U/C proof, not a real provider CAS/SQL guarantee.

### RECOVERY-B13A-CR-R02-F002: CLOSED

Original HIGH: absent current Store row plus already-consumed current Journal acquired recovery_bind and invoked its write before eventual rejection. Original failure chain remains retained.

Current fix: admission.go:694-695 rejects absent-row current history unless its latest entry remains begun. runtime.go:1160-1161 separately constrains bind purpose to absent row + begun + verified receipt. Existing full history, Begin/run/backup, installation, receipt, independent Store load, ctx and ownership checks still precede effects.

Independent evidence: TestR03ReboundConsumedMissingStoreRejectsBind now PASS in normal/race with zero BindRestoredRun calls. Legitimate receipt-only begun/missing-row Resume binds once and performs no Begin/Replace. Existing succeeded Resume does only permitted Exit/revocation. Real Coordinator test AlreadyConsumedReadyWindow starts from the sole ready_to_consume + exact consumed reconciliation window, performs two DB state transitions, ZERO MarkConsumed and reaches revocation/startup. Author's cross-product tests independently inspected/parsed cover invalid present/absent/stage combinations and wrong receipt/run/backup/install/cancel with zero following write calls.

### Original R01 F001-F004: CLOSED, original chain retained

Original public producer/current-run positive, both Stop cancellation probes, actual Verify join, legitimate concurrent known append and original startup append control all pass again against the CURRENT fixture. None required restoration of a private-phase positive or weakened assertion. The previous producer/cancel/join/failclosed conclusions remain within their already-reviewed boundary.

## Actual independent verification

Test-only overlay replaces only internal/recovery/runtime_test.go using its CURRENT final frozen content plus uniquely named original probe bodies and bounded new probes. No old entire test prefix or production implementation is overlaid. All appended source snapshots use .frozen in Temp. Original R01/R02 artifacts are untouched.

Executed through the unchanged copied Main allowlist runner:

`go test [-race] -overlay <this Temp>/probe-overlay.json ./internal/recovery -run '^TestR03' -count=1 -timeout=30s -json`

Primary raw: checks/r03-probes-final-normal and checks/r03-probes-final-race, each result.json + complete stdout.log/stderr.log. Each actually ran 32 nodes / 29 concrete leaves, all 29 PASS, 0 fail, 0 skip, no race warning. Breakdown: original R01 five probes + original R02 two negative/two positive leaves = 9; independent cross-call/Store/receipt/source checks = 15; production Coordinator state-machine seam checks = 5. Ordinary/race initial runs also each passed all 27 leaves; final runs added Exit Store-advance and in-place history-alias negatives. No test compile failure was credited as behavior. See probe-summary.json for every exact leaf, argv, raw hash and production transition trace.

Production Coordinator normal trace from identity_ready: article_deletion_ready -> ready_to_consume -> journal_consumed -> succeeded, MarkConsumed=1; later starting stages execute only their remaining transitions. Starting journal_consumed makes no MarkConsumed; exact ready_to_consume+consumed window also makes no MarkConsumed. All valid cases finish with startup and one revocation. External ports are deterministic synthetic fixtures; no real SQL, external crypto/provider, root or restore is claimed.

Author current raw independently parsed (not Reviewer execution): final-normal/final-race each 251 nodes/230 concrete run+PASS leaves, 8 groups nonzero, exact original 139 retained, no fail/skip/race warning. Source-format check exit0/empty output; seven manifest exits 2/0/0/0/0/0/0. Author red-behavior has two actual assertion failures on the original defects; red-r02 matched no leaf and is NOT Red proof. The author's preliminary expansion used four over-strict expectations around permitted owned Store writes; those are retained auxiliary runs, not hidden or used to waive current behavior. We independently tested owned progression. Main's separate normal/race reruns and protected1183 guard are Lead provenance, not independent Reviewer test claims. No full-seven or stable230 suite was rerun by this reviewer.

## B13A mechanism-batch technical boundary

Combining original R01 full review + R02 original-four repair/new-risk review + this R03 two-finding/related-risk review now supports technical reception of the B13A single-card mechanism batch WITHIN its neutral component U/C scope. The earlier R02 whole-mechanism blockers are removed. No specific unresolved component-internal source/state/interface/current-binding/purpose/cancellation/true-join/failclosed verification blocker remains identified in that combined scope.

This assessment includes full-history/SQL-NULL/canonical-field/default-closed behavior from stable R01, opaque grant production/consumption and independent current binding from R02 plus current positives, two-attempt/ABA/own-cause safeguards retained from prior review, both revised cross-source fences, legal owned Store/Journal progression, cancel/Verify join and controlled Exit ordering. It is based on these actual combined scopes and evidence, not on a single task label or design/planning PASS. It does not set a G4-batch ledger status or close a Task; Main must independently make those final decisions and required reception checks.

Real retained-key verification/provider, same-Store PG transaction/session ACK, real restore/exclusive driver and actual root/three-worker/process exit remain S4.8/10/13/14/16 responsibilities where allocated. Their absence is not a reverse dependency blocking this neutral S4.7 batch. No real PG/DDL/integration/E2E, production readiness, complete project Gate PASS or human signature is claimed. r52 remains draft/human acceptance unknown; this review does not change design or planning status.

## Global format and remaining limits

Root gofmt remains FAIL exit2, never PASS. All original 23 protected old evidence paths retain exact baseline hashes; both global format raw streams are byte-identical to preSOURCE. Main's new exact-object format-exception-main-r03.json was read and checked (SHA256 1E807A07D8C202C44365A8D5946F43966C73010022AECC35501AC463476DAE67); it applies only to these exact five source hashes and this S4.7 existing old-evidence format failure. R02's exception was not extended. Reviewer neither adopts a waiver nor edits protected evidence. The R03 disposition does not waive source formatting, other six checks, findings, independent review, real verification, Task/batch reception or human confirmation. No pending format-decision item remains for this frozen object.

Bounded review cannot prove arbitrary future consumers or actual deployment safety. Current real service/root/provider/SQL behavior was not executed. No shared writes, secrets, network/install, PG, service/worker/restore, integration tag, cleanup or subdelegation occurred. All probes released/joined their synthetic activity. Final five hashes rechecked unchanged. STOP; hand back to Main.
