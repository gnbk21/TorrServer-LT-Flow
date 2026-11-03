import { api, apiClient } from "./client";

export interface ViewedItem {
  hash: string;
  file_index: number;
  timecode?: number;
  position_source?: string;
  position_updated_at?: string;
  completed?: boolean;
}

export const viewedApi = {
  report: (
    hash: string,
    fileIndex: number,
    timecode: number,
    closing = false,
  ) =>
    apiClient("/viewed", {
      method: "POST",
      keepalive: closing,
      body: JSON.stringify({
        action: "set",
        hash,
        file_index: fileIndex,
        timecode,
        position_source: "internal_player",
      }),
    }),
  complete: (hash: string, fileIndex: number, completed: boolean) =>
    api.post(
      "/viewed",
      { action: "set", hash, file_index: fileIndex, completed },
      undefined,
      true,
    ),
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
