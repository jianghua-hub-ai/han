# IR-01 内容发布社区任务 SSOT

> 本文件是 implementation/test task 三态唯一源。46 个 Story 派生条目为绿地 G 模式；`project-bootstrap-B0.1` 是用户直接授权的项目启动初始任务，不经 task-decompose。所有条目初始状态均为 `[ ]`，无日期。Plan 只引用本文件，不复制状态。

## 依据与范围

- 需求：requirements r16，accepted checkpoint `decision-e3af3912166f7287e8324488cad1cf1b87ba1034fca3378d3e0467a23751592d`。
- 架构：architecture r13，accepted checkpoint `decision-5e55881bdc4a7e038b3764d91ff0dfd906a48fe121e93bbb0813ca58173cae40`。
- 边界：identity r11、article r12。
- identity：design r09（checkpoint object digest `sha256:df484400154f7b64c00c40842e02eb254408bdb1c764768c51bf9672dbd09c61`，review PASS，checkpoint `decision-094b56eaec9148d2f91829ae0c3beb9820ce8ec27a347d9af0cc339caa073e02` 为 `deferred:reorder`）、stories r09、test-plan r01。
- article：design r06（checkpoint object digest `sha256:ed1a13b7f3d2e9f2c63b47d66d26469ea19315c42ace9cbe49db9f206880e4e0`，review PASS，checkpoint `decision-b1f45c78289c05b46e3304495e6a5f5112bf47357615df6fce5650f63cf869f8` 为 `deferred:reorder`）、stories r06、test-plan r01。
- ADR：ADR-0001～ADR-0021；Profile：`go-service`。
- 当前事实：无业务代码、`go.mod`、schema、migration、测试框架或项目检查命令；未做 R 模式 gap-map。

## 全局约束与项目启动前置

### 正式项目启动前置：`project-bootstrap-B0.1`

`project-bootstrap-B0.1` 是所有代码与测试任务的硬前置。它是用户明确授权的项目启动期初始任务，不从 Story 派生；完成前 B1～B12 全部暂停。B0 只建立 Go、依赖、迁移与测试执行基线，不实现身份、文章或其他业务行为。

### 共享所有权

- `project-bootstrap-B0.1` 唯一拥有 Go module、最小可执行入口、迁移执行器、测试标签/环境基线、依赖审查记录及 Manifest 检查命令登记；后续 Story task 只消费。
- 共享平台、隐私、审计事实由 `identity-auth-S1.1` 唯一拥有；article 只依赖。
- ReplayStatus v1 消费/fixture 与恢复协调由 `identity-auth-S4.2` / `identity-auth-S1.5` 拥有，article producer 依赖同一 fixture，不复制 schema 状态。
- canonical 用户删除编排由 `identity-auth-S9.1` 拥有；article 端 `OrphanArticlesCanonical` 由 `article-management-S6.1` 拥有。
- Article Deletion Journal、finalizer 与 replay evidence 由 article Story 4 唯一拥有；identity 只消费 DTO。
- 开发 task 自带局部测试和独立 Review；专项集成/安全/韧性/性能及部署后 E2E 另列，不阻塞开发 task 按自身条件关闭。
- 未决 Q1、AM-Q1/Q2、容量、隐私部署值、RPO/RTO、retention 等只阻塞卡片明确标注的范围，不替用户填值。

## 任务统计

- 总数：47
- project-bootstrap：1
- identity-auth：26
- article-management：20
- 类型：项目启动 1；开发 28；运行输入准备 3；专项契约测试 2；专项集成测试 3；专项性能测试 2；专项安全测试 1；E2E 2；专项韧性测试 2；专项安全/集成测试 1；专项契约/韧性测试 1；专项集成/安全测试 1

## project-bootstrap（B0，非 Story）

- [-] project-bootstrap-B0.1 建立 Go 服务、依赖、迁移与测试执行基线

- **目标**：建立不含业务行为的 Go 1.27 项目启动基线，使后续 46 个 Story task 可在同一 module、数据库迁移合同与测试环境约定上实施。
- **类型**：项目启动
- **实施责任**：Backend-Dev；Agent ID `01a15c1c-35d3-7d31-8ce4-86153871411e`；开始@2026-10-21
- **文件**：
  - `go.mod（新增）`
  - `go.sum（新增）`
  - `cmd/han-server/main.go（新增）`
  - `internal/migration/runner.go（新增）`
  - `internal/migration/runner_test.go（新增）`
  - `internal/migration/runner_integration_test.go（新增）`
  - `internal/migration/testdata/0001_create_probe.sql（新增）`
  - `internal/migration/testdata/0002_fail_probe.sql（新增）`
  - `migrations/README.md（新增）`
  - `internal/testenv/env.go（新增）`
  - `internal/testenv/env_test.go（新增）`
  - `docs/testing.md（新增）`
  - `docs/dependencies.md（新增）`
  - `add.manifest.yaml（修改；实施时登记真实检查命令）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go version`
  - `go list -m`
  - `go list -m -f "{{.Path}} {{.Version}}" github.com/jackc/pgx/v5 golang.org/x/crypto`
  - `go mod verify`
  - `gofmt -l .`
  - `go vet ./...`
  - `go build ./cmd/han-server`
  - `go test ./internal/migration ./internal/testenv`
  - `go test -tags=contract ./...`
  - `go test -tags=integration ./internal/migration -run "TestRunner"`
  - `go test -race ./...`
  - `go test ./...`
  - 当前状态：规划阶段未实施、未运行。现场无 PostgreSQL 18、`psql`、Docker、`golangci-lint`、`goimports`，且依赖精确版本尚未因网络超时锁定；不得把这些工具或真实数据库验证写成已通过。
- **依赖**：无（B0 根任务，不依赖自身）
- **并行**：串行；B0 完成并经独立 Review 后才可启动 B1～B12。
- **阻塞注记**：实施时必须取得可用模块源并锁定 `github.com/jackc/pgx/v5` 与既有 `golang.org/x/crypto` 的精确版本，记录许可证与供应链审查；真实 PostgreSQL 18 集成测试环境未就绪时本任务不能关闭。
- **本任务关闭条件**：`go.mod` 声明 Go 1.27、module `han`，`go.sum` 锁定并验证已批准的 pgx/v5 与 x/crypto，`docs/dependencies.md` 记录实际版本及审查证据；最小 `han-server` 可构建且不含业务路由/规则；迁移执行器只承担 `migrations/NNNN_name.sql` 唯一顺序、`schema_migrations`、单迁移事务、PostgreSQL advisory lock 和失败停止，单元测试及 PostgreSQL 18 集成测试覆盖顺序、重复版本、提交记录、回滚与并发锁；`contract`、`integration`、`browser`、`e2e` 标签在 `docs/testing.md`/`internal/testenv` 形成单一约定：`contract` 不依赖外部服务，`integration` 统一读取 `HAN_TEST_POSTGRES_DSN`，`browser`/`e2e` 统一读取 `HAN_TEST_BASE_URL`，浏览器专项读取 `HAN_TEST_BROWSER_BIN`，带标签命令缺必需变量时明确失败而不静默通过；Manifest 的 `check_commands` 登记现场可真实运行的 Go 检查命令，不把不可用工具或未运行验证登记为通过；独立 Review 通过，Lead 核固定工作树 status/diff/所有权后关闭。
- **后续验证关联**：B1～B12；原 46 个 identity-auth/article-management task。
- **设计/测试来源**：architecture r13 §4.1/§4.9/§7.2/§8.1～8.2/§11；ADR-0009/0012/0021；go-service Profile。

## identity-auth（11 Story）

- [ ] identity-auth-S1.1 建立共享 REST、配置、隐私门禁与追加式审计底座

- **目标**：建立共享 REST、配置、隐私门禁与追加式审计底座。
- **类型**：开发（共享基础，identity-auth 所有）
- **文件**：
  - `migrations/0001_shared.sql（新增）`
  - `internal/platform/config.go（新增）`
  - `internal/platform/errors.go（新增）`
  - `internal/platform/logging.go（新增）`
  - `internal/platform/http.go（新增）`
  - `internal/privacy/store.go（新增）`
  - `internal/audit/store.go（新增）`
  - `internal/platform/config_test.go（新增）`
  - `internal/platform/errors_test.go（新增）`
  - `internal/privacy/store_test.go（新增）`
  - `internal/audit/store_integration_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/platform ./internal/privacy ./internal/audit`
  - `go test ./...`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：ProcessingGate 缺值 fail closed、公共错误 envelope/trace 映射、AuditEvent 只追加与事务失败回滚均有失败证据和局部测试；独立 Review 通过；Lead 核范围后关闭。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S1.6、article-management-S1.2
- **设计/测试来源**：identity design §3.3.1/§3.4.2；identity test I-AUD/I-CON；ADR-0014/0016/0017

- [ ] identity-auth-S1.2 建立身份账号、标识与隐私证据数据模型

- **目标**：建立身份账号、标识与隐私证据数据模型。
- **类型**：开发
- **文件**：
  - `migrations/0002_identity.sql（新增）`
  - `internal/identity/model.go（新增）`
  - `internal/identity/repository.go（新增）`
  - `internal/identity/repository_integration_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity`
  - `go test -tags=integration ./internal/identity`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：users、identity_identifiers、sessions、temporary_credentials、隐私/登录桶/恢复核对表的约束与索引落地；真实 PostgreSQL 约束测试通过；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.6、identity-auth-S9.2
- **设计/测试来源**：identity design §3.4.1；identity test I-REG/I-AUTH/I-ADM

- [ ] identity-auth-S1.3 实现注册与当前隐私确认的原子创建

- **目标**：实现注册与当前隐私确认的原子创建。
- **类型**：开发
- **文件**：
  - `internal/identity/handler.go（新增/修改）`
  - `internal/identity/service.go（新增/修改）`
  - `internal/identity/registration.go（新增）`
  - `internal/identity/registration_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestRegister"`
  - `go test ./internal/identity`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`、`identity-auth-S1.2`、`identity-auth-S1.4`、`identity-auth-S2.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：先以失败用例证明旧 notice、缺门禁、并发标识冲突和审计失败不会留下部分账号；注册成功返回 201 且不建 Session；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.6、identity-auth-S3.4
- **设计/测试来源**：identity Story S01；identity design §3.3.2；test I-REG-01～05

- [ ] identity-auth-S1.4 准备隐私、同源、代理与 SMTP 处理门禁运行输入 [P]

- **目标**：准备隐私、同源、代理与 SMTP 处理门禁运行输入。
- **类型**：运行输入准备
- **文件**：
  - `deploy/processing-gate.env.example（新增）`
  - `docs/runbooks/processing-gate.md（新增）`
  - `internal/platform/config_test.go（修改）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/platform -run "TestProcessingGate|TestOrigin|TestTrustedProxy"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：配置键、读取方、无默认值与缺值关闭行为可核；实际地区、控制者、处理依据、canonical origin、可信代理和 SMTP 处理方由责任人登记且未写入凭据；独立 Review 通过。
- **后续验证关联**：identity-auth-S3.3、identity-auth-S5.3、article-management-S7.2
- **设计/测试来源**：identity design Q8/安全配置；identity test §7/§11；article test ProcessingGate

- [ ] identity-auth-S1.5 验证身份 REST、内部端口与 ReplayStatus 消费契约 [P]

- **目标**：验证身份 REST、内部端口与 ReplayStatus 消费契约。
- **类型**：专项契约测试
- **文件**：
  - `test/contract/identity/rest_contract_test.go（新增）`
  - `test/contract/identity/internal_ports_test.go（新增）`
  - `test/contract/identity/replay_status_consumer_test.go（新增）`
  - `testdata/contracts/article-deletion-replay/ready-v1.json（新增）`
  - `testdata/contracts/article-deletion-replay/unavailable-v1.json（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=contract ./test/contract/identity`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.1`、`identity-auth-S4.2`、`identity-auth-S5.1`、`identity-auth-S9.1`、`article-management-S4.2`、`article-management-S6.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：11 US 公共 REST、S1–S16 端口、Ready20/Unavailable6 严格消费者与固定 digest 通过；额外/缺失/null/错类型先红后绿；独立 Review 通过。
- **后续验证关联**：identity-auth-S3.4、article-management-S1.5
- **设计/测试来源**：identity test I-CON-01～03/I-REC-06～08；ADR-0020

- [ ] identity-auth-S1.6 验证身份 PostgreSQL 事务、锁与并发矩阵 [P]

- **目标**：验证身份 PostgreSQL 事务、锁与并发矩阵。
- **类型**：专项集成测试
- **文件**：
  - `test/integration/identity/postgres_test.go（新增）`
  - `test/integration/identity/fixtures.sql（新增）`
  - `test/integration/identity/faults_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/identity`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.3`、`identity-auth-S6.1`、`identity-auth-S9.1`、`identity-auth-S10.1`、`identity-auth-S11.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：隔离 PostgreSQL 上唯一性、bootstrap、会话撤销、幂等、审计失败、治理竞争与删除回滚矩阵通过；无真实 PostgreSQL 时保持未关闭；独立 Review 通过。
- **后续验证关联**：identity-auth-S2.2、identity-auth-S3.4
- **设计/测试来源**：identity test 身份 PostgreSQL 事务与并发专项；I-REG/I-ADM/I-CRED/I-GOV/I-DEL

- [ ] identity-auth-S2.1 实现密码策略与 Argon2id 凭据哈希 [P]

- **目标**：实现密码策略与 Argon2id 凭据哈希。
- **类型**：开发
- **文件**：
  - `internal/security/password.go（新增）`
  - `internal/security/password_test.go（新增）`
  - `internal/security/argon2.go（新增）`
  - `internal/security/argon2_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/security`
  - `go test ./...`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：先以失败用例覆盖长度/字母/数字、参数缺失、资源错误和旧参数；仅保存自描述 Argon2id 编码且无明文；独立 Review 通过。
- **后续验证关联**：identity-auth-S2.2、identity-auth-S3.2
- **设计/测试来源**：identity Story S02；identity design §3.3.3；test I-REG-06/I-AUTH-06；ADR-0012

- [ ] identity-auth-S2.2 建立 identity 容量与性能基线 [P]

- **目标**：建立 identity 容量与性能基线。
- **类型**：专项性能测试
- **文件**：
  - `test/performance/identity/benchmark_test.go（新增）`
  - `docs/benchmarks/identity.md（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -run "^$" -bench . -benchmem ./test/performance/identity`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.6`、`identity-auth-S4.4`、`identity-auth-S5.3`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：容量模型、目标值、RPO/RTO 与 retention 未确认
- **本任务关闭条件**：产出 Argon2id、Session、治理事务、SMTP 与恢复批量基线；不编造通过阈值。工作负载和批准目标未确认时任务保持阻塞注记；独立 Review 通过。
- **后续验证关联**：identity-auth-S3.4
- **设计/测试来源**：identity test I-PERF-01～04；Q4/Q9/Q10/Q11

- [ ] identity-auth-S3.1 实现 Session、同源、CSRF 与受限 Principal 接入 [P]

- **目标**：实现 Session、同源、CSRF 与受限 Principal 接入。
- **类型**：开发（共享接入）
- **文件**：
  - `internal/session/store.go（新增）`
  - `internal/session/store_test.go（新增）`
  - `internal/platform/http_security.go（新增）`
  - `internal/platform/http_security_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/session ./internal/platform -run "TestSession|TestOrigin|TestCSRF|TestCORS"`
  - `go test ./...`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`、`identity-auth-S1.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：先证明恶意/缺失 Origin、跨源凭据 CORS、伪造 CSRF 和旧 Cookie 被拒绝；轮换只存摘要；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S3.3、article-management-S1.3
- **设计/测试来源**：identity Story S03/K1；design §3.3.1/§3.3.4；test I-AUTH-01～04

- [ ] identity-auth-S3.2 实现统一失败登录、会话轮换与持久防暴力

- **目标**：实现统一失败登录、会话轮换与持久防暴力。
- **类型**：开发
- **文件**：
  - `internal/identity/handler.go（修改）`
  - `internal/identity/service.go（修改）`
  - `internal/identity/login.go（新增）`
  - `internal/identity/login_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestLogin|TestCredentialMode|TestLoginAbuse"`
  - `go test ./internal/identity`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.2`、`identity-auth-S2.1`、`identity-auth-S3.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：先以状态矩阵证明账号不存在/停用/删除/recovery/临时凭据失效均统一 401，且不得回退永久密码；成功轮换会话，失败保留旧会话；独立 Review 通过。
- **后续验证关联**：identity-auth-S3.3、identity-auth-S3.4
- **设计/测试来源**：identity Story S03；test I-AUTH-01～08

- [ ] identity-auth-S3.3 执行浏览器认证安全专项 [P]

- **目标**：执行浏览器认证安全专项。
- **类型**：专项安全测试
- **文件**：
  - `test/browser/identity/auth_security_test.go（新增）`
  - `test/browser/identity/testdata/certificates/README.md（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=browser ./test/browser/identity`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.4`、`identity-auth-S3.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：真实浏览器上下文观察 Cookie 属性、Origin/Fetch Metadata、CORS、CSRF 与旧 Cookie 失效；不要求前端 profile；独立 Review 通过。
- **后续验证关联**：identity-auth-S3.4
- **设计/测试来源**：identity test 浏览器认证安全专项；I-AUTH-01/03/04/07

- [ ] identity-auth-S3.4 执行部署后 identity 真实 E2E 旅程 [P]

- **目标**：执行部署后 identity 真实 E2E 旅程。
- **类型**：E2E
- **文件**：
  - `test/e2e/identity/journey_test.go（新增）`
  - `test/e2e/identity/README.md（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=e2e ./test/e2e/identity`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.5`、`identity-auth-S1.6`、`identity-auth-S2.2`、`identity-auth-S3.3`、`identity-auth-S4.4`、`identity-auth-S5.3`、`identity-auth-S9.2`、`article-management-S7.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：部署后从真实入口完成注册/登录、管理员治理、删除用户/无归属处置、凭据恢复与旧备份恢复三层断言；Test-Engineer 独立执行并取得业务验收决定。
- **后续验证关联**：需求 IR-01 真实 E2E
- **设计/测试来源**：identity test identity 真实 E2E 建议；development-delivery e2e

- [ ] identity-auth-S4.1 实现初始管理员 bootstrap 与受保护 onboarding

- **目标**：实现初始管理员 bootstrap 与受保护 onboarding。
- **类型**：开发
- **文件**：
  - `internal/identity/bootstrap.go（新增）`
  - `internal/identity/bootstrap_test.go（新增）`
  - `internal/identity/handler.go（修改）`
  - `internal/identity/service.go（修改）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestBootstrap|TestInitialAdmin|TestOnboarding"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S2.1`、`identity-auth-S3.2`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：Q1 初始管理员轮换与具体改密规则未确认
- **本任务关闭条件**：并发 bootstrap 只产生一个 admin；受保护动作全拒绝；受限会话只开放 me/改密/隐私。Q1 的首次/后续改密规则未确认时不得关闭受影响断言；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.6、identity-auth-S3.4
- **设计/测试来源**：identity Story S04；test I-ADM-01～03

- [ ] identity-auth-S4.2 实现 Recovery Journal、完整 keyring 与恢复协调 [P]

- **目标**：实现 Recovery Journal、完整 keyring 与恢复协调。
- **类型**：开发（共享恢复基础）
- **文件**：
  - `internal/recovery/journal.go（新增）`
  - `internal/recovery/keyring.go（新增）`
  - `internal/recovery/coordinator.go（新增）`
  - `internal/recovery/replay_status_v1.go（新增）`
  - `internal/recovery/journal_test.go（新增）`
  - `internal/recovery/replay_status_v1_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/recovery`
  - `go test ./...`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`、`identity-auth-S1.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：先以失败用例覆盖 retained key 重放、缺 key、部分写、链损坏、namespace 混用、Unavailable 与坏 Ready；只有双 namespace 和二次同 ref/digest 可 MarkConsumed；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S4.4、article-management-S4.2
- **设计/测试来源**：identity Story S04；design §3.4.2/8；test I-REC-02～10；ADR-0018/0019/0020

- [ ] identity-auth-S4.3 准备恢复运行输入、合成备份与 keyring 样本 [P]

- **目标**：准备恢复运行输入、合成备份与 keyring 样本。
- **类型**：运行输入准备
- **文件**：
  - `docs/runbooks/database-recovery.md（新增）`
  - `testdata/recovery/README.md（新增）`
  - `testdata/recovery/keyring-fixtures.json（新增）`
  - `testdata/recovery/backup-manifest.json（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/recovery -run "TestRecoveryFixtures"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S4.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：RPO/RTO、批量容量、key/Journal/tombstone retention 与运行责任未确认
- **本任务关闭条件**：只含合成数据；记录 backup_ref、run、keyring、清理责任与恢复步骤。RPO/RTO/保留值未知时不填数值，真实演练仍保持阻塞；独立 Review 通过。
- **后续验证关联**：identity-auth-S4.4、article-management-S4.4
- **设计/测试来源**：identity test §7 数据环境；Q9/Q10/Q11

- [ ] identity-auth-S4.4 验证 Recovery Journal、keyring 与联合恢复 [P]

- **目标**：验证 Recovery Journal、keyring 与联合恢复。
- **类型**：专项韧性测试
- **文件**：
  - `test/integration/recovery/identity_recovery_test.go（新增）`
  - `test/integration/recovery/journal_faults_test.go（新增）`
  - `test/integration/recovery/joint_readiness_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/recovery`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S4.2`、`identity-auth-S4.3`、`identity-auth-S9.2`、`article-management-S4.2`、`article-management-S4.4`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：真实备份/持久卷环境中覆盖崩溃、损坏、跨 key 版本重放、kept_deleted、双 namespace 与 MarkConsumed 顺序；未有环境/RPO/RTO 时保持未关闭；独立 Review 通过。
- **后续验证关联**：identity-auth-S2.2、identity-auth-S3.4
- **设计/测试来源**：identity test Recovery Journal/keyring 与联合恢复专项；I-REC-01～10

- [ ] identity-auth-S5.1 建立幂等、临时凭据与强制 TLS Mailer 共享内核 [P]

- **目标**：建立幂等、临时凭据与强制 TLS Mailer 共享内核。
- **类型**：开发（共享凭据基础）
- **文件**：
  - `internal/idempotency/store.go（新增）`
  - `internal/idempotency/store_test.go（新增）`
  - `internal/mailer/mailer.go（新增）`
  - `internal/mailer/smtp.go（新增）`
  - `internal/mailer/smtp_test.go（新增）`
  - `internal/identity/temporary_credential.go（新增）`
  - `internal/identity/temporary_credential_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/idempotency ./internal/mailer ./internal/identity -run "TestIdempotency|TestTemporaryCredential|TestSMTP"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`、`identity-auth-S1.2`、`identity-auth-S2.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：先以失败用例覆盖同键异摘要、uncertain/recovery_review_required、旧 generation、TLS 降级/错证书；状态机、purpose 与安全快照符合合同；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S5.2、identity-auth-S10.1、identity-auth-S11.1、article-management-S7.1
- **设计/测试来源**：identity K3；design §3.3.1/§3.4.2/4；test I-CRED

- [ ] identity-auth-S5.2 实现创建其他管理员与双 Gate

- **目标**：实现创建其他管理员与双 Gate。
- **类型**：开发
- **文件**：
  - `internal/identity/handler.go（修改）`
  - `internal/identity/service.go（修改）`
  - `internal/identity/administrator_create.go（新增）`
  - `internal/identity/administrator_create_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestCreateAdministrator|TestAdministratorOnboarding"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.3`、`identity-auth-S1.4`、`identity-auth-S3.2`、`identity-auth-S4.1`、`identity-auth-S5.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：先证明声明过期、越权、SMTP 未受理、同键冲突和残余窗口不会留下可用管理员；双 Gate 仅在 current notice 下开放；独立 Review 通过。
- **后续验证关联**：identity-auth-S5.3、identity-auth-S3.4
- **设计/测试来源**：identity Story S05；test I-CRED-01/02/04

- [ ] identity-auth-S5.3 验证 SMTP/TLS 与凭据残余窗口 [P]

- **目标**：验证 SMTP/TLS 与凭据残余窗口。
- **类型**：专项安全/集成测试
- **文件**：
  - `test/integration/identity/smtp_credentials_test.go（新增）`
  - `test/integration/identity/smtp_test_server.go（新增）`
  - `test/integration/identity/testdata/certificates/README.md（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/identity -run "TestCredentialSMTP|TestSMTPResidualWindow"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.4`、`identity-auth-S5.2`、`identity-auth-S10.1`、`identity-auth-S11.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：可控 SMTP 下覆盖 implicit TLS/required STARTTLS、降级、证书故障、未受理、accepted 后 DB 失败和重试；不发外部邮件；独立 Review 通过。
- **后续验证关联**：identity-auth-S2.2、identity-auth-S3.4
- **设计/测试来源**：identity test SMTP/TLS 与凭据残余窗口专项；I-CRED-01～06

- [ ] identity-auth-S6.1 实现其他管理员查询、停用、恢复、删除与撤权

- **目标**：实现其他管理员查询、停用、恢复、删除与撤权。
- **类型**：开发
- **文件**：
  - `internal/identity/handler.go（修改）`
  - `internal/identity/service.go（修改）`
  - `internal/identity/administrator_governance.go（新增）`
  - `internal/identity/administrator_governance_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestAdministratorGovernance"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.2`、`identity-auth-S4.1`、`identity-auth-S7.1`、`identity-auth-S8.1`、`identity-auth-S9.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：初始 admin 是唯一治理者；旧会话即时失效；重复目标状态稳定；initial admin 永久受保护；失败证据和独立 Review 完成。
- **后续验证关联**：identity-auth-S1.6、identity-auth-S3.4
- **设计/测试来源**：identity Story S06；test I-ADM-04～05

- [ ] identity-auth-S7.1 实现用户与管理员最小披露查询和游标分页

- **目标**：实现用户与管理员最小披露查询和游标分页。
- **类型**：开发
- **文件**：
  - `internal/identity/handler.go（修改）`
  - `internal/identity/query.go（新增）`
  - `internal/identity/cursor.go（新增）`
  - `internal/identity/query_test.go（新增）`
  - `internal/identity/cursor_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestUserQuery|TestIdentityCursor"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.2`、`identity-auth-S3.2`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：普通管理员不可枚举管理员；字段 allowlist、稳定 created_at/id 游标、非法/过滤不匹配游标先红后绿；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S8.1、identity-auth-S9.1
- **设计/测试来源**：identity Story S07；test I-LIST-01～03

- [ ] identity-auth-S8.1 实现停用/恢复与作者状态只读接缝

- **目标**：实现停用/恢复与作者状态只读接缝。
- **类型**：开发
- **文件**：
  - `internal/identity/handler.go（修改）`
  - `internal/identity/account_status.go（新增）`
  - `internal/identity/article_reader.go（新增）`
  - `internal/identity/account_status_test.go（新增）`
  - `internal/identity/article_reader_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestAccountStatus|TestGetAuthorStates"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.2`、`identity-auth-S7.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：目标状态幂等、每次重新授权、停用撤销全部 Session；Article 零写入并只读返回作者状态；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.6、identity-auth-S9.1、article-management-S2.1
- **设计/测试来源**：identity Story S08；test I-GOV-01～05；S2/S3

- [ ] identity-auth-S9.1 实现 canonical 用户删除与标识/retained_email 生命周期

- **目标**：实现 canonical 用户删除与标识/retained_email 生命周期。
- **类型**：开发（跨模块协调）
- **文件**：
  - `internal/identity/deletion.go（新增）`
  - `internal/identity/article_port.go（新增）`
  - `internal/identity/deletion_test.go（新增）`
  - `internal/identity/service.go（修改）`
  - `internal/identity/handler.go（修改）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestCanonicalDelete|TestIdentifierLifecycle"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.2`、`identity-auth-S7.1`、`identity-auth-S8.1`、`article-management-S6.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：先以故障用例证明 User/Identifier/Session/凭据/Article/Audit 任一步失败全回滚；零/多/违规/kept_deleted 和邮箱清理条件成立；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S9.2
- **设计/测试来源**：identity Story S09；test I-DEL-01～07；S4/S5/S14

- [ ] identity-auth-S9.2 验证 canonical 用户删除跨模块事务 [P]

- **目标**：验证 canonical 用户删除跨模块事务。
- **类型**：专项集成测试（identity 主责）
- **文件**：
  - `test/integration/crossmodule/canonical_user_delete_test.go（新增）`
  - `test/integration/crossmodule/canonical_delete_faults_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/crossmodule -run "TestCanonicalUserDelete"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S9.1`、`article-management-S4.1`、`article-management-S6.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：真实 PostgreSQL 上覆盖零/多/违规/deletion_pending/kept_deleted、审计失败和 retained_email 清理；任一模块未实现时不关闭；独立 Review 通过。
- **后续验证关联**：identity-auth-S4.4、identity-auth-S3.4
- **设计/测试来源**：identity/article test canonical 删除跨模块专项

- [ ] identity-auth-S10.1 实现管理员重置用户密码

- **目标**：实现管理员重置用户密码。
- **类型**：开发
- **文件**：
  - `internal/identity/handler.go（修改）`
  - `internal/identity/password_reset.go（新增）`
  - `internal/identity/password_reset_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestPasswordReset"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.4`、`identity-auth-S2.1`、`identity-auth-S3.2`、`identity-auth-S4.1`、`identity-auth-S5.1`、`identity-auth-S7.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：先证明越权、SMTP 未受理、同键冲突和 residual window 保持旧凭据/Session；成功后仅最新临时凭据有效且响应不含秘密；独立 Review 通过。
- **后续验证关联**：identity-auth-S5.3、identity-auth-S1.6
- **设计/测试来源**：identity Story S10；test I-CRED-02/03/05/06

- [ ] identity-auth-S11.1 实现临时密码重发与 generation 替换

- **目标**：实现临时密码重发与 generation 替换。
- **类型**：开发
- **文件**：
  - `internal/identity/handler.go（修改）`
  - `internal/identity/temporary_credential_resend.go（新增）`
  - `internal/identity/temporary_credential_resend_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestTemporaryCredentialResend"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.4`、`identity-auth-S4.1`、`identity-auth-S5.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：按 purpose/角色授权；失败保留旧 generation，成功仅最新 generation 有效且 7 天期限从新生成起；运行 fixture 由 S05 或 S10 提供；独立 Review 通过。
- **后续验证关联**：identity-auth-S5.3、identity-auth-S1.6
- **设计/测试来源**：identity Story S11；test I-CRED-03/05/06

## article-management（8 Story）

- [ ] article-management-S1.1 建立 Article、通知、删除与恢复证据数据模型 [P]

- **目标**：建立 Article、通知、删除与恢复证据数据模型。
- **类型**：开发
- **文件**：
  - `migrations/0003_article.sql（新增）`
  - `internal/article/model.go（新增）`
  - `internal/article/repository.go（新增）`
  - `internal/article/repository_integration_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article`
  - `go test -tags=integration ./internal/article`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：Article 正交状态、TransferNotice/Recipient、DeletionOperation、DeletionReplayRun/Evidence 约束与索引在真实 PostgreSQL 通过；独立 Review 通过。
- **后续验证关联**：article-management-S2.2、article-management-S4.3
- **设计/测试来源**：article design §3.3.1 数据结构；article test A-CREATE/A-DEL/A-REC

- [ ] article-management-S1.2 建立文章策略、cursor、强 ETag 与公共 REST 合同

- **目标**：建立文章策略、cursor、强 ETag 与公共 REST 合同。
- **类型**：开发（共享 article 基础）
- **文件**：
  - `internal/article/policy.go（新增）`
  - `internal/article/model.go（修改）`
  - `internal/article/handler.go（新增）`
  - `internal/article/policy_test.go（新增）`
  - `internal/article/cursor_test.go（新增）`
  - `internal/article/etag_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestVisibility|TestCursor|TestETag|TestArticleErrors"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`、`article-management-S1.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：先以失败用例覆盖状态组合、投影泄露、非法 cursor、weak/multi/* ETag 与错误映射；成员/作者/管理员投影隔离；独立 Review 通过。
- **后续验证关联**：article-management-S1.5、article-management-S2.1、article-management-S3.1
- **设计/测试来源**：article design §3.3.1/§3.4.1；test A-ETAG/A-CON

- [ ] article-management-S1.3 实现创建文章纵切

- **目标**：实现创建文章纵切。
- **类型**：开发
- **文件**：
  - `internal/article/command_service.go（新增/修改）`
  - `internal/article/handler.go（修改）`
  - `internal/article/create_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestCreateArticle"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.1`、`identity-auth-S3.2`、`article-management-S1.1`、`article-management-S1.2`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：AM-Q1 最大长度/格式只阻塞最终 validator 数值，不阻塞非空与文本安全
- **本任务关闭条件**：active Principal 创建 published/clear 文章并返回 201/Location/ETag；客户端 author_id、受限身份、空白与 DB 失败先红后绿；独立 Review 通过。
- **后续验证关联**：article-management-S1.5、article-management-S2.2
- **设计/测试来源**：article Story 1；test A-CREATE-01～03

- [ ] article-management-S1.4 执行部署后 article 真实 E2E 旅程 [P]

- **目标**：执行部署后 article 真实 E2E 旅程。
- **类型**：E2E
- **文件**：
  - `test/e2e/article/journey_test.go（新增）`
  - `test/e2e/article/README.md（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=e2e ./test/e2e/article`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S1.5`、`article-management-S2.2`、`article-management-S2.3`、`article-management-S4.3`、`article-management-S4.4`、`article-management-S7.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：部署后从真实入口完成创建/浏览/修改/删除、停用可见性、无归属处置、通知与旧备份重放三层断言；Test-Engineer 独立执行并取得业务验收决定。
- **后续验证关联**：需求 IR-01 真实 E2E
- **设计/测试来源**：article test article 真实 E2E 建议；development-delivery e2e

- [ ] article-management-S1.5 验证文章 REST、三投影与 S1–S16 内部契约 [P]

- **目标**：验证文章 REST、三投影与 S1–S16 内部契约。
- **类型**：专项契约测试
- **文件**：
  - `test/contract/article/rest_contract_test.go（新增）`
  - `test/contract/article/internal_ports_test.go（新增）`
  - `test/contract/article/projection_contract_test.go（新增）`
  - `test/contract/article/replay_status_producer_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=contract ./test/contract/article`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.5`、`article-management-S1.3`、`article-management-S2.1`、`article-management-S3.1`、`article-management-S4.2`、`article-management-S5.1`、`article-management-S6.2`、`article-management-S7.1`、`article-management-S8.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：16 endpoint、三投影 allowlist、公共错误、S1–S16 与 Ready20/Unavailable6 producer fixture 全部通过；独立 Review 通过。
- **后续验证关联**：article-management-S1.4
- **设计/测试来源**：article test A-CON-01～04/A-REC-01～03

- [ ] article-management-S2.1 实现成员可见列表与详情分页

- **目标**：实现成员可见列表与详情分页。
- **类型**：开发
- **文件**：
  - `internal/article/query_service.go（新增/修改）`
  - `internal/article/handler.go（修改）`
  - `internal/article/query_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestListArticles|TestGetArticle|TestVisibleScan"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S8.1`、`article-management-S1.2`、`article-management-S1.3`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：AM-Q2 limit/批次/P95 数值未确认
- **本任务关闭条件**：先证明隐藏密集区、同时间戳、作者状态缺失与当前状态变化不漏不重；不可见统一 404；独立 Review 通过。
- **后续验证关联**：article-management-S1.5、article-management-S2.2
- **设计/测试来源**：article Story 2；test A-READ-01～06

- [ ] article-management-S2.2 验证文章 PostgreSQL 状态、分页与 ETag 并发 [P]

- **目标**：验证文章 PostgreSQL 状态、分页与 ETag 并发。
- **类型**：专项集成测试
- **文件**：
  - `test/integration/article/postgres_state_test.go（新增）`
  - `test/integration/article/pagination_test.go（新增）`
  - `test/integration/article/etag_concurrency_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/article -run "TestArticleState|TestArticlePagination|TestArticleETag"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S1.3`、`article-management-S2.1`、`article-management-S3.1`、`article-management-S8.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：真实 PostgreSQL 上状态组合、连续隐藏、多 writer、治理竞争和事务回滚通过；无真实 PostgreSQL 时不关闭；独立 Review 通过。
- **后续验证关联**：article-management-S2.3、article-management-S1.4
- **设计/测试来源**：article test 文章 PostgreSQL 状态与分页专项

- [ ] article-management-S2.3 建立 article 容量与性能基线 [P]

- **目标**：建立 article 容量与性能基线。
- **类型**：专项性能测试
- **文件**：
  - `test/performance/article/benchmark_test.go（新增）`
  - `docs/benchmarks/article.md（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -run "^$" -bench . -benchmem ./test/performance/article`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S2.2`、`article-management-S4.3`、`article-management-S4.4`、`article-management-S7.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：AM-Q1/Q2、容量目标、RPO/RTO、retention 未确认
- **本任务关闭条件**：产出内容、分页隐藏密度、dispatcher、finalizer 与 replay 基线；不编造阈值。AM-Q1/Q2、容量和 retention 未确认时保留阻塞注记；独立 Review 通过。
- **后续验证关联**：article-management-S1.4
- **设计/测试来源**：article test A-PERF-01～05

- [ ] article-management-S3.1 实现作者受控详情与强 ETag 修改

- **目标**：实现作者受控详情与强 ETag 修改。
- **类型**：开发
- **文件**：
  - `internal/article/command_service.go（修改）`
  - `internal/article/query_service.go（修改）`
  - `internal/article/handler.go（修改）`
  - `internal/article/update_test.go（新增）`
  - `internal/article/author_projection_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestAuthorArticle|TestUpdateArticle"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S1.2`、`article-management-S2.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：先证明缺/坏 If-Match、双 writer、非作者与字段越界失败；只改内容/version，violating 约束保持且作者投影最小；独立 Review 通过。
- **后续验证关联**：article-management-S1.5、article-management-S2.2
- **设计/测试来源**：article Story 3；test A-ETAG/A-MOD

- [ ] article-management-S4.1 实现永久删除协调、Deletion Journal 与 finalizer

- **目标**：实现永久删除协调、Deletion Journal 与 finalizer。
- **类型**：开发
- **文件**：
  - `internal/article/deletion_coordinator.go（新增）`
  - `internal/article/deletion_finalizer.go（新增）`
  - `internal/article/deletion_journal.go（新增）`
  - `internal/article/command_service.go（修改）`
  - `internal/article/handler.go（修改）`
  - `internal/article/deletion_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestPermanentDeletion|TestDeletionJournal|TestDeletionFinalizer"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S1.3`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：先证明 Journal/fsync/DB/finalizer 失败不返回 204且无重新可见窗口；重复 DELETE/双 finalizer 收敛；tombstone 无 PII；独立 Review 通过。
- **后续验证关联**：article-management-S4.3、article-management-S5.1、article-management-S6.2、article-management-S8.1
- **设计/测试来源**：article Story 4；test A-DEL-01～10；ADR-0020

- [ ] article-management-S4.2 实现 ReplayStatus producer、证据存储与旧备份重放 [P]

- **目标**：实现 ReplayStatus producer、证据存储与旧备份重放。
- **类型**：开发
- **文件**：
  - `internal/article/deletion_replay.go（新增）`
  - `internal/article/replay_evidence.go（新增）`
  - `internal/article/replay_status_v1.go（新增）`
  - `internal/article/deletion_replay_test.go（新增）`
  - `internal/article/replay_status_v1_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestReplayStatus|TestDeletionReplay|TestReplayEvidence"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S4.2`、`article-management-S4.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：先以 strict variant、固定 digest、run/backup 并发、input 漂移和崩溃用例证明旧 ready 失效；相同 input 复用 ref/digest/time；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.5、article-management-S1.5、article-management-S4.4
- **设计/测试来源**：article Story 4/S16；test A-REC-01～07

- [ ] article-management-S4.3 验证 Article Deletion Journal 与 finalizer 韧性 [P]

- **目标**：验证 Article Deletion Journal 与 finalizer 韧性。
- **类型**：专项韧性测试
- **文件**：
  - `test/integration/article/deletion_journal_test.go（新增）`
  - `test/integration/article/deletion_finalizer_test.go（新增）`
  - `test/integration/article/journal_faults_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/article -run "TestDeletionJournal|TestDeletionFinalizer"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S4.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：独立持久卷上覆盖部分写、重复 sequence、链/HMAC/key/namespace/fsync/空间故障、双 finalizer与三类删除；独立 Review 通过。
- **后续验证关联**：article-management-S2.3、article-management-S4.4、article-management-S1.4
- **设计/测试来源**：article test Article Deletion Journal/finalizer 专项

- [ ] article-management-S4.4 验证 ReplayStatus producer 与旧备份恢复 [P]

- **目标**：验证 ReplayStatus producer 与旧备份恢复。
- **类型**：专项契约/韧性测试
- **文件**：
  - `test/integration/recovery/article_replay_test.go（新增）`
  - `test/integration/recovery/replay_evidence_test.go（新增）`
  - `test/integration/recovery/old_backup_restore_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/recovery -run "TestArticleReplay|TestOldBackupRestore"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S4.2`、`identity-auth-S4.3`、`article-management-S4.2`、`article-management-S4.3`、`article-management-S4.5`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：三类 tombstone、operation/audit 对账、双 backup 并发、输入漂移、双次 ReadStatus 与旧备份恢复通过；缺 inventory/RPO/RTO 时不关闭真实演练部分；独立 Review 通过。
- **后续验证关联**：identity-auth-S4.4、article-management-S2.3、article-management-S1.4
- **设计/测试来源**：article test ReplayStatus producer 与旧备份恢复专项；A-REC-01～08

- [ ] article-management-S4.5 准备备份 inventory、tombstone/key/evidence 留存输入 [P]

- **目标**：准备备份 inventory、tombstone/key/evidence 留存输入。
- **类型**：运行输入准备
- **文件**：
  - `docs/runbooks/article-deletion-recovery.md（新增）`
  - `testdata/recovery/article-backup-inventory.json（新增）`
  - `testdata/recovery/article-deletion-fixtures.json（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/recovery -run "TestArticleRecoveryFixtures"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S4.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：tombstone/key/inventory/evidence retention、RPO/RTO 与运行责任未确认
- **本任务关闭条件**：仅使用合成 inventory/备份/删除样本；明确退役证据、清理责任和 fail-closed 条件，不填未确认 retention/RPO/RTO；独立 Review 通过。
- **后续验证关联**：article-management-S4.4
- **设计/测试来源**：article test §7；AM-Q8/Q13

- [ ] article-management-S5.1 实现管理员删除他人普通文章

- **目标**：实现管理员删除他人普通文章。
- **类型**：开发
- **文件**：
  - `internal/article/command_service.go（修改）`
  - `internal/article/handler.go（修改）`
  - `internal/article/admin_delete_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestAdminDeleteOrdinaryArticle"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.2`、`article-management-S3.1`、`article-management-S4.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：管理员仅能从普通文章入口删除 active/published/clear/有作者文章；违规/无归属必须走专用入口，且不能编辑他人；独立 Review 通过。
- **后续验证关联**：article-management-S1.5
- **设计/测试来源**：article Story 5；test A-MODR-01～02

- [ ] article-management-S6.1 实现 canonical 无归属化端口

- **目标**：实现 canonical 无归属化端口。
- **类型**：开发（article 共享接缝，Story 6 所有）
- **文件**：
  - `internal/article/article_port.go（新增）`
  - `internal/article/command_service.go（修改）`
  - `internal/article/orphan_canonical_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestOrphanArticlesCanonical"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.2`、`article-management-S1.1`、`article-management-S1.2`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：零/多/违规/已处置/deletion_pending 矩阵幂等；只写 article-owned 状态与 Audit，不回退已处置文章；独立 Review 通过。
- **后续验证关联**：identity-auth-S9.1、identity-auth-S1.5
- **设计/测试来源**：article Story 6 共享 canonical 端口；test A-ORP-01～02

- [ ] article-management-S6.2 实现无归属文章公开、转移与永久删除

- **目标**：实现无归属文章公开、转移与永久删除。
- **类型**：开发
- **文件**：
  - `internal/article/command_service.go（修改）`
  - `internal/article/handler.go（修改）`
  - `internal/article/orphan_disposition_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestOrphanDisposition"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S8.1`、`identity-auth-S9.1`、`article-management-S4.1`、`article-management-S6.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：只有真实 orphan 可处置；violating 不得直接公开；转移仅 active 目标并保留违规事实；完成事实正确触发 retained_email 清理；独立 Review 通过。
- **后续验证关联**：identity-auth-S9.2、article-management-S1.5、article-management-S7.1
- **设计/测试来源**：article Story 6；test A-ORP-03～07

- [ ] article-management-S7.1 实现转移通知 recipient lease 与 best-effort dispatcher

- **目标**：实现转移通知 recipient lease 与 best-effort dispatcher。
- **类型**：开发
- **文件**：
  - `internal/article/notice_dispatcher.go（新增）`
  - `internal/article/repository.go（修改）`
  - `internal/article/notice_dispatcher_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestTransferNotice|TestNoticeDispatcher"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.4`、`identity-auth-S5.1`、`article-management-S6.2`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：两收件人独立五态；Gate 关闭不 claim；短事务 lease、事务外 SMTP、旧 worker 不覆盖新 owner；accepted 崩溃风险可审计；独立 Review 通过。
- **后续验证关联**：article-management-S1.5、article-management-S7.2
- **设计/测试来源**：article Story 7；test A-NOT-01～08

- [ ] article-management-S7.2 验证通知双 worker、SMTP/TLS 与崩溃恢复 [P]

- **目标**：验证通知双 worker、SMTP/TLS 与崩溃恢复。
- **类型**：专项集成/安全测试
- **文件**：
  - `test/integration/article/notice_dispatcher_test.go（新增）`
  - `test/integration/article/notice_smtp_test.go（新增）`
  - `test/integration/article/notice_crash_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/article -run "TestNotice"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.4`、`article-management-S7.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：双 worker、lease 回收、Gate 竞态、瞬时/永久错误、TLS 降级/证书和 accepted 后崩溃通过；不发外部邮件；独立 Review 通过。
- **后续验证关联**：identity-auth-S3.4、article-management-S2.3、article-management-S1.4
- **设计/测试来源**：article test 通知 lease、双 worker 与 SMTP/TLS 专项；A-NOT-01～09

- [ ] article-management-S8.1 实现违规判定、解除与永久删除治理

- **目标**：实现违规判定、解除与永久删除治理。
- **类型**：开发
- **文件**：
  - `internal/article/command_service.go（修改）`
  - `internal/article/handler.go（修改）`
  - `internal/article/policy.go（修改）`
  - `internal/article/moderation_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestModeration"`
  - 当前状态：因 `go.mod`、代码、测试环境或对应外部输入不存在，未运行；以上为实现后应执行的具体命令。Manifest 未登记项目检查命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.2`、`article-management-S1.2`、`article-management-S3.1`、`article-management-S4.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：先证明非法类别、重复状态、错入口、并发 ETag 和审计失败；restore 只清违规事实，其他隐藏约束保留；独立 Review 通过。
- **后续验证关联**：article-management-S1.5、article-management-S2.2
- **设计/测试来源**：article Story 8；test A-MODR-03～06

