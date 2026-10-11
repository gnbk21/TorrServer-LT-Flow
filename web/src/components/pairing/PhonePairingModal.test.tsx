import { afterEach, expect, test, vi } from "vitest";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { PhonePairingModal } from "./PhonePairingModal";

const mocks = vi.hoisted(() => ({
  status: vi.fn(),
  qr: vi.fn(),
  base: "http://192.168.1.2:8090",
}));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("../common/Modal", () => ({
  Modal: ({
    isOpen,
    children,
  }: {
    isOpen: boolean;
    children: React.ReactNode;
  }) => (isOpen ? <div>{children}</div> : null),
}));
vi.mock("../../hooks/mediaBase", () => ({
  useExternalMediaBase: () => ({ data: mocks.base }),
}));
vi.mock("../../api/certificates", () => ({
  certificatesApi: { status: mocks.status },
}));
vi.mock("../../lib/qr", () => ({ generateQrSvg: mocks.qr }));

afterEach(() => {
  cleanup();
  vi.resetAllMocks();
  mocks.base = "http://192.168.1.2:8090";
});
function view(open: boolean, client: QueryClient) {
  return (
    <QueryClientProvider client={client}>
      <PhonePairingModal
        isOpen={open}
        onClose={() => {}}
        serverIps={["192.168.1.2"]}
      />
    </QueryClientProvider>
  );
}
test("HTTP pairing identifies plaintext and does not fetch irrelevant TLS status", async () => {
  mocks.qr.mockResolvedValue("<svg></svg>");
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  render(view(true, client));
  await screen.findByText("pairing.httpPlaintext");
  expect(screen.getByText("pairing.reachability")).toBeVisible();
  expect(mocks.status).not.toHaveBeenCalled();
  await waitFor(() => expect(mocks.qr).toHaveBeenCalledOnce());
});
test("server certificate trust is explicitly separate from phone trust", async () => {
  mocks.base = "https://192.168.1.2:8091";
  mocks.qr.mockResolvedValue("<svg></svg>");
  mocks.status.mockResolvedValue({ enabled: true, cert: { trusted: true } });
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  render(view(true, client));
  await screen.findByText("pairing.serverTrustOnly");
  expect(screen.getByText("pairing.httpsTrust")).toBeVisible();
  expect(screen.queryByText("pairing.httpPlaintext")).toBeNull();
});
test("closing pairing cancels its certificate request and stops QR work", async () => {
  mocks.base = "https://192.168.1.2:8091";
  mocks.qr.mockResolvedValue("<svg></svg>");
  let signal: AbortSignal | undefined;
  mocks.status.mockImplementation((current: AbortSignal) => {
    signal = current;
    return new Promise(() => {});
  });
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const page = render(view(true, client));
  await waitFor(() => expect(signal).toBeDefined());
  await act(async () => page.rerender(view(false, client)));
  expect(signal?.aborted).toBe(true);
  expect(mocks.qr).toHaveBeenCalledOnce();
  expect(screen.queryByText("pairing.httpsTrust")).toBeNull();
});
