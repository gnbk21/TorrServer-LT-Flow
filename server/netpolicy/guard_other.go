//go:build !windows

package netpolicy

func GuardSnapshot() GuardAvailability {
	return GuardAvailability{State: "UNSUPPORTED"}
}
