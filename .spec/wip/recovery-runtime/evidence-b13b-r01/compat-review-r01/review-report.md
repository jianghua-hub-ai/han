# Independent B13B G0 compatibility code review R01

Conclusion: PASS for this exact bounded two-source compatibility correction. New findings: 0. The specific S4.8 G0 blocker caused by SHA-only admission checks on genuine Journal HMAC values is technically resolved. This is not full S4.8 G0 readiness, SOURCE authorization, a full-repository/Task/batch reception or human signoff. STOP/frozen; return to Main.

Reviewer: original non-author Ptolemy/add-reviewer, 01a18492-af9e-7ea2-9cd5-e4dd368d7bee. Author: original Zeno/add-backend-dev, 01a18486-3e11-7852-9723-ad380e0bb392. Workspace D:/AID/VSCode/Codex/Han, main, HEAD 6da8abbceb0fa197364810a92a8fa2d97b38ddca. B13A historical findings and acceptance records are not reopened or rewritten. No implementation/shared document/Git writes, subdelegation or cleanup.

## Exact object

Author input: .spec/wip/recovery-runtime/evidence-b13b-r01/compat-author-r01/index.json, SHA256 E74330FA635D3758C862AFFC42D4A8EF26CB06372FE3C859193DA02542CF534F. All 52 indexed retained files were independently byte/hash checked, including both before and final frozen sources. Entry and completion live/frozen SHA matched. Complete two-source.diff read; independent delta +5/-2 production, +155/-3 test, no third reviewed source. Test changes update only HMAC fixture generation/signing and add 41 compatibility leaves; existing assertions were not deleted or relaxed.

| Source in internal/recovery | Before SHA256 | Reviewed SHA256, unchanged during review |
|---|---|---|
| admission.go | 0581C8CD4D03B20FB78245904E1E72DB0B5E93734DCEF2B4CF45565055A5D7CC | E234CC71C5A21D42EFD6E8B345F2499800B607F2D8AF66043A0EB1A4D38FD3F0 |
| admission_test.go | 7DC50B6F4BF577D0C1FEC37EC81531CEC91ED07646982A981C3AB61E41BA1682 | 02E9232EAC4A87F81582038304C1B06D6C6381DE2959CD45B95F7286B4CBFB0A |

Producer/adjacent hashes independently matched this correction's protected preimage baseline: keyring.go DDB9B972F83E9CA7A2CEBCB19745179C5BBFF9FCC1D8A29BBB80C62FBE45DEBF; journal.go C2C1806A592299A534D826A7B7DFC5DE5CA18D192AFDA1D1F53E87205C2E218E; replay_status_v1.go 2F2FFA4B7357D9E7D63B2F923B9F212D89CED112EB88E0B453D3C5E8E41FDA82; runtime.go 284886C23EA1038B685F1EA5BD1A6B1BC423077042939687C6EF062392594384. Other whole-tree/metadata guard observations remain Main/author provenance, not a Reviewer rescan.

## Two actual review rounds

Round 1: complete exact production/test diff and surrounding validators/fixtures. Checked that the new private expression at admission.go:28 is exactly ^hmac-sha256:[0-9a-f]{64}$ and applies only to AdmissionRun.JournalSecretFingerprint (:239) and recovery JournalEntry.SecretFingerprint/RecordHMAC (:452). Anchors, lowercase ASCII hex and exact width reject whitespace, alternate prefixes/algorithms, plain SHA and malformed strings. No trim, prefix substitution, dual algorithm, global SHA widening, new exported API or algorithm/wire change.

Round 2: traced actual native producer fingerprintSecret -> hmacDigest, Begin/MarkConsumed -> computeRecordHMAC -> disk JSON -> toEntry, and actual FileRecoveryJournal.validateRecords authentication. Then followed current admission CanonicalAdmissionSnapshot/validateRecoveryEntries -> complete history/exact Store+entry matching -> public Runtime.RunNormal -> Acquire binding. Refreshed r52 canonical/source/immutable Begin bindings; r52 exact SHA remains 5173FF8830142CA7E699AEB7A49945E8D6694C67372A1F3F2D557FF3B119ACAB. The existing same-format HMAC is admitted as an opaque authenticated source value; Runtime does not take the keyring or replace the producer's authentication. Native parser still verifies MAC before a verified-source adapter may assert trust. No future S4.8 implementation was fabricated or assumed to pass a test.

The dedicated SHA expression remains unchanged in replay_status_v1.go. Installation target/commitment, artifact/approval/schema/receipt, evidence/publication, Store canonical digest and Journal chain/prefix/head remain actual SHA fields. Consumer equality still compares exact original HMAC strings, key IDs/versions, run/backup/stage/time. Journal source format and algorithms are independently pinned below, not inferred only from a regex or the author's result.

## Independent executed evidence

Python stdlib HMAC-SHA256 oracle uses only literal synthetic reviewer key/candidate, independent of the author's fixture builders and expected vector. oracle.json fixes the fingerprint, exact ordered MAC payload JSON, two authenticated begun/consumed disk records, and SHA256 record-chain hashes. Probes call real fingerprintSecret, computeRecordHMAC, encodeDiskRecord and toEntry, require exact equality with that oracle, authenticate the records through real validateRecords, then feed the complete corresponding Store/namespace/closure projection into public Runtime.RunNormal and obtain WorkRequest with exact run-a/backup-a binding. No native Journal constructor, file lock or media operation is invoked.

Primary test-only overlay contains the CURRENT final admission_test.go plus independent probes/constants. Only this test path is overlaid; no production path or old entire test file is substituted. All Go source artifacts are .frozen in personal Temp. Executed through a byte-identical copy of Main's OS/path allowlist runner:

`go test [-race] -overlay <Temp>/probe-overlay.json ./internal/recovery -run '^TestCompatReviewer' -count=1 -timeout=30s -json`

Primary raw directories checks/probes-final-normal and checks/probes-final-race each contain result.json, stdout.log, stderr.log. Each actually ran 62 nodes / 58 concrete leaves; every leaf PASS, zero fail/skip and no race warning. Initial checks/probes-normal and probes-race each passed 55 leaves; three valid-format related-risk cases were then added and the expanded final tests rerun. Initial overlay is explicitly reconstructed from the retained final image by removing that appended three-case block; final logs use the retained final overlay. No compile failure, skipped test or synthetic success was counted as a failing-behavior reproduction.

58-leaf breakdown:

- 1 independent real-producer/wire/authentication -> public Runtime/lease positive.
- 4 real authentication controls: original key now verify-only with another active key still validates; modified fingerprint, MAC and backup fail real validateRecords.
- 36 malformed-value cases: three HMAC fields x twelve encodings (plain SHA, bare hex, wrong algorithm, upper hex/prefix, short/long, newline, NUL, Unicode, empty, nonhex). Dedicated consumer rejection and zero public startup/work independently asserted.
- 14 genuine-SHA field checks: target, commitment, evidence, publication, restore target, artifact, approval, schema, receipt, initial/current/previous/prefix hash and Store revision Digest reject HMAC through actual current Runtime proof.
- 3 valid-format risk controls: different legal HMAC in Store/entry still fails exact history binding; in-place valid-HMAC historical mutation with same namespace token during restore verification returns ErrAdmissionDrift before any BindRestoredRun/startup. This avoids mistaking early syntax rejection for evidence that the existing binding/immutable-history fence still works.

See probe-summary.json for every exact leaf, command/environment, result and raw SHA; oracle.json and probes.frozen retain positive/negative inputs. No independent probe failed in this review; there is no new failure evidence to hand back for repair and no finding severity/location/recommendation to invent.

## Author evidence and retained risk boundary

Author raw was independently parsed completely, separately from Reviewer execution: normal/race each 271 concrete run+PASS leaves; all original 230 exact names are retained, 41 new compatibility leaves, eight nonzero risk groups, zero fail/skip/race warning. Red exit1 contains four concrete behavior-failing leaves (three native HMAC positives plus rejection of plain SHA Store fingerprint), not undefined symbols, no-match or timeout. Current list, two-source format and package vet exit0; package regression exit0 has 13 pre-existing Windows/POSIX platform skips, explicitly not POSIX evidence. Full current author raw/result hashes are in evidence-audit.json. The Reviewer did not rerun all original stable tests or all seven repository checks.

The source comparison preserves original B13A assertions; fixtures now derive their three HMAC values from the existing producer algorithm. No canonical-field omissions/NULL changes, proof attempts, state machine, Store effects, current binding, purpose capability, cancel/join or closure logic were changed. Existing 230 paths/current author raw plus unchanged Runtime and the new valid-HMAC exact-binding/history controls support the bounded conclusion that this syntax correction did not weaken those risk boundaries. Original B13A findings remain their historical independently closed chain; this new code is evaluated under this separate review, not grandfathered by old PASS.

## G0 disposition, checks and limitations

The concrete G0 compatibility gap is resolved at the pure component boundary: genuine original producer HMAC can enter the consumer; fake SHA or malformed HMAC cannot; actual SHA remains SHA; true parser authentication, wire bytes and exact history binding remain enforced. This permits Main to remove that specific compatibility blocker when coordinating S4.8. It neither opens the S4.8 eleven-source window nor asserts the future reader/context/publication implementations exist or pass.

No whole-repository seven-check claim is made. Root format's known old23 baseline result remains FAIL exit2 per Main; it was not rerun, relabeled PASS or independently waived here. The S4.7-only exceptions are NOT inherited by this B13B correction. Main owns combined checks and any exact new-object disposition; component code conclusion alone cannot accept a repository/Task/batch gate.

This Windows review executes pure parser/HMAC/canonical/Runtime U/C only. Native POSIX media/locks, actual Anchor/provider, real PG/DDL/SQL/restore/root/services/workers/integration/E2E remain NOT RUN. No real ENV/DSN/key/secret was read, no download/network/install or service launched. Existing OS/path variables alone were carried into Go, with GOENV off, GOTOOLCHAIN local, GOPROXY/GOSUMDB off, GOFLAGS and test DSN empty. r52 remains draft/human acceptance unknown; no design or SSOT status changed. End-of-review two-source and producer/dependency SHA rechecked unchanged. STOP/frozen, Main final reception pending.
