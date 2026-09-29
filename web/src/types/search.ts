export interface SearchResultItem {
  title: string;
  name?: string;
  size: number | string;
  seeders: number;
  leechers: number;
  magnet: string;
  link?: string;
  category?: string;
  date?: string;
  tracker?: string;
}
