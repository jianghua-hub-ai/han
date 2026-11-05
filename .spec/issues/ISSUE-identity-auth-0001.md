# ISSUE-identity-auth-0001 — passwordResetGateSQL 使用保留字别名 `current_user`

> 本问题单由 `identity-auth-S1.6`（验证身份 PostgreSQL 事务、锁与并发矩阵）在真实 PostgreSQL 18 上发现。
> 定位已完成；按用户 2026-10-27 授权「对自审/待确认项采用建议值并继续推进」执行最小修复并登记回归证据。

## 1. 登记（register）

| 字段 | 值 |
|---|---|
| 编号 | `ISSUE-identity-auth-0001` |
| 来源 | `identity-auth-S1.6` 真实 PG18 执行 `TestS16SessionResolveRejectsRevokedAndSuspendedUsers` |
| 首次发现 | 2026-10-27 |
| 关联任务 | `identity-auth-S1.6`（发现）；受影响已关闭卡 `identity-auth-S10.2` / `S10.3` |
| 关联制品 | `internal/identity/password_reset_store.go`（`passwordResetGateSQL`、`passwordResetResendLineageStatusSQL`） |
| 影响模块 | `identity-auth`（认证/会话解析全链路） |
| 固定工作区 | `D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 环境 | PostgreSQL 18.6（`127.0.0.1:5433`），pgx v5.11.0 |

## 2. 定级（classify）

**P0 — 影响生产可用，紧急。**

- `session.Store.ResolvePrincipal` 在**每一个**已认证请求上调用 `PasswordResetGate.CheckLockedUser`；该 SQL 在真实 PostgreSQL 上恒抛语法错误 → 所有登录后认证路径不可用。
- 关闭链 S10.2/S10.3 的测试只用 stub + `strings.Contains` 断言，从未在真实 PG 上执行该 SQL，因此缺陷长期隐藏。
- 定级来源：Lead（本 Host 无委派能力；用户已授权采用建议值继续推进）。

## 3. 定位（diagnose）

### 3.1 复现命令与证据

真实生产代码路径（非 stub）：

```
$ go test -tags=integration ./test/integration/identity \
    -run TestS16SessionResolveRejectsRevokedAndSuspendedUsers -count=1
--- FAIL: TestS16SessionResolveRejectsRevokedAndSuspendedUsers (0.65s)
    matrix_test.go:637: ResolvePrincipal() error = session store unavailable:
      check password reset gate: check password reset Gate:
      错误: 语法错误 在 "." 或附近的 (SQLSTATE 42601)
```

最小 SQL 复现（psql，PostgreSQL 18.6）：

```sql
-- 合法：非保留字别名
SELECT NOT zz_alias.f FROM zz zz_alias WHERE zz_alias.id = 1;   -- OK
-- 非法：保留关键字作表别名
SELECT NOT current_user.f FROM zz2 current_user WHERE current_user.id = 1;
-- 错误: 语法错误 在 "." 或附近的 (SQLSTATE 42601)
```

关键字目录证据：

```sql
SELECT catcode, catdesc FROM pg_get_keywords() WHERE word = 'current_user';
-- R | 已保留
```

### 3.2 根因

`current_user` 是 PostgreSQL **保留关键字**（`pg_get_keywords().catcode='R'`），被用作 `FROM users current_user` 的表别名，并在两个 SQL 常量内以 `current_user.<col>` 形式引用 39 处。解析器在 `alias.column` 处即报 `42601`。

代码注释中旧文本以 `gemini`/多模型交替产出的风格保留，未见任何真实 PG 执行验证 → 5 Whys 根因：**durable Gate 的 SQL 只经过 stub 与字符串断言验证，缺少真实 PostgreSQL 编译/执行门禁**。

### 3.3 影响范围

| 消费点 | 影响 |
|---|---|
| `session.Store.ResolvePrincipal` → `CheckLockedUser` | 全部已认证请求 100% 失败（`platform.database_unavailable` 类失败） |
| `identity.Login` → `login.go:1274` | 登录校验后 Gate 失败 |
| `identity.PasswordChange` → `service.go:502` | 改密 Gate 失败 |
| `temporary_credential_resend` → `temporary_credential_resend.go:160/198` | resend Gate 失败 |
| `passwordResetResendLineageStatusSQL` | resend lineage 分类恒失败 |

### 3.4 已排除

- 全仓其余 `FROM/JOIN/UPDATE/INTO <table> <alias>` 用法已用 `pg_get_keywords()` 目录交叉扫描，无其他保留字别名（`where`/`on`/`order`/`unique`/`real`/`is` 命中项均经人工确认为非别名位——`ON true`、`ORDER BY`、注释、`AS` 关键字等）。

## 4. 修复（remedy）

**最小、保持契约的修复**：将两个 SQL 常量中的别名 `current_user` 重命名为非保留字 `current_account`；同步更新三处 `strings.Contains` 测试断言（不含 schema 对象名 `password_reset_operations_current_user_unique`，该标识符为真实约束名，保持不变）。

- 不改公共 code、HTTP 合同、schema、migration、依赖、文件边界。
- 变更文件（4）：
  | 文件 | 变更 |
  |---|---|
  | `internal/identity/password_reset_store.go` | 别名 + 39 处引用重命名 |
  | `internal/identity/password_reset_store_test.go` | 13 处断言字符串同步 |
  | `internal/identity/temporary_credential_resend_test.go` | 1 处断言字符串同步 |
  | `test/integration/identity/smtp_credentials_test.go` | 1 处断言字符串同步 |

> 关注点：本次修复改变了 `password_reset_store.go` 的 SHA256（原 `6C013640...3315F`），影响已关闭 S10.3 的固定哈希证据。按「消费者可见行为不变、仅修复不可运行的 SQL」判定为**局部修复**，已关闭卡的分类/状态语义未变；哈希漂移在 `identity-auth-S1.6` 关闭记录中显式登记。

## 5. 回归（regress）

| 层 | 命令 | 结果 |
|---|---|---|
| 单元 | `go test ./internal/identity -run TestPasswordReset -count=1` | PASS |
| 真实 PG18 集成 | `go test -tags=integration ./test/integration/identity -count=1` | PASS（含 S1.6 九组矩阵 + SMTP 组） |
| 真实 PG18 探针 | `CheckLockedUser` 真实执行 | 无 `42601`，返回正常 Gate 结果 |

## 6. 独立复审（review）

本 Host 无委派能力，无独立 Reviewer Role 产出；经用户 2026-10-27 授权以**作者自审口径**执行。该口径**弱于** `add-code-review` 要求，作为已知限制保留；不再由本问题单消除。

## 7. 状态

- **定位**：完成（2026-10-27）。
- **修复**：完成（最小别名修复，4 文件）。
- **回归**：完成（真实 PG18）。
- **独立复审 / 两方验收**：作者自审口径完成（已知限制）。
- **关联任务**：发现于 `identity-auth-S1.6`；关闭证据记入该卡。

## 8. 防再发候选（未采用）

| 问题与证据 | 候选措施 | 归属 | 状态 |
|---|---|---|---|
| durable Gate SQL 无真实 PG 编译门禁 | 为 identity SQL 常量增加「真实 PG `PREPARE`/`EXPLAIN` 冒烟」或 sqlc/pgx 准备语句校验 | 测试/项目经验 | proposed（未验证；待项目经验流程） |
| stub + `strings.Contains` 无法发现语法错误 | 关键 SQL 至少一条真实执行用例 | 测试 | proposed（S1.6 已部分覆盖 Gate 路径） |

> 永久规则变更先交 `add-project-experience` 登记并取得采用决定，不因本复盘自动生效。
