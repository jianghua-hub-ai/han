# S412 Independent Code Review R01 - CHANGES REQUIRED; HIGH F001 OPEN

Reviewer: Confucius / add-reviewer, non-author of Lorentz's implementation.
Actual reviewer ID: `01a1890f-a330-73d1-86fb-236777c54751`.
Fixed workspace: `D:/AID/VSCode/Codex/Han`, main, HEAD
`6da8abbceb0fa197364810a92a8fa2d97b38ddca` as bound by Main's grant.
No Git command or Git operation was performed in this actual lane.

The original identity-auth-S4.12 ten-source implementation has one independently
confirmed HIGH finding. **S412 CodeGate FAIL / CHANGES REQUIRED.** The bounded
single-task B13F whole-batch G4 is **BLOCKED** by that same real cross-producer,
state/history, and schema contract defect. These are separate review conclusions,
not a Task/Plan edit, risk acceptance, human sign-off, or project-wide result.

SOURCE0 / LOCAL / IdentityPG were actually released before packaging, pending0.
All five owned jobs completed real parent/process Wait, kernel Job-zero, and
dual actual pipe EOF/pump join. `release.json` was written first. Everything
after that release was own-D text reading/copying/hashing/packaging; no further
Go, SQL, source write, or service operation occurred.

## HIGH S412-ICR-R01-F001 - recovery allocation ignores administrator history

Primary location: `internal/identity/recovery_password_reset.go:406`, with the
selected generation passed to the new credential at line427. Production SHA:
`B2C28DB6923FCAD3C16700BB87C998E805DA23B78C3188611F9207E9CC15A084`.

The new recovery candidate selects MAX(generation) only from credentials whose
purpose is password_reset. The real schema at migrations/0002_identity.sql:125
has UNIQUE(user_id,generation) across both credential purposes. Migration0008
changes current-credential indexes while retaining that table-level uniqueness;
0010 does not remove it. Existing ordinary reset uses the latest generation
across the account's credential purposes.

The existing AdministratorCreateCoordinator.newAdministratorAggregate at
administrator_create.go:914-921 creates new_administrator generation1. Existing
administrator resend can create later generations. Recovery Finalize at
recovery_execution.go:673/:691 freezes the account and invalidates current
credentials without deleting those rows or changing their generations.

A valid recovered noninitial ND0 administrator with that retained history and
no earlier password_reset history is therefore allocated generation1 again.
The existing store insert necessarily collides with the original global unique
constraint. Its candidate transaction rolls back and the account cannot reach
restored, although its approval, actor, Runtime, and restoration eligibility
are valid. Design F002 line3825 requires an increment of the account's existing
maximum and preservation of old history.

This was first established statically and then reproduced independently on the
unchanged final source in ordinary and race modes. The observed error in both:

`SQLSTATE 23505 / temporary_credentials_user_id_generation_key`.

The failing call was the actual restored entry after valid maintenance
authentication. It was not a failed setup insert, invalid FK/trigger fixture,
fabricated producer, provider violation, or assertion changed to force failure.
Raw coordinates: `raw/probe-normal/stdout.log:95` and
`raw/probe-race/stdout.log:95`; exact events are retained in
`failure-markers.json`.

Impact: legitimate recovery restoration is blocked for backed-up administrators
with administrator-credential history and no reset history. This is not claimed
as a privilege bypass, partial committed capability, or second SMTP send.

Minimal repair direction belongs to original Lorentz: allocate from the full
account-wide credential generation sequence under the existing target User
lock, keep overflow refusal before allocation, and preserve old rows and
constraints. Do not change DDL, delete history, weaken uniqueness, or fix this
through a reviewer source edit. The independent history oracle must remain
unchanged for the repair re-review.

## Paired fixture validity and actual results

The independent probe has two cases per mode:

1. The existing clean ND0 approved positive uses the actual Runtime, initial
   administrator trust preparation/strong change, maintenance HTTPAuthorizer,
   approved-source software fixture, Store, durable dispatcher and settlement.
   It succeeds and performs exactly one secret generation and one Send.
2. The history case adds a backed-up administrator through the actual original
   newAdministratorAggregate producer, original temporary-credential transition,
   and original Postgres persistence method before the recovery run/Finalize.
   Its creation timestamp precedes the run. The fixture represents a lawful
   delivered backup state; it does not claim real historical SMTP delivery or
   exercise a public administrator-creation E2E journey.

The history case explicitly checked, before calling restored, that the target
is suspended/required/recovery_locked, its immutable review source is active
and admin, the review is pending, no reset operation exists, and exactly one
generation1 new_administrator credential remains recovery_invalidated after
the real Finalize. Thus the history is legitimate persisted input to the
recovery contract. The independent approval is correctly rebound to that exact
target; no body or backup role is used to self-approve a new authority.

The history call then generates one secret but sends zero messages, encounters
the real unique error, and leaves the full recorded before/after image equal.
The Runtime callback is unwound and Stop/join completes before the outer test
raises the behavioral failure. Fixture cleanup subsequently closes/joins the
pool and retains the schema. The same positive and failing assertions were
used unchanged in both modes.

| Actual probe | Positive | History case | Actual Go exit |
|---|---:|---:|---:|
| ordinary | PASS | behavior FAIL, 23505 | 1 |
| race | PASS | same behavior FAIL, 23505 | 1 |

There is no build/setup/panic/timeout/data-race diagnosis. The product's SQL
error is preserved as the failure evidence. The Python collector returning0
means valid evidence collection and joins; it is not a Go test PASS.

## Two independent static rounds and final revision reconciliation

The fixed Main review-static-input-r02 was reviewed before author runtime
release. All five production files were read in full: recovery760, model383,
password_reset1389, password_reset_store1953, and password_reset_operator1725,
total6,210 lines. The new unit183 and PG940 files were read in full, total1,123
test lines. The other seven files' complete actual changes, including existing
test changes, were read. Unchanged old test bodies are not claimed as freshly
read in full. The full supplied ten-file diff was independently reconstructed
from its before snapshots and equaled current source bytes.

Round1 checked direct implementation and test defects: immutable approval fields,
private actor/phase binding, D0/current-deleted refusal, old authority and holds,
candidate atomicity, one-owner dispatch, accepted-versus-restored state, no-Mailer
driver, due/lease/attempt/retry/permit handling, atomic G+1 and Audits, whole ACK,
terminal intent/readback, and ordinary refusals. F001 is the concrete omission
in the account-wide generation query, not a stylistic objection.

Round2 followed real producers and consumers through more than two layers:
administrator credential creation/resend, Finalize's retained history, new
restored allocation, durable store insertion, original uniqueness, ordinary
reset and Gate, controlled driver and original Claim/Renew SQL, S410 ACK images,
reviewProof/restoredProof, and Complete's common-C/per-target-P full collection.
Migration0010 was read in full for immutable approval, composite run/user/FK
binding, terminal review immutability, failed-pointer replacement constraints,
and revision behavior. No S412 DDL change or new root/Article ownership was
invented. Actual source ranges and SHA are retained under `static/`.

Main's final ten sources equal the reviewed snapshot hashes. The final author
95,456-byte diff was also reconciled with Main's snapshot diff: all hunk headers,
context and edit content agree after excluding file-label and line-terminator
representation differences. Exact source hashes and reconstructed bytes remain
separate, unnormalized evidence. See `final-delta-reconciliation.json`.
The pre-start nonexistent-grant/snapshot diagnostics are orchestration facts,
not product failures, and were never used as a review input or runtime Red.

The original Role/main review Skill and applicable task/checklist/abstraction/
security/core/GoProfile methods were personally read or reused at their exact
same SHA by this same reviewer. Main's reading is not substituted for mine.
Prior readsets and exact method versions remain in the static evidence. No
future Workflow, implementation repair, task mapping change or new G0 was run.

## Actual scope, source protection, and owned process lifecycle

Actual root: `D:/Temp/han-b13f-s412-independent-12k4b0zg`.
Raw root: its `packet/raw` subdirectory. The precise canonical START is retained
byte-for-byte as `grant.received.json`; `freeze36.received.json` binds all ten
current sources, twenty protected prior sources, and six documents.
Every actual command guarded all36 hashes before/after and retained the ten
source before/after bytes. They were unchanged at release.

| Actual owned job | Parent PID | Actual exit | Wait/Job0/2EOF joined |
|---|---:|---:|---|
| preflight | 63376 | 0 | yes |
| go-version | 17336 | 0 | yes |
| probe-normal | 28880 | 1 | yes |
| probe-race | 60992 | 1 | yes |
| physical | 67576 | 0 | yes |

The personally inspected owned Windows helper was reused at exact SHA
6FD58BE2B28FDDDB47EC3F2F534EAFDAF7EE2CB5C4EC54772D18C357EE1085BD.
It assigns the suspended parent to its owned Job before resume and actually
waits the parent, kernel ActiveProcesses0, and both EOF pumps even on failure.
Kernel accounting totaled77 processes. Sixteen optionally observed own PID
values are a subset, not a per-child creation/Wait or whole-Host census.
Raw records four actual Runtime Stop/callback-unwound joins and four pool.Close
joins with acquired connections zero. No unrelated process was inspected or
killed. Main's PG server was retained.

One Go parent ran at a time, GOMAXPROCS1, -p1, -parallel1, count1, offline/local,
readonly modules, with no install/network. Only this reviewer's prior stopped
warm cache was written. All new TEMP/TMP/GOTMPDIR/work/profile paths were in the
unique D root. The installed Go version is retained in the version raw.

Production overlays0; original assertion replacements0. The new probe text is
appended after the complete current original S412 P-test byte prefix. The old
S411 fixture resource adapter changes only its five exact resource/name literals
for this new database/config/schema prefix; inverse replacement equals the full
current source bytes. Source/staged SHA and per-job inverse checks are retained.
No shadow old package or old S412 prefix was substituted.

## Independent new-schema physical evidence

Only Main's selected identity_dsn field was read privately. The value and private
JSON are excluded from the cachefree packet and never printed. Actual preflight
and final SQL verified127.0.0.1:51936, han_b13f_identity, userhan_b13f, PostgreSQL
180006, and data directory
`D:/Temp/han-b13f-lead-20261030-r01/postgres/data`.
The effective grant is sole reviewer LOCAL/PG: Main's author453 read-only SQL
had already completed before START. The earlier proposed joint window was not
used. No old51935, ambient/business DSN, or author schema was used.

Four exact own UUID schemas were physically read in an independent psql
READ ONLY/REPEATABLE READ transaction after both test jobs joined. Every schema
exists with24 named tables. Full User/run/review/operation/credential/job/claim/
Session/Audit/identifier images and zero backends for each exact own app tag
were retained. SQL text and results are in `raw/physical`; full analysis and
leaf-to-schema names are in `physical-analysis.json`.

Both failing postimages retain the original generation1 new_administrator
credential as recovery_invalidated. Their target is locked/required/suspended,
execution counter0, current-operation pointerNULL, reviewpending with no
restoration pointer, and no new operation. Both positive postimages show the
expected committed restored temporary authority. Original before/after images
and the exact product error jointly distinguish rollback from a successful
restoration or a broken fixture.

All four names were compared offline with the sealed author inventory and are
disjoint. There was zero SQL read/write on the author's453 schemas, no prefix
history scan, no DROP/delete/move/archive, and no source or service operation.
All own raw, schemas/data, private config, cache and work artifacts are retained;
only the cachefree packet excludes private/runtime build material.

## Author evidence, NOTRUN scope, and follow-up

The final author report, current matrix, failure chain context, exported actual
result/leaf records and matching source headers were read as author provenance.
The source-aligned author527 identity/neighbor leaves contain ownU37; session60
and maintenance AuthP42 are separate. Author ordinaryP52 and ordinaryAuth42 are
separate jobs; the author race94 splits into S41252 and Auth42. These numbers
are not added into a reviewer test PASS or a whole-batch result. Exact causal
headers and hashes are in `author-reuse-causal.json`.

Per the grant's first-real-finding stop rule, this reviewer did NOT execute
U37/P52/Auth42 ordinary/race or an expanded ordinary regression matrix. Only
the paired valid-generation-history probe, its race counterpart, preflight,
version, and independent physical evidence ran. That intentional NOTRUN scope
must be covered on the actual repaired revision before a future positive
review conclusion; author green counts do not close F001.

The disclosed execution-counter MAX limitation remains accurately bounded:
author P proves original FK/no-gap rejection of an invalid tiny fixture, U
checks application pre-write MAX refusal, and P exercises auth/credential MAX.
No lawful persisted MAX history was independently executed or invented here.
This review does not turn fixture-constraint rejection into product Red, or
waive a genuine uncovered requirement. Original goimports/golangci availability,
rootold23 formatFAIL, and Main's later whole-source project checks remain
separate; none is relabeled PASS by this packet.

## Separate CodeGate and B13F whole-batch conclusion

CodeGate for the original S412 ten-source final revision is FAIL/CHANGES REQUIRED
because a lawful restored target is blocked by a confirmed generation defect.
Original author Lorentz owns repair. Re-review must retain this oracle, reconcile
the final delta, run the required risk-layer ordinary/race/PG evidence and
appropriate affected neighbors, and prove safe preservation of old history.

B13F contains one task, but its whole-batch boundary also covers real old
consumer/interface/source/state/migration/resource and applicable integration/
performance behavior. Its G4 is independently BLOCKED by the same real mismatch
between the new allocator, original producer/history, and live schema contract.
The two static rounds provide that cross-boundary evidence; a single-card label
or sum of leaves would not establish a positive G4-batch. Remaining runtime
coverage is explicitly incomplete, and no separate measured production
performance PASS is claimed from these small fixtures.

Real approval provider/subject/ACL, real SMTP, native restore, production root,
deployment and E2E remain unverified under their original responsibilities.
Software fixtures do not become production fallbacks, and future root/S416/E2E
are not invented as new local prerequisites. Neither conclusion writes a Task,
G4 ledger, risk adoption or human acceptance. Main's packet reception and any
new original-author source/runtime grant are separate next actions.

Current state: HIGH F001 OPEN; CodeGateFAIL; bounded B13F G4BLOCKED;
SOURCE0/LOCAL/IdentityPG actually released; pending0; STOP.
