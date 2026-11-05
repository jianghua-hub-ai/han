# Han IR-01 内容社区首期 — 发布说明（内部草稿）

- 里程碑：IR-01（首期最小可用闭环）
- 状态：**内部草稿，未对外签发**；AI 代理整理，非人签发
- 日期：2026-11-05
- 适用对象：产品、研发、测试、内部演示
- 基线：`main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（工作树快照，非 git tag）

## 一、本版交付

1. **身份认证（identity-auth）**：注册、登录/登出、会话与 CSRF 防护、密码重置，以及管理员创建、停用、恢复等治理闭环。
2. **文章管理（article-management）**：文章创建、修改、删除与浏览；作者归属与无归属文章处置；发布意图、违规状态与作者归属正交。
3. **接口契约**：16 个生产业务 API 端点的精确 method/path 与 405 `Allow`；404/405 先于安全链；三投影 allowlist（成员列表 5 键、成员详情 8 键、作者 4 键、Admin 12 键）；19 条公共错误；S1–S16 内部端口契约；Ready 20 / Unavailable 6。
4. **数据与安全子集（真实 PostgreSQL 18.6）**：应用角色 ACL 12/12、真实 Audit SQL（179 项输入）、producer 真实层双轮 58 pass。

## 二、验证结果

| 项 | 结果 |
|---|---|
| contract suite（普通 / `-race`） | 各 286 run / 287 pass / 0 fail / 0 skip |
| `go test -tags=contract ./... -count=1` | exit 0，28.514s（整卡复审 R02 独立复跑） |
| `go vet -tags=contract ./...` | exit 0，2.218s |
| `gofmt -l cmd internal test` | exit 0，0 行输出 |
| `S15-RE-F008` | 非作者独立复审 R02 判 CLOSED（34 叶子 / 0 fail / 0 skip） |
| S1.5 整卡 | 复审 R02 PASS FOR LEAD GATE，Lead 同 revision Gate 通过，2026-11-05 关闭（AI 代理接收） |

## 三、已知限制（影响使用或评估判断）

- **无部署后真实 E2E**：`identity-auth-S3.4`、`article-management-S1.4` 的专用命令从未执行（ADR-0036 范围决策）。
- **无真实平台 observability 演练**：候选版本未构建/未部署，I-OBS-DEP-01～06 未执行（ADR-0036）。
- **生产恢复链未实现**：CAS anchor provider、key-retirement、restore 接线、POSIX 独占锁、11 端口 production root（ADR-0037）。
- **真实 Journal / HMAC / keyring、旧备份恢复未验证**（S4.x 责任）。
- **Case18 `failed`/`expired`/`undetermined` 与真实部署双系统未验证**（ADR-0034/0035）。
- **容量与性能未测**（ADR-0033）。
- 本版为**测试项目**交付：不得宣称 production readiness、灾备就绪或部署就绪。

## 四、使用与升级注意

- 恢复相关入口保持 fail-closed：未配置 provider/restore/锁时按设计拒绝，不提供 fallback。
- 本版结论绑定工作树快照哈希；未打 tag、未提交，不能由 `git checkout` 重建同一证据集。
- 对外引用任何数字时，请同时引用 `.spec/wip/article-management/verification-s1.5.md` 与 `evidence-s1.5/` 下的原始日志。

