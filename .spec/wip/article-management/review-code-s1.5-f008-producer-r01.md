# S1.5 `S15-RE-F008` 与 producer 真实层证据：非作者独立复审 R01

- 性质：独立只读复审，非作者侧产出。
- 流转说明：本文件由 Lead 转录整理自 `add-reviewer` 子 Agent（actualID `01a19285-8ece-7c20-a161-c0d7b8cce6aa`）的返回正文；该 Agent 为只读委派，未直接写文件。委派时固定工作区 `D:\AID\VSCode\Codex\Han`、分支 `main`、`HEAD` `6da8abb`，禁止任何写入与 Git 写操作。
- 复审对象：`evidence-s1.5/f008-allow-assertions-r01/`（`S15-RE-F008` 修复）与 `evidence-s1.5/producer-real-r01/`（producer 真实 PostgreSQL 证据）。
- 复审时点：2026-10-31。
- 权威边界：本复审不宣称 S1.5 整卡关闭；不代替人的业务接收或后续授权。

## 1. 结论摘要

- `S15-RE-F008` = **NOT CLOSED**。代码层修复成立：三条目标路由的 `Allow` 精确断言和“仅 trace / resolver 零调用”断言均已落地；但证据包把实际 34 个叶子写成 33 个，且记录耗时与保存日志不一致，当前证据不能支持其自身结论，需更正后复审。
- producer 证据 = **接收（有限定）**。可作为 producer 侧真实 PostgreSQL 运行证据消费；不等于 Review PASS、人的业务接收或 S1.5 整卡关闭。

## 2. 逐条裁定

| 编号 | 判定 | 证据出处 |
|---|---|---|
| Q1 `Allow` 是否逐字一致 | 一致 | `cmd/han-server/main_test.go:3402` 比较 `mainOrphanAllow(action)`；`:3413` 为 `GET`；`:3416` 为 `GET, DELETE`；`cmd/han-server/main.go:3613` list `GET`、`:3615` publish `POST`、`:3617` transfer `POST`、`:3653` detail/delete `GET, DELETE` |
| Q2 是否具备负向有效性 | 具备 | 动作集合 `main_test.go:3339`；method 变体改 PATCH `:3368`；405 断言 `:3394`；`Allow` 断言 `:3402`。`GET`/`POST`/`GET, DELETE` 互异，退化常量会使至少一组失败；路径写错会落 404 |
| Q3 是否证明“未触碰 resolver / 无副作用” | 限定成立 | `resolver.calls++` `:3330`；trace seam `:3364`；组合断言 `:3405`；405 分支 `main.go:3669` 设 `Allow` 后 `:3675` 返回，早于 probe/handler；trace 包裹 `main.go:3703`。能证明未调用注入的 resolver、未进入安全链/业务 handler；不能证明“绝对无任何副作用” |
| Q4 日志是否支持报告结论 | 部分支持，存在重大不符 | 33 leaves 不成立：`f008-targeted-normal.log` 末个 subtest RUN 在 `:35`、末个 PASS 在 `:70`，实际 34 subtest、35 RUN/PASS（含顶层），`:72` 为 `ok ... 1.380s`；race/full/vet 支持；gofmt 无保存日志（复审方独立执行输出为空）。报告耗时 `report.md:31/32/33` 为 1.36s / 2.891s / 40.455s，与日志 1.380s / 3.009s / 39.555s 不一致 |
| Q5 F008 是否可关闭 | NOT CLOSED（仅证据记账阻断） | 须更正为 34 leaves、按日志更正耗时、补 gofmt 证据并重算哈希后复审；无需改生产代码 |
| Q6 producer 结论是否被索引支撑 | 是 | `index.json:159-161` 普通 58/0/0；`:182-184` race 58/0/0；`:157`/`:180` `overlay:false`；`:83` `source_stable:true`；10 个源文件哈希复算一致；上游 `leaves.json` 各 58 键全 pass |
| Q7 producer 证据能否接收 | 接收（有边界） | `report.md:35/36` 58/0/0、`:26` 无 overlay、`:47` 10 源清单均有索引与上游结果支撑；PG preflight `required-pg-normal/stdout.log:6` 记录 `version=180006 host=127.0.0.1 port=51934 database=han_b13d_article` |

## 3. 缺陷与风险

- M — 叶子数量错误：实际 34，证据写 33，`report.md:31`、`index.json:51`、`verification-s1.5.md:147`、`tasks.md:1233` 多处同步错误。
- M — 耗时不一致：无法确认报告摘要与保存日志来自同一轮执行。
- L — `gofmt` 结论无保存日志。
- L — race 日志仅单行 `ok`，`-run 'orphan|Orphan'` 较宽，不能仅凭日志证明命中的目标叶子。
- L — `mainOrphanAllow` 的 `default` 返回 `POST`，未知 action 不显式失败。
- L — producer 证据是外部 recovery-runtime 卡产物，`index.json:253` 为 `status:not_reviewed`，仅覆盖 10 个源文件与 `freshD` 单库 Article schema；不覆盖真实 Journal/HMAC/keyring、旧备份、Case18、应用角色 ACL、部署 E2E。

## 4. 复审方实际读取与哈希复核

- 复算一致：F008 的 `index.json`、`report.md`、三份 run 日志与 `go-vet.log`；`main_test.go` `220A49CD…` / 400192 B；`main.go` `6D5373CA…` / 173317 B；producer 的 `index.json` `F840BC2C…` / 19567 B、`report.md` `A9A4ECBB…` / 8023 B，另核 10 个源文件与 11 份上游证据。
- 只读执行：`Get-FileHash`、`Get-Content`、`Select-String`、`rg`、`git rev-parse`、`git status`、`gofmt -l`；未运行 `go test`，未写文件，未执行 Git 写操作。

## 5. 后续处置

Lead 依据本复审结论建立 `evidence-s1.5/f008-allow-assertions-r02/`（单轮执行重采日志、计数与耗时取自日志、补 `gofmt.log`、race 加 `-v`、`mainOrphanAllow` 未知 action 改为 `t.Fatalf`），r01 原样保留追溯，并提请同一非作者复审 R02。