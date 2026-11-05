# Han — 内容发布社区（IR-01）

从零建设的内容发布社区。首期（IR-01）完成**用户身份认证**与**文章管理**的最小可用闭环：注册、登录、创建/修改/删除/浏览文章，以及管理员对无归属文章的处置。评论、收藏、关注和标签不在首期范围。

> 状态：IR-01 已按 86 张任务关闭（其中 10 张为范围决策关闭）。本仓库为测试项目交付，**不宣称生产就绪**，详见下文「边界」。

## 技术栈

- Go 1.27，标准库 `net/http`
- PostgreSQL 18 + `pgx/v5`，版本化 SQL 迁移（`migrations/`，11 个文件）
- 密码与临时凭据使用 Argon2id
- 会话使用服务端不透明 Cookie + CSRF 防护

## 目录结构

```text
cmd/han-server/        HTTP 入口、路由与生产装配
internal/              领域模块（identity、article、session、security、recovery 等）
migrations/            版本化 SQL 迁移
test/contract/         契约测试（无外部依赖）
test/integration/      集成测试（真实 PostgreSQL 18）
test/e2e/              部署后 E2E（当前未执行，见 ADR-0036）
docs/                  测试、依赖与运维 runbook
.add/project/knowledge 需求、设计、决策（ADR）、任务与批次计划
.spec/wip/             本地工作证据（仅摘要入库，原始产物不入库）
```

## 快速开始

### 必需环境变量

| 变量 | 说明 |
| --- | --- |
| `HAN_DATABASE_DSN` | 应用连接串（PostgreSQL 18） |
| `HAN_MIGRATION_DATABASE_DSN` | 迁移专用连接串；缺省时按实现决定回退行为 |
| `HAN_HTTP_ADDR` | HTTP 监听地址，如 `127.0.0.1:8080` |

其他特性开关（审计、恢复、通知等）见 `.add/project/knowledge/decisions/` 与 `docs/`。**禁止提交真实凭据**；测试用 DSN 仅限本机/保留域名，见 ADR-0038。

### 构建与运行

```bash
go build ./...
go run ./cmd/han-server
```

启动时会执行迁移并校验数据库版本；配置缺失或平台不支持时按 fail-closed 退出。

## 测试

| 标签 | 外部依赖 | 必需变量 |
| --- | --- | --- |
| `contract` | 无 | 无 |
| `integration` | PostgreSQL 18 | `HAN_TEST_POSTGRES_DSN` |
| `browser` | 已部署服务、浏览器 | `HAN_TEST_BASE_URL`、`HAN_TEST_BROWSER_BIN` |
| `e2e` | 已部署服务 | `HAN_TEST_BASE_URL`（含浏览器步骤时另需 `HAN_TEST_BROWSER_BIN`） |

```bash
go test -tags=contract ./...   # 无外部依赖，CI 首选
go test ./...
go test -race ./...            # 需要 CGO
```

详见 [docs/testing.md](docs/testing.md)。

## 项目检查

```text
gofmt -l ./cmd ./internal ./test
go mod verify
go vet ./...
go build ./...
go test -tags=contract ./...
go test -race ./...
go test ./...
```

## 接口

生产业务 API 共 16 个端点，覆盖文章读取/写入、作者管理、管理员文章查询与无归属文章处置。每个端点有精确 method/path、405 `Allow`，且 404/405 先于安全链。契约由 `test/contract/` 锁定。

## 文档索引

- [变更日志](CHANGELOG.md)
- [测试约定](docs/testing.md)、[依赖说明](docs/dependencies.md)
- 运维 runbook：[数据库恢复](docs/runbooks/database-recovery.md)、[文章删除恢复](docs/runbooks/article-deletion-recovery.md)、[身份可观测性](docs/runbooks/identity-observability.md)
- 架构决策：`.add/project/knowledge/decisions/README.md`
- 需求/设计/任务：`.add/project/knowledge/`
- 发布材料（内部草稿）：`.add/project/knowledge/release/ir-01-content-community/`

## 边界（未验证 / 未实现）

- 无部署后真实 E2E、无真实平台 observability 演练（ADR-0036）。
- 生产恢复链未实现：CAS anchor provider、key-retirement、restore 接线、POSIX 独占锁、11 端口 production root（ADR-0037）。真实 Journal/HMAC/keyring、旧备份恢复未验证。
- Case18 `failed`/`expired`/`undetermined`、真实部署双系统未验证（ADR-0034/0035）。
- 容量与性能未测（ADR-0033）。

## 提交规范

使用 [Conventional Commits](https://www.conventionalcommits.org/)：`feat`、`fix`、`docs`、`test`、`chore` 等；提交信息带任务编号或模块 scope。

