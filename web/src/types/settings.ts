export interface FlowSwarmCustom {
  ConnectionSpeed: number;
  TorrentConnectBoost: number;
  PeerConnectTimeout: number;
  PieceTimeout: number;
  RequestQueueTime: number;
  MinReconnectTime: number;
}

export interface FlowSettings {
  SchemaVersion?: number;
  GlobalCacheBudgetMB?: number;
  WarmCacheBudgetMB?: number;
  PreparationConcurrency?: number;
  ManagementOrigins?: string;
  ManagementRateLimit?: number;
  SecurityProfile?: string;
  RequirePlaybackToken?: boolean;
  PlaybackTokenTTL?: number;
  TorrentInterface?: string;
  RequireTorrentInterface?: boolean;
  MSXAllowLAN?: boolean;
  Enabled: boolean;
  AdaptiveStartup: boolean;
  BootstrapHeadMB: number;
  BootstrapTailMode: string;
  ProbeGraceMs: number;
  StartupBufferSeconds: number;
  StartupBufferMinMB: number;
  StartupBufferMaxMB: number;
  StartupSafetyFactorPct: number;
  AdaptiveReadAhead: boolean;
  TargetBufferSeconds: number;
  MaxBufferSeconds: number;
  WarmSessionTimeoutSec: number;
  NetworkRetryMinSec: number;
  NetworkRetryMaxSec: number;
  SwarmProfile: string;
  SwarmCustom: FlowSwarmCustom;
  RangeTraceEnabled: boolean;
  RangeClassification: boolean;
  MetricsEnabled: boolean;
  DebugFlow: boolean;
  DiagnosticHistory?: boolean;
  DHTStatePersistence?: boolean;
  PreparationQuotaMB?: number;
  PeerResumeHints?: boolean;
  ScarcePieceHints?: boolean;
  RateAwareDeadlines?: boolean;
  CapacityAwareRequests?: boolean;
  AdaptiveUrgentHorizon?: boolean;
  ContainerBurstHints?: boolean;
}

export interface TMDBSettings {
  APIKey: string;
  APIURL: string;
  ImageURL: string;
  ImageURLRu: string;
}

export interface TorznabConfig {
  Host: string;
  Key: string;
  Name: string;
}

export interface BTSettings {
  Flow: FlowSettings | null;
  CacheSize: number;
  ReaderReadAHead: number;
  PreloadCache: number;
  PadTailPartial: boolean;
  UseDisk: boolean;
  TorrentsSavePath: string;
  RemoveCacheOnDrop: boolean;
  ForceEncrypt: boolean;
  RetrackersMode: number;
  TrackersListURL: string;
  DefaultTrackers: string;
  TorrentDisconnectTimeout: number;
  EnableDebug: boolean;
  EnableDLNA: boolean;
  FriendlyName: string;
  EnableBonjour: boolean;
  EnableRutorSearch: boolean;
  EnableTorznabSearch: boolean;
  TorznabUrls: TorznabConfig[];
  EnableJacRedSearch: boolean;
  JacRedUrl: string;
  JacRedKey: string;
  TMDBSettings: TMDBSettings;
  EnableIPv6: boolean;
  DisableTCP: boolean;
  DisableUTP: boolean;
  DisableUPNP: boolean;
  DisableDHT: boolean;
  DisablePEX: boolean;
  DisableUpload: boolean;
  DisableEndGame: boolean;
  DownloadRateLimit: number;
  UploadRateLimit: number;
  ConnectionsLimit: number;
  DHTConnectionsLimit: number;
  PeersListenPort: number;
  EnableLPD: boolean;
  SslPort: number;
  SslCert: string;
  SslKey: string;
  TrustedProxies: string[];
  ShowFSActiveTorr: boolean;
  StoreSettingsInJson: boolean;
  StoreViewedInJson: boolean;
  TrackTimecode: boolean;
  MergeAllM3U: boolean;
  [key: string]: unknown;
}
