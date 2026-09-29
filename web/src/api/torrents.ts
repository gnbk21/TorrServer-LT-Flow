import { api, sessionToken } from "./client";
import { torrentSchema } from "./schemas";
import { Torrent } from "../types/torrent";

export interface AddTorrentRequest {
  link: string;
  title?: string;
  poster?: string;
  category?: string;
  data?: string;
  save_to_db?: boolean;
}

export interface SetTorrentRequest {
  hash: string;
  title?: string;
  category?: string;
  poster?: string;
  data?: string;
}

export const torrentsApi = {
  prepare: (hash: string, index: number, signal?: AbortSignal) =>
    api.get<Torrent>(
      `/stream?${new URLSearchParams({ link: hash, index: String(index), preload: "true", stat: "true", ss: sessionToken })}`,
      signal,
      130_000,
    ),
  list: (signal?: AbortSignal) =>
    api
      .post<{ action: string }, unknown>(
        "/torrents",
        { action: "list" },
        signal,
      )
      .then((data) => torrentSchema.array().parse(data) as Torrent[]),

  get: (hash: string, signal?: AbortSignal) =>
    api.post<{ action: string; hash: string }, Torrent>(
      "/torrents",
      { action: "get", hash },
      signal,
    ),

  add: (data: AddTorrentRequest, signal?: AbortSignal) =>
    api.post<AddTorrentRequest & { action: string }, Torrent>(
      "/torrents",
      { action: "add", ...data },
      signal,
    ),

  upload: async (
    file: File,
    saveToDb = true,
    title?: string,
    poster?: string,
    category?: string,
  ) => {
    const formData = new FormData();
    formData.append("file", file);
    if (saveToDb) formData.append("save", "true");
    if (title) formData.append("title", title);
    if (poster) formData.append("poster", poster);
    if (category) formData.append("category", category);
    return api.post<FormData, Torrent | Torrent[]>("/torrent/upload", formData);
  },

  set: (data: SetTorrentRequest, signal?: AbortSignal) =>
    api.post<SetTorrentRequest & { action: string }, void>(
      "/torrents",
      { action: "set", ...data },
      signal,
    ),

  remove: (hash: string, signal?: AbortSignal) =>
    api.post<{ action: string; hash: string }, void>(
      "/torrents",
      { action: "rem", hash },
      signal,
    ),

  drop: (hash: string, signal?: AbortSignal) =>
    api.post<{ action: string; hash: string }, void>(
      "/torrents",
      { action: "drop", hash },
      signal,
    ),

  wipe: (signal?: AbortSignal) =>
    api.post<{ action: string }, void>("/torrents", { action: "wipe" }, signal),

  getStreamUrl: (
    hash: string,
    index?: number,
    play = true,
    hostOverride?: string,
  ) => {
    const base =
      hostOverride ||
      (typeof window !== "undefined" ? window.location.origin : "");
    const params = new URLSearchParams();
    params.set("link", hash);
    if (index !== undefined) params.set("index", String(index));
    if (play) params.set("play", "true");
    params.set("ss", sessionToken);
    return `${base}/stream?${params.toString()}`;
  },

  getPlaylistUrl: (hash: string, hostOverride?: string) => {
    const base =
      hostOverride ||
      (typeof window !== "undefined" ? window.location.origin : "");
    return `${base}/stream?${new URLSearchParams({ link: hash, m3u: "true", ss: sessionToken })}`;
  },

  getAllPlaylistUrl: (hostOverride?: string) => {
    const base =
      hostOverride ||
      (typeof window !== "undefined" ? window.location.origin : "");
    return `${base}/playlistall/all.m3u`;
  },
};
