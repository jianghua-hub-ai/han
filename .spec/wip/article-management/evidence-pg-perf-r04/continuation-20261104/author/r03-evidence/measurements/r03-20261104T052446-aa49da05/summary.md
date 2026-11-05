# PGPERF r04 evidence summary

Run: `r03-20261104T052446-aa49da05`  
Started: 2026-11-04T05:24:47Z  
Generated: 2026-11-04T05:25:12Z

All figures are observational on a shared development machine. They are not target-environment SLO evidence and do not imply capacity approval.

> Capacity limits: small <= 1.0 GB / medium <= 5.5 GB / large <= 8.5 GB / total <= 15 GB. 2026-11-04 本轮用户允许采用建议值，Lead START 明确采用；原 r04 small=0.5/total=14 未修改，新值待本轮独立 Review。单位为 GiB（2^30 bytes）。

| Point | Tier | Mode | Concurrency | N | P50 ms | P95 ms | QPS | Errors | Exploratory |
|---|---|---|---:|---:|---:|---:|---:|---:|---|
| R02-fault-measurement | small | TIMED-3REP | 1 | 1 | 0.519 | 0.519 | 1924.928 | 1 | true |

## Seeds

| Schema | Tier | Case | Articles | Hidden | Near-limit | Seed ms | ANALYZE ms |
|---|---|---|---:|---:|---:|---:|---:|
| han_pgperf_r04_sr2ferr_86c4321399bef5bf | small | r2ferr | 200 | 56 | 10 | 43 | 16 |

## Schemas

- `han_pgperf_r04_sr2ferr_86c4321399bef5bf` tier=small case=r2ferr retained=true
