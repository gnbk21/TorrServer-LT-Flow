import { apiClient } from "./client";
import { z } from "zod";

export const certificateStatusSchema = z.object({
  enabled: z.boolean(),
  port: z.string().optional(),
  http_port: z.string(),
  http_enabled: z.boolean(),
  force_https: z.boolean(),
  http_media: z.boolean(),
  read_only: z.boolean(),
  cert_from_flags: z.boolean(),
  revision: z.string(),
  cert: z.object({
    source: z.enum(["none", "self-signed", "uploaded", "user"]),
    cert_file: z.string().optional(),
    key_file: z.string().optional(),
    subject: z.string().optional(),
    issuer: z.string().optional(),
    dns_names: z.array(z.string()).nullable().optional(),
    ips: z.array(z.string()).nullable().optional(),
    not_before: z.string().optional(),
    not_after: z.string().optional(),
    trusted: z.boolean(),
    error: z.string().optional(),
  }),
});
export type CertificateStatus = z.infer<typeof certificateStatusSchema>;
export const certificatesApi = {
  status: (signal?: AbortSignal) =>
    apiClient("/ssl/status", { signal }).then((data) =>
      certificateStatusSchema.parse(data),
    ),
  change: (
    action: "paths" | "upload" | "selfsigned" | "regenerate",
    revision: string,
    body?: FormData | { cert: string; key: string },
    signal?: AbortSignal,
  ) =>
    apiClient(`/ssl/${action}`, {
      method: "POST",
      headers: { "If-Match": revision },
      body:
        body instanceof FormData
          ? body
          : body
            ? JSON.stringify(body)
            : undefined,
      signal,
    }).then((data) => certificateStatusSchema.parse(data)),
};
