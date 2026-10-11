package netpolicy

// Presence is not installation or packet-level verification. A restricted
// service must never elevate itself or execute a helper from an HTTP request.
type GuardAvailability struct {
	Supported              bool   `json:"supported"`
	Available              bool   `json:"available"`
	State                  string `json:"state"`
	DNSVerified            bool   `json:"dns_verified"`
	LeakProtectionVerified bool   `json:"leak_protection_verified"`
}
