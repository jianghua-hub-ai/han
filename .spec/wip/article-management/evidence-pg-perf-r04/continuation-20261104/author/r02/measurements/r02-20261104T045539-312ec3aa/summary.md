# PGPERF r04 evidence summary

Run: `r02-20261104T045539-312ec3aa`  
Started: 2026-11-04T04:55:41Z  
Generated: 2026-11-04T04:55:48Z

All figures are observational on a shared development machine. They are not target-environment SLO evidence and do not imply capacity approval.

> Capacity limits: small <= 1.0 GB / medium <= 5.5 GB / large <= 8.5 GB / total <= 15 GB. 2026-11-04 本轮用户允许采用建议值，Lead START 明确采用；原 r04 small=0.5/total=14 未修改，新值待本轮独立 Review。单位为 GiB（2^30 bytes）。

| Point | Tier | Mode | Concurrency | N | P50 ms | P95 ms | QPS | Errors | Exploratory |
|---|---|---|---:|---:|---:|---:|---:|---:|---|

## Seeds

| Schema | Tier | Case | Articles | Hidden | Near-limit | Seed ms | ANALYZE ms |
|---|---|---|---:|---:|---:|---:|---:|

## Schemas

- `han_pgperf_r04_sbc001_7176a8833f8045b8` tier=small case=bc001 retained=true
