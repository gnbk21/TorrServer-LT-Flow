import { z } from "zod";
export const coverageSchema = z.object({
  schema_version: z.literal(1),
  known: z.boolean(),
  sampled_at_ms: z.number().nonnegative(),
  availability_sampled_at_ms: z.number().nonnegative(),
  truncated: z.boolean(),
  piece_length: z.number().nonnegative(),
  pieces: z
    .array(
      z.object({
        piece: z.number().int().nonnegative(),
        resident: z.boolean(),
        verified: z.boolean(),
        suppliers: z.number().int().min(0).max(512).optional(),
        queue_known: z.boolean(),
        requested: z.number().int().nonnegative(),
        receiving_bytes: z.number().nonnegative(),
        writing: z.number().int().nonnegative(),
        finished: z.number().int().nonnegative(),
        required_head: z.boolean(),
        required_index: z.boolean(),
        blocker: z.boolean(),
      }),
    )
    .max(256),
  frontiers: z
    .array(
      z.object({
        file_index: z.number().int().positive(),
        piece: z.number().int().nonnegative(),
        readable_bytes: z.number().nonnegative(),
        verified_bytes: z.number().nonnegative(),
        sampled_at_ms: z.number().nonnegative(),
      }),
    )
    .max(8),
});
export type PieceCoverageSnapshot = z.infer<typeof coverageSchema>;
