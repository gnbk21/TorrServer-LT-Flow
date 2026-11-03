import { afterEach, expect, test, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { ResourceStatus } from "./ResourceStatus";
import type { RuntimeStatus } from "../../types/runtime";
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
afterEach(cleanup);
const status = (sampled_at?: string) =>
  ({
    resources: {
      sampled_at,
      budget_bytes: 0,
      warm_budget_bytes: 0,
      ram_resident_bytes: 0,
      disk_cache_bytes: 0,
      memory_pressure: false,
      protected_overcommit: false,
      background_limited: false,
    },
  }) as RuntimeStatus;
test.each([undefined, "0001-01-01T00:00:00Z", "malformed"])(
  "unmeasured resources (%s) are unknown, not zero budgets",
  (timestamp) => {
    render(<ResourceStatus status={status(timestamp)} />);
    expect(
      screen.getByText("runtime.ramBudget").nextElementSibling,
    ).toHaveTextContent("status.unknown");
    expect(screen.queryByText("0 B")).not.toBeInTheDocument();
  },
);
test("a measured zero remains distinct from missing evidence", () => {
  render(<ResourceStatus status={status(new Date().toISOString())} />);
  expect(
    screen.getByText("runtime.warmBudget").nextElementSibling,
  ).toHaveTextContent("0 B");
});
