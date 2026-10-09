package torr

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"server/diagnostics"
	"server/flow"
	"server/settings"
	"sync"
	"time"
)

var ErrSettingsConflict = errors.New("settings changed; reload before applying your draft")

type pendingConfiguration struct {
	Base     string           `json:"base"`
	Settings *settings.BTSets `json:"settings"`
}
type ConfigurationState struct {
	Revision   string                  `json:"revision"`
	Saved      *settings.BTSets        `json:"saved"`
	Effective  *settings.BTSets        `json:"effective"`
	Pending    *settings.BTSets        `json:"pending,omitempty"`
	Applying   bool                    `json:"applying"`
	Error      string                  `json:"error,omitempty"`
	Recovery   settings.RecoveryStatus `json:"recovery"`
	DataPath   string                  `json:"data_path"`
	Executable string                  `json:"executable"`
}

var configControl struct {
	sync.Mutex
	operation   sync.Mutex
	once        sync.Once
	applying    bool
	effective   *settings.BTSets
	pending     *pendingConfiguration
	error       string
	integration func(*settings.BTSets)
}

func pendingPath() string { return filepath.Join(settings.Path, "flow-settings-pending.json") }
func configurationRevisionLocked() string {
	s := settings.CloneSettings(settings.BTsets())
	// Revision covers queued intent too, so a second tab cannot silently replace it.
	if configControl.pending != nil {
		return settings.SettingsRevision(s) + ":" + settings.SettingsRevision(configControl.pending.Settings)
	}
	return settings.SettingsRevision(s)
}
func ConfigurationSnapshot() ConfigurationState {
	configControl.Lock()
	defer configControl.Unlock()
	executable, _ := os.Executable()
	s := ConfigurationState{Revision: configurationRevisionLocked(), Saved: settings.CloneSettings(settings.BTsets()), Effective: settings.CloneSettings(settings.BTsets()), Error: configControl.error, Recovery: settings.SettingsRecovery(), DataPath: settings.Path, Executable: executable}
	s.Applying = configControl.applying
	if s.Applying {
		s.Effective = settings.CloneSettings(configControl.effective)
	} else if helperEngine() != nil && helperEngine().Session() == nil {
		s.Effective = nil
	}
	if configControl.pending != nil {
		s.Pending = settings.CloneSettings(configControl.pending.Settings)
		s.Saved = settings.CloneSettings(s.Pending)
	}
	return s
}

func ApplyConfiguration(next *settings.BTSets, revision, when string) error {
	configControl.operation.Lock()
	defer configControl.operation.Unlock()
	configControl.Lock()
	defer configControl.Unlock()
	if revision != "" && revision != configurationRevisionLocked() {
		return ErrSettingsConflict
	}
	if when != "now" && when != "idle" {
		return errors.New("apply mode must be now or idle")
	}
	if err := settings.ValidateSettings(next); err != nil {
		return err
	}
	current := settings.BTsets()
	if current != nil && (current.SslCert != next.SslCert || current.SslKey != next.SslKey) {
		if err := validateCertificateChange(next); err != nil {
			return err
		}
	}
	if next.Flow != nil && next.Flow.RequirePlaybackToken && !settings.HttpAuth {
		return errors.New("enable HTTP authentication before requiring playback capabilities")
	}
	if settings.ReadOnly {
		return errors.New("database is read-only")
	}
	next = settings.NormalizeConfiguration(next)
	restart := settings.NeedsEngineRestart(settings.BTsets(), next)
	if when == "idle" && settings.NeedsEngineRestart(settings.BTsets(), next) {
		pending := &pendingConfiguration{settings.SettingsRevision(settings.BTsets()), next}
		data, _ := json.Marshal(pending)
		if err := diagnostics.AtomicPrivateFile(pendingPath(), data); err != nil {
			return errors.New("cannot persist scheduled settings")
		}
		configControl.pending = pending
		configControl.error = ""
		return nil
	}
	if err := applyConfigurationLocked(next, restart); err != nil {
		configControl.error = err.Error()
		return err
	}
	configControl.pending = nil
	configControl.error = ""
	retirePendingLocked()
	return nil
}

func validateCertificateChange(next *settings.BTSets) error {
	if settings.Args != nil && (settings.Args.SslCert != "" || settings.Args.SslKey != "") {
		return errors.New("certificate paths are controlled by startup flags")
	}
	if (next.SslCert == "") != (next.SslKey == "") {
		return errors.New("both certificate paths are required")
	}
	if next.SslCert == "" {
		if settings.Ssl {
			return errors.New("use the certificate controls to generate a self-signed pair")
		}
		return nil
	}
	pair, err := tls.LoadX509KeyPair(next.SslCert, next.SslKey)
	if err != nil {
		return errors.New("certificate and key must be readable and match")
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil || time.Now().Before(leaf.NotBefore) || !time.Now().Before(leaf.NotAfter) {
		return errors.New("certificate must be currently valid")
	}
	return nil
}

// ApplyCertificateConfiguration serializes file selection/generation with every
// settings mutation. Queued settings are retained; resolve them before changing
// certificate identity, rather than silently discarding another tab's draft.
func ApplyCertificateConfiguration(revision string, change func(commit func(string, string) error) error, retire func(string, string)) error {
	configControl.operation.Lock()
	defer configControl.operation.Unlock()
	configControl.Lock()
	defer configControl.Unlock()
	if revision == "" || revision != configurationRevisionLocked() {
		return ErrSettingsConflict
	}
	if configControl.pending != nil {
		return errors.New("resolve scheduled settings before changing the certificate")
	}
	if settings.ReadOnly {
		return errors.New("database is read-only")
	}
	if settings.Args != nil && (settings.Args.SslCert != "" || settings.Args.SslKey != "") {
		return errors.New("certificate paths are controlled by startup flags")
	}
	return change(func(cert, key string) error {
		next := settings.CloneSettings(settings.BTsets())
		if next == nil {
			return errors.New("settings are unavailable")
		}
		oldCert, oldKey := next.SslCert, next.SslKey
		next.SslCert, next.SslKey = cert, key
		if err := validateCertificateChange(next); err != nil {
			return err
		}
		if err := settings.SetBTSetsChecked(next); err != nil {
			return err
		}
		if retire != nil && (oldCert != cert || oldKey != key) {
			retire(oldCert, oldKey)
		}
		return nil
	})
}
func CancelPendingConfiguration(revision string) error {
	configControl.operation.Lock()
	defer configControl.operation.Unlock()
	configControl.Lock()
	defer configControl.Unlock()
	if revision != configurationRevisionLocked() {
		return ErrSettingsConflict
	}
	if err := os.Remove(pendingPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	configControl.pending = nil
	configControl.error = ""
	return nil
}

// One process worker survives engine restarts. Its maintenance lease fences new
// streaming/preload work and it rechecks detached work before applying.
func StartConfigurationWorker(integration func(*settings.BTSets)) {
	configControl.once.Do(func() {
		configControl.Lock()
		configControl.integration = integration
		if data, err := os.ReadFile(pendingPath()); err == nil {
			var p pendingConfiguration
			if len(data) <= 1<<20 && json.Unmarshal(data, &p) == nil && settings.ValidateSettings(p.Settings) == nil {
				if settings.SettingsRevision(p.Settings) == settings.SettingsRevision(settings.BTsets()) {
					retirePendingLocked()
				} else {
					configControl.pending = &p
				}
			} else {
				configControl.error = "INVALID_PENDING_SETTINGS"
			}
		}
		configControl.Unlock()
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				applyPendingConfiguration()
			}
		}()
	})
}
func applyPendingConfiguration() {
	configControl.operation.Lock()
	defer configControl.operation.Unlock()
	configControl.Lock()
	defer configControl.Unlock()
	p := configControl.pending
	if p == nil || configControl.error != "" || helperEngine() == nil || helperEngine().Session() == nil || FlowHasActiveWork() {
		return
	}
	if p.Base != settings.SettingsRevision(settings.BTsets()) {
		configControl.error = "PENDING_SETTINGS_CONFLICT"
		return
	}
	token, _ := flow.Maintenance.Acquire(120 * time.Second)
	if token == "" {
		return
	}
	defer flow.Maintenance.Release(token)
	if FlowHasActiveWork() {
		return
	}
	if err := applyConfigurationLocked(p.Settings, true); err != nil {
		configControl.error = err.Error()
		return
	}
	configControl.pending = nil
	configControl.error = ""
	retirePendingLocked()
}

func retirePendingLocked() {
	if err := os.Remove(pendingPath()); err != nil && !os.IsNotExist(err) {
		configControl.error = "SETTINGS_APPLIED_PENDING_FILE_CLEANUP_FAILED"
	}
}

// Keep status polling responsive while native teardown/reconnection runs.
// operation serializes mutations; Mutex only protects the observable state.
func applyConfigurationLocked(next *settings.BTSets, restart bool) error {
	configControl.applying = true
	configControl.effective = settings.CloneSettings(settings.BTsets())
	callback := configControl.integration
	configControl.Unlock()
	err := applySettings(next)
	if err == nil && restart && callback != nil {
		callback(next)
	}
	configControl.Lock()
	configControl.applying = false
	configControl.effective = nil
	return err
}
