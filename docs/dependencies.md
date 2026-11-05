# Go 依赖审查

审查日期：2026-10-21。

## 直接依赖

| Module | 版本 | 用途 | 许可证 | 许可证证据 |
| --- | --- | --- | --- | --- |
| `github.com/jackc/pgx/v5` | `v5.11.0` | PostgreSQL 原生 API 与 `pgxpool` | MIT | Go module cache 中该版本根目录的 `LICENSE` |
| `golang.org/x/crypto` | `v0.58.0` | S2.1 Argon2id 实现已使用；业务消费者尚未接线 | BSD-3-Clause | Go module cache 中该版本根目录的 `LICENSE` |

版本由 Go 1.27.0 工具实际解析并写入 `go.mod`/`go.sum`，不是凭记忆填写。

## 已解析的传递依赖

| Module | 版本 | 许可证 |
| --- | --- | --- |
| `github.com/jackc/pgpassfile` | `v1.0.0` | MIT |
| `github.com/jackc/pgservicefile` | `v0.0.0-20240606120523-5a60cdf6a761` | MIT |
| `github.com/jackc/puddle/v2` | `v2.2.2` | MIT |
| `golang.org/x/sync` | `v0.24.0` | BSD-3-Clause |
| `golang.org/x/sys` | `v0.49.0` | BSD-3-Clause |
| `golang.org/x/text` | `v0.43.0` | BSD-3-Clause |

许可证类型来自各已解析 module cache 版本根目录的 `LICENSE`。`golang.org/x/sys v0.49.0` 由 `golang.org/x/crypto v0.58.0` 的 Argon2id 实现依赖图引入；其余传递依赖由 pgx/v5 的 `pgxpool` 依赖图产生，未新增其他项目直接依赖。

## 完整性与供应链限制

- 解析时仅对命令进程设置 `GOPROXY=https://goproxy.cn,direct`。
- `GOSUMDB` 保持启用，命令进程使用 `sum.golang.org https://sum.golang.google.cn`；未执行 `go env -w`，未关闭校验数据库。
- `go mod verify` 已执行并返回 `all modules verified`。
- 本批次未安装或运行额外的许可证扫描器、SBOM 生成器或 `govulncheck`，因此未声称完成自动化漏洞或完整供应链审计。
- 项目未 vendor 依赖；后续构建仍依赖可访问的 module proxy/cache 与 checksum database。
