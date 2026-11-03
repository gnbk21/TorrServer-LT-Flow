import { expect, it, vi, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import { PieceCoverage } from "./PieceCoverage";
import type { PieceCoverageSnapshot } from "../../api/coverage";
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
afterEach(cleanup);
const sample: PieceCoverageSnapshot = {
  schema_version: 1,
  known: true,
  sampled_at_ms: 1000,
  availability_sampled_at_ms: 1000,
  truncated: false,
  piece_length: 1024,
  frontiers: [],
  pieces: [
    {
      piece: 7,
      resident: true,
      verified: false,
      suppliers: 0,
      queue_known: true,
      requested: 1,
      receiving_bytes: 32,
      writing: 0,
      finished: 1,
      required_head: true,
      required_index: false,
      blocker: true,
    },
  ],
};
it("does not call received or resident bytes verified", () => {
  const view = render(<PieceCoverage coverage={sample} now={1001} />);
  const button = screen.getByRole("button");
  expect(button).toHaveAttribute("data-state", "partial");
  fireEvent.click(button);
  expect(screen.getByText("0")).toBeVisible();
  view.rerender(<PieceCoverage coverage={sample} now={7000} />);
  expect(button).toHaveAttribute("data-state", "unknown");
  expect(screen.queryByText("0")).toBeNull();
});
