# S1.5/S4.4 模拟环境收口 R01 评审处置

- 评审对象：`.spec/wip/article-management/review-sim-closeout-r01.md`（SHA256 `3BD9139322745DC907C59F81FCBB114879FDA348FC18FA0613174CADAA31DAAE`，154 行）
- 评审结论：PASS WITH MINOR（1 MEDIUM + 3 LOW）
- 授权来源：ADR-0028 + 用户 2026-10-31 授权原文
- 声明：AI 代理登记与处置，非真人签字

## 处置结果

| Finding | 级别 | 处置 | 证据 |
|---|---|---|---|
| S15-SIM-F001：泄漏断言禁词 `ref` 随机命中 26 位 Trace | MEDIUM | **已修复**。新增 `s15AssertNoRecoveryLeak`，先将已校验为合法的 Trace ID 掩码为 `<trace>`，再调用共享断言；其他禁词检查不放宽。 | `evidence-s1.5/sim-closeout-r01/s15-matrix-stress-r02.log`（`-count=25` 通过）、`s15-dispatcher-and-matrix-r02.log` |
| S15-SIM-F002：authorization/origin 未观测，缺 guard-open 正控 | LOW | **已修复**。closed/unknown 两态装配 `seam.authorized` 与 `seam.protectOrigin` 记录器并断言零进入；新增 `guard_open_reaches_authorization_and_origin`，在 `public_ready` 下请求 `POST /api/v1/articles`，断言 201、origin=1、authorization=`create`、acquire/release=(1,1)。 | 同上日志；代码 SHA256 `DA02943B9848C50E1960ECCDBF210C3E69D3AA8A64034F9A6321740E519E9115` |
| S15-SIM-F003：`CheckPhase` 在 `test/integration/recovery` 当前路径不可达 | LOW | **接受为已披露边界**。该方法是目的/上下文校验镜像，只有 `EnsureCurrentPublication`/`ImportLegacyPublication` 被调用时才执行；当前三路径未触达，已在证据中标注“正确但未执行的镜像”。 | `evidence-s4.4/sim-closeout-r01/index.json` `scope.not_closed` |
| S15-SIM-F004：证据命令与并发 S4.20 写 `main.go` 时间交叠 | LOW | **接受为已披露边界**。整树未跟踪，无法用 Git 证明写集；`production_code_changed:false` 只能读作“本交付未改”，已在证据中如实记录并保留并发事实。 | `evidence-s1.5/sim-closeout-r01/index.json` `authorization.disclaimer` 与 `notes` |

## 复跑命令（均 exit 0）

`go build ./...`、`go vet ./...`、`go vet -tags=integration ./cmd/han-server`、`gofmt -l`（空）、`go test -tags=integration ./cmd/han-server -run 'TestRecoveryMainBusinessDispatcher|TestRecoveryIsolationMatrix' -count=1`、`go test -tags=integration ./cmd/han-server -run '^TestRecoveryIsolationMatrix$' -count=25`、`go test -tags=integration ./cmd/han-server -run TestAppendOnlyApplicationACL -count=1 -v`（12/12 PASS）。

## 仍未证明（不变）

Case18 真实双系统全局恢复隔离 `NOT RUN/unknown`；真实 POSIX Journal fsync/flock、真实 keyring/HMAC、真实旧备份恢复与 RPO/RTO、真实部署未验证；不宣称 production readiness；整卡状态不变。
## 记录更正

评审报告表格中 `internal/article/deletion_journal_hash_test.go` 的 SHA256 有 1 字符笔误（`...ACE626BB2BF...`）。工作树实测为 `10F940D8C18984BD5D806BB9D5D6379DC644B22ACE626EB2BF6E4A59AC33FB48`（1349B），与 `evidence-s4.4/sim-closeout-r01/index.json` 一致。以实测与证据索引为准；评审报告作为独立记录未作改动。