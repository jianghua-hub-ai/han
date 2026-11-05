C5 count=1 C4 count=1 C3 count=1 C2 count=1 C1 count=1 # B7 G4-batch 批次整体核对 r01

- **日期**：2026-10-27
- **执行**：主会话 Lead（Host 无委派能力，无 Reviewer Role 产出）；**依据用户 2026-10-26 明确授权「由作者自审 Code Review（本 Host 可以自审）」**
- **objectType**：code（批次整体核对，非单 task 复审）
- **工作区 / 分支 / HEAD**：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **批次**：B7「删除/恢复纵切与 article 读取」
- **批次范围**：`identity-auth-S5.3`、`identity-auth-S9.1`、`article-management-S2.1`、`article-management-S4.2`、`article-management-S4.3`、`article-management-S4.5`
- **结论**：**B7 G4-batch Gate：PASS（r01 后补记）**。r01 时点为 PARTIAL / OPEN（5 张 task-level CLOSED，`article-management-S4.3` 仍 active 且受外部环境阻塞）；2026-10-27 用户明确确认本项目为测试项目、这版不做真卷韧性验证且后续无需验证，Lead 据此落 **ADR-0024** 并将 S4.3 以范围决策关闭，B7 六卡至此全部闭合。**注**：该 PASS 不含 S4.3 的真卷韧性证据（经决策不验证），弱于「六卡各自取得真实环境验证」的整体接纳。

> 独立性声明：由 Lead 自审口径执行，弱于非作者独立批次评审，作为已知限制保留。

## 1. 批次成员状态（以 tasks SSOT 为准）

| task | 状态 | 关闭证据 |
|---|---|---|
| `identity-auth-S5.3` | `[x]` CLOSED@2026-10-25 | R02 `57FC8599…3ABA`；Lead 同 revision Gate |
| `identity-auth-S9.1` | `[x]` CLOSED@2026-10-27 | `.spec/wip/identity-auth/review-code-s9.1-r02.md`；F001～F004 CLOSED |
| `article-management-S2.1` | `[x]` CLOSED@2026-10-25 | R02 `01373C88…8D4F`；Lead 同 revision Gate |
| `article-management-S4.2` | `[x]` CLOSED@2026-10-27 | `.spec/wip/article-management/review-code-s4.2-r02.md`；F001～F005 CLOSED |
| `article-management-S4.5` | `[x]` CLOSED@2026-10-25 | Gauss R02 `49340355…1F48`；Lead 同 revision Gate |
| `article-management-S4.3` | `[x]` CLOSED@2026-10-27 | **ADR-0024 范围决策关闭**（经决策不验证，非验证通过）；r01 时点为 `[-]` active / 受真实 Unix 独立持久卷阻塞 |

## 2. 固定对象与漂移核对

本轮实测以下已关闭卡片的最终对象 SHA256，与 Plan 记录逐项一致，**0 mismatch**：

| 对象 | 本轮 SHA256 | 与 Plan |
|---|---|---|
| `test/integration/identity/smtp_credentials_test.go`（S5.3） | `9CB60DE64DB1B00B3AA92EB32788CA042A3390FE583D60709FA1F295CA4C8826` | 一致 |
| `test/integration/identity/smtp_test_server.go`（S5.3） | `FEFF245A580D7D848A780EB63C5AD1D0E2B13582ADDCC3BE6C9506C9F82DFB26` | 一致 |
| `test/integration/identity/testdata/certificates/README.md`（S5.3） | `B9CE292F5A8701756646170C128825E6D955BF302D422E928B6664FE4847F73F` | 一致 |
| `docs/runbooks/article-deletion-recovery.md`（S4.5） | `41ECB0CF593511119A1BBC910C3B707D094282D1D06875134F531D58F6DD4A09` | 一致 |
| `testdata/recovery/article-backup-inventory.json`（S4.5） | `1355094A5A6751E6E479F4769C9D6E41C993F2A054D23ACF948D3F7E5C4E9B8C` | 一致 |
| `testdata/recovery/article-deletion-fixtures.json`（S4.5） | `E86F32A1731E04F3B1590A87282F71F4332B488F2B6040E74AAF7F5876CF7D81` | 一致 |
| `test/integration/recovery/article_fixtures_test.go`（S4.5） | `88FD408A466CAF208E780112F82B6EB82BCCDB58D5CB1E848C07B52DFDC3C4F6` | 一致 |
| `internal/article/query_service.go`（S2.1） | `539795E92A4F4276887C9ED3BCD9EEA08C51B1523271CB671D65D2B30B80C777` | 一致 |
| `internal/article/query_test.go`（S2.1） | `DF40F63117A0C8F7569A255EABBF9DD95F81CA86FBAE17F88EE93A8F78A0C53E` | 一致 |

S9.1 与 S4.2 的固定对象哈希已在各自 r02 报告中记录，本轮未重复改动；两卡相关文件自 2026-10-27 关闭后无写入。

## 3. 跨任务依赖与接口一致性（Round 2）

- **writer 隔离**：S5.3（`test/integration/identity` 三文件）、S2.1（四文件，含 `cmd/han-server`）、S4.5（四文件 `testdata/recovery` + `docs/runbooks` + `test/integration/recovery`）、S9.1（`internal/identity` 五文件）、S4.2（`internal/article` 19 文件，含 `cmd/han-server`）写集相互不重叠；`cmd/han-server` 由 S2.1 与 S4.2 串行消费，未发现并行 owner 冲突。
- **接口一致性**：S4.2 生产的 ReplayStatus v1（Ready20/Unavailable6）由 `internal/recovery` 单一 authority 定义，S5.3/S9.1/S2.1 未复制第二份 schema authority；S9.1 的 canonical 删除经 article-owned `OrphanArticlesCanonical` 端口协作，未复制 article SQL/状态 authority。
- **依赖收敛**：B7 各卡 direct requires 均已 CLOSED，批次内无未满足依赖。
- **原仅剩阻断（已由范围决策解除）**：`article-management-S4.3` 原需真实 Unix 独立持久卷执行 Journal/finalizer 韧性专项（部分写、重复 sequence、链/HMAC/key/namespace/fsync/空间故障、双 finalizer、三类删除）；本机为 Windows，无受支持平台。2026-10-27 用户明确决定本项目为测试项目、这版不做该验证且后续无需验证，Lead 落 ADR-0024 并以范围决策关闭 S4.3；**该验证仍未执行、无实测证据，不得外推为真卷结论**。

## 4. 本轮真实执行的回归命令与结果

```text
go test ./internal/article ./internal/identity -count=1                                  -> ok
go test -tags=integration ./test/integration/recovery -run '^TestArticleRecoveryFixtures$' -count=1 -> ok
go test -tags=integration ./test/integration/identity -run 'TestCredentialSMTP|TestSMTPResidualWindow' -count=1 -> ok 2.593s
go test -tags=contract ./...                                                             -> ok
```

`HAN_TEST_POSTGRES_DSN` 使用 User scope 值（127.0.0.1:5433），仅在隔离 schema 内使用，未打印或持久化凭据。

## 5. 限制与未验证风险

- **独立性**：Lead 自审口径（用户已授权）。
- r01 时点批次整体未闭合（S4.3 未执行）；S4.3 关闭后 B7 六卡闭合，但本报告仍不把「5 张单卡 PASS + 1 张范围决策关闭」相加冒充六卡真实环境整体接纳。
- 未运行真实 Unix 持久卷、真实浏览器、真实外部 SMTP（S5.3 仅 loopback 合成）、部署或 E2E。

## 6. Gate

**B7 G4-batch Gate：PASS（r01 后补记）**。

已确认：5 张 task-level CLOSED 卡的对象无漂移、跨任务依赖与接口一致、相关回归 PASS；`article-management-S4.3` 已于 2026-10-27 按 ADR-0024 以范围决策关闭（真卷韧性经决策不验证，三文件未创建、未执行，未验证边界保留）。**该 PASS 的性质**：B7 六卡在「已完成的验证范围」内闭合，但 S4.3 的真实卷 `flock`/`fsync`/崩溃/磁盘故障结论**不存在**；本 PASS 不证明 production readiness、灾备就绪、容量/P95，也不替代 article S4.4（ReplayStatus producer 与旧备份恢复）与 article S1.4（部署后 E2E）的真实环境验证。据此启动依赖 B7 的后续批次或部署/E2E 前，须先核其自身 requires 与该放开范围是否匹配。
