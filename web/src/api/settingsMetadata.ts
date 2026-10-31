import { z } from "zod";
import { api, ApiError } from "./client";

export const settingsMetadataSchema = z.object({
  schema_version: z.literal(1),
  fields: z.record(
    z.string().max(128),
    z.object({
      type: z.enum(["integer", "boolean", "string"]),
      default: z.union([z.number(), z.boolean(), z.string()]),
      min: z.number().int().optional(),
      max: z.number().int().optional(),
      choices: z
        .array(z.union([z.number(), z.string()]))
        .max(32)
        .optional(),
      unit: z.string().max(32).optional(),
      restart_required: z.boolean(),
      experimental: z.boolean(),
      deprecated: z.boolean(),
      read_only: z.boolean(),
      invalid_policy: z.enum(["normalize", "reject", "legacy"]).optional(),
      legacy_zero: z.boolean().optional(),
    }),
  ),
});
export type SettingsMetadata = z.infer<typeof settingsMetadataSchema>;
export async function getSettingsMetadata(signal?: AbortSignal) {
  try {
    return settingsMetadataSchema.parse(
      await api.get("/settings/metadata/v1", signal),
    );
  } catch (error) {
    // Only absence means an older server. Authentication, rate limiting and a
    // malformed modern contract remain visible instead of silently falling back.
    if (error instanceof ApiError && [404, 405].includes(error.status))
      return null;
    throw error;
  }
}
