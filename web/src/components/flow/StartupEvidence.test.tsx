import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StartupEvidence } from "./StartupEvidence";
import type { FlowStartup } from "../../types/flow";

describe("startup preparation evidence", () => {
  it("separates an unknown operation from a measured zero", () => {
    render(
      <StartupEvidence
        startup={
          {
            phases: [
              { phase: "CONNECT", basis: "OPERATION" },
              { phase: "PROBE", basis: "OPERATION", duration_ms: 0 },
            ],
          } as FlowStartup
        }
      />,
    );
    expect(screen.getByText(/flow.stream.unknown/)).toBeInTheDocument();
    expect(screen.getByText(/flow.stream.milliseconds/)).toBeInTheDocument();
    expect(screen.getByText("flow.stream.timingHint")).toBeInTheDocument();
  });
  it("shows metadata recovery before a preload progress snapshot exists", () => {
    render(
      <StartupEvidence
        startup={
          {
            metadata_recovery: {
              state: "METADATA_HTTP",
              source: "PUBLIC_CACHE",
              attempts: 2,
            },
          } as FlowStartup
        }
      />,
    );
    expect(
      screen.getByText("flow.metadataRecovery.METADATA_HTTP"),
    ).toBeInTheDocument();
    expect(screen.getByText("flow.metadataAttempts")).toBeInTheDocument();
    expect(screen.queryByText("flow.startupVerified")).not.toBeInTheDocument();
  });
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
  it("shows sampled zero supply, partial evidence and a progress extension", () => {
    render(
      <StartupEvidence
        startup={
          {
            progress: {
              head_bytes: 1,
              head_target_bytes: 1,
              head_verified_bytes: 1,
              tail_bytes: 0,
              tail_target_bytes: 1,
              tail_verified_bytes: 0,
              missing_head_pieces: 0,
              missing_tail_pieces: 1,
              blocked_on: "INDEX",
              wait_extended: true,
              wait_budget_ms: 150000,
              tail_supply: {
                suppliers: 0,
                unchoked_suppliers: 0,
                queue_known: true,
                requested_blocks: 0,
                unrequested_blocks: 1024,
                finished_blocks: 0,
                writing_blocks: 0,
                truncated: true,
              },
            },
          } as FlowStartup
        }
      />,
    );
    for (const key of [
      "startupSuppliers",
      "startupUnchoked",
      "startupBlocks",
      "startupSupplyTruncated",
      "startupWaitExtended",
    ]) {
      expect(screen.getByText(`flow.${key}`)).toBeInTheDocument();
    }
  });
});
