//go:build !windows

package netchange

func Watch() (<-chan struct{}, func()) { return nil, func() {} }
