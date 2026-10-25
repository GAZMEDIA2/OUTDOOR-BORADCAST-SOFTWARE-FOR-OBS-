package netx

import (
	"net"
	"net/netip"
	"sort"
)

// LocalAddrs lists this computer's usable addresses: public IPv6 first,
// then IPv4 (local network), then private IPv6. Loopback and link-local
// addresses are skipped.
func LocalAddrs() []netip.Addr {
	var v6pub, v4, v6priv []netip.Addr
	ifs, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, ifc := range ifs {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			pfx, err := netip.ParsePrefix(a.String())
			if err != nil {
				continue
			}
			ip := pfx.Addr().Unmap()
			switch {
			case ip.IsLoopback(), ip.IsLinkLocalUnicast(), ip.IsMulticast(), ip.IsUnspecified():
				continue
			case ip.Is4():
				if ip.Compare(netip.MustParseAddr("169.254.0.0")) >= 0 && ip.Compare(netip.MustParseAddr("169.254.255.255")) <= 0 {
					continue
				}
				v4 = append(v4, ip)
			case ip.IsPrivate(): // fc00::/7
				v6priv = append(v6priv, ip)
			case ip.IsGlobalUnicast():
				v6pub = append(v6pub, ip)
			}
		}
	}
	sortAddrs(v6pub)
	sortAddrs(v4)
	sortAddrs(v6priv)
	out := append(append(v6pub, v4...), v6priv...)
	return out
}

// HasPublicIPv6 reports whether any interface has a global IPv6 address.
func HasPublicIPv6() bool {
	for _, a := range LocalAddrs() {
		if a.Is6() && !a.IsPrivate() {
			return true
		}
	}
	return false
}

func sortAddrs(a []netip.Addr) {
	sort.Slice(a, func(i, j int) bool { return a[i].Less(a[j]) })
}
