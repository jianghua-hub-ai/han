# S3.3 安全专项补证：Lead 核收

日期：2026-10-28。原任务 `identity-auth-S3.3`。固定工作区 `D:/AID/VSCode/Codex/Han` / `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。

Lead 已核原前置、当前成果、实际执行、原 Reviewer 逐 finding 复审、工作树与原变更后关闭条件，判定条件满足，决定于本日关闭且仅关闭 S3.3。Tasks SSOT 由 Task-Planner 按该决定同步；真实浏览器保持 ADR-0025 的 **WAIVED / NOT RUN**，不记为 PASS。

## 本轮成果与问题闭环

Test-Engineer Bacon `01a18077-e5df-7240-baac-bdee83a21685` 先核原 I-AUTH-01/03/04/07，定向非浏览器普通/race 各 26 top / 84 leaf / 97 节点通过，同时披露 G01/G02。非作者 Bernoulli `01a18077-e81e-7472-8311-f13b823c954f` 的 [R01](review-verification-s3.3-r01.md) 将它们确认为 `IA-S3.3-R01-F001/F002` 两项 MEDIUM 未豁免验证缺口，非已复现产品 Bug。

Lead 采用补齐建议，Task-Planner 在原卡内登记两份新增 integration 测试；Backend Erdos `01a18086-c383-7b72-bcfe-75635bd6c600` 实施并自测。仅新增：

- `test/integration/identity/auth_security_postgres_test.go`：418 行，SHA256 `0A96A652CFBE74B7E222E1774BAAD4C50606588C298593982D0F9820A602A4AA`。
- `test/integration/identity/auth_security_fixture_test.go`：911 行，SHA256 `0DF966B9F17AD4DAF3275F3BE6428A37A4E600619524C484279C9EE4E26ED088`。

原 183 项应用源码、迁移、module 与既有测试保持原字节。当前完整输入 185 项（163 Go），平坦清单 SHA256 `CC86009A6350314DA549D81F551B85ADEB1CA9AFA83F7298E860BAD80D8D75C7`。原两 browser 计划文件依豁免仍未创建，没有补造浏览器证据。

[原 Reviewer R02](review-code-s3.3-r02.md)，SHA256 `2FDAEA71618E280CE4301E7B4AA5591F3B8F500A37014CFE8EF50C0F09001F45`，完成两轮新测试/直接消费者审查和独立 PG：**Code Review PASS，F001/F002 CLOSED，0 新 finding**。原 R01 失败链及作者归属保留。

## 实际验证与 Lead 复核

| 执行者 | 实际结果 |
|---|---|
| Backend 作者 | PG smoke 1 leaf/2 nodes；normal/race 各 13 leaf/15 nodes，并各 3 个独立 OS 进程 helper；tagged vet exit0、无 DSN |
| 原非作者 Reviewer | 独立新 schema 的 normal/race 各 13 leaf/15 nodes、各 3 helper；tagged vet exit0、无 DSN |
| Lead | source185 新执行 gofmt/modules/vet/build/contract/race/unit 七项全部 exit0，格式输出空；全部输入前后相同 |

各 Go child 实际 wait 完成，无业务用例 fail/skip/遗漏或额外节点。Lead 亲自解析双方原 JSON、预登记选择、argv、退出/超时、原流 hash、去敏环境与每次全部 185 live/snapshot 路径/hash；另读取全 13 个作者成功 leaf 的最终 SQL 行，核会话撤销/保留、审计 outcome、三维桶精确计数。

作者保留 27 个新 schema、33 目录、220 份 21 表阶段像；Reviewer 独立保留 26 个新 schema、32 目录、214 份阶段像，无复用作者 schema。Lead 重算共 434 份阶段像的 9114 个脱敏表摘要，核 110 组提交前隔离/提交/回滚/请求后/并发镜像关系，及 12 个 helper 的独立 PID、实际 wait、退出与原流摘要、16 个 advisory blocker 收据。所有 schema、行、原始日志和失败历史保留。

F001 使用真实 LoginHandler/Coordinator/Service、SessionStore、PasswordReset Gate、Audit Store 与 HTTPAuthorizer：仅摘要读回，active 用户旧会话因成功登录轮换失效，新会话可达；错密码和真实 Audit 23514 失败回滚均保留旧会话，失败审计独立持久化。缺失/伪造 CSRF 拒绝且终端零调用。

F002 使用真实持久 store/protector：三维分类与退避、缺失/已有/共享维度的并发无丢计数、真实第三桶 23514 的全回滚、关闭 parent 业务池后的三个新 OS 进程继续读取同一已提交状态。pair-only 状态通过 public store.Save 明示构造，不解释为完整 HTTP 自然达成的独立 pair 封禁。

连接实际检查 literal loopback:5433、全部 fallback、PG 180006、user/database、owner/search_path/current_schema；新随机 schema 独立 CREATE 提交后才执行冻结九迁移。仅授权新 schema 内固有迁移演进（含原 constraint/index DROP）；无 public/旧 schema 写入或额外清理。DSN 只在 runner 内存及集成/helper child，未进入 argv、文件或输出。磁盘敏感字段只保存 hash；原始秘密值及 HMAC/凭据比较由实际执行的断言证明，不能从脱敏文件重构。

## 失败历史与证据导航

首次静态 list 的 PID int32/uint32 编译错误已在两新文件内修正，保留原 raw，不计产品 RED。G0 乱码草稿保留为失败证据，采用的是可读 ASCII 交接。Reviewer normal 的 Go/13 用例实际通过，收据工具因 WORK 在 JSON build-output 而初记汇总 false；原 result/STOP 保留，以同一 raw 的独立复核及 work-reconciliation 更正，未重跑或吞掉测试失败。作者临时 binary 已不在原路径，未虚填其后验 hash；Reviewer 用 -work 保留并核同轮三 helper binary hash。

- [准备与原 183 底座](evidence-s3.3-s3.5/prep-r01/index.json)：`F7CFD669744933FF61A3812D62F2884DD607366C93868577B553FC2EC8354401`。
- [完整 185 字节绑定、Lead 新七项与原流/SQL 行核验](evidence-s3.3-s3.5/s3.3-fixed185-r01/index.json)：`1FF370C14A15DCA1ED4A4F13F193EB7E2F680C88DC3AAE86CDD0A92D297111E6`。早期包的 source_binding_index 指原 183 底座，完整 185 以本索引为准。
- [作者真实 PG](evidence-s3.3-s3.5/s3.3-author-pg-r01/index.json)：`012D0563F9F5BAB9EA4352BF009067C1B0FC48FAE46FCAD08759C5692599982C`。
- [原 Reviewer R02 完整归档](evidence-s3.3-s3.5/s3.3-review-r02/index.json)：`0DDB6355B88AC2A416105161F9B05F961343B4002B3602E47D8DFE91AD290C52`。
- [最终台账 CAS、前后像与 Lead 范围核验](evidence-s3.3-s3.5/s3.3-close-r01/index.json)：`B271C0DE76F51BDDC4B618193D2F9751EA2DB4B86A5AEC97741284499D711CD8`。Lead 回读确认 S3.3 已 `[x]`，总数 59 / 47 completed / 3 active / 9 todo；原 R 对账后的其他 58 卡及本轮开始时其他 57 卡逐字节保持。四路径、三 requires、变更后完整关闭条件、后续关联、开始日期、BOM/原 LF 与 CRLF 保留；Plan 仅本卡 B5A 子句与当前接续段变化。完整 185 live/snapshot、原 183、五材料、其他安全文档及 main/HEAD、空暂存区核验一致；未关闭 B5A 整批。该统计仅为本次关闭时点，后续以 tasks SSOT 为准。
- 原 [组件核证](evidence-s3.3-s3.5/s3.3-verification-r01/index.json)、[独评 R01](evidence-s3.3-s3.5/s3.3-review-r01/index.json)、[G0](evidence-s3.3-s3.5/s3.3-g0-r01/index.json)、[原卡 R 对账](evidence-s3.3-s3.5/s3.3-reconcile-r01/index.json)及 [CODE](evidence-s3.3-s3.5/s3.3-code-r01/index.json)均保留，不覆盖旧失败或待验状态。

## 接收边界

本次是 HTTP/Go/PostgreSQL 安全集成证据：hasher、clock、observer 与终端受控，issuer/source 使用真实实现的合成输入。没有真实浏览器、production factory、TLS listener、完整业务旅程、生产 Argon2 profile/阈值、真实代理、容量、部署或恢复验收结论。S3.4、S3.6 及所有 article 任务保持原范围；本卡关闭不代人的 accepted/checkpoint、整体验收或发布/部署/清理授权。
