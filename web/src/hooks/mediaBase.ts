import { useQuery } from "@tanstack/react-query";
import { z } from "zod";
import { api } from "../api/client";
import { safeMediaBase } from "../lib/mediaBase";

export function useExternalMediaBase(enabled = true) {
  return useQuery({
    queryKey: ["external-media-base"],
    queryFn: async ({ signal }) => {
      const value = z
        .object({ base: z.string() })
        .parse(await api.get("/mediabase", signal));
      return safeMediaBase(value.base);
    },
    enabled,
    staleTime: 30_000,
    retry: false,
  });
}
