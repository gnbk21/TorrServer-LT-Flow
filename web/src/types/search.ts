export interface SearchResultItem {
  title: string;
  name?: string;
  size: number | string | null;
  seeders: number | null;
  leechers: number | null;
  sources?: string[];
  magnet: string;
  link?: string;
  category?: string;
  date?: string;
  tracker?: string;
}
