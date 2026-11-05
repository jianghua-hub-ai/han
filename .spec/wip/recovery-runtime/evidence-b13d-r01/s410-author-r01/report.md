# S4.10 author delivery report - STOP / frozen

Original Zeno / add-backend-dev, agent 01a18486-3e11-7852-9723-ad380e0bb392. Author implementation and self-test evidence only. Independent Code Review, Main acceptance and Task/G4 closure are separate and are not claimed here.

## Exact object and changes

Fixed Han/main/6da8abbceb0fa197364810a92a8fa2d97b38ddca. The three authorized paths were all absent at actual SOURCE start and are now the only repository writes by this actor: 5,616 added lines (2,694 production, 1,053 U, 1,869 P). No fourth source, migration, canonical encoder, Runtime, Article, design, Task/Plan or protocol edit. Full additions are in diff/source3-full.diff, original absence in source-before/initial.json, final bytes in source-final/.

- `internal/identity/recovery_execution.go`: `7C24CA226D0D8DF5FC722A35630C090DD01B26F655C56F79F6C4F9FBEE85E407`
- `internal/identity/recovery_execution_test.go`: `E51D1B82BC7E510A3DAB944C2CA16B85BFB49B132ED28AF12EED45EF596BE9CC`
- `internal/identity/recovery_execution_postgres_test.go`: `5907393CAC84B4D92B590A77B7B10A462D4D1C94D3510A063071D31787202980`

Production provides typed cross-package construction and a construction-only, one-seal original-five-method forwarder for subsequent root DI. It uses the original Store/pool and public Runtime producer rather than inventing permits. Bind/Finalize/Trust/Complete/Session Exit retain original binding, User -> Store -> domain/record/Audit locking, same-transaction audit and finite-context resource ownership. Failed-idempotency recovery consumes the r56 immutable source wrapper outside SQL, then verifies original six-field claim/cohort/actor and exact audit. Terminal retries make zero new writes. Linked reset unknown/hold/protection remains domain-owned. Canonical A/B and the actual Article same-transaction consumer are exercised with explicit synthetic fixtures.

ACK confirmation uses a distinct physical connection and action-scoped complete relevant collections, original bindings and unique audit. It no longer compares thirteen unrelated full tables for a narrow action. Aggregate Bind/Finalize/Complete/Exit retain the original required full collections and canonical counter/digest fence. Related differences reject; unrelated authorized changes remain valid. No COUNT-only proof or generic failure settlement is substituted.

Restored Complete now reuses passwordResetIdempotencyTerminal and the existing full success envelope. The snapshot read explicitly joins the same operation/user/all six claim fields/run/install. Existing 0008 PK/composite FK uniqueness and actual review/audit/reset-settlement provenance are retained. Generic non-NULL terminal data is not domain success proof. The lawful full F002 positive and linked protections remain.

## Executed failure evidence and fixes

- ACK scope: raw/s410-author-red-ack-05 is a real compiled seven-leaf execution: five unrelated-change positive cases fail under the old full-database image, while two related-difference negative controls pass. raw/s410-author-ack-green-01 passes the exact same seven cases with unchanged U/P bytes. See tdd-ack-proof.json.
- Restored domain proof: raw/s410-author-restored-red-exact is a real compiled one-leaf behavioral failure on the generic-terminal negative. raw/s410-author-restored-green passes that unchanged probe, the full persisted F002 positive and five linked-protection leaves (seven total). See tdd-restored-proof.json. Final complete P has 79 leaves, retaining all original 78 plus the new negative.
- red-ack-01/02/03 were compile or legal-fixture diagnostics (actual Article repository shape, SQL casts, trace shape); red-ack-04 was not accepted as the clean Red because an additional SQL type diagnostic contaminated coverage. All remain in raw/ and raw-index.json.
- pg-01 had 76/78 passes and two setup errors (42P08 parameter cast, 23514 incomplete legal hold tuple). The full legal tuple and explicit casts were fixed only in the owned P fixture; pg-fix-01 passed the six affected branches. Schema/domain validators and assertions were not weakened.
- u-final/pg-final compile diagnostics from an explicit pgx fixture receiver refactor were fixed within the owned test source; they are not Red/Green.
- frozen-fmt and fmt-comment-diff genuinely failed due to CRLF reintroduced by an author text write. format-lf normalized line endings; format-only-proof.json proves that correction only changes CRLF to LF. The final nine delivery jobs all ran afterward on current 7C24/E51D/5907. Old 949F/5852 results are historical, never the final hash claim.

All 51 command records and all original failures are preserved/classified in raw-index.json; exact argv, source-before/after, stdout, stderr, result, parsed leaves, verdict and joins remain beside each raw job. No compile/setup/timeout/Skip result is counted as behavioral Red.

## Final exact-version validation

| Risk group | U normal / race | P normal / race |
|---|---:|---:|
| FinalizeOnceAndAtomicAudit | 2 / 2 | 6 / 6 |
| UserStoreReviewLockOrder | 14 / 14 | 3 / 3 |
| KeptDeletedCanonicalAndDenied | 4 / 4 | 9 / 9 |
| IdempotencyUnknownAndCompleteProof | 40 / 40 | 45 / 45 |
| SucceededRecoverySessionExitAtomic | 2 / 2 | 4 / 4 |
| RecoveryExitAuditAckReadback | 2 / 2 | 12 / 12 |

Totals: U 64 normal + 64 race; P 79 normal + 79 race. All six groups in each mode are nonzero; no fail, skip, unfinished leaf or race report on the final object. Original 64 U/78 P leaf names and assertions are retained; risk-leaves-final.json enumerates every final leaf. Main separately reported exact-name retention verification; this report does not substitute that for independent Review.

All nine final labels in validation-final.json exit 0, with unchanged current three SHA before/after: four directed normal/race runs, list U/P, native vet, static integration vet, and source3 gofmt -l (empty). No additional repeated check was run merely to package evidence. Global Manifest7/Linux/extra cross-platform/full integration were not run by this author; Main owns the later combined window.

## Physical PG, data retention and process ownership

Only the granted new D identity database was used: loopback 127.0.0.1:51934, database han_b13d_identity, user han_b13d, server_version_num 180006, fixed migrations 0001-0010. Only the new exact P top received its synthetic explicit DSN; ordinary commands had empty test DSN. No old autoDROP/old C database/ambient secret was used.

758 unique schemas logged by all raw jobs, including failed historical attempts, exactly match the final physical read-only namespace/table/run/audit observations. They are not 758 distinct final test leaves. See schema-physical-summary.json and raw/s410-author-delivery-pg-race/retained-schemas-physical.json. All schemas and data are retained; semantic DELETE fault injections are not schema cleanup. Main independently reported physical psql/dataDirectory verification in identity-final-raw-pre-reception-r01.json; that is separate Lead evidence, not a SQL operation rerun or signed by this author.

Every one of 51 owned command records has actual parent Wait, kernel Job-empty and stdout/stderr EOF/pump joins; pending owned jobs are zero. Optional PID observation recorded 79 unique PIDs and kernel accounting on jobs with that field totals 1,829 processes. The observed PID set is explicitly incomplete for short-lived descendants; kernel Job zero is the true all-descendant proof. Main separately observed 100 known PIDs. These are different evidence scopes, not contradictory exhaustive totals. All author Rows/Tx/pools/callback goroutines complete and close; cluster stop belongs to Main. Cache/work/module/binary/fixture roots remain outside this packet.

## Scope guard and remaining boundaries

Original 19 protected source objects are byte-exact against the actual SOURCE-start baseline (guard-final.json). This bounded guard is not an assertion that the whole shared dirty tree is clean or that Main's current18 objects were checked by this actor. Existing Main-authorized metadata changes are preserved; no stale full Task/Plan hash is asserted. No Git command/write/cleanup, source outside the three, real production root, approval authority or schema migration was added.

Software fixtures plus the isolated real PostgreSQL execution do not establish provider ACL/approval-subject authority, production root wiring, SMTP/native restore/real maintenance workers, or real restore/full E2E. Missing real providers remain closed for their affected branch. No independent CodePASS, finding/Task closure or G4 approval is claimed. There is no unresolved author test failure on the final exact source; all historical failures remain visible. STOP: source and packet frozen, sole execution window released to Main for original independent reviewers.
