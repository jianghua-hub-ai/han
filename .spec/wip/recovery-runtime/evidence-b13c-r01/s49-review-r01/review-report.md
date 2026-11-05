B13C identity-auth-S4.9 — Independent Code Review R01

STOP / FROZEN. Result: PASS for the exact S4.9 six-source software/component scope. Findings: zero; no scoped blocking defect or adoption decision remains. This independently supports technical reception of this card's AdmissionStore/0010/publication work, not Task closure, B13C cross-card/batch acceptance, production rollout, future C1/C2 implementation, recovery Root/E2E or human design acceptance. B13A/B13B and the accepted three-fixture compatibility repair remain unchanged.

Reviewer: original Ptolemy/add-reviewer 01a18492-af9e-7ea2-9cd5-e4dd368d7bee, independent of author Zeno/add-backend-dev 01a18486-3e11-7852-9723-ad380e0bb392. Fixed D:/AID/VSCode/Codex/Han/main/HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca. Six reviewed sources STOP/frozen throughout; entry/exit SHA match, staged0 and direct dependencies unchanged. User authorized these reads, own Temp/test-only overlays and only the explicit fresh identity PG resource. Shared source/Task/Plan/design/evidence writes0, Git writes0, production overlays0, delegation0, cleanup/DROP0.

Round 1 — entire fixed six-source change

Read all four new sources in full: recovery_admission_store.go197 lines, encoding tests369, new PG harness/tests922 and0010 SQL230. Read the complete modified recovery_run_store.go344 lines and its full diff; read all431 lines of the modified old integration test and its complete diff/before assertions. The entire source6.diff2047 lines (+1843/-39) is covered by those complete new texts and two modified diffs; independently reconstructed the exact full diff from before/final bytes. All111 canonical author-wrapper payload hashes verified; source-index is retained as the original author's package index and owner is Zeno, not Main/self-review.

Production Store: one identity-owned RecoveryRunStore implements AdmissionStore. Snapshot owns a short actual RR READ ONLY transaction, reads singleton installation/revision and ALL recovery_runs LEFT JOIN restore fields in C byte order, checks each Scan and final Rows.Err/ctx.Err, closes Rows before Commit, owns rollback on error/panic and preserves original/cleanup causes. Query/scan/commit failure returns no successful partial set. Revision read redoes the full snapshot/canonical digest, with no cache/latest/limit/status filter, xmin proof or write/repair. Installation InstalledAt, all nullable evidence/consumption/publication/audit/restore and five initial-admin fields retain their real SQL NULL distinction. Complete data validation/digest uses recovery.CanonicalAdmissionSnapshot as the single encoder.

0010: inspected all installation/restore FKs, immutable bindings and paired nullable publication; initial user/bootstrap pair, changed generation/audit/time tuple and exact successor; sessions ordinary default/four-null versus maintenance/four-present +positive generation; seven-field operation approval pair/role/status/digest/actor FK and immutability; review pointer's account AND run/account composite FKs and deferred transactional use; immutable review source/terminal metadata and bounded candidate replacement. Installation internal revision updates do not recurse; run INSERT/UPDATE (including failed/same outcome), restore insert, review and bound-operation writes increment the same installation row in their transaction. Ordinary operations/reads do not bump; bigint overflow aborts the business update atomically. Original0005 one-way/delete/binding guards and0008 authority/lineage constraints remain. Migration does not invent installation, restore/proof/publication or success for legacy rows.

Extended Load/Transition use a separate 18-column published projection of the SAME Store, validate/persist/return publication already supplied by real Coordinator, and join cleanup causes. Failed Commit remains ErrRecoveryWriteUnknown with original cause for independent classification/readback. Old scanRecoveryRun and selectRecoveryRunSQL16-column text are independently exact versus before, and deletion.go is unchanged. Original Create's synthetic nonempty fingerprint compatibility remains while new admission validates canonical HMAC. Old integration fixture migration is moved to the new retention-only harness (full9→10); UTC/mainline/duplicate/CAS business assertions were manually traced through the helper split, with original assertion-message counts retained. The old autoDROP helper is still historical code but not reached/executed.

Round 2 — real contracts and consumers

Read the complete current S4.9 task card and its four minimum risk groups. Independently read current identity design Store/canonical1518–1524, scope3708, initial proof3710–3712, operation/review3783 and3801–3803; TP r24 P0 Store/RR/counter/NULL/Rows.Err/retention contract and F001/F002 metadata responsibility. Design r53 is a known C1/C2-only successor; no draft C1 obligation was imposed on S4.9, and design adoption/human acceptance is not inferred. Original r52 contract/TP/S4.7/S4.8 stable context reused.

Followed actual recovery/admission.go complete AdmissionSnapshot/AdmissionRun encoder+validation1–337, readProofAttempt364–397, full-history completeAdmissionRun468–515 and recheckProof587–603. Runtime verifies recomputed complete Store digest/counter and source closure; a structurally readable legacy succeeded row is not sufficient for admission. Followed real Coordinator publication producer195–225, transition/load/readback317–365 and current binding validation426–431; Store's change mapping matches its expected readback. Followed deletion.go760–796 and994–1008 old transaction-scoped lock-only consumer; retained16 columns have no publication projection bleed. Read unchanged migration.Load and old0002/0005/0008 relevant schema/authority/review/lineage constraints. Direct source boundaries remain neutral and identity-owned; no reverse import or second authority introduced.

Independent actual LOCAL / P

Fresh personal Temp test-only overlays append probes to CURRENT frozen encoding/PG test files and register them under the two authorized exact tops; original38 encoding and22 PG concrete leaf names are retained and freshly execute. Production overlays0. Unit additional3 leaves inject installation-query error, run-query error and original query panic into a test-only Tx adapter: empty result/no Commit, one Rollback, original+cleanup causes or identical panic survive. This adapter is explicitly synthetic; real pgx/PG behavior is separately executed below.

Current encoding normal/race: each48nodes/41concrete leaves,41PASS,0skip/fail/unfinished/race warning. Current PG normal/race: each31nodes/26leaves,26PASS,0skip/fail/unfinished/race warning. Required original PG groups remain exactly4/7/4/7. Each mode combined67 leaves = original60 + reviewer7, all PASS; normal/race exact leaf sets equal. Lists matched both exact tops. No parent/list/compile-only result substitutes for leaf events.

Additional independent PG probes, beyond the retained22:

1. Explicit Python stdlib JSON/hash oracle contains the fixed complete field order/fixture values, with three runs (including failed and byte-order changes), all six distinct counters, nullable evidence/publication/audit, complete restore and five proof fields, installation and expected revision9. Real Store's canonical bytes and returned SHA digest match it exactly. No tested encoder generates the expected bytes/hash; projection-oracle.json is retained. Recomputed actual digest confirms the single-encoder consumer chain.
2. Actual bound operation UPDATE increments revision inside its real transaction; independent pool connection still observes the old revision. Rollback preserves old counter/data. A proof UPDATE likewise increments inside the same transaction, is invisible outside until Commit, then proof+counter appear together in an independent snapshot. No long transaction spans external file/CAS work.
3. Valid alternative backup role/status/deleted_at mutations on a pending review each return real55000 and unchanged revision; a same-outcome actual UPDATE increments once. This exercises the metadata trigger's source/actual-write branches, not just schema grep.
4. Actual public Runtime.Verify consumes the REAL Store/PG snapshot after9→10 upgrade. A legacy succeeded row without provenance is closed with ErrAdmissionIncomplete and lease released; after explicit synthetic proof/restore/evidence/publication setup, the same real Runtime/Store chain reaches internal_ready, makes the exact historical closure request, releases its temporary lease and still denies ordinary WorkRequest. Journal/closure/exclusive endpoints are test-only synthetic sources; no actual restore/root/provider/ordinary worker is claimed. Runtime algorithm and cryptography were not copied into an oracle or production.

Retained original PG cases also actually verify RR READ ONLY trace/public Store, independent-writer snapshot isolation, real PostgreSQL22012 delivered via pgx Rows.Err using a query-fault Tx wrapper, missing-table closed, cancellation/no leaked pool acquisition, concurrent Store transition single winner with true join, real Commit + synthetic lost ACK for read/write with independent readback, revision writers/rollback/ordinary exclusions, overflow, NULL/UTC, source/FK pairs, legacy upgrade and atomic DDL rollback. The injected query and lost ACK are synthetic fault adapters around REAL PG transactions; they are not a real network outage or production approval/provider.

Exact execution and resource boundary

Fresh endpoint only127.0.0.1:59672/han_b13c_identity as han_b13c from Main postgres-resource identity_dsn, privately injected only for exact TestRecoveryAdmissionStorePostgres. Main local_s49/pg_s49/postgres START checked. Standard OS/path allowlist only; GOENVoff, GOTOOLCHAINlocal, GOPROXY/GOSUMDBoff, GOFLAGS empty, testDSN empty for other commands; pgpass/service inheritance isolated. No existing DSN/secret ENV/article DB read, no printed/copied connection resource. Fixture preflight actually checks host/port/version180006/user/db before DDL; quoted per-schema search_path applies to pool connections/transactions. All52 independently created unique han_b13c_s49_* schemas are recorded and retained in retained-schemas.json; failed-build attempts created zero. No DROP/delete/cluster stop occurred; Main owns final cluster stop/join. Pool/Rows/Tx/context/test goroutines are closed/joined on command completion.

Actual successful raw directories:

- checks/review-s49-encoding-list and review-s49-pg-list
- checks/review-s49-encoding-normal and review-s49-encoding-race-r2
- checks/review-s49-postgres-normal-r2 and review-s49-postgres-race-r2

Actual tests, each -count=1:

go test [-p=1] [-race] -overlay <Temp>/probe-overlay.json ./internal/identity -run '^TestRecoveryAdmissionStoreEncoding$' -count=1 -timeout=120s -json
go test -p=1 [-race] -tags=integration -overlay <Temp>/probe-overlay.json ./internal/identity -run '^TestRecoveryAdmissionStorePostgres$' -count=1 -timeout=120s -json

Full argv/results/timestamps, exact leaf paths, group counts, raw SHA, fault classification and schema names are probe-summary.json. Test snapshots remain .frozen, with no raw .go files in repository evidence.

Diagnostics retained, never credited as behavioral Red/Green: the reviewer's first runner rejected differently separated Windows path spellings before Go/PG began; corrected only that Temp path normalization. Initial concurrent builds then hit VirtualAlloc errno1455/out-of-memory for encoding-race and both PG builds (zero test run/schema); encoding-normal passed. Original failure raw remain, retries were serial with -p=1/GOMAXPROCS2 and unchanged source/assertions, all missing validations completed. No OOM/peer compile diagnostic is called a product finding or skip. No peer compile failure appeared in successful runs.

Author/Main evidence provenance

Canonical index owner is actual Zeno; Main wrapper retention is not independent authorship. Author current encoding38/PG22 leaf sets are independently compared to our new executions, and original publication compiled behavioral Red is reparsed (one actual fail); author recovery390 and other checks remain separately attributed self-test evidence. Author source-five fmt/current static vet evidence is inspected as applicable, not labeled additional reviewer execution. We did not run global seven, broad integration, old business/autoDROP or other reviewer's Article tests. No current global format PASS/exception is adopted or extended; source fmt is not root fmt. Main joint validation/format applicability and Curie's/other batch reviewers remain separate.

Exact six-source objects (entry=exit)

| Source | Before | Reviewed SHA256 |
|---|---|---|
| internal/identity/recovery_admission_store.go | absent (new) | BF82CE97E42E42A8B4EBDF664A0F80D7BE40F71698A40B47ED78EB8B957B8DD6 |
| internal/identity/recovery_admission_store_postgres_test.go | absent (new) | BA9DC2902C60C8814D41F6CB1D5AF2DFA6645289AE1820B38168E9CD7A02A403 |
| internal/identity/recovery_admission_store_test.go | absent (new) | C146E38306FA5B607DD9D6AD3D8921F8FF5B957FD6DD6EBBE61DB07EEE820F3B |
| internal/identity/recovery_run_store.go | 519D7E98A4908A7AC4EF10C91D990575C4C8B3D3B621B4B69E6BD7589E722056 | A816375E9A5B0FEAD2B46360BFC5FFF7FB166E47524744CB341352FB418BDCE9 |
| internal/identity/recovery_run_store_integration_test.go | CADC4A4529ECAB2135FA2BAF3CCF632A5FC0E0B922272F2B528C50C78B7DB1F2 | 26C3B4C592EB65C31CB89021A492891F3E16C168C87CC00602862D0A4B9833B7 |
| migrations/0010_recovery_runtime_metadata.sql | absent (new) | 874FE07A24BF4CC763D7FB53C7E4B50CA389996D609FF2E772C93246ADD37A24 |

Author receipt SHA 13C6B8AFEBCF50AE9CC75F0A7ED9AFF72316A522C9189110442B7309F5EA7C94; canonical index SHA 915C736A2BCAE52B0FD393D9194EA828BC946608FC6EDB75A5CF140B38F4E09C; full source6.diff SHA 83CB9C7783A68F4DC9BFC4C3028001D0E987FA7CAD6EE639398D4DED4EC52680. All reviewed six and thirteen necessary direct dependency hashes match entry/exit. Metadata writers and Noether's known peer source window are excluded from any broad frozen-source assertion; no Article failure is attributed to S4.9.

Limits and handback

No scoped technical blocker remains from these two rounds and current67-leaf-per-mode independent U/C/P evidence. This supports S4.9 local mechanism reception (full Store projection, canonical/revision,0010 metadata and publication/lock-only compatibility), not a blanket whole-project/Task/G4-batch conclusion. Full live restoration, approval/provider/ACL, Session issuance/resolution and business recovery finalization, article C1/C2, real root/worker lifecycle, native POSIX/filesystem crash durability, production migration rollout/rollback, deployment/E2E and human design acceptance were not exercised. Metadata fixtures/real SQL do not claim complete password/Session domain transactions. Old PG business tops and POSIX integration top were deliberately not run; their modified fixture/assertion compatibility was inspected and relevant risks exercised only in the permitted new tops. Counter+digest is local drift detection, not external rollback/ABA proof; external Begin/deployment exclusion obligations remain.

Findings/adoption items: none in this fixed object. Any subsequent six-source/dependency change requires new freeze/hashes and impact re-review. No self-fix, SSOT update or Task/batch close was made. All LOCAL complete, no pending process/subagent; STOP/frozen and return to Main.
