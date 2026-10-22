import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, render, screen } from "@testing-library/react";
import { FlowDiagnosticsDrawer } from "./FlowDiagnosticsDrawer";
import type { FlowStatusResponse } from "../../types/flow";
vi.mock("../../hooks/queries", () => ({
  useFlowDiagnostics: () => ({}),
  useVisible: () => true,
}));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: { defaultValue?: string }) =>
      key === "flow.startupStages.HEAD_INDEX"
        ? "Loading header and index"
        : key.startsWith("flow.startupStages.")
          ? (options?.defaultValue ?? key)
          : key,
  }),
}));
afterEach(() => {
  cleanup();
  vi.useRealTimers();
});
describe("startup explanation", () => {
  it("expires supplier evidence and advice without another server response", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-10-10T00:00:00Z"));
    render(
      <FlowDiagnosticsDrawer
        isOpen
        onClose={() => {}}
        status={
          {
            hash: "fixture",
            sampled_at: new Date().toISOString(),
            sparse: {
              known: true,
              sampled_at_ms: Date.now(),
              sampled_peers: 1,
            },
            sessions: [
              {
                group: "fixture",
                file_index: 1,
                wait_explanation: {
                  reason: "MISSING_CONNECTED",
                  confidence: "high",
                  actions: ["WAIT_SUPPLIERS"],
                },
              },
            ],
          } as FlowStatusResponse
        }
      />,
    );
    expect(screen.getByText("flow.sparsePeers")).toBeInTheDocument();
    expect(
      screen.getByText("flow.waitActions.WAIT_SUPPLIERS"),
    ).toBeInTheDocument();
    act(() => vi.advanceTimersByTime(5001));
    expect(screen.queryByText("flow.sparsePeers")).not.toBeInTheDocument();
    expect(
      screen.getByText("flow.waitActions.WAIT_SUPPLIERS"),
    ).toBeInTheDocument();
    act(() => vi.advanceTimersByTime(5000));
    expect(
      screen.queryByText("flow.waitActions.WAIT_SUPPLIERS"),
    ).not.toBeInTheDocument();
    expect(screen.getAllByText("flow.sparseUnknown")).toHaveLength(2);
  });
  it("shows bounded urgent progress only for fresh snapshots", () => {
    const urgent = {
      piece: 1,
      priority: 7,
      blocks: 256,
      unrequested: 100,
      requested: 120,
      writing: 6,
      finished: 30,
      duplicate_requests: 0,
      verified: false,
      receiving_blocks: 1,
      receiving_bytes: 8192,
      oldest_request_age_ms: 123,
    };
    const { rerender } = render(
      <FlowDiagnosticsDrawer
        isOpen
        onClose={() => {}}
        status={{
          hash: "fixture",
          sparse: { known: true, sampled_at_ms: Date.now(), urgent: [urgent] },
        }}
      />,
    );
    expect(screen.getByText(/flow.urgentProgress/)).toBeInTheDocument();
    expect(screen.getByText(/flow.urgentAge/)).toBeInTheDocument();
    rerender(
      <FlowDiagnosticsDrawer
        isOpen
        onClose={() => {}}
        status={{
          hash: "fixture",
          sparse: { known: true, sampled_at_ms: 1, urgent: [urgent] },
        }}
      />,
    );
    expect(screen.queryByText(/flow.urgentProgress/)).not.toBeInTheDocument();
    expect(screen.queryByText(/flow.urgentAge/)).not.toBeInTheDocument();
  });
  it("labels stale availability instead of displaying it as current", () => {
    render(
      <FlowDiagnosticsDrawer
        isOpen
        onClose={() => {}}
        status={{
          hash: "fixture",
          sparse: { known: true, sampled_at_ms: 1, sampled_peers: 0 },
        }}
      />,
    );
    expect(screen.getByText("flow.sparseUnknown")).toBeInTheDocument();
    expect(screen.queryByText("flow.sparsePeers")).not.toBeInTheDocument();
  });
  it("shows a bounded fresh sample and warns when peers were omitted", () => {
    render(
      <FlowDiagnosticsDrawer
        isOpen
        onClose={() => {}}
        status={{
          hash: "fixture",
          sparse: {
            known: true,
            sampled_at_ms: Date.now(),
            sampled_peers: 512,
            truncated: true,
          },
        }}
      />,
    );
    expect(screen.getByText("flow.sparseTruncated")).toBeInTheDocument();
    expect(screen.getByText("flow.sparseHint")).toBeInTheDocument();
    expect(screen.queryByText("flow.sparseUnknown")).not.toBeInTheDocument();
  });
  it("distinguishes unobserved timings from a measured zero and explains the wait", () => {
    const status = {
      hash: "fixture",
      startup: {
        state: "BOOTSTRAP_PRELOAD",
        file_index: 1,
        wait_reason: "HEAD_INDEX",
        first_useful_block_ms: -1,
        first_peer_ms: 0,
      },
    } as FlowStatusResponse;
    render(<FlowDiagnosticsDrawer isOpen onClose={() => {}} status={status} />);
    expect(
      screen.getAllByText("Loading header and index").length,
    ).toBeGreaterThan(0);
    expect(screen.getByText("0 ms")).toBeInTheDocument();
    expect(screen.getAllByText("—").length).toBeGreaterThan(0);
    expect(screen.getByText("flow.startupTimingHint")).toBeInTheDocument();
  });

  it("shows localized wait actions only while the server observation is fresh", () => {
    const status = {
      hash: "fixture",
      sampled_at: new Date().toISOString(),
      sessions: [
        {
          group: "fixture",
          file_index: 1,
          wait_explanation: {
            reason: "MISSING_CONNECTED",
            confidence: "high",
            actions: ["WAIT_SUPPLIERS", "CHECK_DISCOVERY"],
          },
        },
      ],
    } as FlowStatusResponse;
    const { rerender } = render(
      <FlowDiagnosticsDrawer isOpen onClose={() => {}} status={status} />,
    );
    expect(
      screen.getByText("flow.waitActions.WAIT_SUPPLIERS"),
    ).toBeInTheDocument();
    expect(screen.getByText("flow.waitConfidence")).toBeInTheDocument();
    rerender(
      <FlowDiagnosticsDrawer
        isOpen
        onClose={() => {}}
        status={{ ...status, sampled_at: "2000-01-01T00:00:00Z" }}
      />,
    );
    expect(
      screen.queryByText("flow.waitActions.WAIT_SUPPLIERS"),
    ).not.toBeInTheDocument();
    expect(screen.getByText("flow.sparseUnknown")).toBeInTheDocument();
  });
});
