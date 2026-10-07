//go:build windows

// Read the installed policy through SCM instead of parsing localized sc.exe
// display text. This probe does not modify the service or its configuration.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
)

func run() error {
	manager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer manager.Disconnect()
	handle, err := windows.OpenService(manager.Handle, windows.StringToUTF16Ptr("TorrServer-Flow"), windows.SERVICE_QUERY_CONFIG)
	if err != nil {
		return err
	}
	service := &mgr.Service{Handle: handle}
	defer service.Close()
	actions, err := service.RecoveryActions()
	if err != nil {
		return err
	}
	reset, err := service.ResetPeriod()
	if err != nil {
		return err
	}
	report := struct {
		Reset   uint32               `json:"reset_seconds"`
		Actions []mgr.RecoveryAction `json:"actions"`
	}{reset, actions}
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		return err
	}
	expected := []mgr.RecoveryAction{{Type: mgr.ServiceRestart, Delay: 15 * time.Second}, {Type: mgr.ServiceRestart, Delay: 30 * time.Second}, {Type: mgr.ServiceRestart, Delay: 60 * time.Second}, {Type: mgr.NoAction}}
	if reset != 86400 || len(actions) != len(expected) {
		return fmt.Errorf("unexpected recovery reset/count: %d/%d", reset, len(actions))
	}
	for i, action := range actions {
		if action != expected[i] {
			return fmt.Errorf("unexpected recovery action %d: %+v", i, action)
		}
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
