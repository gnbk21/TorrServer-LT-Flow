import { z } from "zod";
import { api } from "./client";

const proofSchema = z.object({
  id: z.string().regex(/^[A-Z2-7a-z0-9]{20,64}$/i),
  expires_at: z.string().datetime({ offset: true }),
  reached_at: z.string().datetime({ offset: true }).optional(),
  different_client: z.boolean(),
  evidence: z
    .array(
      z.object({
        at: z.string().datetime({ offset: true }),
        code: z.enum([
          "AUTH_REQUIRED",
          "AUTH_ACCEPTED",
          "MANAGEMENT_ORIGIN_DENIED",
          "MANAGEMENT_HOST_DENIED",
        ]),
        origin: z.string().max(512).optional(),
      }),
    )
    .max(128),
});
export const pairingApi = {
  create: async (signal?: AbortSignal) =>
    z
      .object({
        schema_version: z.literal(1),
        proof: proofSchema.extend({
          token: z.string().regex(/^[A-Z2-7a-z0-9]{20,64}$/i),
        }),
        startup: z.object({
          listeners_ready: z.boolean(),
          engine_ready: z.boolean(),
        }),
        advertised_addresses: z.array(z.string()).nullable(),
        listener_addresses: z.array(z.string()),
        auth_required: z.boolean(),
      })
      .parse(await api.post("/flow/pairing", {}, signal, true)),
  status: async (id: string, signal?: AbortSignal) =>
    proofSchema.parse(
      await api.get(`/flow/pairing/status/${encodeURIComponent(id)}`, signal),
    ),
};
