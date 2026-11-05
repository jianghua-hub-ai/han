B13C historical migration-fixture compatibility — Independent Code Review R01

STOP / FROZEN. Result: PASS for the exact three-source compatibility repair. Findings: zero (no CRITICAL/HIGH/MEDIUM/LOW findings in reviewed scope). No adoption decision or risk waiver is requested by this review. This supports Main's technical reception of the historical-fixture live-directory compatibility correction only; it does not close Task, S4.9 G0 as a whole, source14, B13C batch, PG/E2E or human acceptance. B13A/B13B historical closures remain unchanged.

Reviewer: original Ptolemy/add-reviewer 01a18492-af9e-7ea2-9cd5-e4dd368d7bee, independent of Zeno/add-backend-dev 01a18486-3e11-7852-9723-ad380e0bb392. Workspace D:/AID/VSCode/Codex/Han; main; HEAD 6da8abbceb0fa197364810a92a8fa2d97b38ddca. User authorized readonly source review, personal Temp and isolated exact pure-top LOCAL. Shared source/Task/Plan/design writes0; production overlays0; no subdelegation, Git write, cleanup, real environment, PG or services.

Round 1 — complete exact change

Read the entire author receipt/source3.diff (+236/-4), three before/final snapshots for independent equality checks, actual fixture imports/callers and every appended helper/test in the delta. Independently checked live=final frozen, old images=before hashes. Verified all58 original author index payloads; canonical receipt and diff match original Temp. Canonical index hash and exact artifacts are recorded in probe-summary.json.

The crossmodule cdFrozenMigrations helper at canonical_user_delete_test.go:1264 reads only its nine exact named historical paths into fstest.MapFS before calling unchanged migration.Load. The identity helper s33FrozenIdentityMigrations at auth_security_fixture_test.go:916 reads the existing same-package migrationFiles nine-list and calls the same real production loader. No whole-directory migration.Load occurs before selection, no load-then-truncate, count9→10, replacement SQL parser or production loader rewrite. Helpers are private test-only functions under the existing integration build tag; no public API, new dependency, shared-list write or future S4.9 migration mechanism is introduced.

Full-file independent preservation check: the entire old source prefix/body is byte-equivalent after only the declared standard imports and one loader-call replacement per file; helpers/tests are appended to two files. password_reset_maintenance_test.go has exactly its import removal plus one helper-call change, with the rest of the complete file unchanged. All original business assertions/fixture transaction logic/count9/contiguous-version checks remain. This was recomputed directly from frozen before and current sources, not accepted from the author's preservation JSON.

Round 2 — real callers, shared contract and direct loader

Followed actual cdMigrate (228–243), s33Fixture.createAndMigrate (342–366) and maintenancePGFixture.applyMigrations (794–811). Each now obtains the selected chain before its unchanged length9/version i+1/tx.Exec/Commit assertions. Both real identity callers use the one tested helper in the same package. The shared test/integration/identity/postgres_test.go:42–52 migrationFiles currently lists exactly0001–0009; its declaration and other consumers are unchanged. It remains the explicitly selected shared historical-list contract, not an invented new source of truth.

Followed production internal/migration/runner.go Load42–84 and loadMigration149–174: directory enumeration occurs only against the selected MapFS, names/duplicate versions/order are processed by the real loader, historical SQL bytes pass through its real rejectTransactionControl validation, and result metadata is derived as before. Read the necessary sqlcheck.go1–260 transaction-control/quote handling. No migration runner or SQL execution is invoked by these pure tops. Existing loader validation is not a full PostgreSQL syntax/semantic proof; rejection claims below concern its existing transaction-control/lexical guards.

Independent behavior on current frozen sources

Two exact pure tops were listed and executed through the isolated allowlist runner with test-only overlays reconstructed from CURRENT frozen files. No production source is overlaid. Each overlay adds three subtests inside its existing authorized pure top and appends independent FS-oracle code; the existing nine tests/expectations remain exact.

Each top now actually runs12 concrete leaves: original9 + reviewer3. Combined normal and combined race each run26 nodes /24 leaves,24PASS,zero skip/fail/unfinished/race warning; exact normal/race leaf sets equal. All original18 leaf names are retained and newly pass in our executions. No parent-only PASS, compile-only success, no-match, skip, author's self-test or Main rerun substitutes for these actual events.

The original cases independently re-executed are HistoricalNine, Future0010Ignored, FutureBadSQLIgnored, FutureBadNameIgnored, MissingHistoricalRejected, WrongHistoricalNameRejected, InvalidHistoricalSQLRejected, NoDirectoryEnumeration and RepositoryHistoricalChain (nine per package). Repository positives only read actual historical SQL files; no SQL is applied.

Additional independent FS controls (three per package):

1. ReviewerExactInputAndMetadata supplies independent explicit nine paths, unique SQL bytes (including a valid quoted COMMIT string), and inaccessible poisoned future0010/invalid-name files. The input FS records each ReadFile and rejects every Open/ReadDir/unknown-path access. Result: exactlynine named reads in expected order, zero Open/ReadDir, zero future reads; all real output Version/Name/Path/SQL bytes exact. Thus the test proves actual input access/producer output and not just a grep or successful row count.
2. ReviewerReadErrorCauseAndShortCircuit independently injects a unique read error at each of the nine historical files. All nine positions return nil result and preserve errors.Is(original cause), stop at exactlythat read, do not read later/future paths and do not enumerate. Existing missing/wrong-name tests also pass; no historical-name fallback or silent shortened chain is accepted.
3. ReviewerRealLoaderValidation supplies top-level BEGIN and an unterminated quoted string in the final historical item, separately. Both are rejected by the actual production migration.Load validation, with no SQL execution. The valid exact-byte positive proves the helper does not merely reject all input.

Actual commands / raw

For each of the two corresponding package/top pairs, isolated runner executed once:

go test -tags=integration <package> -list '^<exact pure top>$'
go test -tags=integration [-race] -overlay <this Temp>/probe-overlay.json <package> -run '^<exact pure top>$' -count=1 -timeout=30s -json

Pairs: ./test/integration/crossmodule / TestB13CFrozenCanonicalDeleteMigrations and ./test/integration/identity / TestB13CFrozenIdentityMigrations. Raw/results are checks/list-canonical, list-identity, independent-canonical-normal/race and independent-identity-normal/race. All six commands exit0. Full actual argv/times, leaf names, FS access/error logs and stdout/stderr SHA are in probe-summary.json; retained test-only snapshots are canonical-test-overlay.frozen, identity-test-overlay.frozen and probes-template.frozen.

The runner inherits only the stated OS/path allowlist, forces GOENVoff/GOTOOLCHAINlocal/GOPROXYoff/GOSUMDBoff/empty GOFLAGS and HAN_TEST_POSTGRES_DSN, and assigns isolated nonexistent pgpass/service paths. No init/TestMain was found in either selected directory. Only exact two pure-top selectors (and -list) were used; no old business integration/autoDROP/newSchemaHarness path was called. All logs record isolated_pg=false. The copied runner's legacy descriptive sentence mentions an isolated-PG exception, but its actual implementation always forces DSN empty and contains no PG lane; no such exception was authorized or used here.

Author and Main evidence remain distinct

Independently parsed author raw: original red-canonical and red-identity each9 leaves3PASS/6behaviorFAIL, no skip/compiler failure; frozen Red helper bodies actually delegated migration.Load on unfiltered input. Current author normal/race each18PASS,0skip/fail; author source3 format check empty/exit0 and integration static vet exit0. Original Red/self-test evidence remains immutable and is not represented as a reviewer Red rerun. Our current FS-oracle executions provide the independent technical conclusion.

Main separately reports current pure18 normal/race/source3fmt PASS; this is Main evidence, not our independent test count. Main later opened Noether's article8 stable-core peer SOURCE/U-C window while coordC1/PG remained closed. Our Go executions had already all completed exit0, and none were repeated after this notice. No peer compile failure occurred in our runs. Article peer/source14 closures, C1 consultation, unknown future producer and complete B13C scope are not reviewed or decided here. No claim of a globally frozen dependency graph is made from a three-fixture review.

Exact source3 — before and reviewed SHA256

| Source | Before | Reviewed final (entry=exit) |
|---|---|---|
| test/integration/identity/auth_security_fixture_test.go | 0DF966B9F17AD4DAF3275F3BE6428A37A4E600619524C484279C9EE4E26ED088 | ECBFA61C9531AE51BFD7211D25EE282A841EE4BF08CB674134C6E2B1BAAB8B6F |
| test/integration/identity/password_reset_maintenance_test.go | D4B0C837F27E2654784C779A3C08065F0CE4ECCE78B4ED63072B69125DEA2204 | FED5C115DFF1DB5F140D6BEBF46B406768EB77566B647D0F46C426ECB584EE75 |
| test/integration/crossmodule/canonical_user_delete_test.go | 2C88891DCD6F2E7F2B1B256492679FC6F4341F2F0816451D38E5841E3FE9FA20 | 30D0C3628AEE8AFE9A43E2998FBF7E9FD66FF715F9F188022420BDAF48A405B7 |

Source3 entry/exit SHA match. The direct fourteen dependencies hashed (loader, SQL validator, shared postgres fixture/list, go.mod/go.sum and historical nine SQL files) also match entry/exit; this direct dependency guard is separate from Main's originalfeature14/global ownership guard. Branch/HEAD remain fixed and staged set empty. No shared file was changed by the reviewer. Author primary receipt SHA 84518109EF4C66A64F66C6A9FD6C1C72C38C746E563840CCCDBD55935A5C1651; source3.diff SHA 8C4436AB14E4C4EF4A64B8576FC1BBFE27FC32085371B5865DB1A7856C711A07.

Limits / reception boundary

This correction freezes the INPUT FILE SET for historical fixtures; it does not freeze a second copy of SQL semantics or replace the new S4.9 full-upgrade proof. Reading historical SQL and validating migration metadata/lexical restrictions is portable pure evidence; it does not prove PG execution, schema constraints, rollback/application, real fixture business behavior, full0010 upgrade, native provider/OS behavior, recovery/root/services/workers or E2E. Existing business assertion preservation is proven textually but their PG tests were deliberately not run. No global seven checks or old PG suites were rerun; no global format PASS/exception or batch acceptance is asserted.

No current finding/repair remains in this bounded object. The verified pre-selection mechanism can support Main resolving the specific historical liveLoad/count9 compatibility gap; Main retains G0/Task/batch decisions and successor full-upgrade verification. Any further change to these sources/shared list/loader requires a new fixed-object review of its relevant impact. Original Zeno owns any future repair; this reviewer made none. Work STOP/frozen, all test processes joined, no pending LOCAL/subagent.
