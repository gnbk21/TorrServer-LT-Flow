package settings

// FlowSettings keeps the first Flow milestone independent of upstream settings.
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
	WarmSessionTimeoutSec  int
	RangeTraceEnabled      bool
	RangeClassification    bool
	MetricsEnabled         bool
	DebugFlow              bool
}

func DefaultFlowSettings() *FlowSettings {
	return &FlowSettings{
		Enabled: true, AdaptiveStartup: true, BootstrapHeadMB: 16,
		BootstrapTailMode: "upstream-auto", ProbeGraceMs: 1500,
		StartupBufferSeconds: 6, StartupBufferMinMB: 32,
		StartupBufferMaxMB: 128, StartupSafetyFactorPct: 130,
		WarmSessionTimeoutSec: 600, RangeClassification: true,
		MetricsEnabled: true,
	}
}

// Normalize bounds user-supplied values without changing explicit false flags.
func (f *FlowSettings) Normalize() {
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
	if f.WarmSessionTimeoutSec < 30 || f.WarmSessionTimeoutSec > 1800 {
		f.WarmSessionTimeoutSec = 600
	}
}

func CurrentFlow() *FlowSettings {
	if s := BTsets(); s != nil && s.Flow != nil {
		return s.Flow
	}
	return DefaultFlowSettings()
}
