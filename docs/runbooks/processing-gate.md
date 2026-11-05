# ProcessingGate、同源、可信代理与 SMTP 运行输入

本机 local-dev 登记状态：用户于 **2026-10-22** 作为本次部署、隐私与安全责任人，批准下列参考占位值，并确认九个必填键已设置在 User scope；`HAN_SMTP_CA_BUNDLE` 未设置。

本次批准只适用于本机 local-dev，不是生产批准、法律结论、SMTP 处理方实际批准或部署后 E2E 证据。`deploy/processing-gate.env.example` 是始终保持空值的模板，不承载 User-scope 值；不得向示例、本文、日志、错误或版本库写入 DSN、密码或其他凭据。

## 当前运行时读取方边界

`internal/platform/config.go` 的 `LoadDeploymentConfig` 当前读取并校验以下十项非秘密部署输入：

- `HAN_PROCESSING_REGION_REF`
- `HAN_PROCESSING_CONTROLLER_REF`
- `HAN_PROCESSING_LEGAL_BASIS_REF`
- `HAN_SMTP_PROCESSOR_REF`
- `HAN_PUBLIC_ORIGIN`
- `HAN_TRUSTED_PROXY_CIDRS`
- `HAN_SMTP_TLS_MODE`
- `HAN_SMTP_HOST`
- `HAN_SMTP_SERVER_NAME`
- `HAN_SMTP_CA_BUNDLE`

前九项均为必填项；缺失或仅为空白时配置加载 fail closed。`HAN_SMTP_CA_BUNDLE` 为可选受控 CA 文件路径，缺失或空白表示使用系统信任根；reader 只保存清理后的路径，不读取文件，也不验证文件是否存在。配置错误只标识键名和稳定原因，不包含已配置值。

当前校验包括：

- 四个 ProcessingGate 引用必须存在且非空白。
- `HAN_PUBLIC_ORIGIN` 必须是 canonical HTTP(S) origin；拒绝 userinfo、query、fragment、非根路径和非法端口，并移除默认端口与尾斜杠。
- `HAN_TRUSTED_PROXY_CIDRS` 必须包含至少一个合法 CIDR；逗号分隔项不得为空，并保存规范化网段。
- `HAN_SMTP_TLS_MODE` 仅允许 `implicit_tls` 或 `starttls_required`。
- `HAN_SMTP_HOST` 与 `HAN_SMTP_SERVER_NAME` 必须存在且非空白。

`cmd/han-server` 在确认数据库 DSN 非空后调用 `LoadDeploymentConfig`，该调用发生在数据库配置解析、数据库连接与 ping、迁移、监听和服务启动之前。任一部署配置校验失败时，这些后续步骤均不执行；CLI 顶层仍只输出通用启动失败消息。

reader 已落地不等于消费者已启用，以下边界保持不变：

- 尚未实现 Origin middleware，`HAN_PUBLIC_ORIGIN` 未被 HTTP 请求处理链消费。
- 尚未读取或信任转发来源 Header，`HAN_TRUSTED_PROXY_CIDRS` 未被代理 Header 处理消费。
- 尚未实现 mailer、SMTP 连接或发送；SMTP 配置仅完成启动期读取与校验。
- `HAN_SMTP_PROCESSOR_REF` 的读取不等于实际 SMTP 处理方已批准，也不等于 TLS 连接或证书校验已经执行。

当前 reader 状态是代码事实，必须由独立 Reviewer/人工直接核对真实读取方。机械测试不解析本文，也不证明本文不存在后追加的矛盾声明；出现与代码不一致的描述应作为 Review finding 处理。

## 2026-10-22 local-dev 输入登记

| 输入 | local-dev User scope 状态 | 登记责任 | 缺失或未实现时的行为 |
|---|---|---|---|
| `HAN_PROCESSING_REGION_REF` | 已设置：`local-dev/region` | 用户（本次 local-dev 部署/隐私/安全责任人） | 缺失或空白时 `LoadDeploymentConfig` 拒绝启动 |
| `HAN_PROCESSING_CONTROLLER_REF` | 已设置：`local-dev/controller` | 用户（本次 local-dev 部署/隐私/安全责任人） | 缺失或空白时 `LoadDeploymentConfig` 拒绝启动 |
| `HAN_PROCESSING_LEGAL_BASIS_REF` | 已设置：`local-dev/not-approved-legal-basis` | 用户（本次 local-dev 部署/隐私/安全责任人） | 缺失或空白时 `LoadDeploymentConfig` 拒绝启动；该占位引用不是法律结论 |
| `HAN_SMTP_PROCESSOR_REF` | 已设置：`local-dev/no-approved-smtp-processor` | 用户（本次 local-dev 部署/隐私/安全责任人） | 缺失或空白时 `LoadDeploymentConfig` 拒绝启动；该占位引用不代表处理方已批准 |
| `HAN_PUBLIC_ORIGIN` | 已设置：`http://127.0.0.1:8080` | 用户（本次 local-dev 部署/隐私/安全责任人） | 缺失、空白或非法时拒绝启动；Origin middleware 尚未实现 |
| `HAN_TRUSTED_PROXY_CIDRS` | 已设置：`127.0.0.1/32,::1/128` | 用户（本次 local-dev 部署/隐私/安全责任人） | 缺失、空白或非法时拒绝启动；转发来源 Header 仍不得受信任 |
| `HAN_SMTP_TLS_MODE` | 已设置：`starttls_required` | 用户（本次 local-dev 部署/隐私/安全责任人） | 缺失、空白或不是 `implicit_tls` / `starttls_required` 时拒绝启动；尚无 SMTP 连接 |
| `HAN_SMTP_HOST` | 已设置：`smtp.local.invalid` | 用户（本次 local-dev 部署/隐私/安全责任人） | 缺失或空白时拒绝启动；保留域名不是外部 SMTP 服务批准，且尚无 SMTP 连接 |
| `HAN_SMTP_SERVER_NAME` | 已设置：`smtp.local.invalid` | 用户（本次 local-dev 部署/隐私/安全责任人） | 缺失或空白时拒绝启动；尚无 SMTP 连接或证书校验 |
| `HAN_SMTP_CA_BUNDLE` | 未设置（使用系统信任根） | 用户（本次 local-dev 部署/隐私/安全责任人） | 可空；空白使用系统信任根，非空路径只保存而不读取；尚无 SMTP 连接 |
| SMTP TLS 安全基线 | unknown | 安全责任人 | 尚无获批准数值或已命名配置键；不得发明，邮件能力保持关闭 |
| SMTP 端口与认证凭据键 | unknown | 部署/安全责任人 | 不在当前十键合同内；不得发明、追加或写入仓库 |

这些 local-dev 参考占位值的批准不延伸到生产、外部 SMTP、法律结论或 SMTP 处理方实际批准。获批准 SMTP 处理方、SMTP 主机、ServerName、CA、TLS 安全基线、端口和认证配置是不同事实，不能用一个非空 `HAN_SMTP_PROCESSOR_REF` 代替其他事实。任何未来 SMTP 实现都必须验证证书链、有效期和主机名，禁止明文、机会式降级及 `InsecureSkipVerify` 或等价配置。

## 登记与启用步骤

1. 用户已于 2026-10-22 承担本次 local-dev 的部署、隐私与安全登记责任，并批准上表参考占位值。
2. 九个必填键已设置在 User scope；`HAN_SMTP_CA_BUNDLE` 未设置，以选择系统信任根。该登记不证明进程已成功启动或配置已完成部署后 E2E。
3. 保持 `deploy/processing-gate.env.example` 十个赋值为空，不把 User-scope 值复制到模板。
4. 生产或外部 SMTP 使用前，必须另行登记并批准真实地区、控制者、处理依据、canonical origin、可信代理、SMTP 处理方、主机、ServerName、信任根或受控 CA、TLS 基线、端口与凭据承载方式。凭据只进入受控环境，不进入本文、示例、日志、错误或版本库。
5. 启动期 reader 会在数据库连接、迁移和监听前 fail closed，但不会启用下述消费者。
6. 运行下列机械检查。平台测试证明十键读取、校验及 env 示例允许键集合精确且全部为空；启动链测试证明配置失败发生在数据库连接、迁移和监听之前。它们不证明本文 reader 状态真实，也不证明 Origin middleware、代理 Header 消费或 SMTP 连接已经实现：

   ```text
   go test ./internal/platform -run "TestProcessingGate|TestOrigin|TestTrustedProxy|TestLoadDeploymentConfig"
   go test ./cmd/han-server -run "^TestRun_StartupGate$"
   go test ./internal/platform
   ```

7. Origin middleware、可信代理 Header 消费以及 mailer/SMTP 连接必须由后续已授权任务分别实现和验证；本卡不启用这些能力。

## 当前阻塞与 Lead 决策

- 本次 local-dev 的九个必填参考占位值已在 User scope 设置，CA 未设置；这不是生产部署值或生产批准。
- 实际生产地区、控制者、处理依据、canonical origin、可信代理、获批准 SMTP 处理方、SMTP 主机、ServerName、CA、TLS 安全基线、端口与凭据仍为 unknown。
- `local-dev/not-approved-legal-basis` 与 `local-dev/no-approved-smtp-processor` 明确表示未形成法律结论、未批准实际 SMTP 处理方。
- 十键启动期 reader 已落地；尚无本次 local-dev 启动成功或部署后 E2E 证据，不能声称已经部署。
- Origin middleware、可信代理 Header 消费、mailer/SMTP 连接与发送仍未实现；需由后续任务明确消费者边界并完成独立验证。
- 本次批准不延伸到上述后续消费者；在它们分别实现并完成独立验证前，不得声称对应能力已启用。
