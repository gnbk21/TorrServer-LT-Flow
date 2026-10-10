# Five-repeat streaming comparisons

Checkpoint: `ff5df2fa`; [CI run 38069527761](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/38069527761).

These are original-byte, controlled Linux native peer/HTTP tests, not Android decoder playback. All 160 trials completed byte-integrity checks. Each row contains five rotated runs. Values below are medians; five trials are too few for precise population tail estimates. Server read blocking is not automatically a player-visible stutter.

Stable baseline: `MatriX.145.Flow-v1.0.1`, source `55c16dc399b8f9b14bd5ae76b62450af027c6b81`; published checksums and provenance were verified before running.

Stable executable SHA-256: `8ff5d9f4263da477b3aba4d78ffdae4409d750208792c7ca819c936ea19c195c`.
Candidate executable SHA-256: `b7211a2e717bbd8189c0c3232a83c88a0bca38cad6321267d1f99deabf258f0b`.

Matched settings: 120 Mb/s generated media, 60 seconds, RAM cache, 512 MiB cache, 32 MiB startup reserve, 4 MiB pieces. Healthy, outage, burst and mixed-peer scenarios each use the same bytes across compared variants. Profile and transport cohorts ran separately; CPU timings across cohorts must not be compared.

## Profiles

| Scenario/profile | Passed | Startup ms | Blocking over 250 ms, total ms | Seek P99 ms | Peak RSS MiB | CPU seconds | Native duplicate MiB |
|---|---:|---:|---:|---:|---:|---:|---:|
| bursts/previous-legacy | 5/5 | 5804.17 | 0.00 | 718.92 | 1159.01 | 12.10 | 0.02 |
| bursts/previous-adaptive | 5/5 | 5804.31 | 0.00 | 649.62 | 1159.70 | 12.33 | 0.00 |
| bursts/legacy | 5/5 | 5803.41 | 0.00 | 1086.22 | 1119.93 | 12.75 | 0.02 |
| bursts/adaptive | 5/5 | 5804.73 | 0.00 | 47.20 | 1117.30 | 12.98 | 0.00 |
| healthy/previous-legacy | 5/5 | 5803.92 | 0.00 | 502.52 | 1115.90 | 9.97 | 0.00 |
| healthy/previous-adaptive | 5/5 | 5803.00 | 0.00 | 1082.09 | 1117.12 | 9.79 | 0.00 |
| healthy/legacy | 5/5 | 5803.11 | 0.00 | 869.23 | 1117.53 | 10.32 | 0.00 |
| healthy/adaptive | 5/5 | 6003.44 | 0.00 | 387.97 | 1116.61 | 10.42 | 0.00 |
| mixed-peers/previous-legacy | 5/5 | 5805.27 | 0.00 | 286.07 | 1120.66 | 11.02 | 2.50 |
| mixed-peers/previous-adaptive | 5/5 | 5804.85 | 0.00 | 220.48 | 1121.21 | 11.02 | 7.09 |
| mixed-peers/legacy | 5/5 | 6004.27 | 0.00 | 351.86 | 1124.07 | 11.79 | 16.08 |
| mixed-peers/adaptive | 5/5 | 6002.29 | 0.00 | 911.65 | 1116.77 | 11.88 | 2.06 |
| outages/previous-legacy | 5/5 | 5405.91 | 14524.30 | 783.84 | 1118.12 | 23.38 | 0.00 |
| outages/previous-adaptive | 5/5 | 5407.36 | 16521.01 | 697.39 | 1119.05 | 23.43 | 0.02 |
| outages/legacy | 5/5 | 5405.77 | 18732.62 | 1081.41 | 1118.17 | 24.72 | 0.02 |
| outages/adaptive | 5/5 | 5406.64 | 15353.05 | 1081.37 | 1116.82 | 24.88 | 0.00 |

## Transport buffers

Native direct is 0 KiB; 64, 256 and 1024 KiB use the existing buffered HTTP reader. These runs all use the candidate Adaptive profile.

| Scenario/buffer KiB | Passed | Startup ms | Blocking over 250 ms, total ms | Seek P99 ms | Peak RSS MiB | CPU seconds | Native duplicate MiB |
|---|---:|---:|---:|---:|---:|---:|---:|
| bursts/adaptive-transport-0 | 5/5 | 5405.01 | 0.00 | 47.09 | 1116.67 | 26.17 | 0.00 |
| bursts/adaptive-transport-64 | 5/5 | 5405.36 | 0.00 | 656.40 | 1115.80 | 25.88 | 0.00 |
| bursts/adaptive-transport-256 | 5/5 | 5405.86 | 0.00 | 502.21 | 1117.20 | 26.06 | 0.02 |
| bursts/adaptive-transport-1024 | 5/5 | 5403.37 | 0.00 | 908.73 | 1117.41 | 26.14 | 0.02 |
| healthy/adaptive-transport-0 | 5/5 | 5604.11 | 0.00 | 325.39 | 1115.93 | 13.92 | 0.02 |
| healthy/adaptive-transport-64 | 5/5 | 5404.75 | 0.00 | 596.33 | 1116.14 | 13.87 | 0.02 |
| healthy/adaptive-transport-256 | 5/5 | 5404.00 | 0.00 | 46.00 | 1114.66 | 13.75 | 0.00 |
| healthy/adaptive-transport-1024 | 5/5 | 5403.43 | 0.00 | 561.35 | 1116.90 | 13.86 | 0.00 |
| mixed-peers/adaptive-transport-0 | 5/5 | 5605.80 | 0.00 | 638.31 | 1116.41 | 24.92 | 12.73 |
| mixed-peers/adaptive-transport-64 | 5/5 | 5606.43 | 0.00 | 419.27 | 1117.10 | 24.36 | 2.11 |
| mixed-peers/adaptive-transport-256 | 5/5 | 5606.10 | 0.00 | 465.02 | 1121.17 | 24.58 | 7.27 |
| mixed-peers/adaptive-transport-1024 | 5/5 | 5605.73 | 0.00 | 397.08 | 1117.94 | 24.54 | 5.39 |
| outages/adaptive-transport-0 | 5/5 | 6006.67 | 17260.76 | 1077.93 | 1114.72 | 18.50 | 0.00 |
| outages/adaptive-transport-64 | 5/5 | 6005.90 | 16690.78 | 559.93 | 1110.28 | 18.74 | 0.00 |
| outages/adaptive-transport-256 | 5/5 | 6005.34 | 12976.86 | 1071.50 | 1114.46 | 18.83 | 0.00 |
| outages/adaptive-transport-1024 | 5/5 | 6005.28 | 15903.31 | 840.63 | 1116.60 | 18.54 | 0.00 |

## Decisions and limits

- Healthy, burst and mixed-peer trials had no server reads blocked for more than 250 ms. Outage trials retained substantial blocking in every profile; passing byte integrity does not mean uninterrupted playback.
- Candidate Adaptive outage blocking had a 15.35 s median versus 16.52 s for stable Adaptive. Stable Legacy was 14.52 s; candidate Legacy was 18.73 s. The data does not establish that the candidate is universally faster.
- The 256 KiB transport had the lowest median outage blocking (12.98 s versus 15.90 s at 1024 KiB). Other scenarios and five-run variability do not support an automatic default change. The 1024 KiB default and Legacy default remain unchanged.
- Profile cohort candidate CPU medians increased about 5-8% in several cases. Representative profiling and final-head comparisons remain necessary before making a performance claim.
- These reports predate the subsequent v2/session/expiry corrections. Final acceptance must repeat the required comparisons for the release candidate. Eight-hour endurance, physical Android telemetry and Windows recovery checks are separate gates.
