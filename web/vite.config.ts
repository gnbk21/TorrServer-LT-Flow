import { defineConfig, type ViteDevServer } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { readFileSync } from "node:fs";
import { createHash } from "node:crypto";

const appearanceSource = readFileSync(
  new URL("./public/appearance.js", import.meta.url),
  "utf8",
);
const appearanceName = `appearance.js?v=${createHash("sha256").update(appearanceSource).digest("hex").slice(0, 12)}`;
const mockedAPI = process.env.FLOW_TEST_API_STUB === "1";

const endpoints = [
  "echo",
  "torrents",
  "torrent",
  "settings",
  "ssl",
  "mediabase",
  "flow",
  "runtime",
  "search",
  "torznab",
  "jacred",
  "stream",
  "play",
  "viewed",
  "playlist",
  "playlistall",
  "cache",
  "waf",
  "tmdb",
  "gst",
  "shutdown",
  "storage",
  "download",
  "ffp",
  "stat",
  "subs",
  "metrics",
  "dav",
  "hls",
];
function installMockAPI(server: Pick<ViteDevServer, "middlewares">) {
  if (!mockedAPI) return;
  server.middlewares.use((request, response, next) => {
    const name = new URL(
      request.url || "/",
      "http://fixture.invalid",
    ).pathname.split("/")[1];
    if (!name || !endpoints.includes(name)) return next();
    response.statusCode = 404;
    response.setHeader("Content-Type", "application/json");
    response.end(
      JSON.stringify({ error: "endpoint omitted by isolated browser fixture" }),
    );
  });
}
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    {
      name: "flow-isolated-browser-fixtures",
      configureServer: installMockAPI,
      configurePreviewServer: installMockAPI,
    },
    {
      name: "flow-prepaint-identity",
      apply: "build",
      transformIndexHtml(html) {
        return html.replace("./appearance.js", `./${appearanceName}`);
      },
    },
  ],
  base: "./",
  build: { outDir: "build", emptyOutDir: true, sourcemap: false },
  server: {
    host: "127.0.0.1",
    port: 5173,
    proxy: mockedAPI
      ? undefined
      : Object.fromEntries(
          endpoints.map((name) => [
            `/${name}`,
            {
              target: process.env.FLOW_DEV_SERVER || "http://127.0.0.1:8090",
              changeOrigin: false,
            },
          ]),
        ),
  },
});
