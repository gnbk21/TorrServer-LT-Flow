package netbind

import (
	"net"
	"sort"

	"github.com/wlynxg/anet"
)

// LocalIPs supplies startup, certificates and setup with the same addresses,
// including before the torrent engine/network watcher is ready.
func LocalIPs() ([]string, error) {
	ifaces, err := anet.Interfaces()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, _ := anet.InterfaceAddrsByInterface(&iface)
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && !ip.IsUnspecified() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() {
				seen[ip.String()] = true
			}
		}
	}
	list := make([]string, 0, len(seen))
	for ip := range seen {
		list = append(list, ip)
	}
	sort.Strings(list)
	return list, nil
}
