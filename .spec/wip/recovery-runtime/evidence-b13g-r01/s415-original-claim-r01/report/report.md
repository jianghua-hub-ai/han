# 原 B13G / identity-auth-S4.15 实际认领对账

正式 TaskPlanner Nash 实际 ID：`01a18b95-c084-7f90-9952-a5462254cc15`。已按精确匹配的 Main START grant 完成两 SSOT 的一次有界写入，固定 main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，未执行 Git 命令。

本次只登记原 S4.15 的真实 owner `Sartre/add-backend-dev/01a18b91-4da7-72c3-92c7-0e2b5dbe0823`、开始日期、实际认领登记时间 `2026-10-30T14:58:40+08:00` 与当前 READONLY G0/作者本人 D 候选权限；原卡 `[ ]` 改 `[-]`，统计 60 completed / 3 active / 11 todo 改为 60 / 4 / 10。Plan B13G 仅增加一条实际认领来源，上下文旧状态按原时点保留。本次没有新建任务、依赖、批次、命令或技术合同，不重开技术 Planning Gate。

原七项 requires 全部现场核为 `[x]`，原 74 张卡 ID/顺序不变、其他 73 张整卡逐字节相等、319 条正式 edges 有序全等、原 9 批移除唯一认领注记后逐字节相等。两文件全部原命令代码块相等；对两文件执行本次补丁的内存逆向后，完整字节均还原精确 before。逐卡 SHA、319 edges、9 批 SHA 与 inverse 断言见 `beforeafter/validation.json`；精确文本 diff、原始 before/after 与 edits 在 `beforeafter/`。逆向仅在内存验证，没有回滚 SSOT。

现场实际编码：tasks 为 UTF-8 BOM + LF，FF=2；Plan 为 UTF-8 BOM + CRLF。与输入口述顺序相反，实际现场字节已保留。Main 指定 before 快照与 grant SHA 和本次写前完整文件完全相等。

原卡 10 源边界（2 个尚不存在）与全仓 Go/SQL/go.mod/go.sum 文件集合及 SHA 均保持。具体文件 SHA 见 preparation/validation/source-after；这些仅为只读 hash 核证，未运行 Go、test、gofmt、SQL、PG、服务、provider、SMTP、native 或 E2E，没有实现/测试 PASS。没有子委派、清理、删除、移动、归档或 Git 写入。

开发实现与局部测试/非作者 Review 仍按原卡关闭条件，跨层软件专项与部署后真实 E2E 仍按原关联独立闭环。本次认领不满足任何实现验收，SOURCE=0，作者 G0 尚未被本登记认定通过。r56/TP26、Article r27/TP21、B13F/S4.12 完成历史以及人的精确接纳 unknown 原样保留，不代签人、不代独立 Review、不声称 Main 已完成现场复核。Profile README 已读取，检查命令仍引用原卡，本次不运行也不改其技术命令。

已读取角色、方法与规则全文：

- `.codex/agents/add-task-planner.toml`
- `.codex/skills/add-batch-plan/SKILL.md`
- `.codex/skills/add-route-work/SKILL.md`
- `.codex/skills/add-delivery-workflow/SKILL.md`
- `.codex/guidance/task-execution-workflow.md`
- `.codex/guidance/multi-agent-coordination.md`
- `.codex/guidance/design-doc-writing-style.md`
- `.codex/profiles/go-service/README.md`
- `.spec/wip/recovery-runtime/evidence-b13g-r01/s415-original-claim-metadata-start-r01.json`

返回 Main 后由其现场核对两 SSOT diff/status 与 receipt；本角色 metadata 所有权已 RELEASED，SOURCE=0，STOP，不安排下一任务。
