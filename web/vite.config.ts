import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

const endpoints = [
  "echo",
  "torrents",
  "torrent",
  "settings",
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
  "dav",
  "hls",
];
export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: "./",
  build: { outDir: "build", emptyOutDir: true, sourcemap: false },
  server: {
    host: "127.0.0.1",
    port: 5173,
    proxy: Object.fromEntries(
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
