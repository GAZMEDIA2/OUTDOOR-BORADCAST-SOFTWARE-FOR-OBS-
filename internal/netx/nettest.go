package netx

import (
	"net"
	"net/netip"
	"time"
)

// TestResult is the outcome of the network test, in plain language.
type TestResult struct {
	At           time.Time `json:"at"`
	IPv6Address  string    `json:"ipv6Address"`  // public IPv6 of this computer, if any
	IPv6Internet bool      `json:"ipv6Internet"` // IPv6 actually reaches the internet
	LocalIPv4    string    `json:"localIpv4"`
	PublicIPv4   string    `json:"publicIpv4"`
	SharedIPv4   bool      `json:"sharedIpv4"` // behind a router / shared IP
	Gate         string    `json:"gate"`       // "relaxed", "strict" or "unknown"
	Verdict      string    `json:"verdict"`    // "great", "good", "limited"
	Summary      string    `json:"summary"`
	Advice       []string  `json:"advice"`
}

// RunTest checks how this computer's internet connection will behave for a
// direct van <-> studio link. allowSTUN=false skips the public address check.
func RunTest(allowSTUN bool, servers []string) TestResult {
	if servers == nil {
		servers = DefaultSTUNServers
	}
	r := TestResult{At: time.Now(), Gate: "unknown"}
	for _, a := range LocalAddrs() {
		if a.Is6() && !a.IsPrivate() && r.IPv6Address == "" {
			r.IPv6Address = a.String()
		}
		if a.Is4() && r.LocalIPv4 == "" {
			r.LocalIPv4 = a.String()
		}
	}
	if allowSTUN {
		// IPv4: ask two different mirrors from the same port. Same public
		// port both times = a "relaxed" gate that direct dialling gets through.
		if c, err := net.ListenUDP("udp4", nil); err == nil {
			var seen []netip.AddrPort
			for _, srv := range servers {
				if a, ok := stunOnce(c, "udp4", srv); ok {
					seen = append(seen, a)
				}
				if len(seen) == 2 {
					break
				}
			}
			c.Close()
			if len(seen) > 0 {
				r.PublicIPv4 = seen[0].Addr().String()
				r.SharedIPv4 = r.PublicIPv4 != r.LocalIPv4
			}
			if len(seen) == 2 {
				if seen[0].Port() == seen[1].Port() {
					r.Gate = "relaxed"
				} else {
					r.Gate = "strict"
				}
			}
		}
		if r.IPv6Address != "" {
			if c, err := net.ListenUDP("udp6", nil); err == nil {
				for _, srv := range servers {
					if _, ok := stunOnce(c, "udp6", srv); ok {
						r.IPv6Internet = true
						break
					}
				}
				c.Close()
			}
		}
	} else if r.IPv6Address != "" {
		r.IPv6Internet = true // assume; cannot check without a mirror
	}

	switch {
	case r.IPv6Internet:
		r.Verdict = "great"
		r.Summary = "Excellent: this connection has working IPv6, the most reliable route for Van Link."
	case r.PublicIPv4 != "" && !r.SharedIPv4:
		r.Verdict = "great"
		r.Summary = "Excellent: this computer has its own public internet address."
	case r.Gate == "relaxed":
		r.Verdict = "good"
		r.Summary = "Good: this connection allows direct dialling over IPv4."
	case r.Gate == "strict":
		r.Verdict = "limited"
		r.Summary = "Limited: this network is strict. Van Link will still connect if the other end has IPv6, a public address, or a relaxed network."
	case r.PublicIPv4 == "" && r.IPv6Address == "":
		r.Verdict = "limited"
		r.Summary = "No internet connection was detected."
	default:
		r.Verdict = "good"
		r.Summary = "This connection should work for Van Link."
	}
	if !r.IPv6Internet {
		r.Advice = append(r.Advice, "If your internet provider or router offers IPv6, turning it on makes Van Link connect more reliably.")
	}
	if r.Gate == "strict" {
		r.Advice = append(r.Advice, "Strict networks are common on mobile data. Starlink or a fixed line at the other end usually solves it.")
	}
	return r
}

func stunOnce(c *net.UDPConn, network, server string) (netip.AddrPort, bool) {
	dst, err := ResolveUDP(network, server)
	if err != nil {
		return netip.AddrPort{}, false
	}
	buf := make([]byte, 1500)
	for try := 0; try < 2; try++ {
		tx := NewTxID()
		if _, err := c.WriteToUDPAddrPort(BindingRequest(tx), dst); err != nil {
			return netip.AddrPort{}, false
		}
		c.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))
		for {
			n, _, err := c.ReadFromUDPAddrPort(buf)
			if err != nil {
				break
			}
			if got, a, err := ParseBindingResponse(buf[:n]); err == nil && got == tx {
				return netip.AddrPortFrom(a.Addr().Unmap(), a.Port()), true
			}
		}
	}
	return netip.AddrPort{}, false
}
