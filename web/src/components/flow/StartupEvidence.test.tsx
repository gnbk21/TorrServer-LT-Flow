import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StartupEvidence } from "./StartupEvidence";
import type { FlowStartup } from "../../types/flow";

describe("startup preparation evidence", () => {
  it("keeps older servers compatible", () => {
    const { container } = render(
      <StartupEvidence startup={{} as FlowStartup} />,
    );
    expect(container).toBeEmptyDOMElement();
  });
  it("separates verified head/index bytes and retains blocker zero", () => {
    render(
      <StartupEvidence
        startup={
          {
            progress: {
              head_bytes: 1048576,
              head_target_bytes: 1048576,
              head_verified_bytes: 0,
              tail_bytes: 1048576,
              tail_target_bytes: 1048576,
              tail_verified_bytes: 1048576,
              missing_head_pieces: 1,
              missing_tail_pieces: 0,
              head_blocker_piece: 0,
              blocked_on: "HEAD",
            },
          } as FlowStartup
        }
      />,
    );
    expect(screen.getByText("flow.startupBlockers.HEAD")).toBeInTheDocument();
    expect(screen.getAllByText("flow.startupVerified")).toHaveLength(2);
    expect(screen.getAllByText("flow.startupBlockerPiece")).toHaveLength(1);
  });
});
