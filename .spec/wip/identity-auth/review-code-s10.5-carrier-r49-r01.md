# S10.5 r49 carrier and actual PG/B evidence independent review R01

**CHANGES REQUIRED.** Two bounded rounds are complete. One new MEDIUM finding is open: `S105-CODE-CARRIER-R49-F001`. No HIGH/CRITICAL finding was confirmed. The author's executed PG/B success facts are verified within their actual layers; they do not erase the carrier failure-path defect or close the full software CodeGate.

Reviewer: Laplace, original non-author Reviewer. Fixed Han/main/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`. Sole new review object: `test/integration/identity/password_reset_maintenance_test.go`, SHA256 `D45215F862FB3F733FE1ED3906E45CC25AE3EB03A61FDA3757C177C2927D630D`. The carrier was not authored or repaired here. Only this NEW retained Temp was written.

## Finding

### [MEDIUM] S105-CODE-CARRIER-R49-F001: preflight verification abort bypasses pool closure

- Location: carrier lines **710-713**; failure exits at **234-242 / 745-763**.
- Current behavior: the preflight pool is created, `verifyEndpoint` is called, and the only `pool.Close()` comes afterward. There is no defer or test Cleanup registration for this local pool.
- Trigger: QueryRow/Scan fails (for example, connectivity/auth/query failure), or actual version/endpoint/current_user/database/read-only values mismatch. `maintenanceMust` or `verifyEndpoint` calls `t.Fatal`, ending the calling test goroutine before line713.
- Impact: the failed fixture leaves no close owner for its pool. Installed pgx v5.11.0 starts a background health goroutine at pool.go:334-338; that loop waits on closeChan (506-520), which is closed by Pool.Close (470-475). An established idle connection and the goroutine can survive the aborted test until process exit. Cancelling the fixture context is not closing this pool.
- Basis: coding-standards section3 requires resource ownership/failure convergence; enabled go-service section1.1 requires deferred release. This is a fixture lifecycle defect, not a claim that existing successful runtime records leaked.
- Evidence: **STATIC_CONFIRMED**, frozen source plus exact installed SDK implementation. See `F001-static-evidence.txt` and readset SDK snapshot. No new PG or failure-injection run was performed.
- Recommendation: original author registers closure immediately after successful pool creation, before verification, while preserving prompt normal closure. Same carrier can hold a regression; no eighth source or public API change is needed.
- Re-review condition: actual new frozen carrier SHA, failure-path resource proof for query/metadata abort, relevant regression and source7 drift check. Reviewer does not repair the author's file.

## Fixed input and actual rounds

The author final index SHA256 is `C42F24F21F0A28B40332C014EC59A010CF92D0281DD421D4874E022395503BA0`, rooted at the parent `combined-r49-r01/`. All1454 payloads matched their recorded hashes, including raw logs/observations/proofs and referenced binaries. The durable publication index `AE422A4BAA81B1F81E1D4BAF8073A31EFF3CC02242D33CB84A2A94BD05D0ED44` and its1449 payloads were also verified. Main onsite receipt `DEC89DB19B070BE01453FA9B9D2E5431B37AC7B77CBC7F52AF23BDF9FE5BFD1C` was checked as Main verification evidence, not a Main PG rerun or independent review.

Round1 directly read all1858 frozen carrier lines, preparation diff, original failed raw log and new normal/race raw. It reviewed child binary provenance, process/producer/pipe ownership, preflight/fixture/9-migration scope, all assertions, exact audit/snapshots and failure/cleanup paths, including the complete proxy/COMMIT barrier implementation.

Round2 reconstructed the real consumers and state consequences: current Root context/typed errors/output, same-SHA Domain Requeue/transaction/Audit/replay, real worker Claim/Renew/Fail, migration.Load ordering and applicable schema constraints, and real pgx pool behavior. All274 observations, both manifests/case lists/mandatory prefixes/build records, ACK/worker/guard proof files and binary hashes were independently inspected. Hash inventory is distinguished from content reading in `readset.json`; no whole-repository pre-read or new workflow/delegation occurred.

The17 full Role/Skill/core/security/go-service texts were reused only after exact hash verification; earlier direct-consumer reading was reused at its unchanged version. Upstream six-object manifest remains `478E11DB183DB40B53BF3980BF897753FD8E50331992A4B468455A8A0A4E5BF5`; current ADRr04/designr49/test-planr23 were used, including the changed cooperative C/join contract.

## Actual evidence and precise interpretation

| Execution owner/layer | Verified result |
|---|---|
| Author normal PG/B harness | 121 unique nodes/110 leaves,0 fail/skip;137 observations with actual Wait/producer join and0 watchdog Kill |
| Author race PG/B harness | Same121/110 and137 joins,0 fail/skip/race diagnostic; parent harness is race-instrumented |
| Actual child binary | Default build of ./cmd/han-server, source digests unchanged, same SHA `17B897979630006DE8C8B5B643CFD786E0AA65DC4AACEB9AEFC7AA03FD4DFDF1`; **not built with -race** |
| Fixtures | Ten retained fresh schemas, five per final run, each full nine-file SQL chain and observed PostgreSQL180006/identity/approved loopback5433 metadata |
| Mandatory prefixes | All11 are nonempty with passing leaves in each raw run; prefix count is not equivalent to full Case closure |
| Laplace own command | `go vet -mod=readonly -tags=integration ./test/integration/identity`: exit0,2.578s, stdout/stderr empty, carrier hash unchanged; no tests/PG executed |

All PG/B runtime results above belong to Dalton's actual execution. This Reviewer did not read the actual DSN, launch a child/listener, connect to PG, execute DDL or make new signing material. The local proxy described below existed in author execution, not a new Reviewer action.

**ACK-loss causality:** carrier1171-1309 installs an additional deferred constraint trigger on the exact request UUID inside the new fixture schema. A private advisory lock stops that backend specifically at COMMIT; observer checks the matching COMMIT/advisory waiter and PID. The forwarding mutex then arms opaque server-to-client byte discard before lock release. Observer sees committed ready+Audit, discarded bytes are observed, and only then is the proxy disconnected. The child returns outcome_unknown5/empty stdout; a fresh invocation returns already_applied0, with exact audit/after and unchanged table images/auditCount1. Each run records18 discarded bytes, one connection and two joined pumps/listener. This is real committed-ACK-loss causality, not just a timeout or a trigger that rejects commit.

Both actual ACK barrier records say **negotiated_TLS=false**. Configuration was preserved; this is not TLS ACK-loss validation. All default build args and source-to-binary before/after records were read and verified. No hidden child race coverage is inferred from the parent race command.

**Oracles:** complete typed snapshot is read independently from SQL, persisted original P/signature/after/receipt bytes and SHA256 are compared, int64 above2^53 is preserved, six allowed state combinations are exercised, stale25 fields are rejected, guard/Get adapter counters distinguish real-PG instrumentation from CLI observations, and real worker2-to5 produces attempts3/4/5 with old-request replay and fresh approval. Tables are compared as eight sorted JSONB image hashes. These prove named final-state properties; they are not an exhaustive SQL statement trace.

**Original failure:** normal-initial exit1 and its three HAN_RECOVERY_MODE leaf failures remain retained (five FAIL nodes including parents). The prior oracle expected invalid_config; the actual contracted category is mode_conflict. Current carrier429-433 changes only that expectation for this key. Frozen Root/FD error mapping supports the correction, so this is a contract-backed oracle repair, not weakening an unexplained failure. Historical run120/109 is not relabeled as final121/110.

## Per-Case coverage and limitations

The rows below describe evidence layers, not Case acceptance. Prior same-SHA Domain U and attributed Dewey Root U/SDK evidence supplement named boundaries; they do not manufacture missing PG/B runs.

| Case | Inspected source and evidence | Remaining boundary |
|---|---|---|
| I-S105-01 | carrier:409-416; B bad/mixed argv and no-arg startup failure control; positive maintenance PG leaves | No normal server listener journey is claimed. |
| I-S105-02 | 417-443,1584-1608; B six pre-DSN keys, missing DSN through valid auth, endpoint refusals and actual loopback success | Full numeric IPv6/MaxConns and zero-lookup counters use Root U; these B negatives alone do not measure connect=0. |
| I-S105-03 | 112-145,445-483; Actual current Windows owner constructor; wrong signed subject | No second actual Windows SID. |
| I-S105-04 | 136-145,1797-1823; Independent synthetic approver registration/signature and new-invocation trust mutation | No natural-person/production signing or launcher proof. |
| I-S105-05 | 417-443,445-483; B config/canonical refusals and legal signed request | Remaining strict-config submatrix relies on current Root/Domain U. |
| I-S105-06 | 271-313,485-509; Four mandatory no-EOF/blocked-producer B leaves; parent holds writer until child exit | Exact native positive-available-before-Read cancellation remains Root U. |
| I-S105-07 | 445-483,970-1009; B duplicate/noncanonical request and exact persisted all-string byte carriers | No complete B replay of every strict JSON subcase. |
| I-S105-08 | 395-400,445-483,970-1009; Real plain Ed25519 and wrong domain; persisted original signature verified | Full field-binding matrix is Domain U plus static wire check. |
| I-S105-09 | 382-407,445-483; Canonical UUID generation/payload and malformed request controls | All reason/ref UTF8/length boundaries are not separately B-covered here. |
| I-S105-10 | 445-483; Actual expired/future request refusals | All key/executor time endpoints and no-skew tests are Domain U. |
| I-S105-11 | 1471-1486,1538-1561; Real locked User cancellation, empty output/nonzero and unchanged images; actual Domain recording call | Opaque cross-context/final-cancel/sticky-end behavior is prior same-SHA Domain/Root U. |
| I-S105-12 | 1311-1335,970-1009; Six legal state combinations x3 fixture nextAt offsets; typed actual after, eight-table images, precision above2^53 | Zero offset is fixture-time equality, not guaranteed equality with later product clock; unrelated job/audit rows are not individually compared on first apply. |
| I-S105-13 | 1338-1369,1826-1847; Real state/hold/MaxInt64/foreign/missing refusals; real worker establishes legal succeeded/invalidated terminals | Failed terminal/state-space remainder relies on Domain U. |
| I-S105-14 | 1705-1750; All25 signed before fields changed independently; state_conflict and unchanged eight-table image | This changes the signed expected snapshot, not every field via a simultaneous external writer. |
| I-S105-15 | 1012-1309,1311-1335; Exact replay and real causal COMMIT-ack-loss through opaque proxy; committed observer, exit5 then fresh replay0 | Both actual ACK trials negotiated TLS=false; no TLS ACK assertion. |
| I-S105-16 | 1380-1417; B giant metadata/giant foreign text/corrupt metadata refusal; real-PG Domain adapter counts guard/Get/update/append | No measured client byte/allocation bound or exhaustive64KiB expansion-edge matrix; recording adapter is distinct from CLI scheduling. |
| I-S105-17 | 1492-1536,1751-1795; Deterministic real-PG absent-guard foreign-UUID insertion/PK failure and exact job rollback; append-only guard/Get; concurrent B requests | Process pairs do not by themselves prove lock overlap. The deterministic adapter barrier proves the specified UUID interleaving. |
| I-S105-18 | 1515-1536,1643-1703; Real worker APIs/leases and separate concurrent maintenance processes | No recorded simultaneous worker versus maintenance/reset/resend acceptance race; this PG/B subset remains unverified. |
| I-S105-19 | 1643-1703; Actual Claim/Renew/SQL with synthetic Audit failure, MaxRetries2 then5, counts3/4/5, old replay/fresh approval | Injected Audit failure is explicitly a test seam; no external mail. |
| I-S105-20 | 1492-1513; Real Audit PK collision after job update rolls back job; appended foreign winner retained | Not a complete B matrix of all Audit errors or post-Append actual text-bound failures. Persistent-state proof does not prove zero attempted SQL. |
| I-S105-21 | 1230-1309; Real committed-but-ACK-lost unknown, empty stdout, one audit and immutable fresh replay | Actual rolled-back and unprovable ACK-loss variants are not newly recorded; Domain U covers error categories. |
| I-S105-22 | 271-363,1118-1128,1538-1579; 274 actual Wait/writer joins, no watchdog kills; proxy listener/pumps join; selected backend absence checks | B does not alone prove every internal SDK/transport/timer join. Dewey R02 supplies Root U/real SDK peer evidence. F001 exposes a fixture failure cleanup gap. |
| I-S105-23 | 409-416; inherited Main six checks; No-arg/argv boundary and Main current-source unit/race/contract regressions | No new production server listener/deployment/E2E run. |
| I-S105-24 | 112-145,258-313; Synthetic local harness controls process env/anonymous pipe and real current SID | Production launcher/ACL/key issuance, cross-dup ownership and second actual SID NOT RUN. |
| I-S105-25 | 1797-1823; New-invocation epoch/key/executor changes refused; username ENV ignored | No real revocation drain, restore exclusion/new epoch operation or cross-backup proof. |

## Cross-seven-file software assessment

This assessment checks a connected chain instead of adding local PASS labels. The frozen seven are:

| File | SHA256 |
|---|---|
| `cmd/han-server/main.go` | `833D3E1ECBC39C6CED1E91D59E63651A32A4AD3A1D04C1CC7662560244BDA6AD` |
| `cmd/han-server/main_test.go` | `4EF46AA9AF542C15FEF8D4688208ABDD75A6674C0A6F134D6645C86F4068D73E` |
| `cmd/han-server/maintenance_stdin_windows.go` | `620583ABAEB1551761359F93EAA7A4D0B79BE8A4E312B2FFD9C1D69A82663A43` |
| `cmd/han-server/maintenance_stdin_other.go` | `A711C46F5D0AD017CA54569BC8862F30EFCDF2989EA13B0C856A47F615D854D9` |
| `internal/identity/password_reset_operator.go` | `A1C2039503A748F82529A01EA57E4EB14C478B9ACD34329770D7474E546D98B5` |
| `internal/identity/password_reset_operator_test.go` | `5923422EFB2F81518ACA07FF178DC3FFFB63310CFE2A75F034EEDA530672C31D` |
| `test/integration/identity/password_reset_maintenance_test.go` | `D45215F862FB3F733FE1ED3906E45CC25AE3EB03A61FDA3757C177C2927D630D` |

Root actual operator preparation occurs under original operation D and before DSN. Current main4559-4602 checks live operation before authority preparation; its final input/error check runs before its own cancel. The real Domain assembly forwards the same context and typed error; no success output is written on Domain failure. Existing Domain report `385BA0CBD60F57BD2C24B84F969C59392176446FD6679F75DE79595CD2B0A57B` still applies exactly to A1C203/592342. Actual child PG/B now demonstrates the shared API/wire/audit/state pipeline at those versions; precision and one-enqueue are tied to actual schema/worker consumers, not solely mock expectations.

Dewey final Root R02 report `56427E3CC2668AC15537A786EEEFE4D0BF5C373527470AAC1B3FEB07EC6B0985` and index `D84FF282DC9045B66B6FD2F4AB71036A88660A9F5D446D6ACC1A2BEC42231E58` were read/hash-verified. Original full R01 plus exact repair diff bridges to current833D/4EF46/adapters620583/A711C46; F003/F004 closed and necessary F001/F002 current regressions remain closed. R02 independently ran39 normal leaves/11 race leaves/vet; these are Dewey's evidence, not a Laplace re-execution. No old-version F001/F002 closure is blindly applied to the new Root.

The exact software conclusion remains **CHANGES REQUIRED**: carrier MEDIUM F001 needs original-author repair/re-review or explicit responsible risk disposition. I-S105-18 and other listed PG/B subcase gaps remain visible. The prior Root and Domain conclusions remain valid within their scopes; they do not imply acceptance of this carrier or all Case layers.

Main manifest index `06D7F466F0FCB54B4FA0C4BF1DD5325E97AB12E91963A062F4CF0882E64416F6`/50payloads was checked. Actual mod verify, vet, build, contract tests, full race and full unit commands each exit0. Product source7 gofmt is0/empty. **Exact whole-workspace `gofmt -l .` exits2**, with retained evidence/probe formatting and two empty absent-file preimage parse errors. This is not all seven checks PASS; no historical evidence was edited or cleaned. Main must preserve/explicitly disposition that check result rather than replacing it with the scoped format success.

## Unverified risks and handoff

1. All runtime PG/B described here is original-author execution whose source/oracles/raw records were independently inspected. Reviewer did not rerun PG, read actual DSN/credentials, execute DDL, start a CLI/listener or create a fixture.
2. Parent harness normal and race records each show121 unique nodes/110 leaves, zero failure/skip/race diagnostic. The real product child was built without -race; its same SHA17B897979630006DE8C8B5B643CFD786E0AA65DC4AACEB9AEFC7AA03FD4DFDF1 is verified. Do not claim child race instrumentation.
3. Both causal ACK-loss runs record negotiated_TLS=false. TLS configuration was preserved; no TLS-enabled ACK-loss result exists. Root TLS deadline-cap validation is a separate Dewey SDK-peer layer, not this PG TLS run.
4. Eight-table image comparisons prove selected final persisted states, not absence of attempted-and-rolled-back/no-op SQL or all database table changes. No SQL/audit/transaction trace establishes universal no-extraSQL; use same-SHA static Domain logic and named real adapter counters only within their scope.
5. I-S105-18 concurrent worker/reset/resend acceptance race, exhaustive64KiB JSONB expansion/wire-allocation boundaries and some Case20 fault branches lack this carrier runtime evidence. Full subcase closure cannot be inferred from a name/prefix or from the prior unit results.
6. Observed process/producer/proxy joins and selected backend absence are not standalone proof of all SDK internals. No r48 hard1.5s completion assertion is restored; watchdog Kill remains test failure. r49 allows already-started noncancelable CPU/native stages to exceed C but requires real join and sticky final failure.
7. Exact Main gofmt -l . exit2 remains a failed configured check due retained probes/empty immutable preimages. Seven product sources gofmt check is0/empty and six non-format commands are0; this is not all seven Manifest checks PASS. No evidence cleanup/formatting was authorized or performed.
8. The carrier fixture directly applies the nine loaded SQL files in a transaction; it does not exercise migration.Runner upgrade ledger/rollback or live production migration. Ten fresh schemas were retained in the inspected author runs.
9. Production launcher/ACL/trust custody/signing, second actual SID, revocation/restore drill and deployment/E2E/human acceptance remain outside this software review.

Optional, nonblocking oracle improvement: first-apply checks exempt the entire jobs/audit tables while checking the selected row, so comparing every unaffected row would strengthen single-target write evidence. This suggestion is separate from F001 and does not allege the current production SQL updates other rows.

**STOP.** Return F001 and this fixed-version report to Main/original Dalton. Do not patch product/carrier or wait for future PG runs to hide the current result. Any repair requires an actual new carrier SHA and relevant independent re-review. No Tasks/Plan/Case/E2E/human acceptance or next-action authority is issued.

Artifacts: `report.json`, `readset.json`, `rules-reuse.json`, `all-payload-hashes.json`, `independent-evidence-inspection.json`, original failure inspection, `F001-static-evidence.txt`, own vet logs/result, final source checks and the attributed Root/Main evidence bridges.
