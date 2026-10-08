# Upstream maintenance and high bitrate streaming

Scope accepted 8 October 2026: adapt the reviewed LT 1.1.10 fixes and improve
high bitrate streaming within the existing Flow architecture. The original Flow
and Modern Web specifications, free sources, bounded cache and preserved user
settings remain constraints. Live playback is not restarted during development.

## Maintenance

- [ ] Protect user certificates, validate without binding, atomically write managed
      certificates, restrict generated keys on Unix and Windows, reject partial
      configuration, and fail clearly on invalid explicitly enabled HTTPS.
- [ ] Preserve posters on uncertain checks, validate URLs and image responses,
      avoid full image allocation, and cover persistence decisions.
- [ ] Add Linux/Entware CA discovery before HTTPS work; respect explicit roots.
- [ ] Adapt AAC profile handling and test the optional browser HLS path.
- [ ] Preserve macOS installer configuration and launch context.
- [ ] Add bounded transient engine reconnect retries within existing settings
      rollback and error reporting; retain prompt cancellation and readiness.
- [ ] Run the added packages in CI; preserve newer Flow toolchain/native pins.

## Streaming

- [ ] Extend bounded asynchronous native diagnostics with urgent block states,
      native priorities, duplicate requests and progress, with freshness and
      truncation. Wire backend, frontend contracts, translations and history.
- [ ] Identify and reproduce concrete high bitrate controller/cache defects;
      fix them with meaningful regression coverage and bounded resources.
- [ ] Integrate an opt-in Adaptive Streaming profile only with coherent backend,
      settings migration/export, UI, translated explanations and Legacy rollback.
      Preserve the full graded deadline ramp unless comparative evidence supports
      a different policy; keep experimental switches independent.
- [ ] Compare actual Flow builds at equal demand/startup constraints, exact Range
      bytes and seek/cancel behavior. Record results and limits, not a guaranteed
      public-swarm or phone speed claim.
- [ ] Build a Windows executable; run native/platform/frontend gates; review final
      diff and reconcile this checklist and original specification constraints.

Separate acceptance: representative public swarms, real Lampa/Just Player playback,
long sessions and physical network/VPN checks. No paid provider, full episode
preparation, router alteration or automatic profile switch is required.
