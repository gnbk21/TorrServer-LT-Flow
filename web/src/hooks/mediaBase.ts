import { useQuery } from "@tanstack/react-query";
import { z } from "zod";
import { api } from "../api/client";
import { safeMediaBase } from "../lib/mediaBase";
import { useVisibilityGate } from "./queries";

export function useExternalMediaBase(enabled = true) {
  const visible = useVisibilityGate(["external-media-base"]);
  return useQuery({
    queryKey: ["external-media-base"],
    queryFn: async ({ signal }) => {
      const value = z
        .object({ base: z.string() })
        .parse(await api.get("/mediabase", signal));
      return safeMediaBase(value.base);
    },
    enabled: enabled && visible,
    staleTime: 30_000,
    retry: false,
  });
}
