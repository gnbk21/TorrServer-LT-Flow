# Native CPU profiling

The upgrade CI captures CPU evidence separately from performance comparisons.
Profiling changes timing, so these runs must not promote a streaming profile,
HTTP buffer size or PGO setting.

## Scope and outputs

- Go: the owned synthetic server's loopback profiler captures CPU, runtime
  trace, heap, allocations, mutex, block and goroutine profiles.
- Linux: pinned Perfetto v58.2 captures 99 Hz CPU samples and call stacks for
  the exact owned process ID. It enables no system scheduling, process scan or
  log source. Decoding must find nonzero samples/stacks and no other process.
- Windows: built-in WPR records CPU on a disposable GitHub-hosted runner.
  `tracerpt` must decode the ETL. Filter the recorded `owned_pid` in WPA for
  analysis. Windows CPU recording is system-scoped and is refused on the
  workstation. It uses a unique recording instance and cleans up only that
  instance, including a failed stop or a start timeout.

Captures last at most 30 seconds. Perfetto uses a 32 MiB buffer and 64 MiB file
limit; accepted trace files are limited to 128 MiB. Reports retain the executable
and trace hashes, owned PID, tool hashes where applicable and measurement scope.
An empty, oversized, undecodable or out-of-scope capture fails acceptance.

`flow-native-profile-linux-amd64` and `flow-native-profile-windows-amd64` are
separate fixtures built with `TS_PROFILE_SYMBOLS=1`. They preserve symbols and
are never release assets. Ordinary binaries retain stripping and PGO-off
defaults. The resulting `flow-native-cpu-perfetto` and `flow-native-cpu-etw`
artifacts include traces, decoder results, bounded reports and generated-workload
integrity evidence. Raw captures do not by themselves establish a CPU reduction
or known native attribution; reports keep that distinction explicit.

Perfetto's archive is pinned to the verified upstream SHA-256
`32c739f71b2d39721afd294c0b038f2499a44a71b5b6bcdd85b83faca0b240b9`.
These tools are used only in CI and are not bundled with Flow releases.

References: [Perfetto CPU profiling](https://perfetto.dev/docs/quickstart/callstack-sampling),
[Linux tracing cookbook](https://perfetto.dev/docs/getting-started/linux-cookbook),
[Windows WPR commands](https://learn.microsoft.com/en-us/windows-hardware/test/wpt/wpr-command-line-options),
[pinned Perfetto release](https://github.com/google/perfetto/releases/tag/v58.2).
