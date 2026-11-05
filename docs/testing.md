# 测试约定

## 默认检查

无外部依赖的默认检查：

```text
go test ./...
go test -race ./...
go test -tags=contract ./...
```

`contract` 测试不得连接数据库、已部署服务或浏览器；缺少外部环境不应影响该命令。

## 标签与环境变量

| 标签 | 外部依赖 | 必需环境变量 |
| --- | --- | --- |
| `contract` | 无 | 无 |
| `integration` | PostgreSQL 18 | `HAN_TEST_POSTGRES_DSN` |
| `browser` | 已部署服务、浏览器 | `HAN_TEST_BASE_URL`、`HAN_TEST_BROWSER_BIN` |
| `e2e` | 已部署服务 | `HAN_TEST_BASE_URL`；包含浏览器步骤时还需 `HAN_TEST_BROWSER_BIN` |

带外部依赖的测试必须通过 `internal/testenv` 的 `Require...` helper 读取必需值。变量缺失或仅包含空白时测试必须明确失败，不得 `Skip` 或静默通过。

PostgreSQL 迁移集成测试命令：

```text
go test -tags=integration ./internal/migration -run "TestRunner"
```

集成测试会读取 DSN、连接真实数据库并核对 PostgreSQL major 为 18。测试使用独立随机 schema，并验证迁移顺序、成功记录、失败回滚/停止和并发 advisory lock。当前仓库不包含真实 DSN、服务地址或浏览器路径。
