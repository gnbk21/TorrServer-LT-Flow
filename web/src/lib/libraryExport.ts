import { z } from "zod";
import { api, ApiError } from "../api/client";
import { torrentsApi } from "../api/torrents";
import type { Torrent } from "../types/torrent";
import { redactDiagnostic } from "./redact";
import { torrentMagnet } from "./torrentLink";

const exportSchema = z
  .array(
    z.object({
      hash: z.string().regex(/^(?:[a-f0-9]{40}|[a-f0-9]{64})$/i),
      title: z.string().max(4096),
      category: z.string().max(512),
      torrs_hash: z
        .string()
        .regex(/^[A-Za-z0-9]{1,24576}$/)
        .optional(),
    }),
  )
  .max(5000);
type ExportRow = z.infer<typeof exportSchema>[number];
export type LibraryExportFormat = "json" | "magnets" | "torrs";

// Legacy list DTOs retain unknown fields. Never spread them or reuse an opaque
// token: both may contain private tracker credentials or custom client data.
export function portableLibraryRows(rows: Torrent[]): ExportRow[] {
  return exportSchema.parse(
    rows.map((row) => ({
      hash: row.hash.toLowerCase(),
      title: redactDiagnostic(row.title),
      category: redactDiagnostic(row.category ?? ""),
    })),
  );
}

export async function loadLibraryExport(): Promise<ExportRow[]> {
  try {
    return exportSchema.parse(await api.get<unknown>("/flow/library/export"));
  } catch (error) {
    // Preserve older-server compatibility without treating auth/offline errors
    // as permission to read the legacy endpoint.
    if (!(error instanceof ApiError) || error.status !== 404) throw error;
    return portableLibraryRows(await torrentsApi.list());
  }
}

export function formatLibraryExport(
  rows: ExportRow[],
  format: LibraryExportFormat,
): string {
  const body =
    format === "json"
      ? JSON.stringify(rows, null, 2)
      : rows
          .map((row) =>
            format === "torrs" && row.torrs_hash && row.hash.length === 40
              ? `torrs://${row.torrs_hash}`
              : torrentMagnet(row.hash, row.title),
          )
          .join("\n");
  if (new TextEncoder().encode(body).byteLength > 16 * 1024 * 1024)
    throw new Error("torrent.exportTooLarge");
  return body;
}
