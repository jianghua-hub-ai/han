# 数据库迁移

迁移资产使用 `NNNN_name.sql` 命名：

- `NNNN` 是四位十进制版本号，在目录内必须唯一。
- `name` 使用小写字母、数字和单下划线分隔的名称。
- 执行顺序只由版本号升序决定；非 `.sql` 文件和子目录不参与执行。

`internal/migration` 提供项目内执行器。一次运行在同一 PostgreSQL 连接上持有 advisory lock，创建并读取 `schema_migrations`，然后逐个应用未记录的版本。每个 SQL 文件及其成功记录位于同一独立事务中；任一版本失败时回滚该版本并立即停止，后续版本不会执行。

迁移 SQL 与包含执行器的应用代码必须同批交付。

`han-server` 启动时先校验必填的 `HAN_DATABASE_DSN` 与 `HAN_HTTP_ADDR`，再验证数据库连接并自动执行迁移。监听地址只接受单个 IPv4 或方括号 IPv6 `IP:port`，port 范围为 `1–65535`，没有代码默认值，也不会回退到客户端测试变量 `HAN_TEST_BASE_URL`。

任一配置、数据库连接或迁移失败时，服务会在建立 HTTP listener 前 fail closed 并以非零状态退出。迁移成功后才调用 `net.Listen("tcp", addr)` 并启动不含业务路由的最小 HTTP 服务；监听或 Serve 失败会清理 listener 和数据库连接池，并以非零状态退出。
