import { render, screen, cleanup } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { ApiError } from "../../api/client";
import { RequestError } from "./RequestState";
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { exists: () => false },
  }),
}));
afterEach(cleanup);
test("503 is starting only with an explicit engine code; 401/403/429 differ", () => {
  const states = [
    [
      new ApiError(503, "Maintenance", { code: "MAINTENANCE_BUSY" }),
      "status.requestFailed",
    ],
    [
      new ApiError(503, "Starting", { code: "ENGINE_STARTING" }),
      "status.starting",
    ],
    [new ApiError(401, "Unauthorized", {}), "status.authRequired"],
    [new ApiError(403, "Forbidden", {}), "reliability.forbidden"],
    [new ApiError(429, "Slow down", {}, 2000), "reliability.cooldown"],
  ] as const;
  for (const [error, message] of states) {
    render(<RequestError error={error} />);
    expect(screen.getByRole("alert").textContent?.toLowerCase()).toContain(
      message.toLowerCase(),
    );
    cleanup();
  }
});
