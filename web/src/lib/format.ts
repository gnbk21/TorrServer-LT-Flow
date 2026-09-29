export function humanizeBytes(bytes?: number): string {
  if (bytes == null || !Number.isFinite(bytes) || bytes < 0) return "—";
  const units = ["B", "KiB", "MiB", "GiB", "TiB"];
  const index =
    bytes === 0
      ? 0
      : Math.max(0, Math.min(4, Math.floor(Math.log(bytes) / Math.log(1024))));
  const value = bytes / 1024 ** index;
  return `${value.toFixed(index === 0 || value >= 100 ? 0 : 1)} ${units[index]}`;
}
export function humanizeSpeed(value?: number): string {
  return value == null || !Number.isFinite(value)
    ? "—"
    : `${humanizeBytes(value)}/s`;
}
export function humanizeBitrate(value?: number): string {
  return value == null || !Number.isFinite(value) || value <= 0
    ? "—"
    : `${(value / 1e6).toFixed(2)} Mbps`;
}
export function formatDuration(value?: number): string {
  if (value == null || !Number.isFinite(value) || value < 0) return "—";
  const seconds = Math.round(value);
  return seconds < 60
    ? `${seconds}s`
    : seconds < 3600
      ? `${Math.floor(seconds / 60)}m ${seconds % 60}s`
      : `${Math.floor(seconds / 3600)}h ${Math.floor(seconds / 60) % 60}m`;
}
export interface EpisodeInfo {
  isEpisode: boolean;
  season?: number;
  episode?: number;
  displayTitle: string;
  code?: string;
}
export function parseEpisodeInfo(filename: string): EpisodeInfo {
  const name = filename.split(/[\\/]/).pop() || filename;
  const patterns = [
    /(?:^|[\s._-])s(\d{1,2})[\s._-]*e(\d{1,3})(?=$|[\s._-])/i,
    /(?:^|[\s._-])(\d{1,2})x(\d{1,3})(?=$|[\s._-])/i,
    /(?:^|[\s._-])season[\s._-]*(\d{1,2})[\s._-]*episode[\s._-]*(\d{1,3})(?=$|[\s._-])/i,
  ];
  for (const pattern of patterns) {
    const match = pattern.exec(name);
    if (match?.[1] && match[2]) {
      const season = Number(match[1]),
        episode = Number(match[2]);
      const code = `S${String(season).padStart(2, "0")}E${String(episode).padStart(2, "0")}`;
      return { isEpisode: true, season, episode, code, displayTitle: name };
    }
  }
  const match =
    /(?:^|[\s._-])(?:episode|ep|e)[\s._-]*(\d{1,3})(?=$|[\s._-])/i.exec(name);
  if (match?.[1]) {
    const episode = Number(match[1]);
    return {
      isEpisode: true,
      episode,
      code: `E${String(episode).padStart(2, "0")}`,
      displayTitle: name,
    };
  }
  return { isEpisode: false, displayTitle: name };
}
