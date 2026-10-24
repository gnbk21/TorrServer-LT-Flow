import { afterEach, describe, it, expect, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { FlowHealthPanel } from "./FlowHealthPanel";
import type { FlowSession } from "../../types/flow";
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
afterEach(cleanup);
const session = {
  state: "PLAYING",
  buffer_sampled_at_ms: Date.now(),
  playback_consumption_rate: 0,
  buffer_ahead_seconds: 90,
  buffer_ahead_bytes: 1000,
  download_rate: 0,
  sustainability_ratio: 0,
  connected_peers: 1,
  piece_wait_p95_ms: 0,
  seek_count: 0,
  buffer_warning: false,
} as FlowSession;
describe("Flow health presentation", () => {
  it("treats missing or stale measurement timestamps as unknown", () => {
    const view = render(
      <FlowHealthPanel
        session={{
          ...session,
          buffer_sampled_at_ms: undefined,
          playback_consumption_rate: 200,
        }}
      />,
    );
    expect(screen.getByText("flow.health.UNKNOWN")).toBeInTheDocument();
    view.rerender(
      <FlowHealthPanel
        session={{
          ...session,
          buffer_sampled_at_ms: Date.now() - 11000,
          playback_consumption_rate: 200,
        }}
      />,
    );
    expect(screen.getByText("flow.health.UNKNOWN")).toBeInTheDocument();
    expect(screen.queryByText("90.0 s")).not.toBeInTheDocument();
  });

  it("does not convert an unknown demand into healthy or stalled playback", () => {
    render(<FlowHealthPanel session={session} />);
    expect(screen.getByText("flow.health.UNKNOWN")).toBeInTheDocument();
    expect(screen.queryByText("90.0 s")).not.toBeInTheDocument();
  });
  it("retains warm idle state with zero network throughput", () => {
    render(<FlowHealthPanel session={{ ...session, state: "WARM_IDLE" }} />);
    expect(screen.getByText("flow.health.WARM_IDLE")).toBeInTheDocument();
  });
  it("honors an authoritative buffer warning", () => {
    render(
      <FlowHealthPanel
        session={{
          ...session,
          playback_consumption_rate: 200,
          buffer_warning: true,
        }}
      />,
    );
    expect(screen.getByText("flow.health.BUFFER_RISK")).toBeInTheDocument();
  });
  it.each([
    ["HIGH", "BUFFER_RISK"],
    ["ELEVATED", "MARGINAL"],
  ])("keeps the health heading consistent with %s risk", (level, health) => {
    render(
      <FlowHealthPanel
        session={{
          ...session,
          playback_consumption_rate: 200,
          sustainability_ratio: 2,
          risk: {
            level,
            reason: "PIECE_WAIT",
            score: 65,
            confidence: "medium",
            target_seconds: 120,
          },
        }}
      />,
    );
    expect(screen.getByText(`flow.health.${health}`)).toBeInTheDocument();
  });
});
