import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { BufferingEvidence } from "./BufferingEvidence";
import type { FlowSession } from "../../types/flow";

afterEach(cleanup);

const session = {
  active_readers: 1,
  supply: {
    known: true,
    state: "SUPPLY_DEFICIT",
    demand_rate: 12.5e6,
    delivery_rate: 10e6,
    deficit_rate: 2.5e6,
  },
  blocker: {
    known: true,
    piece: 0,
    piece_length: 16 * 1048576,
    blocks: 1024,
    finished: 1022,
    requested: 2,
    unrequested: 0,
    writing: 0,
    oldest_request_age_ms: -1,
    suppliers: 0,
    reason: "WAIT_BLOCKS",
  },
} as FlowSession;

describe("buffering evidence", () => {
  it("explains a deficit independently of media name or bitrate", () => {
    render(<BufferingEvidence session={session} fresh />);
    expect(
      screen.getByText("flow.buffering.states.SUPPLY_DEFICIT"),
    ).toBeInTheDocument();
    expect(screen.getByText("flow.buffering.blocker")).toBeInTheDocument();
    expect(screen.getByText("flow.buffering.hint")).toBeInTheDocument();
  });
  it("does not show stale blocker and supply as live", () => {
    render(<BufferingEvidence session={session} fresh={false} />);
    expect(screen.getByText("flow.sparseUnknown")).toBeInTheDocument();
    expect(
      screen.queryByText("flow.buffering.blocker"),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("flow.buffering.states.SUPPLY_DEFICIT"),
    ).not.toBeInTheDocument();
  });
  it("keeps older servers and idle streams compatible", () => {
    const { container, rerender } = render(<BufferingEvidence fresh />);
    expect(container).toBeEmptyDOMElement();
    rerender(
      <BufferingEvidence session={{ ...session, active_readers: 0 }} fresh />,
    );
    expect(container).toBeEmptyDOMElement();
  });
  it("does not confuse finished blocks with a verified piece", () => {
    render(
      <BufferingEvidence
        session={{
          ...session,
          blocker: { ...session.blocker!, reason: "VERIFYING" },
        }}
        fresh
      />,
    );
    expect(screen.getByText("flow.buffering.verifying")).toBeInTheDocument();
  });
});
