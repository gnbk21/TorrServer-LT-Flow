//go:build windows

package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"server"
)

const flowServiceName = "TorrServer-Flow"

func serviceDataDir() string {
	base := os.Getenv("ProgramData")
	if base == "" {
		drive := os.Getenv("SystemDrive")
		if drive == "" {
			drive = "C:"
		}
		base = filepath.Join(drive+`\`, "ProgramData")
	}
	return filepath.Join(base, flowServiceName)
}

func serviceCommand(command string, p *args) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	if command == "install" {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		path := p.Path
		if path == "" {
			path = serviceDataDir()
		}
		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(path, 0750); err != nil {
			return fmt.Errorf("create service state directory: %w", err)
		}
		if p.UI {
			return errors.New("--ui is interactive and cannot be installed in a service")
		}
		installedArgs := []string{"--service", "run"}
		for i := 1; i < len(os.Args); i++ {
			a := os.Args[i]
			if a == "--service" {
				i++
				continue
			}
			if strings.HasPrefix(a, "--service=") {
				continue
			}
			if a == "--path" || a == "-d" {
				i++
				continue
			}
			if strings.HasPrefix(a, "--path=") || strings.HasPrefix(a, "-d=") {
				continue
			}
			installedArgs = append(installedArgs, a)
		}
		installedArgs = append(installedArgs, "--path", path)
		s, err := m.CreateService(flowServiceName, exe, mgr.Config{
			DisplayName:      "TorrServer-Flow",
			Description:      "Torrent streaming server for local playback",
			StartType:        mgr.StartAutomatic,
			DelayedAutoStart: true,
		}, installedArgs...)
		if err != nil {
			return err
		}
		defer s.Close()
		fmt.Println("Installed", flowServiceName, "with state at", path)
		return nil
	}

	s, err := m.OpenService(flowServiceName)
	if err != nil {
		return err
	}
	defer s.Close()
	switch command {
	case "start":
		return startService(s)
	case "stop":
		return stopService(s)
	case "restart":
		if err := stopService(s); err != nil {
			return err
		}
		return startService(s)
	case "uninstall":
		if err := stopService(s); err != nil {
			return err
		}
		return s.Delete()
	default:
		return fmt.Errorf("unknown service command %q", command)
	}
}

func waitForServiceState(s *mgr.Service, want svc.State) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		status, err := s.Query()
		if err != nil {
			return err
		}
		if status.State == want {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s to reach state %d", flowServiceName, want)
}

func startService(s *mgr.Service) error {
	status, err := s.Query()
	if err != nil {
		return err
	}
	if status.State == svc.Running {
		return nil
	}
	if err := s.Start(); err != nil {
		return err
	}
	return waitForServiceState(s, svc.Running)
}

func stopService(s *mgr.Service) error {
	status, err := s.Query()
	if err != nil {
		return err
	}
	if status.State == svc.Stopped {
		return nil
	}
	if _, err := s.Control(svc.Stop); err != nil {
		return err
	}
	return waitForServiceState(s, svc.Stopped)
}

type flowService struct{}

func (flowService) Execute(_ []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}
	started := make(chan struct{})
	go func() {
		server.Start()
		close(started)
	}()
	select {
	case <-started:
	case <-time.After(25 * time.Second):
		return false, 1
	}
	wait := make(chan string, 1)
	go func() { wait <- server.WaitServer() }()
	if err := waitForHTTPListener(); err != nil {
		go server.Stop()
		return false, 1
	}
	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		select {
		case result := <-wait:
			if result != "" {
				return false, 1
			}
			return false, 0
		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				changes <- request.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				stopped := make(chan struct{})
				go func() { server.Stop(); close(stopped) }()
				select {
				case <-stopped:
				case <-time.After(30 * time.Second):
					return false, 1
				}
				return false, 0
			}
		}
	}
}

func waitForHTTPListener() error {
	host := "127.0.0.1"
	if len(params.IPs) > 0 && params.IPs[0] != "" {
		host = params.IPs[0]
	}
	if host == "0.0.0.0" {
		host = "127.0.0.1"
	} else if host == "::" || host == "[::]" {
		host = "::1"
	}
	addr := net.JoinHostPort(host, params.Port)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 250*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("HTTP listener did not become ready at %s", addr)
}

func runWindowsService() error {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return err
	}
	if !isService {
		return errors.New("--service run must be launched by Windows Service Control Manager")
	}
	return svc.Run(flowServiceName, flowService{})
}
