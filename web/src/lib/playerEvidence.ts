import { z } from "zod";

const duration = z
  .number()
  .int()
  .min(0)
  .max(30 * 86400 * 1000);
const count = z.number().int().min(0).max(1_000_000_000);
const sampleSchema = z
  .object({
    sequence: count.positive().optional(),
    elapsed_ms: duration,
    recorded_at_ms: z.number().int().positive(),
    startup_ms: duration.nullable(),
    rebuffer_events: count,
    rebuffer_ms: duration,
    seek_events: count,
    seek_buffer_ms: duration,
    paused_ms: duration,
    play_ms: duration,
    buffered_ms: duration,
    dropped_frames: count,
    bandwidth_bps: z.number().int().nonnegative(),
    state: z.number().int().min(1).max(4),
    playing: z.boolean(),
    session_id: z
      .string()
      .regex(/^[A-Za-z0-9_-]{16,64}$/)
      .nullable(),
  })
  .strict();
export const playerReportSchema = z
  .object({
    schema_version: z.union([z.literal(1), z.literal(2)]),
    source: z.literal("flow-player-media3"),
    clock: z.literal("player-elapsed-and-wall; server-clock-alignment-unknown"),
    samples: z.array(sampleSchema).min(1).max(128),
    summary: sampleSchema
      .extend({
        total_samples: count.positive(),
        observed_min_buffered_ms: duration.nullable(),
      })
      .strict()
      .optional(),
  })
  .strict()
  .superRefine((report, ctx) => {
    let elapsed = -1,
      sequence = 0;
    for (const sample of report.samples) {
      if (
        sample.elapsed_ms < elapsed ||
        (sample.sequence !== undefined && sample.sequence <= sequence) ||
        (report.schema_version === 2 && sample.sequence === undefined)
      )
        ctx.addIssue({ code: "custom", message: "Invalid sample ordering" });
      elapsed = sample.elapsed_ms;
      sequence = sample.sequence ?? sequence;
    }
    if (
      report.schema_version === 2 &&
      (!report.summary ||
        report.summary.sequence !== sequence ||
        report.summary.elapsed_ms !== elapsed ||
        report.summary.total_samples !== sequence)
    )
      ctx.addIssue({ code: "custom", message: "Invalid session summary" });
  });
export type PlayerReport = z.infer<typeof playerReportSchema>;
export async function readPlayerReport(file: File): Promise<PlayerReport> {
  if (file.size > 262144) throw new Error("flow.playerReportInvalid");
  try {
    return playerReportSchema.parse(JSON.parse(await file.text()));
  } catch {
    throw new Error("flow.playerReportInvalid");
  }
}
