import { api } from "./client";

export interface ViewedItem {
  hash: string;
  file_index: number;
  timecode?: number;
}

export const viewedApi = {
  list: (hash?: string, signal?: AbortSignal) =>
    api.post<{ action: string; hash?: string }, ViewedItem[]>(
      "/viewed",
      { action: "list", hash },
      signal,
    ),

  set: (hash: string, fileIndex: number, signal?: AbortSignal) =>
    api.post<{ action: string; hash: string; file_index: number }, void>(
      "/viewed",
      { action: "set", hash, file_index: fileIndex },
      signal,
    ),

  remove: (hash: string, fileIndex: number, signal?: AbortSignal) =>
    api.post<{ action: string; hash: string; file_index: number }, void>(
      "/viewed",
      { action: "rem", hash, file_index: fileIndex },
      signal,
    ),
};
