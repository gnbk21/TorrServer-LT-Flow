import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { FlowDiagnosticsDrawer } from "./FlowDiagnosticsDrawer";
import type { FlowStatusResponse } from "../../types/flow";
vi.mock("../../hooks/queries", () => ({ useFlowDiagnostics: () => ({}) }));
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
afterEach(cleanup);
describe("startup explanation", () => {
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
    expect(screen.getByText("—")).toBeInTheDocument();
    expect(screen.getByText("flow.startupTimingHint")).toBeInTheDocument();
  });
});
