export interface TransferResult {
  bytes: number;
  mbps: number;
  ttfbMs: number;
  maxReadGapMs: number;
  p95ReadGapMs: number;
}

// Browser read gaps include browser scheduling/coalescing. They are not packet
// loss or player rebuffer events. Keep each run bounded and cancellable.
export async function measureLAN(signal: AbortSignal): Promise<TransferResult> {
  const start = performance.now();
  const response = await fetch("/flow/lan-test?mib=16", {
    credentials: "same-origin",
    cache: "no-store",
    signal,
  });
  if (!response.ok || !response.body)
    throw new Error(`Transfer test: HTTP ${response.status}`);
  const reader = response.body.getReader();
  const gaps: number[] = [];
  let bytes = 0;
  let first: number | undefined;
  try {
    for (;;) {
      const before = performance.now();
      const part = await reader.read();
      const at = performance.now();
      if (part.done) break;
      if (first === undefined) first = at;
      else if (gaps.length < 32768) gaps.push(at - before);
      bytes += part.value.byteLength;
      if (bytes > 16 * 1048576)
        throw new Error("Transfer exceeded its byte budget");
    }
  } finally {
    await reader.cancel().catch(() => {});
    reader.releaseLock();
  }
  if (bytes !== 16 * 1048576 || first === undefined)
    throw new Error("Transfer was truncated");
  const elapsed = performance.now() - start;
  gaps.sort((a, b) => a - b);
  return {
    bytes,
    mbps: (bytes * 8) / Math.max(1, elapsed) / 1000,
    ttfbMs: first - start,
    maxReadGapMs: gaps.at(-1) ?? 0,
    p95ReadGapMs:
      gaps[Math.min(gaps.length - 1, Math.floor(gaps.length * 0.95))] ?? 0,
  };
}
