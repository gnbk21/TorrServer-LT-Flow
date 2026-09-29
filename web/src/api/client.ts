export class ApiError extends Error {
  status: number;
  statusText: string;
  data: unknown;

  constructor(status: number, statusText: string, data: unknown) {
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
  }

  get isStarting() {
    return this.status === 503;
  }

  get isUnauthorized() {
    return this.status === 401 || this.status === 403;
  }
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
  options: RequestInit & { timeoutMs?: number } = {},
): Promise<T> {
  const url = new URL(endpoint, window.location.origin);
  if (url.origin !== window.location.origin)
    throw new Error("Cross-origin API request rejected");
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
    options.timeoutMs ?? 30_000,
  );
  try {
    const response = await fetch(url, {
      ...options,
      headers,
      credentials: "same-origin",
      signal: controller.signal,
    });
    const text = await response.text();
    let data: unknown = text;
    if (
      text &&
      response.headers.get("content-type")?.includes("application/json")
    ) {
      try {
        data = JSON.parse(text);
      } catch {
        throw new ApiError(response.status, "Invalid JSON response", null);
      }
    }
    if (!response.ok)
      throw new ApiError(response.status, response.statusText, data);
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
    }),
};
