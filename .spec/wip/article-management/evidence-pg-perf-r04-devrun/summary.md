# PGPERF r04 evidence summary

Run: `r04d-20261103T155443`  
Started: 2026-11-03T15:54:45Z  
Generated: 2026-11-03T16:06:57Z

All figures are observational on a shared development machine. They are not target-environment SLO evidence and do not imply capacity approval.

| Point | Tier | Mode | Concurrency | N | P50 ms | P95 ms | QPS | Errors | Exploratory |
|---|---|---|---:|---:|---:|---:|---:|---:|---|
| W01-typical | small | TIMED-3REP | 1 | 170403 | 0.560 | 1.052 | 1893.336 | 0 | false |
| W01-near-limit | small | TIMED-3REP | 1 | 97003 | 0.887 | 1.509 | 1077.789 | 0 | false |
| R01-limit20 | small | TIMED-3REP | 1 | 37133 | 2.283 | 3.264 | 412.563 | 0 | false |
| R04-10rows | small | TIMED-3REP | 1 | 248059 | 0.518 | 0.707 | 2756.205 | 0 | false |
| R04-empty | small | SINGLE-RUN | 1 | 331151 | 0.000 | 0.549 | 11038.274 | 0 | true |
| R04-all-hidden | small | SINGLE-RUN | 1 | 3135 | 9.064 | 13.687 | 104.488 | 0 | true |
| R04-last-page | small | TIMED-3REP | 1 | 159110 | 0.568 | 1.033 | 1767.866 | 0 | false |
| D04-100 | small | SINGLE-RUN | 1 | 100 | 1.166 | 1.882 | 784.081 | 0 | true |

## Seeds

| Schema | Tier | Case | Articles | Hidden | Near-limit | Seed ms | ANALYZE ms |
|---|---|---|---:|---:|---:|---:|---:|
| han_pgperf_r04_sw01a_a35a89a943dac228 | small | w01a | 1000 | 325 | 10 | 145 | 46 |
| han_pgperf_r04_sw01b_a35a89a943dac228 | small | w01b | 1000 | 325 | 10 | 145 | 48 |
| han_pgperf_r04_sr01a_a35a89a943dac228 | small | r01a | 1000 | 0 | 10 | 101 | 43 |
| han_pgperf_r04_sr04a_a35a89a943dac228 | small | r04a | 10 | 0 | 1 | 18 | 11 |
| han_pgperf_r04_sr04b_a35a89a943dac228 | small | r04b | 0 | 0 | 0 | 0 | 0 |
| han_pgperf_r04_sr04c_a35a89a943dac228 | small | r04c | 1000 | 1000 | 10 | 137 | 43 |
| han_pgperf_r04_sr04d_a35a89a943dac228 | small | r04d | 1000 | 0 | 10 | 133 | 49 |
| han_pgperf_r04_sx01a_a35a89a943dac228 | small | x01a | 0 | 0 | 0 | 0 | 0 |
| han_pgperf_r04_sx01b_a35a89a943dac228 | small | x01b | 10 | 0 | 1 | 22 | 13 |
| han_pgperf_r04_sd04s_a35a89a943dac228 | small | d04s | 200 | 0 | 10 | 26 | 13 |

## Schemas

- `han_pgperf_r04_sw01a_a35a89a943dac228` tier=small case=w01a retained=true
- `han_pgperf_r04_sw01b_a35a89a943dac228` tier=small case=w01b retained=true
- `han_pgperf_r04_sr01a_a35a89a943dac228` tier=small case=r01a retained=true
- `han_pgperf_r04_sr04a_a35a89a943dac228` tier=small case=r04a retained=true
- `han_pgperf_r04_sr04b_a35a89a943dac228` tier=small case=r04b retained=true
- `han_pgperf_r04_sr04c_a35a89a943dac228` tier=small case=r04c retained=true
- `han_pgperf_r04_sr04d_a35a89a943dac228` tier=small case=r04d retained=true
- `han_pgperf_r04_sx01a_a35a89a943dac228` tier=small case=x01a retained=true
- `han_pgperf_r04_sx01b_a35a89a943dac228` tier=small case=x01b retained=true
- `han_pgperf_r04_sd04s_a35a89a943dac228` tier=small case=d04s retained=true
