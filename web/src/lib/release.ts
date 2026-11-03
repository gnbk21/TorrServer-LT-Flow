export interface ReleaseLabels {
  resolution?: string;
  codec?: string;
  hdr?: string;
  release?: string;
  audio?: string;
  language?: string;
}
// Bounded parsing describes filename claims, never verified media quality.
export function releaseLabels(value: string): ReleaseLabels {
  const text = value
    .slice(0, 8192)
    .normalize("NFKC")
    .toUpperCase()
    .replace(/[._]/g, " ");
  const match = (re: RegExp) => text.match(re)?.[1];
  return {
    resolution:
      match(/\b(2160P|1080P|1080I|720P|480P)\b/)?.toLowerCase() ??
      (/\b4K\b/.test(text) ? "2160p" : undefined),
    codec: /\b(?:X265|H\s?265|HEVC)\b/.test(text)
      ? "HEVC"
      : /\b(?:X264|H\s?264|AVC)\b/.test(text)
        ? "AVC"
        : match(/\b(AV1|VP9)\b/),
    hdr: /\b(?:DOLBY\s*VISION|DV|DOVI)\b/.test(text)
      ? "Dolby Vision"
      : /\bHDR10\+/.test(text)
        ? "HDR10+"
        : match(/\b(HDR10|HDR|HLG|SDR)\b/),
    release: match(/\b(REMUX|WEB[ -]?DL|WEBRIP|BLURAY|BDRIP|HDTV)\b/),
    audio: match(
      /\b(TRUEHD|ATMOS|DTS[ -]?HD|DTS|EAC3|DDP|AC3|AAC|FLAC|OPUS)\b/,
    ),
    language: match(
      /\b(RUS|RUSSIAN|ENG|ENGLISH|UKR|FRENCH|FRE|ROMANIAN|RUM|CHINESE|CHI|ZHO|BULGARIAN|BUL|MULTI)\b/,
    ),
  };
}
export function matchesRelease(
  labels: ReleaseLabels,
  filters: Partial<ReleaseLabels>,
) {
  return Object.entries(filters).every(
    ([key, value]) => !value || labels[key as keyof ReleaseLabels] === value,
  );
}
