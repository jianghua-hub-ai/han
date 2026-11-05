# S1.5 producer 真实层证据盘点 r01（消费 recovery-runtime B13d 外部卡证据）

- 性质：只读盘点。本轮不执行测试、不改生产代码、不改任务状态、不做清理。
- 执行者：Lead（本会话主会话），非子代理产出，无 Role 交接。
- 时点：2026-10-31；工作区 `D:\AID\VSCode\Codex\Han`；分支 `main`；`HEAD` `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 授权来源：用户 2026-10-31 持续授权“对待确认或决策项采用建议值并继续本项目”，按 ADR-0028 在不宣称真人签收的前提下做可执行层核对与登记。
- 盘点对象：S1.5 卡内“真实 Publish/ReadCurrent/source 证据先消费原 S4.2 的精确版本/现役 PG 验证”这一输入条件。

## 1. 结论

现役 `internal/article` 的 `PostgresReplayEvidenceStore.Publish` 与 `ReadCurrent` 已在真实 PostgreSQL 18.6 上取得普通与 `-race` 双轮证据，每轮 58 个叶子测试 PASS、0 fail、0 skip、exit 0；运行时绑定的 10 个源文件 SHA256 与当前工作树逐字节一致。S1.5 卡对 producer 真实层的“消费外部精确版本 + 现役 PG 验证”输入条件因此可视为已满足。

边界先说清楚：该双轮证据由 recovery-runtime B13d 的 `a48-f001-repair-author-r02` 产生，**不是 S1.5 本轮执行**，本目录只做只读盘点与消费登记。它不覆盖真实 Journal/HMAC/keyring、旧备份恢复、Case18 全局恢复隔离、应用角色 ACL 或部署 E2E。

## 2. 证据来源

- 证据目录：`.spec/wip/recovery-runtime/evidence-b13d-r01/a48-f001-repair-author-r02/`
- 作者：Noether / add-backend-dev，actualID `01a1861b-9405-7942-8dd2-2db8b8d39eb2`（来源卡 Article S4.8 F001 局部返工）。
- 来源索引 `index.json`：`968A8FBECD3F553D69728DD252D0C8F73C59F6A88C5B867142DFB08EE338BBD4` / 32021 B。
- 源码保护清单 `source-protection.json`：`C9D35B0B4B32D0AF636FF8093C7BEBB92F7C839F5A13F24FC8651FD3822B196E` / 49186 B；其中 `internal/article/replay_evidence.go` entry/exit 均为 `7C14E74D…` 且 `unchanged:true`。
- 十源前后对照 `source10-before-after.json`：`6AE6D329E52497CB2958870A26DF97F1CC347E4A679206523B6ECB3B6275AEC7` / 2726 B。
- 来源任务说明 `report.md`：`9C152CADFACA8A32156BE07B1B2C37F16DBF49F4C133A3BC2EE3B17FB1A8769C` / 3749 B；第 13 行记载 r02 physical preflight 实际证明 PG 18.6、`127.0.0.1:51934`、`han_b13d_article`。

## 3. 双轮运行

两轮命令（下列为 shell 展示形式，`-run` 的单引号是 shell 引用；逐字 argv 见 `commands.json` 的 `required-pg-normal` / `required-pg-race` 项，均无 `-overlay`，即测试二进制直接使用现役源文件）：

```
go test -p=1 -vet=off -tags=integration ./internal/article -run '^(TestReplayPublicationPostgres|TestDeletionReplayPostgres)$' -count=1 -json
go test -race -p=1 -vet=off -tags=integration ./internal/article -run '^(TestReplayPublicationPostgres|TestDeletionReplayPostgres)$' -count=1 -json
```

| 轮次 | exit | pass | fail | skip | stdout SHA256 | stdout 字节 |
|---|---|---|---|---|---|---|
| `required-pg-normal` | 0 | 58 | 0 | 0 | `52BBB9828525F14DA8AC2E04DF7E437B9FAED5444E83880143B721E0058531F9` | 275985 |
| `required-pg-race` | 0 | 58 | 0 | 0 | `2602BD3C6F679CFEDD7D649B4D0B8AB31B9181521C8CAFB5751A21D62FAAA8F9` | 275949 |

- `leaves.json`：普通 `2F0F01CD9D2D3366E2A8CD56D043040747A37820CBDBCF0712769F3CFF8E1819` / 21180 B；race `B44F93D1F892B6239F525EFEF8562CD31A41D62A1F09AAE2411CE22A47AC3180` / 21174 B。逐个叶子 action 均为 `pass`，无 `fail`、无 `skip`。
- `result.json`：普通 `A306259A59726C009161E73D4366F2F941461242ABCF02777F3BF1B1AEA4C160` / 2135 B；race `A4449600EA0C84AA2ECE8227101731C38ED0B50DED255ECAB768B12BE8A7D6C0` / 2149 B；均 `wait_returned:true`、`race_diagnostic:false`、`pg:"freshD Article only"`。

运行环境（`run.py` 第 31、37-38 行）：独立 `TEMP/TMP/GOCACHE/GOTMPDIR`；`GOENV=off`、`GOWORK=off`、`GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`、`GOFLAGS=-mod=readonly`、`GOMAXPROCS=2`；`PGPASSFILE/PGSERVICEFILE/PGSYSCONFDIR` 指向包内不存在的 sentinel 路径（`packet/no-pgpass`、`packet/no-service`、`packet/no-sysconf`，用于禁用既有配置读取）；`HAN_TEST_POSTGRES_DSN` 先置空，随后断言取值含 `@127.0.0.1:51934/han_b13d_article` 才注入，不存在回落到环境既有 DSN 的路径。

## 4. 源一致性（现役 vs B13d 绑定）

两轮 `result.json` 的 `source10` 与本轮实算的现值逐项相同：

| 文件 | SHA256 | 字节 |
|---|---|---|
| `internal/article/replay_evidence.go` | `7C14E74DCEF68A1B8067DCEC2BF50BA4D1FA9BA165AEE5C52B3828D0D60417FB` | 34397 |
| `internal/article/replay_publication.go` | `34B698A231B32101ACBF1B91AB7843FF1710D529C748072CFDF27BA1CF180B8E` | 37500 |
| `internal/article/replay_publication_integration_test.go` | `8F5DD24DD40BE6A1A6CD4FBB9EF5990EE7EDBCEB964FF9ACF6BD41E5967540FD` | 65102 |
| `migrations/0011_article_replay_publication.sql` | `7C7F1B1F0BEFDBF572B0EE768FAC3ADD987D58DF8CA4525072CD1EE269F84CC2` | 2762 |
| `internal/article/replay_publication_test.go` | `412A145A50A374C689D29EA3050DE48BCB9C827E7D7DB4E927CE08A828F69B05` | 20382 |
| `internal/article/deletion_replay.go` | `33AB20A7EB3A963889B9A8F027D89A8407C67F3D9C7DB85FC7814F9D1B3B577D` | 16175 |
| `internal/article/deletion_replay_source.go` | `485A4621FA22AD7442D9ACF36A9B36118C6CE3C96E37E6DA8EB2FCFA2293DF61` | 10144 |
| `internal/article/deletion_replay_test.go` | `A4923E7B58D74EAB743C21F681F1010E8AAC592396A723E7BF3B3D0B181B02F4` | 30151 |
| `internal/article/deletion_replay_source_test.go` | `2F8E38D81F4CC88E03E02AFC0602501ACC2241E40FC0270D94B270D756EA0695` | 15341 |
| `internal/article/deletion_replay_integration_test.go` | `CB2E859535E0D8BC421298BDE13452271DC0B6B3EAE005BEA9CF76E8224A5F50` | 15900 |

`cmd/`、`internal/`、`test/` 未纳入 Git（`git ls-files` 为 0），因此一致性判断只依据 SHA256 与字节，不依据 `git status`。

## 5. 真实调用链

- 集成 fixture `publicationPGFixture` 直接持有 `store *PostgresReplayEvidenceStore`（`replay_publication_integration_test.go:194`），由 `newPublicationPGFixtureOnPool` 用真实连接池构造（`:241`）。
- 回滚/原子性用例 `publicationPGRollback` 直接调用 `f.store.Publish(f.ctx, request)`（`:455`）与 `f.store.ReadCurrent(f.ctx, f.run, f.backup, request.Refresh)`（`:461`），中间不经过替身或内存实现。
- 顶层叶子 `TestReplayPublicationPostgres/PublicationCurrentEvidenceAtomic/RefreshOutsideRunAndRecheck/Publish` 与 `.../ReadCurrent` 在普通与 `-race` 两轮 leaves 中均存在且 PASS。
- PG 门禁 `newReplayPublicationPool`（`:31`）：`HAN_TEST_POSTGRES_DSN` 为空即 `t.Fatal("explicit isolated Article DSN is required; no fallback")`（`:35`）；host 非 `127.0.0.1` 即 `t.Fatal`（`:42`）；源码无 `t.Skip` 分支。两轮各 58 leaves、0 skip，说明门禁是真实满足而非跳过。

## 6. 边界与不覆盖

1. 本证据是 recovery-runtime B13d 外部卡产物，不是 S1.5 本轮执行；S1.5 只消费其固定哈希结果。
2. 覆盖范围仅为 Article schema 的 `freshD` 单库（`pg:"freshD Article only"`），不证明真实部署集群、多库/多命名空间或生产备份介质。
3. 不覆盖真实 Journal/HMAC/keyring 密钥链、旧备份恢复、Case18 全局恢复隔离、应用角色 ACL 载体与部署 E2E；这些仍是 S1.5 未满足项。
4. B13d 内 `red-*` / `green-*` 轮次为 F001 回归使用 test-only Go overlay，与 `required-pg-*` 两轮不同；本目录只消费后者，不把 overlay 轮次当现役源证据。
5. 本目录未做独立评审；producer 项应记为“已消费可核证据（待独立评审/接收）”，不记为 Review PASS。
6. 不关闭 S1.5 整卡：`[-]`、十二项 requires 与完整关闭条件均不变；不代签人的业务接收。

## 7. 对 S1.5 的推进

- producer 真实层从 unknown 推进为“已消费外部可核证据”。
- `S15-RE-F008`、真实 Journal/旧备份、Case18 全局恢复隔离、`article-management-S8.2` 整卡、真实部署 E2E 与最终接收仍未满足，整卡继续 `[-]`。