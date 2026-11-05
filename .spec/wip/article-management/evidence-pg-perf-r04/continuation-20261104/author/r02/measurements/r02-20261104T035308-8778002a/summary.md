# PGPERF r04 evidence summary

Run: `r02-20261104T035308-8778002a`  
Started: 2026-11-04T03:53:11Z  
Generated: 2026-11-04T03:53:30Z

All figures are observational on a shared development machine. They are not target-environment SLO evidence and do not imply capacity approval.

> Capacity limits: small <= 1.0 GB / medium <= 5.5 GB / large <= 8.5 GB / total <= 15 GB. 2026-11-04 本轮用户允许采用建议值，Lead START 明确采用；原 r04 small=0.5/total=14 未修改，新值待本轮独立 Review。单位为 GiB（2^30 bytes）。

| Point | Tier | Mode | Concurrency | N | P50 ms | P95 ms | QPS | Errors | Exploratory |
|---|---|---|---:|---:|---:|---:|---:|---:|---|

## Seeds

| Schema | Tier | Case | Articles | Hidden | Near-limit | Seed ms | ANALYZE ms |
|---|---|---|---:|---:|---:|---:|---:|

## Schemas

- `han_pgperf_r04_sr2dsk_b307a8c7d8de752d` tier=small case=r2dsk retained=true
