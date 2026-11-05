# S412 F001 Independent Re-review R02

Reviewer: Confucius / add-reviewer, independent non-author.
Actual ID: 01a1890f-a330-73d1-86fb-236777c54751.
Original author: Lorentz. This reviewer made no implementation repair.
Workspace: D:/AID/VSCode/Codex/Han; Main-bound main at
6da8abbceb0fa197364810a92a8fa2d97b38ddca. No Git command or operation was used.

**S412-ICR-R01-F001 HIGH: CLOSED at this exact repaired revision.**
**S412 component CodeGate: PASS within the original ten-source scope.**
**B13F substantive whole-batch independent technical review: PASS, bounded below.**
New confirmed findings: zero. These are three separate judgments. Neither the
technical whole-review result nor the component result signs a Task, a G4 ledger,
human acceptance, project-wide check waiver, deployment, native recovery or E2E.
Main retains formal reception, project-check disposition and Task/G4 recording.

Current reviewer work was actually released before packaging: SOURCE0 / LOCAL /
IdentityPG, current pending0. Release SHA256:
1AFAAFB5AF1CEC97A303A71076A201E8B5F4791CFC3C449DA8ADEBD0E9DF59A5.
The historical interrupted physical job is separately UNKNOWN; it is not included
in the35 successful truejoined jobs and its original raw was not overwritten.

## Fixed object and exact current source

The final Main input is
D:/Temp/han-b13f-lead-20261030-r01/F001-review-input-r02.
Its ten sources equal the earlier personally reviewed immutable F001-static input
byte-for-byte. The complete three-file255-line repair delta was read, independently
applied to this reviewer's sealed R01 baseline, and rebuilt current source bytes
exactly. Other33 entries in final36 match R01. Full snapshot/freeze reconciliation
is under static-r02; every successful actual job also guarded current36 before
and after, and guard-final.json verifies the same36 at release.

| Original source under internal/identity | SHA256 |
| --- | --- |
| recovery_password_reset.go | B41595D82D7EDDE97ABEF01EE6EAB2DDAEB01328926627A2A0E7DB468D696625 |
| recovery_password_reset_test.go | 3D62E4D78F2008C3FB37F227F5AA2CF4CD84AD9BE7D36788918D69B9A83B9920 |
| recovery_password_reset_postgres_test.go | 3A68C94A588F5FC3031E624827A55CED56ACFA06B8A3B671995E789AB46D6AD1 |
| model.go | F5EAF92F4F74392B781F61340C692CF960F3D097889B4A97E2462CE7256C9E30 |
| password_reset.go | 99EFF799CBBBB119DFC7F5DD7D5B00E9ACA5C108983F3A5500BCE9F0E4824DE9 |
| password_reset_store.go | 388CD8DF8E319E11E8D96B55574B11627A42960620386861AF2A1F4E2C638E89 |
| password_reset_operator.go | D3BC9D822961E03D99FA72A5BD12089B145A16F27916101D4FB859D466BE92C7 |
| password_reset_test.go | 1DCCB3BFA888E2C45F82DE4D848D67F6E0A5902A5B9433F018DBA735D4EB4570 |
| password_reset_store_test.go | CE33D8CBCC527AB4024A3F1C4E7F87C264BB13E8D9CC4617DACA0F8DE0BE6543 |
| password_reset_operator_test.go | C6285DDB71BB9FAA6D710DD0C105BE78399AC43702291C99128171F1B50573BE |

Design r56 SHA75AA4C5D449E2CABEF6C1F66D731CB98BF679693B6AE4CD0E6D8BA0494B200BF
and TP26 SHA34826759C79A58EB0B4A296D2DD5C9157014406859153DC6CA0FA6E2BD921DAB
remain the original technical basis. Their draft/human acceptance status is not
changed by this code review. Original Task10 paths, seven risks, required layers,
source approval and r55/r56 terminal/Complete contracts remain in scope.

## Original HIGH F001 and independent closure

R01 found that recovery_password_reset.go:406 selected MAX(generation) only for
purpose=password_reset. Original migrations/0002_identity.sql:125 requires
UNIQUE(user_id,generation) across purposes. A backed-up noninitial administrator
with retained new_administrator history and no password_reset history was thus
allocated generation1 again and could not be restored. The actual R01 ordinary
and race failures were SQLSTATE23505 / temporary_credentials_user_id_generation_key,
with clean ND0 positives passing. Those failures remain immutable in prior-R01
and the original full sealed packet. They were real Go exit1, not a green test
expecting failure, invalid FK setup, approval failure or provider violation.

The repaired query at line406 removes only the purpose restriction. It now
selects the full target account's maximum. Candidate holds the authoritative
target User FOR UPDATE from settlement.locked through create and Store writes.
The original generation<0 or MaxInt64 guard still executes before secret/ID
allocation. Query/context errors return immediately. The constructor receives
generation+1 only after those checks. No DDL, constraint, history or approval
contract was relaxed.

The original independent4917-byte paired oracle is unchanged, SHA256:
23925EB7228619D421EBAAE988E856C915CE3DECFA03429E2F15CE7C698F64A4.
It was appended after the entire current original P-test source prefix. In R02,
both original leaves pass in both ordinary and race modes, actual Go exit0:
clean ND0 restoration and the actual original-producer administrator-history
restoration. The producer uses newAdministratorAggregate, the original
credential transition and Postgres persistence before real Finalize, maintenance
authentication and ResolveRestored. This is a lawful software backup fixture,
not historical SMTP/provider or public administrator-creation E2E proof.

Independent execution also covered the author's three actual history variants:
consumed generation1 becomes a new reset generation2; active generation1 is
retained as recovery_invalidated with new generation2; actual resend history
generations1/2 remains superseded/invalidated with new generation3. Each variant
runs in both modes. New independent SQL confirms all six final states and that
the old administrator rows equal their full after-Finalize values. The Go
assertions also require one secret/Send and terminal replay with no new image,
ID, secret or Send. Four added U guards pass for MaxInt64, negative value, query
error and context error with zero preparation effects/writes.

This paired Red-to-Green, exact production delta, real old producer/history,
locked allocator, durable Store/schema contract and actual whole postimages are
the closure basis. Author green counts alone were not used to close HIGH F001.

## Two independent review rounds and exact reuse

Round1 checked the full actual repair delta and all new U/PG bodies: account-wide
allocation, overflow/read cancellation, preserved old assertions, valid original
history producers, exactly one durable owner/Send and immutable terminal replay.
No new direct defect was established. The seven unchanged sources were compared
byte-for-byte to R01; the only production edit is the MAX query.

Round2 followed real sources/consumers through at least two layers: original
administrator generation1 and resend producer; original User-first locks;
Finalize's retained consumed/invalidated history; recovery candidate allocator;
Store's operation/credential/job/claim transaction; global migration uniqueness;
ordinary reset's existing account-wide latest generation; maintenance authority,
Session, driver, ACK and terminal/Complete consumers. Relevant current ranges
and exact SHA are in the readsets. Migration0008 changes current-state indexes
by purpose while retaining account-wide generation uniqueness. No new migration
or Article/root ownership was invented.

This same reviewer reuses R01's actual full5production6210-line review and
fullnewtests1123-line review, complete original ten-file delta and real-consumer
rounds. Current R02 additions and wiring were read completely, and exact byte
reconstruction proves the final revision relationship. This is not a claim of
mechanically re-reading every unchanged old test body or all6210 lines anew.
The Role/main Skill were refreshed in full; all19 personally read/reused
Role/Workflow/checklist/abstraction/security/core/GoProfile versions match their
original SHA. Main's reading does not substitute for this reviewer's reading.

## Actual independent current execution

Actual root: D:/Temp/han-b13f-s412-f001-actual-r02-vgg1cji5.
All commands, raw stdout/stderr, exact leaf names, sources and process receipts
are under packet/raw. No test was rerun after the Host/PG incident.

| Actual selection | Ordinary | Race |
| --- | ---: | ---: |
| Current original S412 U | 41 PASS | 41 PASS |
| Current original S412 PG | 55 PASS | 55 PASS |
| Actual adjacent maintenance Auth PG | 42 PASS | 42 PASS |
| Original unchanged independent paired oracle | 2 PASS | 2 PASS |

Each mode has140 executed leaves; total280 PASS, zero FAIL/SKIP, matching leaf
sets and no data-race/build/setup diagnosis. These are execution counts, not a
formula for whole-G4 approval. All seven P risk groups retain their original
names. Their actual per-mode distribution is15/7/9/5/8/4/7:

| Risk group | Actual semantic coverage |
| --- | --- |
| TrustedApprovalAndDailyRejection | Source version/run/install/actor/target/role/status/ref/digest binding, D0/current-deleted refusal, legitimate ND0 and ordinary refusal |
| OldAuthorityUnknownBeforeBoundaryAfter | Old source/lineage and unknown protection at real time boundaries without erasing history |
| CandidateAtomicAndCrossTargetCAS | Same-key ownership, cross-target isolation, atomic failures/counters and all three account-wide histories |
| DurableSingleSendAndAckLoss | Durable owner-only dispatch, real rejection/failure pointer and commit ACK reconciliation |
| ControlledDueLeaseDriverNoMailer | Driver due/lease/retry/permit rules, no ordinary worker authority upgrade and no-Mailer settlement |
| AcceptedAtomicRestoredAndSuspended | Accepted evidence versus complete atomic restored/suspended capability, audit and G+1 |
| TerminalWholeReadbackAndComplete | Immutable intent, whole proof/negative partials, terminal retry and S410 full Complete collection |

Auth42 is a separate real Runtime/maintenance Session/strong-change/exit
regression, including old-cookie rejection, current-cookie use, CSRF, Gate,
expiry, runtime/generation/lineage, audit proof and exit ACK boundaries. It is
not counted as S412 P55. Original ordinary reset/administrator neighbors are
additionally reused from source-aligned author U554, which contains ownU41.
Author P99 is exactly own55 + Auth42 + original oracle2. The four author result
headers and full raw JSON leaf streams were independently parsed; both modes
have equal names, source10 before/after equals current and actual author exit0.
Author evidence remains labeled author evidence, not additional reviewer runs.

One Go parent ran at a time, GOMAXPROCS1/-p1/-parallel1/count1, offline/local and
readonly modules, with this reviewer's stopped warm D cache and private D work,
profile and TEMP paths. No other agent cache was written. Production overlays0,
old assertion replacements0, no shadow package. The unchanged S411 resource
adapter changes exactly five resource/name literal groups; its inverse equals
the full current source. Actual source/staged/prefix/oracle SHA proofs are kept.
Message authorization and Main/hash guards are not claimed to be OS source ACL
or an OS CPU-affinity restriction.

## Independent physical postimages and process receipts

The initial post-test physical collector was interrupted when the PG connection
closed during a separate Host/test-resource incident. Original physical raw has
190 complete schemas, one partial and seven missing. There is no final result
or join receipt for that job: PID76304 Wait/Job0/twoEOF remains UNKNOWN. A later
current-PID absence was not used to invent historical Wait. Original stdout,
GB18030 stderr, command and last lifecycle record are preserved unchanged.
This is an environment/resource incident; no product defect or native recovery
success is inferred. Main's old helper24773 and failed short startup budget also
remain separately attributed historical evidence, not this reviewer's joins.

Main recovered the exact retained test data directory without initdb/delete/DDL
and supplied a new physical-only grant. No Go/build/test ran in that continuation.
The new proof reads the same exact198 own schema names from the immutable
original inventory. It uses25 independent psql processes, each one independent
READ ONLY REPEATABLE READ transaction, maximum8 names per process (last6).
This also releases per-backend catalog cache between batches. No old author/Main
schema, namespace discovery/prefix scan, DROP or history write was used.

All25 new batch processes actually exit0 and independently complete suspended
Job assignment, process/parent Wait, kernel ActiveProcesses0 and both actual EOF
pump joins. New SQL verifies the same host127.0.0.1/port51936/database
han_b13f_identity/userhan_b13f/PostgreSQL180006 and data directory
D:/Temp/han-b13f-lead-20261030-r01/postgres/data.
Only the permitted identity_dsn field was consumed privately; no DSN value or
private JSON is in the packet or printed.

All198 schemas exist:114 with24 tables and84 with23 tables, total4668 complete
table images. Every exact own app-tag backend count is0. The last raw retained
User/run/Session/Audit images match new SQL field-for-field for all198 schemas,
792 comparisons. New physical history values and old-row preservation close
the original schema/history contract in addition to the test assertions.

The35 successful actual owned jobs are10 original completed jobs plus25 new
physical processes. All35 have actual Wait/Job0/dual EOF receipts. Kernel total
accounting is275 processes;76 optional observed own PID values include known
parents and are a subset, not a per-child creation/Wait or whole-Host census.
Raw retains216 actual Runtime Stop/callback-unwound and198 pool.Close/acquired0
records. Rows/Tx lifecycle is supported by source cleanup, actual pool joins
and exact backend exit; no nonexistent per-object instrumentation is claimed.
The interrupted job is excluded from every successful-join total.

One own-D analysis parser initially attempted a long raw row from a single Go
Output frame and failed JSON parsing. It was corrected to concatenate Output
for each Test before parsing; all792 values then matched. That collector-only
diagnostic caused no SQL/test rerun and is not a product Red.

release.json was written before formal packaging, current pending0. Historical
UNKNOWN remains explicit and is not retrospectively signed. After release only
own-D text read/copy/hash/packaging occurred. Main subsequently reported separate
reception,4probe/96table independent SQL equality, backend0/source36 stability,
and pg_ctl/server parent Wait0 plus supervisor44359 exit0. Those are Main
observations from its later message, not reviewer execution or fake old joins.

## Separate component and substantive B13F whole-review judgments

Component CodeGate PASS follows closure of the original root cause at current
B415/3D62/3A68 source, independent unchanged-oracle Green, legal consumed/active/
resend histories, error/overflow pre-write guards, original seven-risk execution,
affected Auth regression, new physical whole images and no new blocker.

B13F's bounded whole-batch technical independent review also PASSes because its
new recovery interface composes with actual old sources and consumers: source
approval/maintenance actor binding; ND0 versus D0; User-locked account generation;
immutable administrator history and original migration uniqueness; same Store
and atomic claim/job/credential/User/audit ownership; ordinary Gate and old
unknown protection; owner-only mail and controlled no-Mailer driver; whole ACK,
terminal retry and S410 Complete. These were separately followed through real
code and actual PostgreSQL boundaries, not inferred from one Task or280 leaves.
There is no remaining confirmed implementation blocker within this boundary.

Necessary performance/resource applicability was examined: the account-leading
unique index supports the scoped maximum, the repair adds no transaction,
goroutine, network operation or widened lock order, driver/permit behavior is
covered, and new physical readers use bounded processes/transactions. No
production benchmark/load/SLA result is claimed, and no broad new perf work is
invented for this one-line allocator repair.

This whole-review PASS is a bounded technical review output for Main's G4
assembly; it is not a declaration that every project check or business gate is
green. Main's same-version project evidence still records rootfmt FAIL2 on the
original23 with owned30 format empty/PASS. goimports/golangci unavailability is
NOT PASS. This reviewer neither accepts/waives those failures nor imports an old
finite disposition as a new one. Main must retain the current project-check
results and any actual authorized disposition in its separate formal closeout.

The execution-counter MAX boundary remains honestly limited: scripted U
pre-write refusal, invalid tiny-history P rejection by original FK/no-gap rules,
and actual auth/credential MAX coverage are not a legally persisted MAX-chain
execution. No such chain was fabricated, no uncovered requirement was waived,
and no fixture rejection was relabeled as product failure.

Real approval provider/subject/ACL, real SMTP, native restore, production root,
deployment/E2E and human acceptance remain unverified. Software positives do not
become production fallbacks. Future root/S416 obligations are not invented as
new local prerequisites. No source, metadata, Task, Plan, DDL, Git or service
change, cleanup/delete/move/archive, subdelegation or author risk adoption was
performed by this reviewer.

Final: original HIGH F001 CLOSED; current component CodeGate PASS; bounded
substantive B13F whole-batch technical independent review PASS; new findings0;
current work released/pending0; historical interrupted lifecycle UNKNOWN retained.
STOP. Main reception and formal project/Task/G4/human decisions remain separate.
