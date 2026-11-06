# Web 前端部署接入（同源反代）

> 本文是**配置样例与上线核对清单**，不表示任何环境已经部署。样例未在真实
> 主机上执行过验证；实际启用需由部署责任方按目标环境改写域名、证书路径、
> 产物目录与后端地址。

## 1. 前提

| 项 | 要求 | 原因 |
|---|---|---|
| 协议 | 仅 HTTPS | 会话 Cookie 名为 `__Host-han_session`，浏览器要求 `Secure` + `Path=/` + 无 `Domain` |
| 源 | 前后端同源 | Cookie 为 `SameSite=Lax`，跨站 XHR 不携带；CSRF 头也依赖同源 |
| 前端产物 | `web/dist` | `npm run build` 输出 |
| 后端 | 仅监听回环地址 | 由反代统一暴露，避免绕过 TLS 直连 |

## 2. 构建

```bash
cd web
npm ci
npm run build     # tsc --noEmit && vite build → web/dist
```

产物只有 `index.html` 与 `assets/`，不含服务端代码，也不需要 Node 运行时。

## 3. 反向代理要点

完整样例见 `deploy/nginx-han.conf.example`，必须覆盖以下四点：

1. **`/` 静态托管 + SPA 回退**：`try_files $uri $uri/ /index.html`，否则刷新
   `/articles/xxx` 会 404。
2. **`/api/**` 转发到后端**：保留原始 `Host`、`Cookie` 与 `X-CSRF-Token`；
   不要改写路径前缀（后端路由就是 `/api/v1/...`）。
3. **HTTPS 与安全头**：`Strict-Transport-Security`、`X-Content-Type-Options`、
   `X-Frame-Options: DENY`（或 CSP `frame-ancestors 'none'`）。
4. **缓存分层**：`index.html` 用 `no-store`；`/assets/*`（带内容哈希）可
   `immutable` 长缓存。

## 4. 上线核对清单

- [ ] `npm run build` 成功，`web/dist/index.html` 与 `web/dist/assets/*` 存在
- [ ] 只开放 443；80 端口仅做跳转
- [ ] `GET /` 返回 `index.html`；`GET /articles/<任意 id>` 也返回 `index.html`
- [ ] `POST /api/v1/sessions` 能登录并下发 `__Host-han_session`（`Secure`、`HttpOnly`、`SameSite=Lax`、`Path=/`）
- [ ] 登录后 `GET /api/v1/articles?limit=1` 返回 200；未登录返回 401
- [ ] 写请求带 `X-CSRF-Token` 时成功，缺失时被拒（403 `auth.csrf_failed`）
- [ ] `POST /api/v1/sessions/logout` 返回 204 且清 Cookie；之后同一 Cookie 访问受保护接口返回 401
- [ ] 浏览器控制台无混合内容、无 CSP 违例
- [ ] 刷新页面后仍能判定登录态（受保护只读探测，不新增 `/users/me`）

## 5. 已知边界

- 前端不读取任何后端环境变量，只依赖同源路径 `/api/v1`。
- 不做 SSR/SEO，爬虫拿到的是空壳 `index.html`。
- 管理员治理页、密码自助不在首期范围。
- `POST /api/v1/users`（注册）与 `GET /api/v1/users/me/articles`（我的文章列表）
  已在 IR-02 的 GAP-1/GAP-2 处置中挂载/实现，但**注册仍依赖部署侧真实数据**：
  `privacy_notices` 必须存在 `current` 版本且隐私 processing gate 已正确配置，
  否则注册会返回 `privacy.notice_outdated` 或 `platform.processing_gate_closed`。
  上线前请按 §4 清单实测一次注册。
- 前端注册页使用**静态隐私声明文案（v1.0.0）**；若服务端 `current` 版本号与之
  不一致，注册会被拒绝。升级文案时必须同步改动 `web/src/pages/register.tsx`
  的 `privacyNoticeVersion` 并重新构建。
