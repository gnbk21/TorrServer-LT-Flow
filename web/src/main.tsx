import {
  lazy,
  Suspense,
  useState,
  useEffect,
  Component,
  type ReactNode,
} from "react";
import { createRoot } from "react-dom/client";
import { QueryClientProvider, useQuery } from "@tanstack/react-query";
import {
  createHashRouter,
  RouterProvider,
  Outlet,
  NavLink,
  useBlocker,
} from "react-router-dom";
import { useTranslation } from "react-i18next";
import {
  Activity,
  Library,
  Search,
  Settings as SettingsIcon,
  Smartphone,
} from "lucide-react";
import i18n from "./i18n";
import "./index.css";
import {
  queryClient,
  useVersion,
  useNetwork,
  useVisible,
} from "./hooks/queries";
import { DirtyProvider, useDirty } from "./hooks/dirty";
import { flowApi } from "./api/flow";
import { ApiError } from "./api/client";
import { Button } from "./components/common/Button";
import { Modal } from "./components/common/Modal";
import { Appearance } from "./components/common/Appearance";
import { ObservedPlayback } from "./components/flow/ObservedPlayback";
import { Loading, RequestError } from "./components/common/RequestState";
const Dashboard = lazy(() => import("./pages/Dashboard"));
const Torrents = lazy(() => import("./pages/Torrents"));
const Add = lazy(() => import("./pages/Add"));
const Settings = lazy(() => import("./pages/Settings"));
const Pairing = lazy(() =>
  import("./components/pairing/PhonePairingModal").then((m) => ({
    default: m.PhonePairingModal,
  })),
);
function Shell() {
  const { t, i18n } = useTranslation();
  const version = useVersion();
  const network = useNetwork();
  const visible = useVisible();
  useEffect(() => {
    if (!visible)
      void queryClient.cancelQueries({
        predicate: (query) =>
          [
            "active",
            "library",
            "flow",
            "flow-diagnostics",
            "cache",
            "torrent",
            "runtime",
            "network",
            "tray",
          ].includes(String(query.queryKey[0])),
      });
  }, [visible]);
  const tray = useQuery({
    queryKey: ["tray"],
    queryFn: ({ signal }) => flowApi.getTray(signal),
    refetchInterval: visible ? 5000 : false,
  });
  const [pairing, setPairing] = useState(false);
  const [controlling, setControlling] = useState(false);
  const [controlError, setControlError] = useState<unknown>();
  const paused = tray.data?.server_state === "PAUSED";
  const control = async () => {
    setControlling(true);
    setControlError(undefined);
    try {
      const result = await flowApi.control(paused ? "resume" : "pause");
      queryClient.setQueryData(["tray"], result);
      await queryClient.invalidateQueries({ queryKey: ["runtime"] });
    } catch (error) {
      setControlError(error);
    } finally {
      setControlling(false);
    }
  };
  const { dirty, setDirty } = useDirty();
  const blocker = useBlocker(dirty);
  useEffect(() => {
    document.documentElement.lang =
      i18n.resolvedLanguage === "ua" ? "uk" : i18n.resolvedLanguage || "en";
  }, [i18n.resolvedLanguage]);
  const error = tray.error || network.error || version.error;
  const status =
    (error instanceof ApiError && error.isStarting) ||
    tray.data?.server_state === "STARTING"
      ? "starting"
      : error instanceof ApiError && error.isUnauthorized
        ? "authRequired"
        : error
          ? "offline"
          : !tray.data
            ? "starting"
            : paused
              ? "paused"
              : network.data?.connectivity === "ONLINE"
                ? "online"
                : network.data?.connectivity === "DEGRADED"
                  ? "degraded"
                  : "localOnly";
  const links = [
    { to: "/", key: "dashboard", Icon: Activity },
    { to: "/torrents", key: "torrents", Icon: Library },
    { to: "/add", key: "add", Icon: Search },
    { to: "/settings", key: "settings", Icon: SettingsIcon },
  ];
  return (
    <div className="min-h-dvh">
      <a
        href="#main"
        className="sr-only focus:not-sr-only"
        onClick={(event) => {
          event.preventDefault();
          document.getElementById("main")?.focus();
        }}
      >
        {t("nav.skip")}
      </a>
      <aside className="hidden lg:flex fixed left-0 inset-y-0 w-60 p-5 bg-slate-900 border-r border-slate-800 flex-col gap-8">
        <div>
          <strong className="text-xl">
            TorrServer<span className="text-blue-400"> Flow</span>
          </strong>
          <p className="text-xs text-slate-400 break-all mt-2">
            {version.data || "—"}
          </p>
        </div>
        <nav className="space-y-2">
          {links.map(({ to, key, Icon }) => (
            <NavLink
              end={to === "/"}
              key={to}
              to={to}
              className={({ isActive }) =>
                `flex items-center gap-3 p-3 rounded-xl ${isActive ? "bg-blue-600" : "hover:bg-slate-800"}`
              }
            >
              <Icon size={20} />
              {t(`nav.${key}`)}
            </NavLink>
          ))}
        </nav>
      </aside>
      <div className="lg:ml-60">
        <header className="sticky top-0 z-20 bg-slate-950/95 border-b border-slate-800 p-3 flex flex-wrap gap-3 items-center">
          <strong className="mr-auto text-sm lg:text-base">
            {t(`status.${status}`)}
          </strong>
          <label className="sr-only" htmlFor="language">
            {t("Language")}
          </label>
          <select
            id="language"
            className="max-w-32 text-sm"
            value={i18n.resolvedLanguage || "en"}
            onChange={(e) => void i18n.changeLanguage(e.target.value)}
          >
            {Object.entries({
              en: "English",
              ru: "Русский",
              ua: "Українська",
              bg: "Български",
              fr: "Français",
              ro: "Română",
              zh: "中文",
            }).map(([code, label]) => (
              <option key={code} value={code}>
                {label}
              </option>
            ))}
          </select>
          <Button
            onClick={() => setPairing(true)}
            icon={<Smartphone size={18} />}
          >
            {t("pairing.connect")}
          </Button>
          <Appearance />
          <Button
            disabled={controlling || !tray.data || !!tray.error}
            onClick={() => void control()}
          >
            {t(paused ? "flow.resume" : "flow.pause")}
          </Button>
        </header>
        <ObservedPlayback />
        <main
          id="main"
          tabIndex={-1}
          className="max-w-7xl mx-auto p-4 sm:p-6 pb-28 lg:pb-8"
        >
          {!!controlError && (
            <RequestError error={controlError} retry={() => void control()} />
          )}
          <Suspense fallback={<Loading />}>
            <Outlet />
          </Suspense>
          <footer className="mt-8 text-xs text-slate-400">
            {t("app.install")}
          </footer>
        </main>
      </div>
      <nav
        aria-label={t("nav.main")}
        className="lg:hidden fixed bottom-0 inset-x-0 z-30 grid grid-cols-4 border-t border-slate-700 bg-slate-900 pb-[env(safe-area-inset-bottom)]"
      >
        {links.map(({ to, key, Icon }) => (
          <NavLink
            end={to === "/"}
            key={to}
            to={to}
            className={({ isActive }) =>
              `flex flex-col items-center justify-center min-h-16 p-1 text-xs gap-1 ${isActive ? "text-blue-300 bg-slate-800" : "text-slate-300"}`
            }
          >
            <Icon size={20} />
            {t(`nav.${key}`)}
          </NavLink>
        ))}
      </nav>
      {pairing && (
        <Suspense fallback={<Loading />}>
          <Pairing
            isOpen
            onClose={() => setPairing(false)}
            serverIps={network.data?.addresses ?? undefined}
          />
        </Suspense>
      )}
      <Modal
        isOpen={blocker.state === "blocked"}
        onClose={() => blocker.reset?.()}
        title={t("settings.unsaved")}
      >
        <p>{t("settings.unsavedHint")}</p>
        <div className="actions mt-4">
          <Button onClick={() => blocker.reset?.()}>
            {t("settings.keepEditing")}
          </Button>
          <Button
            variant="danger"
            onClick={() => {
              setDirty(false);
              blocker.proceed?.();
            }}
          >
            {t("settings.discard")}
          </Button>
        </div>
      </Modal>
    </div>
  );
}
class ErrorBoundary extends Component<
  { children: ReactNode },
  { failed: boolean }
> {
  override state = { failed: false };
  static getDerivedStateFromError() {
    return { failed: true };
  }
  override render() {
    return this.state.failed ? (
      <div className="p-6">
        <h1>TorrServer-Flow</h1>
        <p>{i18n.t("app.error")}</p>
        <button onClick={() => window.location.reload()}>
          {i18n.t("app.reload")}
        </button>
      </div>
    ) : (
      this.props.children
    );
  }
}
const router = createHashRouter([
  {
    element: (
      <DirtyProvider>
        <Shell />
      </DirtyProvider>
    ),
    children: [
      { index: true, element: <Dashboard /> },
      { path: "torrents", element: <Torrents /> },
      { path: "add", element: <Add /> },
      { path: "settings", element: <Settings /> },
    ],
  },
]);
createRoot(document.getElementById("root")!).render(
  <ErrorBoundary>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </ErrorBoundary>,
);
