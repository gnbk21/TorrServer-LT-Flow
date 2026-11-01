package settings

import (
	"encoding/json"
	"strings"
)

// FlowSettings keeps Flow controls independent of upstream settings.
// A nil Flow field is migrated to these defaults when existing settings load.
type FlowSettings struct {
	SchemaVersion            int
	Enabled                  bool
	AdaptiveStartup          bool
	BootstrapHeadMB          int
	BootstrapTailMode        string
	ProbeGraceMs             int
	StartupBufferSeconds     int
	StartupBufferMinMB       int
	StartupBufferMaxMB       int
	StartupSafetyFactorPct   int
	AdaptiveReadAhead        bool
	TargetBufferSeconds      int
	MaxBufferSeconds         int
	WarmSessionTimeoutSec    int
	NetworkRetryMinSec       int
	NetworkRetryMaxSec       int
	SwarmProfile             string
	SwarmCustom              FlowSwarmCustom
	RangeTraceEnabled        bool
	RangeClassification      bool
	MetricsEnabled           bool
	DebugFlow                bool
	DiagnosticHistory        bool
	IncidentFlightRecorder   bool // advanced opt-in private raw Go traces; off by default
	DHTStatePersistence      bool
	PreparationQuotaMB       int
	PeerResumeHints          bool // opt-in local peer identities, outside history
	MetadataFallback         bool // supplied HTTPS sources, with bounded recovery
	PublicMetadataCache      bool // opt-in hash disclosure to free public caches
	ScarcePieceHints         bool // bounded scheduling experiment, off by default
	RateAwareDeadlines       bool // measured scheduling experiment, off by default
	CapacityAwareRequests    bool // honor remote and local urgent queue caps, opt-in
	FrontierRecovery         bool // bounded late-block recovery through alternate peers
	AdaptivePeerRequests     bool // measured per-peer service horizon and paced credit
	AdaptiveUrgentHorizon    bool // demand-sized urgent subset, opt-in
	ContainerBurstHints      bool // resident index estimates only, opt-in
	StreamTransportBufferKiB int  // 0/direct, 64, 256 or 1024; existing 1024 KiB default
	NextEpisodeWarmup        bool // opt-in header/index only, <=32 MiB in existing spare cache
	GlobalCacheBudgetMB      int  // aggregate RAM eviction budget; zero chooses bounded auto policy
	WarmCacheBudgetMB        int
	PreparationConcurrency   int
	ManagementOrigins        string // comma-separated browser origins, local dashboard always allowed
	ManagementRateLimit      int    // requests/minute/client; zero disables compatibility limit
	SecurityProfile          string // legacy setting retained for API compatibility; origin checks always apply
	RequirePlaybackToken     bool
	PlaybackTokenTTL         int
	TorrentInterface         string // adapter name; empty retains OS routing
	RequireTorrentInterface  bool
	MSXAllowLAN              bool // explicit approval of literal private/loopback proxy targets
}

// Zero custom values leave libtorrent's own setting unchanged.
type FlowSwarmCustom struct {
	ConnectionSpeed     int
	TorrentConnectBoost int
	PeerConnectTimeout  int
	PieceTimeout        int
	RequestQueueTime    int
	MinReconnectTime    int
}

// UnmarshalJSON migrates newly added fields without overriding explicit false
// values in an existing Flow object.
func (f *FlowSettings) UnmarshalJSON(data []byte) error {
	type plain FlowSettings
	defaults := plain(*DefaultFlowSettings())
	if err := json.Unmarshal(data, &defaults); err != nil {
		return err
	}
	*f = FlowSettings(defaults)
	return nil
}

func DefaultFlowSettings() *FlowSettings {
	return &FlowSettings{
		SchemaVersion: 1,
		Enabled:       true, AdaptiveStartup: true, BootstrapHeadMB: 16,
		BootstrapTailMode: "upstream-auto", ProbeGraceMs: 1500,
		StartupBufferSeconds: 6, StartupBufferMinMB: 32,
		StartupBufferMaxMB: 128, StartupSafetyFactorPct: 130,
		AdaptiveReadAhead: true, TargetBufferSeconds: 45,
		MaxBufferSeconds:      180,
		WarmSessionTimeoutSec: 600, RangeClassification: true,
		NetworkRetryMinSec: 2, NetworkRetryMaxSec: 60,
		SwarmProfile:   "legacy",
		MetricsEnabled: true, DHTStatePersistence: true, PreparationQuotaMB: 4096,
		MetadataFallback:  true,
		WarmCacheBudgetMB: 512, PreparationConcurrency: 1, SecurityProfile: "compatible", PlaybackTokenTTL: 3600,
		StreamTransportBufferKiB: 1024,
	}
}

// Normalize bounds user-supplied values without changing explicit false flags.
func (f *FlowSettings) Normalize() {
	if f.SchemaVersion == 0 {
		f.SchemaVersion = 1
	}
	f.SwarmProfile = strings.ToLower(strings.TrimSpace(f.SwarmProfile))
	normalizeFlowFields(f)
	f.TorrentInterface = strings.TrimSpace(f.TorrentInterface)
	if f.StartupBufferMaxMB < f.StartupBufferMinMB || int64(f.StartupBufferMaxMB) > settingRules["Flow.StartupBufferMaxMB"].Max {
		f.StartupBufferMaxMB = max(DefaultFlowSettings().StartupBufferMaxMB, f.StartupBufferMinMB)
	}
	if f.MaxBufferSeconds < f.TargetBufferSeconds || int64(f.MaxBufferSeconds) > settingRules["Flow.MaxBufferSeconds"].Max {
		f.MaxBufferSeconds = max(DefaultFlowSettings().MaxBufferSeconds, f.TargetBufferSeconds)
	}
	if f.NetworkRetryMaxSec < f.NetworkRetryMinSec || int64(f.NetworkRetryMaxSec) > settingRules["Flow.NetworkRetryMaxSec"].Max {
		f.NetworkRetryMaxSec = max(DefaultFlowSettings().NetworkRetryMaxSec, f.NetworkRetryMinSec)
	}
}

func CurrentFlow() *FlowSettings {
	if s := BTsets(); s != nil && s.Flow != nil {
		return s.Flow
	}
	return DefaultFlowSettings()
}
