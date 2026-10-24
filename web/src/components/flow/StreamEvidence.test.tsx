import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StreamEvidence } from "./StreamEvidence";
import type { StreamEvidence as Evidence } from "../../types/flow";

const evidence: Evidence = {
  known: true,
  first_dial_ms: 0,
  first_connect_ms: 10,
  first_handshake_ms: 25,
  first_payload_ms: 100,
  connect_duration_ms: 10,
  handshake_duration_ms: 15,
  useful_bytes: 1048576,
  redundant_bytes: 16384,
  obsolete_bytes: 0,
  recovery_requests: 1,
  recovery_suppressed: 0,
  active_hedges: 0,
  peak_hedges: 1,
  active_hedge_bytes: 0,
  budget_changes: 2,
  frontiers: 1,
  recovery_scan_truncated: false,
  service: [
    {
      transport: "tcp",
      address_family: "IPV4",
      samples: 100,
      p50_ms: 100,
      p95_ms: 250,
    },
  ],
};

describe("native stream evidence", () => {
  it("keeps older servers compatible", () => {
    const { container } = render(<StreamEvidence fresh />);
    expect(container).toBeEmptyDOMElement();
  });
  it("does not present stale counters as current", () => {
    render(<StreamEvidence evidence={evidence} fresh={false} />);
    expect(screen.getByText("flow.sparseUnknown")).toBeInTheDocument();
    expect(
      screen.queryByText("flow.stream.recovery_requests"),
    ).not.toBeInTheDocument();
  });
  it("labels service measurements separately from network RTT", () => {
    render(<StreamEvidence evidence={evidence} fresh />);
    expect(screen.getByText(/TCP \/ IPV4/)).toBeInTheDocument();
    expect(screen.getByText("flow.stream.hint")).toBeInTheDocument();
  });
});
