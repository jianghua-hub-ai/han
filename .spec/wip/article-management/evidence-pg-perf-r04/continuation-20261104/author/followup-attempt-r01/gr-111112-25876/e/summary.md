# PGPERF r04 evidence summary

Run: `fu_gro_111112_25876`  
Started: 2026-11-04T11:11:14Z  
Generated: 2026-11-04T11:11:41Z

All figures are observational on a shared development machine. They are not target-environment SLO evidence and do not imply capacity approval.

> Capacity limits: small <= 1.0 GB / medium <= 5.5 GB / large <= 8.5 GB / total <= 15 GB. 2026-11-04 本轮用户允许采用建议值，Lead START 明确采用；原 r04 small=0.5/total=14 未修改，新值待本轮独立 Review。单位为 GiB（2^30 bytes）。

| Point | Tier | Mode | Concurrency | N | P50 ms | P95 ms | QPS | Errors | Exploratory |
|---|---|---|---:|---:|---:|---:|---:|---:|---|

## Seeds

| Schema | Tier | Case | Articles | Hidden | Near-limit | Seed ms | ANALYZE ms |
|---|---|---|---:|---:|---:|---:|---:|
| han_pgperf_r04_sr2seed_bc59478014d93ae6 | small | r2seed | 1000 | 0 | 1 | 11303 | 18 |

## Schemas

- `han_pgperf_r04_sr2seed_bc59478014d93ae6` tier=small case=r2seed retained=true
