package settings

import (
	"encoding/json"
	"strings"
)

// FlowSettings keeps Flow controls independent of upstream settings.
// A nil Flow field is migrated to these defaults when existing settings load.
type FlowSettings struct {
	Enabled                bool
	AdaptiveStartup        bool
	BootstrapHeadMB        int
	BootstrapTailMode      string
	ProbeGraceMs           int
	StartupBufferSeconds   int
	StartupBufferMinMB     int
	StartupBufferMaxMB     int
	StartupSafetyFactorPct int
	AdaptiveReadAhead      bool
	TargetBufferSeconds    int
	MaxBufferSeconds       int
	WarmSessionTimeoutSec  int
	NetworkRetryMinSec     int
	NetworkRetryMaxSec     int
	SwarmProfile           string
	SwarmCustom            FlowSwarmCustom
	RangeTraceEnabled      bool
	RangeClassification    bool
	MetricsEnabled         bool
	DebugFlow              bool
	DiagnosticHistory      bool
	DHTStatePersistence    bool
	PreparationQuotaMB     int
	PeerResumeHints        bool // opt-in local peer identities, outside history
	ScarcePieceHints       bool // bounded scheduling experiment, off by default
	RateAwareDeadlines     bool // measured scheduling experiment, off by default
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
		Enabled: true, AdaptiveStartup: true, BootstrapHeadMB: 16,
		BootstrapTailMode: "upstream-auto", ProbeGraceMs: 1500,
		StartupBufferSeconds: 6, StartupBufferMinMB: 32,
		StartupBufferMaxMB: 128, StartupSafetyFactorPct: 130,
		AdaptiveReadAhead: true, TargetBufferSeconds: 45,
		MaxBufferSeconds:      180,
		WarmSessionTimeoutSec: 600, RangeClassification: true,
		NetworkRetryMinSec: 2, NetworkRetryMaxSec: 60,
		SwarmProfile:   "legacy",
		MetricsEnabled: true, DHTStatePersistence: true, PreparationQuotaMB: 4096,
	}
}

// Normalize bounds user-supplied values without changing explicit false flags.
func (f *FlowSettings) Normalize() {
	if f.PreparationQuotaMB < 64 || f.PreparationQuotaMB > 1048576 {
		f.PreparationQuotaMB = 4096
	}
	if f.BootstrapHeadMB < 1 || f.BootstrapHeadMB > 128 {
		f.BootstrapHeadMB = 16
	}
	if f.BootstrapTailMode != "upstream-auto" {
		f.BootstrapTailMode = "upstream-auto"
	}
	if f.ProbeGraceMs < 0 || f.ProbeGraceMs > 10000 {
		f.ProbeGraceMs = 1500
	}
	if f.StartupBufferSeconds < 1 || f.StartupBufferSeconds > 60 {
		f.StartupBufferSeconds = 6
	}
	if f.StartupBufferMinMB < 1 || f.StartupBufferMinMB > 1024 {
		f.StartupBufferMinMB = 32
	}
	if f.StartupBufferMaxMB < f.StartupBufferMinMB || f.StartupBufferMaxMB > 2048 {
		f.StartupBufferMaxMB = max(128, f.StartupBufferMinMB)
	}
	if f.StartupSafetyFactorPct < 100 || f.StartupSafetyFactorPct > 300 {
		f.StartupSafetyFactorPct = 130
	}
	if f.TargetBufferSeconds < 10 || f.TargetBufferSeconds > 180 {
		f.TargetBufferSeconds = 45
	}
	if f.MaxBufferSeconds < f.TargetBufferSeconds || f.MaxBufferSeconds > 600 {
		f.MaxBufferSeconds = max(180, f.TargetBufferSeconds)
	}
	if f.WarmSessionTimeoutSec < 30 || f.WarmSessionTimeoutSec > 1800 {
		f.WarmSessionTimeoutSec = 600
	}
	if f.NetworkRetryMinSec < 1 || f.NetworkRetryMinSec > 60 {
		f.NetworkRetryMinSec = 2
	}
	if f.NetworkRetryMaxSec < f.NetworkRetryMinSec || f.NetworkRetryMaxSec > 600 {
		f.NetworkRetryMaxSec = max(60, f.NetworkRetryMinSec)
	}
	f.SwarmProfile = strings.ToLower(strings.TrimSpace(f.SwarmProfile))
	switch f.SwarmProfile {
	case "legacy", "conservative", "balanced", "aggressive", "custom":
	default:
		f.SwarmProfile = "legacy"
	}
	if f.SwarmCustom.ConnectionSpeed < 0 || f.SwarmCustom.ConnectionSpeed > 500 {
		f.SwarmCustom.ConnectionSpeed = 0
	}
	if f.SwarmCustom.TorrentConnectBoost < 0 || f.SwarmCustom.TorrentConnectBoost > 500 {
		f.SwarmCustom.TorrentConnectBoost = 0
	}
	if f.SwarmCustom.PeerConnectTimeout < 0 || f.SwarmCustom.PeerConnectTimeout > 120 {
		f.SwarmCustom.PeerConnectTimeout = 0
	}
	if f.SwarmCustom.PieceTimeout < 0 || f.SwarmCustom.PieceTimeout > 120 {
		f.SwarmCustom.PieceTimeout = 0
	}
	if f.SwarmCustom.RequestQueueTime < 0 || f.SwarmCustom.RequestQueueTime > 30 {
		f.SwarmCustom.RequestQueueTime = 0
	}
	if f.SwarmCustom.MinReconnectTime < 0 || f.SwarmCustom.MinReconnectTime > 600 {
		f.SwarmCustom.MinReconnectTime = 0
	}
}

func CurrentFlow() *FlowSettings {
	if s := BTsets(); s != nil && s.Flow != nil {
		return s.Flow
	}
	return DefaultFlowSettings()
}
