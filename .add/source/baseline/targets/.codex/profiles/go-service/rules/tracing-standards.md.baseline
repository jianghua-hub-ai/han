# 链路追踪规范

## 1. 通用准则（核心规则，跨项目通用）

### 1.1 trace_id 设计

- **格式**：使用 ULID（26 字符）或同等强度的全局唯一 ID
- **生成位置**：在请求入口（如 API Gateway）统一生成；下游服务从请求头提取，禁止手动生成
- **传递机制**：
  - 跨服务 RPC：通过 gRPC Metadata 传递（标准 HTTP header 也兼容）
  - 跨进程调用（如 fork 子进程）：通过环境变量传递
- **日志关联**：所有日志必须携带 `trace_id` 字段；SDK Logger 应自动注入，避免业务代码每次手填

### 1.2 反例

- ❌ 业务代码手动生成 trace_id（应由 Gateway 统一生成）
- ❌ 跨服务调用不传递 trace_id（链路断开）
- ❌ 日志缺少 trace_id 字段（无法关联追溯）

### 1.3 正例（gRPC 服务间传递）

```go
// 服务间调用：从 context 取 trace_id，写入 gRPC metadata
md := metadata.Pairs("x-trace-id", ctx.TraceID())
ctx = metadata.NewOutgoingContext(ctx, md)
client.SomeMethod(ctx, req)
```

### 1.4 正例（子进程获取）

```go
// 子进程从环境变量获取（父进程 fork 时注入）
traceID := os.Getenv("<TRACE_ENV_KEY>")
```

## 2. 本项目实现（按选型可替换）

### 2.1 命名

- 请求头 key：`x-trace-id`
- 环境变量 key：`<待填充:env-prefix>_TRACE_ID`
- gRPC metadata key：`x-trace-id`

### 2.2 生成位置

- `<待填充:request-entry>` 统一入口生成（或从可信客户端的 `x-trace-id` 请求头提取并按安全策略沿用）

### 2.3 SDK 注入

- 使用 `<待填充:sdk-name>` Logger 自动注入：`sdk.Log().Info("...")`
- 日志 JSON 输出示例：

```json
{"trace_id":"01ARZ3NDEKTSV4RRFFQ69G5FAV","service":"<service>","level":"INFO","msg":"..."}
```

### 2.4 跨项目复用指引

未来其他项目复用本规则：

- §1 通用准则直接复制
- §2 本项目实现按新项目选型替换：
  - 环境变量 key（如 `MYAPP_TRACE_ID`）
  - 请求入口 / 网关实现（替换 `<待填充:request-entry>`）
  - SDK Logger 实现（替换 `<待填充:sdk-name>`）
