# go-service Profile

仅供明确采用 Go 服务端技术栈的项目启用。`rules/` 是 Go 实现细则和编码前安全加载门禁；`../../skills/add-tdd-go/` 提供 Go 专用 TDD 方法。

Go 版本、服务框架、公共库、检查命令和例外只引用项目已有单一依据（架构/ADR 或已登记项目 manifest），不在各 Rule 再填表。Go 不隐含资源协议、DDD/CQRS 或事件系统；资源专用方法只在明确启用 resource-platform 时读取，复杂模型按真实选型适用。tdd-go 的方法保留完整，采用 profile 不表示可选测试工具已安装。
