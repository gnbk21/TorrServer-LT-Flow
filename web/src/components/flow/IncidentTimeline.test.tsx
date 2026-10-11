import { afterEach, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { IncidentTimeline } from "./IncidentTimeline";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    i18n: { language: "en" },
    t: (key: string) => key,
  }),
}));
afterEach(cleanup);

it("labels cache evidence separately and preserves measured zero and unknown", () => {
  render(
    <IncidentTimeline
      status={{
        hash: "fixture",
        timeline: {
          source: "server",
          dropped: 2,
          events: [
            {
              type: "storage",
              stage: "CACHE_WRITE_FAILED",
              time: "2026-10-10T00:00:01Z",
              elapsed_ms: 1000,
              operation_ms: 0,
            },
            {
              type: "metadata",
              time: "2026-10-10T00:00:02Z",
              elapsed_ms: 2000,
              operation_ms: null,
            },
          ],
        },
      }}
    />,
  );
  expect(
    screen.getByText("flow.storageStages.CACHE_WRITE_FAILED"),
  ).toBeInTheDocument();
  expect(screen.getByText("flow.storageTimelineHint")).toBeInTheDocument();
  expect(screen.getByText("0 ms")).toBeInTheDocument();
  expect(screen.getAllByText(/ ms$/)).toHaveLength(1);
  expect(screen.getByText("flow.timelineTruncated")).toBeInTheDocument();
});
