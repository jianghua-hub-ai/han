# Han identity-auth observability runbook

## 1. Scope and readiness boundary

This runbook is the repository SSOT for the provider-neutral identity-auth
observability materials:

- [`identity-alerts.yaml`](../../deploy/observability/identity-alerts.yaml)
- [`identity-alerts.test.yaml`](../../deploy/observability/identity-alerts.test.yaml)
- [`identity-grafana-dashboard.json`](../../deploy/observability/identity-grafana-dashboard.json)
- [`identity-alertmanager-routing.yaml`](../../deploy/observability/identity-alertmanager-routing.yaml)

These files are S3.5 deployment inputs. They do not prove that logs are
retained, metrics are scraped, the dashboard imports, alerts evaluate, a
receiver delivers, or an exercise succeeds. Those real-platform checks and
their evidence belong to S3.6 / I-OBS-DEP-01 through I-OBS-DEP-06. The
production observability Gate remains closed until the applicable platform
validators, independent review, and S3.6 exercise pass.

No provider, account, paid plan, network implementation, receiver integration,
secret, individual rota, or exercise window is selected by these materials.

> **2026-10-27 生效范围（[ADR-0025](../../.add/project/knowledge/decisions/framework.md#adr-0025)）**：按用户最终多选答复，测试项目 Han 的 S3.5 免除 promtool/platform validator、Grafana 导入/查询、Alertmanager 配置/路由等真实平台和校验工具验证作为关闭前提；既有五文件材料静态 Review PASS 仅指原固定对象，本段仍待独立范围评审，未运行的检查不记为 PASS。本文中这些校验作为 S3.5 关闭前提的原表述仅在该任务被豁免；用于 S3.6 时完整保留，§3～§12 的真实部署、日志留存/查询、scrape/网络、告警/10 分钟恢复、Dashboard、receiver 送达/升级、非作者演练、证据和确认条件均不变；§12 的外部配置/owner/索引、账号/secret/排班、平台命令、候选版本、网络与窗口等缺口仍为 S3.6 未满足前置。该决定另仅豁免 S3.3 真实浏览器环境准备/连接/执行，明确未执行真实浏览器验证，不据此宣称 HTTP/Go 替代测试全部完成；不改变产品安全合同，不构成全项目免测、任务关闭或生产就绪，tasks 交 Lead 维护。

## 2. Required application configuration

All values below are mandatory. The application has no code default for them.
Missing, empty, unparsable, non-finite, or out-of-range values fail closed
during startup; a deployment must not silently substitute a value.

| Environment variable | Deployment baseline | Accepted value |
|---|---:|---|
| `HAN_METRICS_ADDR` | No universal address baseline | Exactly one IPv4 `IP:port` or bracketed IPv6 `[IP]:port`, with port `1..65535`, for the independent metrics listener. Hostnames, URLs, paths, multiple addresses, and an empty host are rejected. The actual IP and port still require deployment/network approval. |
| `HAN_LOGIN_OBSERVATION_BUDGET` | `250ms` | `1ms..5s` |
| `HAN_ORIGIN_OBSERVATION_BUDGET` | `250ms` | `1ms..5s` |
| `HAN_LOGIN_FAILURE_AUDIT_BUDGET` | `2s` | `10ms..30s` |
| `HAN_SHUTDOWN_BUDGET` | `10s` | `1s..60s` |
| `HAN_LOGIN_DURATION_BUCKETS_SECONDS` | `0.005,0.01,0.025,0.05,0.1,0.25,0.5,1,2.5,5` | Comma-separated list of 1 to 32 positive, finite, strictly increasing, non-duplicate seconds; maximum value `60`. |

Legal non-baseline values still require deployment review. Record the reviewed
values with the deployed revision; never place credentials or secrets in this
runbook or its evidence.

## 3. Logs

Production logging is one `slog.JSONHandler` JSON object per stdout line
(JSON Lines). The deployment platform must collect stdout without multiline
joining or text rewriting and provide at least 30 days of queryable retention.
S3.6 must prove both the configured policy and the oldest queryable sample over
the required window; a policy declaration alone is insufficient.

Stable query dimensions currently emitted by the relevant adapters include
`service`, `module`, `event`, `trace_id`, and, where applicable, `endpoint`,
`outcome`, `reason_bucket`, `error_code`, `failure_stage`,
`transaction_outcome`, `state_changed`, `audit_persisted`,
`rollback_outcome`, `reason`, and `response_write`. Treat `trace_id` as a log
correlation field, not a metric label.

Queries and sampled evidence must not contain or index secrets, passwords,
temporary credentials, cookies, authorization or CSRF tokens, database DSNs,
HMAC keys/digests, full email addresses, full source addresses, message bodies,
or raw internal causes. If any denylisted value appears, stop the exercise,
restrict evidence access, and start the security incident process.

PostgreSQL `AuditEvent` records are the authoritative business audit. JSON
Lines are diagnostic and correlation evidence only: they must not replace an
`AuditEvent`, be used to backfill or rewrite one, or by themselves authorize a
retry when transaction or audit persistence is unknown. Resolve unknown audit
state only through the approved database-backed reconciliation path; never
infer a committed or absent business event solely from a log line.

## 4. Metrics endpoint and scrape

`HAN_METRICS_ADDR` owns a listener separate from the business HTTP listener.
It exposes only `/metrics`. The response format is fixed Prometheus text
exposition `0.0.4` with
`Content-Type: text/plain; version=0.0.4; charset=utf-8`; `Accept` is ignored
and there is no content negotiation. Encoding is completed and validated
before status 200 is committed; encoding failure returns a fixed, redacted
500 response.

Network policy must allow the approved Prometheus-compatible scraper to reach
the metrics listener and block unapproved networks. It must also prove that
the listener exposes neither its root path nor business paths. The listener
must not be made publicly reachable merely to complete validation.

The deployment scrape configuration must assign the target the scrape label
`job="han-identity-auth"`. This is deployment metadata used by the `up` alert
and dashboard; it is not an application metric label. S3.6 must verify the
label against the real target configuration before enabling the rules.

The application-owned metrics and labels are:

| Metric | Type | Application labels |
|---|---|---|
| `han_identity_login_attempts_total` | Counter | `endpoint,outcome,reason_bucket,error_code` |
| `han_identity_login_internal_failures_total` | Counter | `endpoint,reason_bucket,error_code,failure_stage` |
| `han_identity_login_duration_seconds` | Classic histogram | `endpoint,outcome,reason_bucket,error_code` plus generated `le` on buckets |
| `han_platform_origin_rejections_total` | Counter | `reason,response_write` |

Do not add deployment, user, trace, source-address, digest, or other
high-cardinality/sensitive labels to these application series. A real,
continuous backend scrape is an S3.6 requirement and does not replace the
application's independent text-format conformance evidence.

## 5. Dashboard

Import `identity-grafana-dashboard.json` as **Han / Identity Auth** and bind
`${DS_PROMETHEUS}` to the approved Prometheus-compatible datasource. Verify
each panel against raw queries over the same time range:

1. scrape health for `job="han-identity-auth"`;
2. login attempts by fixed `outcome`;
3. login duration P50, P95, and P99;
4. internal failures by fixed `failure_stage`;
5. Origin rejections by fixed `reason,response_write`;
6. current firing identity-auth alerts.

The fixed `failure_stage` values are `source_evidence`,
`transaction_begin`, `abuse_check`, `user_query`,
`temporary_credential_query`, `credential_verify`, `credential_rehash`,
`recovery_mode`, `abuse_record`, `session_issue`, `session_persist`,
`audit_append`, `failure_audit`, `transaction_commit`, `cookie_output`,
`response_write`, and `unknown`.

An importable JSON document or an empty panel is not a pass. S3.6 must record
the imported dashboard revision, datasource mapping, time window, raw-query
comparison, and reviewer result.

## 6. Alerts and stable resolution

Load `identity-alerts.yaml` into the approved rule evaluator. Every rule uses
`keep_firing_for: 10m`: after its expression becomes false, the alert remains
firing until the signal has stayed below the threshold for 10 minutes.

| Alert | Trigger | Hold | Severity and initial route |
|---|---|---:|---|
| `HanIdentityAuthScrapeDown` | `up{job="han-identity-auth"} == 0` | `2m` | `paging` to production on-call role |
| `HanIdentityAuthInternalFailureRatioHigh` | 10-minute attempts `>=20` and internal failures / attempts `>5%` | `10m` | `paging` to production on-call role |
| `HanIdentityAuthLoginDurationP95High` | 10-minute duration samples `>=20` and P95 `>2s` | `15m` | `warning` to team operations alerting role |
| `HanIdentityAuthOriginRejectionsHigh` | 10-minute Origin rejections `>=20` | `10m` | `warning` to team operations alerting role |

Before deployment, run the applicable Prometheus rule validator and execute
`identity-alerts.test.yaml`. The repository test data covers all four firing
conditions and a scrape alert that remains firing during, then exits after,
the 10-minute stable-resolution period. Static YAML parsing is not equivalent
to rule validation or time-series execution.

## 7. Routing and escalation

`identity-alertmanager-routing.yaml` contains role placeholders only:

- `warning` routes to `role-team-operations-alerting`;
- `paging` routes to `role-production-on-call`.

The receiver entries intentionally contain no provider integration, address,
account, or secret. Platform/operations ownership must map them to approved
real receivers outside the repository and run the target Alertmanager route
test before S3.6.

The response and escalation chain is:

1. production on-call role;
2. Han service owner / identity technical responsibility role;
3. security on-call role for suspected or confirmed security events.

Do not replace role names in this repository with individuals. S3.6 must verify
real receipt, grouping, silence/inhibition behavior, and the approved role
mapping without committing receiver details or secrets.

## 8. Triage and response

For every alert:

1. record the deployed application revision, rule/dashboard revision,
   environment identifier, alert start time, and affected instances;
2. open **Han / Identity Auth** and compare the alert expression with its raw
   series over at least the alert window plus its hold duration;
3. correlate JSON Lines by bounded time, `service`, `event`, and `trace_id`;
4. distinguish application failures from log-pipeline, scraper, network,
   dashboard, rule-evaluator, or receiver failures;
5. preserve redacted evidence and link any issue record;
6. after mitigation, verify the expression stays false for the full 10-minute
   resolution period before declaring recovery.

### 8.1 Scrape down

- Confirm whether the process is running and whether both listeners originally
  bound successfully.
- Check scraper target state and approved-network reachability to the metrics
  listener; do not expose the listener to a wider network as a shortcut.
- Check `/metrics` status, content type, and parse result from the approved
  scraper path. Do not treat a business endpoint response as metrics health.
- If the server or metrics encoder is failing, escalate to the Han service
  owner / identity technical responsibility role.
- **Immediate containment decision:** the production on-call role stops
  promotion when the target revision cannot expose valid metrics or its process
  health is uncertain. Isolate candidate application business traffic through
  the approved deployment control, while keeping the target visible to service
  discovery, the approved scraper, and rule evaluation so `up == 0` or target
  absence cannot be hidden. Do not widen metrics access, move metrics onto the
  business listener, disable authentication controls, or treat missing
  telemetry as healthy.
- **Approved maintenance exception:** if an approved operational reason
  requires the target to leave service discovery or scraping, use the approved,
  traceable maintenance/silence process and retain an independent
  expected-target inventory plus absence evidence. A silence or absent series
  is not recovery and must not close the alert or incident.
- **Exit or rollback:** resume promotion or remove containment only after the
  target is visible on the approved scraper path, explicitly reports
  `up{job="han-identity-auth"} == 1`, and remains healthy continuously for
  10 minutes. If maintenance removed the target from discovery or scraping,
  restore target visibility before starting that 10-minute verification;
  absence never satisfies the exit condition. If the failure follows a
  candidate rollout and cannot be safely corrected within the approved
  response window, the Han service owner / identity technical responsibility
  role restores the last validated application or external observability
  revision, restores scrape/rule visibility, and repeats the same health
  verification.

### 8.2 Internal failure ratio

- Confirm the attempt denominator is at least 20 and compare
  `han_identity_login_internal_failures_total` by `failure_stage`,
  `reason_bucket`, and `error_code`.
- Correlate `login_internal_failure` logs without exposing raw causes or
  credentials.
- Prioritize persistence, transaction commit, session, audit, and response
  failures; verify whether state changed and whether audit persistence is
  known before any retry or repair.
- Do not reinterpret authentication failures or rate limiting as internal
  failures.
- **Immediate containment decision:** the production on-call role stops
  promotion when the paging condition fires. The Han service owner / identity
  technical responsibility role decides whether to remove the candidate from
  service or restore the last validated revision after checking authoritative
  PostgreSQL state and `AuditEvent` evidence. Escalate suspected security
  causation to the security on-call role.
- **Prohibited actions:** do not retry, compensate, backfill, or rewrite
  business/audit state from JSON Lines; do not suppress the alert by excluding
  failure stages or changing its denominator.
- **Exit or rollback:** leave containment only when the authoritative state is
  reconciled, the failure ratio expression is false, and it stays false for
  10 minutes. Roll back the candidate when its revision is correlated with the
  failures and forward repair cannot preserve transaction and audit contracts.

### 8.3 Login P95 duration

- Confirm at least 20 histogram samples in the same 10-minute window.
- Compare P50/P95/P99 and outcomes; check database, credential processing,
  session, audit, and response stages without changing histogram buckets
  during the incident.
- Preserve the configured bucket list and deployment revision with evidence.
  Capacity tuning requires separate approval and measured evidence.
- **Immediate containment decision:** the team operations alerting role pauses
  promotion and asks the Han service owner / identity technical responsibility
  role to compare the candidate against the last validated latency baseline.
  The owner may reduce or remove candidate traffic through the approved
  deployment control; no capacity or timeout value is invented during the
  incident.
- **Prohibited actions:** do not change histogram buckets, lower the sample
  gate, relax authentication/security work, or hide slow outcomes to clear the
  warning.
- **Exit or rollback:** resume only after at least 20 samples show P95 at or
  below 2 seconds and the expression remains false for 10 minutes. Restore the
  last validated application revision when the degradation is candidate-bound
  and cannot be safely mitigated under approved capacity limits.

### 8.4 Origin rejections

- Break down `han_platform_origin_rejections_total` by the fixed `reason` and
  `response_write` labels.
- Correlate the `origin_rejected` event in JSON Lines and distinguish expected
  hostile/invalid traffic from an incorrect public-origin or proxy deployment.
- Escalate suspected attacks or sensitive evidence immediately to the security
  on-call role. Do not weaken Origin checks as an incident workaround.
- **Immediate containment decision:** the team operations alerting role stops
  promotion while the Han service owner / identity technical responsibility
  role checks origin and proxy configuration. The security on-call role owns
  containment decisions for suspected hostile traffic; approved network-edge
  controls may be used without changing application Origin validation.
- **Prohibited actions:** keep strong Origin enforcement. Do not add broad
  origins, bypass the guard, trust unapproved proxy headers, enlarge metrics
  exposure, or discard rejection evidence merely to clear the warning.
- **Exit or rollback:** resume only after the configuration or traffic cause is
  understood, the expression stays false for 10 minutes, and security confirms
  any required containment. If a candidate configuration caused legitimate
  traffic rejection, restore the last validated origin/proxy or application
  revision rather than weakening the guard.

## 9. Shutdown and process result

`HAN_SHUTDOWN_BUDGET` is one shared deadline for the business and metrics
servers. On `SIGTERM` or `SIGINT`, or after a server error, the coordinator:

1. calls both server `Shutdown` operations concurrently with the shared
   deadline;
2. if either shutdown fails or the deadline expires, calls `Close` on both
   servers;
3. waits for both Serve loops;
4. closes both listeners;
5. closes the database pool last.

Only a signal-triggered path with all shutdown and cleanup operations
successful exits with code 0. A server failure, shutdown failure/timeout, Serve
failure during shutdown, listener cleanup failure, or database-pool cleanup
failure exits non-zero. Deployment rollout and termination checks must capture
the signal source, elapsed shutdown time, exit code, and redacted lifecycle
error. A clean process exit does not prove that external log or metrics data
was retained.

## 10. Rollback

If rules, dashboard, or routing fail target-platform validation:

1. stop promotion and keep the production observability Gate closed;
2. restore the last independently validated external rule, dashboard, and
   routing revisions using the platform-owned rollback process;
3. do not change application metric names/labels, broaden metrics network
   access, weaken Origin enforcement, reduce log retention, or substitute a
   mock receiver to make validation pass;
4. verify the restored scrape and alert state, then retain redacted failure and
   rollback evidence for review.

If no previously validated external revision exists, leave the candidate
undeployed and escalate the missing production control; file existence in this
repository is not a safe fallback.

## 11. Exercise and evidence index

S3.6 must use an approved environment, synthetic non-sensitive traffic, a
reviewed window, and non-author runbook review. Exercise all four alerts,
warning and paging delivery, escalation, and recovery. Each run records:

- application and five-file revisions;
- environment and platform configuration references;
- synthetic trigger and cleanup procedure;
- alert pending/firing/resolved timestamps, including the stable 10 minutes;
- dashboard and raw-query comparison;
- receiver and role-confirmation result;
- shutdown/exit result where exercised;
- redacted log, scrape, rule, dashboard, route, and issue evidence indexes;
- participating responsibility types, result, reviewer, and unresolved risks.

Store I-OBS-DEP-01 through I-OBS-DEP-06 Case/Run evidence only in the S3.6
project evidence location selected by that task. Do not write production data,
credentials, receiver endpoints, account identifiers, or natural-person rota
details into this repository.

## 12. External configuration ownership and S3.6 prerequisites

| External configuration | Owner type | Repository index | Validator/review responsibility | Required before S3.6 |
|---|---|---|---|---|
| stdout collection, query access, and at-least-30-day retention | Platform/operations configuration owner | Sections 3 and 11 | Platform validator plus non-author operations reviewer | Real policy, current and oldest-sample query paths, approved evidence location |
| metrics target, `job="han-identity-auth"`, and network restriction | Platform/network configuration owner | Sections 4 and 11; alert/dashboard files | Prometheus-compatible target validator plus network reviewer | Deployed candidate, approved scraper path, denied-path test plan, I-F005-19/20 conformance evidence |
| Grafana datasource, folder, import, and access | Observability platform configuration owner | Section 5; dashboard JSON | Target Grafana import/query validator plus non-author reviewer | Real datasource mapping and dashboard access |
| rule loading and evaluation | Observability platform configuration owner | Section 6; alert rules/tests | Applicable rule validator plus non-author reviewer | Validator PASS and controlled trigger plan |
| warning/paging receivers, credentials, grouping, silence/inhibition, and role mapping | Operations/security configuration owners | Section 7; routing YAML | Target Alertmanager route test plus operations/security reviewers | Approved real receivers, protected secrets, role mapping, and exercise window |
| exercise records and issue links | S3.6 Test-Engineer with operations, identity, security, business/release confirmation | Section 11 | Case review and three-layer result review | Approved Case, deployed revisions, participants by role, evidence store |

If external configuration is not stored in this repository, its owner must
provide a stable, access-controlled reference in the S3.6 evidence index.
Missing ownership, reference, validator, reviewer, platform access, deployed
candidate, receiver, network policy, or exercise window is
`blocked-for-execution`; it must not be inferred or replaced with a local
parse check.
