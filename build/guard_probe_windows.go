//go:build ignore

// Isolated WFP acceptance fixture. No DNS, credentials or public peers.
package main

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"syscall"
	"time"
)

type result struct {
	Success      bool `json:"success"`
	AccessDenied bool `json:"access_denied"`
	Unavailable  bool `json:"unavailable"`
}

func observation(err error) result {
	return result{Success: err == nil, AccessDenied: errors.Is(err, syscall.Errno(10013)),
		Unavailable: errors.Is(err, syscall.Errno(10051)) || errors.Is(err, syscall.Errno(10047))}
}
func localTCP(network, address string) result {
	listener, err := net.Listen(network, address)
	if err != nil {
		return observation(err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err == nil {
			connection.Close()
		}
		done <- err
	}()
	connection, err := net.DialTimeout(network, listener.Addr().String(), time.Second)
	if err == nil {
		connection.Close()
		err = <-done
	}
	return observation(err)
}
func localUDP(network, address string) result {
	listener, err := net.ListenPacket(network, address)
	if err != nil {
		return observation(err)
	}
	defer listener.Close()
	connection, err := net.DialTimeout(network, listener.LocalAddr().String(), time.Second)
	if err != nil {
		return observation(err)
	}
	defer connection.Close()
	_, err = connection.Write([]byte("Flow fixture"))
	if err == nil {
		listener.SetReadDeadline(time.Now().Add(time.Second))
		var payload [32]byte
		var n int
		n, _, err = listener.ReadFrom(payload[:])
		if err == nil && string(payload[:n]) != "Flow fixture" {
			err = syscall.EINVAL
		}
	}
	return observation(err)
}
func public(network, address string) result {
	connection, err := net.DialTimeout(network, address, 500*time.Millisecond)
	if err == nil {
		defer connection.Close()
		if network == "udp4" || network == "udp6" {
			_, err = connection.Write([]byte("Flow fixture"))
		}
	}
	return observation(err)
}
func main() {
	data := map[string]result{
		"lan_tcp4":    localTCP("tcp4", "127.0.0.1:0"),
		"lan_udp4":    localUDP("udp4", "127.0.0.1:0"),
		"lan_tcp6":    localTCP("tcp6", "[::1]:0"),
		"lan_udp6":    localUDP("udp6", "[::1]:0"),
		"public_tcp4": public("tcp4", "192.0.2.1:49191"),
		"public_udp4": public("udp4", "192.0.2.1:49191"),
		"public_tcp6": public("tcp6", "[2001:db8::1]:49191"),
		"public_udp6": public("udp6", "[2001:db8::1]:49191"),
	}
	if json.NewEncoder(os.Stdout).Encode(data) != nil {
		os.Exit(1)
	}
}
