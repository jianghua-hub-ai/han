# Article 删除恢复合成输入 Runbook

## 适用范围与非声明

本 runbook 只使用 synthetic-only 输入，用于仓内严格解析、引用对账与 fail closed 自动检查。它不构成灾备演练，不构成真实恢复证明，不构成 retirement 授权，也不构成 production readiness。

## Synthetic-only 数据规则

真实凭据、PII 和生产数据禁止进入本制品。Article key material、备份引用、tombstone、operation 与 evidence 都必须是公开合成值；不得使用真实备份、真实用户内容、生产路径或连接信息。

## Entry Gate

真实恢复 entry gate 保持关闭。entry gate 只有在实际责任人完成独立核项后才可能由其他获授权流程处理；本任务不打开真实恢复、retirement 或 cleanup。

## Inventory、tombstone、key 与 evidence 对账

自动检查逐字核对 inventory、key、tombstone 与 evidence。缺少 inventory、key、tombstone 或 evidence、引用不一致、版本未知时均 fail closed。Ready 的 inventory_digest 只与固定合成引用 sha256:bbbb 逐字比较，不定义生产 inventory digest 算法。

## Retirement evidence 与人工授权

旧可恢复备份未退役时必须 fail closed。retirement evidence 的真实性与批准必须由实际责任人记录；合成 retirement 引用只覆盖测试分支，不代表真实证据或人工授权。

## Fail-closed 条件

unknown retention / RPO / RTO 阻止真实恢复和灾备演练。缺输入、错误类型、重复键、trailing data、状态混合、摘要或交叉引用不一致均 fail closed，不得用默认值或推断值继续。

## Cleanup 责任

cleanup owner、范围和授权未确认时不得清理。人工核项必须明确 cleanup owner、允许范围和回滚边界；本任务不删除、移动或退役任何真实资源。

## Unknown：retention / RPO / RTO

retention、RPO 与 RTO 均保持 unknown。本任务不填写期限、恢复点或恢复时间目标；retention / RPO / RTO 决定由实际责任人另行批准。

## 自动检查与人工核项

自动检查只验证合成 JSON、Ready20/Unavailable6、固定跨引用、mutation 负例和本文结构。人工核项还必须覆盖真实恢复 entry gate、retirement evidence 的真实性与批准、cleanup owner、允许范围和回滚边界、retention / RPO / RTO 决定、真实运行窗口以及最终 readiness；自动通过不能替代签字。

## 关闭边界

本任务只关闭四个仓内合成制品的机械验证，不运行 PostgreSQL、网络、Host、部署或 E2E，不声称真实旧备份已恢复，也不声称灾备、生产运行或最终 readiness 已完成。
