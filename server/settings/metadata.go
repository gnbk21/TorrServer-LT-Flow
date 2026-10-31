package settings

import (
	"fmt"
	"reflect"
	"strings"
)

// One registry owns numeric/discrete constraints. Some older API fields accept
// zero as a migration sentinel; normal UI bounds describe the effective value.
type settingRule struct {
	Min, Max                  int64
	Choices                   []any
	RejectInvalid, LegacyZero bool
}

var settingRules = map[string]settingRule{
	"CacheSize":       {Min: 1, Max: 16 << 30, RejectInvalid: true, LegacyZero: true},
	"ReaderReadAHead": {Min: 5, Max: 100}, "PreloadCache": {Min: 0, Max: 100},
	"PeersListenPort":     {Min: 0, Max: 65535, RejectInvalid: true},
	"ConnectionsLimit":    {Min: 1, Max: 10000, RejectInvalid: true, LegacyZero: true},
	"DHTConnectionsLimit": {Min: 1, Max: 100000}, "SslPort": {Min: 0, Max: 65535},
	"DownloadRateLimit":                    {Min: 0, Max: 9007199254740991, RejectInvalid: true},
	"UploadRateLimit":                      {Min: 0, Max: 9007199254740991, RejectInvalid: true},
	"TorrentDisconnectTimeout":             {Min: 1, Max: 86400},
	"RetrackersMode":                       {Min: 0, Max: 3, Choices: []any{0, 1, 2, 3}},
	"Flow.StreamTransportBufferKiB":        {Min: 0, Max: 1024, Choices: []any{0, 64, 256, 1024}},
	"Flow.GlobalCacheBudgetMB":             {Min: 0, Max: 1048576, RejectInvalid: true},
	"Flow.WarmCacheBudgetMB":               {Min: 0, Max: 65536, RejectInvalid: true},
	"Flow.PreparationConcurrency":          {Min: 1, Max: 16, RejectInvalid: true, LegacyZero: true},
	"Flow.ManagementRateLimit":             {Min: 0, Max: 60000, RejectInvalid: true},
	"Flow.PlaybackTokenTTL":                {Min: 30, Max: 86400, RejectInvalid: true, LegacyZero: true},
	"Flow.BootstrapHeadMB":                 {Min: 1, Max: 128},
	"Flow.ProbeGraceMs":                    {Min: 0, Max: 10000},
	"Flow.StartupBufferSeconds":            {Min: 1, Max: 60},
	"Flow.StartupBufferMinMB":              {Min: 1, Max: 1024},
	"Flow.StartupBufferMaxMB":              {Min: 1, Max: 2048},
	"Flow.StartupSafetyFactorPct":          {Min: 100, Max: 300},
	"Flow.TargetBufferSeconds":             {Min: 10, Max: 180},
	"Flow.MaxBufferSeconds":                {Min: 10, Max: 600},
	"Flow.WarmSessionTimeoutSec":           {Min: 30, Max: 1800},
	"Flow.NetworkRetryMinSec":              {Min: 1, Max: 60},
	"Flow.NetworkRetryMaxSec":              {Min: 1, Max: 600},
	"Flow.PreparationQuotaMB":              {Min: 64, Max: 1048576},
	"Flow.SwarmCustom.ConnectionSpeed":     {Min: 0, Max: 500},
	"Flow.SwarmCustom.TorrentConnectBoost": {Min: 0, Max: 500},
	"Flow.SwarmCustom.PeerConnectTimeout":  {Min: 0, Max: 120},
	"Flow.SwarmCustom.PieceTimeout":        {Min: 0, Max: 120},
	"Flow.SwarmCustom.RequestQueueTime":    {Min: 0, Max: 30},
	"Flow.SwarmCustom.MinReconnectTime":    {Min: 0, Max: 600},
	"Flow.BootstrapTailMode":               {Choices: []any{"upstream-auto"}},
	"Flow.SwarmProfile":                    {Choices: []any{"legacy", "adaptive", "conservative", "balanced", "aggressive", "custom"}},
	"Flow.SecurityProfile":                 {Choices: []any{"compatible", "restricted"}},
}

type SettingMetadata struct {
	Type            string `json:"type"`
	Default         any    `json:"default"`
	Min             *int64 `json:"min,omitempty"`
	Max             *int64 `json:"max,omitempty"`
	Choices         []any  `json:"choices,omitempty"`
	Unit            string `json:"unit,omitempty"`
	RestartRequired bool   `json:"restart_required"`
	Experimental    bool   `json:"experimental"`
	Deprecated      bool   `json:"deprecated"`
	ReadOnly        bool   `json:"read_only"`
	InvalidPolicy   string `json:"invalid_policy,omitempty"`
	LegacyZero      bool   `json:"legacy_zero,omitempty"`
}
type MetadataSnapshot struct {
	SchemaVersion int                        `json:"schema_version"`
	Fields        map[string]SettingMetadata `json:"fields"`
}

func settingValue(root reflect.Value, path string) reflect.Value {
	for _, part := range strings.Split(path, ".") {
		if root.Kind() == reflect.Pointer {
			if root.IsNil() {
				return reflect.Value{}
			}
			root = root.Elem()
		}
		root = root.FieldByName(part)
		if !root.IsValid() {
			return root
		}
	}
	return root
}
func ruleValid(rule settingRule, value reflect.Value) bool {
	if value.Kind() >= reflect.Int && value.Kind() <= reflect.Int64 {
		v := value.Int()
		if v < rule.Min || v > rule.Max {
			return false
		}
	}
	if len(rule.Choices) > 0 {
		for _, choice := range rule.Choices {
			if fmt.Sprint(value.Interface()) == fmt.Sprint(choice) {
				return true
			}
		}
		return false
	}
	return true
}
func normalizeFlowFields(f *FlowSettings) {
	current, defaults := reflect.ValueOf(f), reflect.ValueOf(DefaultFlowSettings())
	for path, rule := range settingRules {
		if !strings.HasPrefix(path, "Flow.") {
			continue
		}
		path = strings.TrimPrefix(path, "Flow.")
		if path == "StartupBufferMaxMB" || path == "MaxBufferSeconds" || path == "NetworkRetryMaxSec" {
			continue
		}
		v := settingValue(current, path)
		if !ruleValid(rule, v) {
			v.Set(settingValue(defaults, path))
		}
	}
}
func validateRegisteredSettings(s *BTSets) error {
	for path, rule := range settingRules {
		if !rule.RejectInvalid {
			continue
		}
		v := settingValue(reflect.ValueOf(s), path)
		if !v.IsValid() || (rule.LegacyZero && v.Int() == 0) {
			continue
		}
		if !ruleValid(rule, v) {
			return fmt.Errorf("%s exceeds supported bounds", path)
		}
	}
	return nil
}
func metadataUnit(path string) string {
	switch {
	case path == "CacheSize":
		return "bytes"
	case strings.HasSuffix(path, "MB"):
		return "MiB"
	case strings.HasSuffix(path, "KiB"):
		return "KiB"
	case path == "DownloadRateLimit" || path == "UploadRateLimit":
		return "KiB/s"
	case path == "Flow.ManagementRateLimit":
		return "requests/min"
	case strings.HasSuffix(path, "Ms"):
		return "ms"
	case strings.HasSuffix(path, "Pct") || path == "ReaderReadAHead" || path == "PreloadCache":
		return "%"
	case strings.HasSuffix(path, "Seconds") || strings.HasSuffix(path, "Sec") || strings.HasSuffix(path, "Timeout") || strings.HasSuffix(path, "ReconnectTime") || strings.HasSuffix(path, "QueueTime") || path == "Flow.PlaybackTokenTTL":
		return "s"
	default:
		return ""
	}
}
func SettingsMetadata() MetadataSnapshot {
	base := NewDefaultConfig()
	out := MetadataSnapshot{SchemaVersion: 1, Fields: map[string]SettingMetadata{}}
	var walk func(reflect.Value, string)
	walk = func(v reflect.Value, prefix string) {
		if v.Kind() == reflect.Pointer {
			v = v.Elem()
		}
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			path := prefix + v.Type().Field(i).Name
			if field.Kind() == reflect.Struct || field.Kind() == reflect.Pointer {
				walk(field, path+".")
				continue
			}
			kind := field.Kind()
			if kind != reflect.String && kind != reflect.Bool && kind != reflect.Int && kind != reflect.Int64 {
				continue
			}
			m := SettingMetadata{Type: "string", Default: field.Interface(), Unit: metadataUnit(path), ReadOnly: path == "Flow.SchemaVersion"}
			probe := CloneSettings(base)
			target := settingValue(reflect.ValueOf(probe), path)
			switch kind {
			case reflect.Bool:
				m.Type = "boolean"
				target.SetBool(!field.Bool())
			case reflect.Int, reflect.Int64:
				m.Type = "integer"
				target.SetInt(field.Int() + 1)
			case reflect.String:
				target.SetString(field.String() + "-metadata-probe")
			}
			m.RestartRequired = NeedsEngineRestart(base, probe)
			if rule, ok := settingRules[path]; ok {
				m.Choices = rule.Choices
				if m.Type == "integer" {
					lo, hi := rule.Min, rule.Max
					m.Min, m.Max = &lo, &hi
				}
				m.InvalidPolicy = "normalize"
				// Historical APIs accept some editor-out-of-range values. Do not
				// silently tighten saved-configuration compatibility in a metadata
				// endpoint: distinguish these from enforced/normalized constraints.
				if !strings.HasPrefix(path, "Flow.") && path != "ReaderReadAHead" && path != "PreloadCache" {
					m.InvalidPolicy = "legacy"
				}
				if rule.RejectInvalid {
					m.InvalidPolicy = "reject"
				}
				m.LegacyZero = rule.LegacyZero
			}
			m.Deprecated = path == "Flow.SecurityProfile"
			switch path {
			case "Flow.ScarcePieceHints", "Flow.RateAwareDeadlines", "Flow.CapacityAwareRequests", "Flow.FrontierRecovery", "Flow.AdaptivePeerRequests", "Flow.AdaptiveUrgentHorizon", "Flow.ContainerBurstHints":
				m.Experimental = true
			}
			out.Fields[path] = m
		}
	}
	walk(reflect.ValueOf(base), "")
	return out
}
