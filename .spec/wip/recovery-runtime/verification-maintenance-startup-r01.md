# 普通服务器维护态拒启：有界修复交接

本轮修复已由 Lead 现场接收。合法配置显式为 `HAN_RECOVERY_MODE=maintenance` 时，普通无参数服务器清理已加载配置拥有的秘密，再于依赖准备、数据库解析/连接、迁移、bootstrap、监听和 worker 之前返回启动失败。生产改动仅 `cmd/han-server/main.go` 四行；另在原测试文件追加边界回归。正常启动及原 password-reset 命令分派保持原顺序。

采用依据为既有 ADR-0018/0020 的恢复期间不接业务义务及 [Pascal 有界方案](architecture-maintenance-startup-r01.md)。这次没有新增模式、命令、公共错误码、HTTP 接口、authority、ADR 或任务。

- 作者先保留 compiled overlay 的 1 个行为失败、8 个通过对照；正常与错误配置/原 argv 有明确控制。
- 作者启动相关普通、竞态各 28 叶，维护命令兼容各 87 叶通过；最终仅命名修订经精确字节桥接，最终测试版本另直接运行 9 叶和 vet。
- [Dewey 两轮独立评审](review-code-maintenance-startup-r01.md) PASS、零 finding；评审者直接对最终版本普通、竞态各 29 叶及 vet/格式检查通过。
- [Lead 项目检查](evidence-startup-r01/lead-manifest/index.json)：六项非格式配置命令通过、交付两文件格式净；全仓 `gofmt -l .` 仍因保留的旧 S10.5 证据 FAILED2，按本修复限定例外保留，未记全七项通过。
- [Lead 最终接收及范围核证](evidence-startup-r01/lead-reception/verification.json)：仅原两文件改动，其他 186 项源码、33 份知识材料、依赖/Manifest/已有可执行文件保持；分支和 HEAD 固定，无暂存、提交、推送或清理。

新维护态分支未进入 prepared 阶段；其秘密清理依据为实际调用顺序及原 helper 语义，不能拿零个别名观测当动态清零证明，也不承诺不可变字符串物理擦除。所有新边界执行使用合成配置和受控工厂；未连接真实数据库、启动产品服务器或证明实际 HTTP 503。

完整恢复 runtime、Coordinator 消费、正常模式下 unfinished 恢复核验、恢复后成功开流和真实旧备份仍待后继；此修复不授权实际恢复后改为 normal 直接开流。父任务状态只取 [Tasks SSOT](../../../.add/project/knowledge/tasks/tasks.md)，article S1.5/S4.4/S8.2 及完整 Case18 不因本修复关闭。应用数据库角色 ACL 还需 [独立事实回执](../article-management/research-s1.5-acl-readiness-r01.md)，其未验状态不被 SQL 回滚、trigger 拒绝或本修复替代。
