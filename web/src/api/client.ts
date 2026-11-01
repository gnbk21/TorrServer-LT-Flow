export class ApiError extends Error {
  status: number;
  statusText: string;
  data: unknown;
  retryAfterMs: number;

  constructor(
    status: number,
    statusText: string,
    data: unknown,
    retryAfterMs = 0,
  ) {
    super(
      typeof data === "string"
        ? data
        : data &&
            typeof data === "object" &&
            "error" in data &&
            typeof data.error === "string"
          ? data.error
          : statusText || `HTTP ${status}`,
    );
    this.name = "ApiError";
    this.status = status;
    this.statusText = statusText;
    this.data = data;
    this.retryAfterMs = retryAfterMs;
  }

  get isStarting() {
    return (
      this.status === 503 &&
      this.data !== null &&
      typeof this.data === "object" &&
      "code" in this.data &&
      this.data.code === "ENGINE_STARTING"
    );
  }

  get isUnauthorized() {
    return this.status === 401;
  }
}
let cooldownUntil = 0;
export function pollingDelay(base: number): number {
  return Math.max(base, cooldownUntil - performance.now());
}
export function resetManagementCooldown() {
  cooldownUntil = 0;
}
export function retryAfterMilliseconds(
  value: string | null,
  serverDate: string | null,
): number {
  if (!value) return 0;
  const seconds = Number(value);
  const milliseconds = Number.isFinite(seconds)
    ? seconds * 1000
    : Date.parse(value) - Date.parse(serverDate ?? "");
  return Number.isFinite(milliseconds)
    ? Math.min(300000, Math.max(0, milliseconds))
    : 0;
}

// Session token for Flow stream grouping
export const sessionToken = (() => {
  if (
    typeof window !== "undefined" &&
    window.crypto &&
    window.crypto.randomUUID
  ) {
    return window.crypto.randomUUID().replace(/-/g, "");
  }
  return Math.random().toString(36).slice(2, 12) + Date.now().toString(36);
})();

export async function apiClient<T>(
  endpoint: string,
  options: RequestInit & { timeoutMs?: number; manual?: boolean } = {},
): Promise<T> {
  const url = new URL(endpoint, window.location.origin);
  const { manual = false, timeoutMs = 30_000, ...requestOptions } = options;
  if (url.origin !== window.location.origin)
    throw new Error("Cross-origin API request rejected");
  const remaining = cooldownUntil - performance.now();
  if (remaining > 0 && !manual)
    throw new ApiError(
      429,
      "Retry later",
      { code: "POLLING_COOLDOWN" },
      remaining,
    );
  const headers = new Headers(options.headers || {});
  if (!(options.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const controller = new AbortController();
  const abort = () => controller.abort(options.signal?.reason);
  if (options.signal?.aborted) abort();
  else options.signal?.addEventListener("abort", abort, { once: true });
  const timeout = window.setTimeout(
    () =>
      controller.abort(new DOMException("Request timed out", "TimeoutError")),
    timeoutMs,
  );
  try {
    const response = await fetch(url, {
      ...requestOptions,
      headers,
      credentials: "same-origin",
      signal: controller.signal,
    });
    const retryAfterMs = [429, 503].includes(response.status)
      ? retryAfterMilliseconds(
          response.headers.get("Retry-After"),
          response.headers.get("Date"),
        )
      : 0;
    if (retryAfterMs > 0)
      cooldownUntil = Math.max(cooldownUntil, performance.now() + retryAfterMs);
    const text = await response.text();
    let data: unknown = text;
    if (
      text &&
      response.headers.get("content-type")?.includes("application/json")
    ) {
      try {
        data = JSON.parse(text);
      } catch {
        throw new ApiError(
          response.status,
          "Invalid JSON response",
          null,
          retryAfterMs,
        );
      }
    }
    if (!response.ok)
      throw new ApiError(
        response.status,
        response.statusText,
        data,
        retryAfterMs,
      );
    return data as T;
  } finally {
    window.clearTimeout(timeout);
    options.signal?.removeEventListener("abort", abort);
  }
}

export const api = {
  get: <T>(endpoint: string, signal?: AbortSignal, timeoutMs?: number) =>
    apiClient<T>(endpoint, { method: "GET", signal, timeoutMs }),

  post: <TRequest, TResponse>(
    endpoint: string,
    body?: TRequest,
    signal?: AbortSignal,
    manual = false,
  ) =>
    apiClient<TResponse>(endpoint, {
      method: "POST",
      body:
        body instanceof FormData
          ? body
          : body
            ? JSON.stringify(body)
            : undefined,
      signal,
      manual,
    }),
};
