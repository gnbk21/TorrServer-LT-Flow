package settings

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"server/diagnostics"
	"strings"
	"sync"
)

type RecoveryStatus struct {
	Source        string `json:"source"`
	Issue         string `json:"issue,omitempty"`
	SchemaVersion int    `json:"schema_version"`
}

var recovery struct {
	sync.Mutex
	status RecoveryStatus
}

func SettingsRecovery() RecoveryStatus {
	recovery.Lock()
	defer recovery.Unlock()
	return recovery.status
}
func recordRecovery(source, issue string) {
	recovery.Lock()
	recovery.status = RecoveryStatus{source, issue, 1}
	recovery.Unlock()
}

func SettingsRevision(s *BTSets) string {
	b, _ := json.Marshal(s)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func CloneSettings(s *BTSets) *BTSets {
	if s == nil {
		return nil
	}
	b, _ := json.Marshal(s)
	var out BTSets
	_ = json.Unmarshal(b, &out)
	return &out
}

func ValidateSettings(s *BTSets) error {
	if s == nil {
		return errors.New("settings are required")
	}
	if s.CacheSize < 0 || s.CacheSize > 16<<30 || s.PeersListenPort < 0 || s.PeersListenPort > 65535 || s.ConnectionsLimit < 0 || s.ConnectionsLimit > 10000 || s.DownloadRateLimit < 0 || s.UploadRateLimit < 0 {
		return errors.New("settings exceed supported bounds")
	}
	if f := s.Flow; f != nil {
		if f.SchemaVersion != 0 && f.SchemaVersion != 1 {
			return errors.New("unsupported Flow settings schema")
		}
		if f.SecurityProfile != "" && f.SecurityProfile != "compatible" && f.SecurityProfile != "restricted" {
			return errors.New("unknown security profile")
		}
		if len(f.ManagementOrigins) > 8192 || len(f.TorrentInterface) > 256 {
			return errors.New("network policy exceeds supported bounds")
		}
		for _, origin := range strings.Split(f.ManagementOrigins, ",") {
			origin = strings.TrimSpace(origin)
			if origin == "" {
				continue
			}
			u, err := url.Parse(origin)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
				return errors.New("management origins must be exact http(s) origins")
			}
		}
		if f.RequireTorrentInterface && f.TorrentInterface == "" {
			return errors.New("a required torrent interface must be selected")
		}
	}
	return nil
}

func saveKnownGood(s *BTSets) error {
	if Path == "" || ReadOnly {
		return nil
	}
	if err := ValidateSettings(s); err != nil {
		return err
	}
	b, err := json.Marshal(struct {
		SchemaVersion int     `json:"schema_version"`
		Settings      *BTSets `json:"settings"`
	}{1, s})
	if err != nil {
		return err
	}
	return diagnostics.AtomicPrivateFile(filepath.Join(Path, "flow-settings-good.json"), b)
}
func readKnownGood() (*BTSets, error) {
	f, err := os.Open(filepath.Join(Path, "flow-settings-good.json"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var saved struct {
		SchemaVersion int     `json:"schema_version"`
		Settings      *BTSets `json:"settings"`
	}
	st, err := f.Stat()
	if err != nil || st.Size() > 1<<20 {
		return nil, errors.New("invalid recovery snapshot size")
	}
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&saved)
	if err != nil {
		return nil, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil, errors.New("trailing data in recovery snapshot")
	}
	if saved.SchemaVersion != 1 {
		return nil, errors.New("unsupported recovery schema")
	}
	if err = ValidateSettings(saved.Settings); err != nil {
		return nil, err
	}
	return saved.Settings, nil
}

// Hot application is deliberately an allowlist. Anything touching native
// transports, disk ownership or integrations still requires an idle restart.
func NeedsEngineRestart(old, next *BTSets) bool {
	if old == nil || next == nil {
		return true
	}
	a, b := CloneSettings(old), CloneSettings(next)
	b.CacheSize = a.CacheSize
	b.ReaderReadAHead = a.ReaderReadAHead
	b.PreloadCache = a.PreloadCache
	b.EnableDebug = a.EnableDebug
	b.TrackTimecode = a.TrackTimecode
	b.ShowFSActiveTorr = a.ShowFSActiveTorr
	if a.Flow != nil && b.Flow != nil {
		av, bv := reflect.ValueOf(a.Flow).Elem(), reflect.ValueOf(b.Flow).Elem()
		for _, name := range []string{"AdaptiveStartup", "BootstrapHeadMB", "ProbeGraceMs", "StartupBufferSeconds", "StartupBufferMinMB", "StartupBufferMaxMB", "StartupSafetyFactorPct", "AdaptiveReadAhead", "TargetBufferSeconds", "MaxBufferSeconds", "WarmSessionTimeoutSec", "NetworkRetryMinSec", "NetworkRetryMaxSec", "RangeTraceEnabled", "RangeClassification", "MetricsEnabled", "DebugFlow", "PreparationQuotaMB", "ScarcePieceHints", "RateAwareDeadlines", "GlobalCacheBudgetMB", "WarmCacheBudgetMB", "PreparationConcurrency", "ManagementOrigins", "ManagementRateLimit", "SecurityProfile", "RequirePlaybackToken", "PlaybackTokenTTL"} {
			bv.FieldByName(name).Set(av.FieldByName(name))
		}
	}
	return !reflect.DeepEqual(a, b)
}
