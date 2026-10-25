// Package netx holds the small networking helpers Van Link needs: a minimal
// STUN client (only to learn "what address do I appear from"), local address
// discovery and the network test.
package netx

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"net/netip"
)

const stunMagic = 0x2112A442

// DefaultSTUNServers are public address mirrors. They only ever see a
// 20-byte "what is my address?" question; video never goes near them.
var DefaultSTUNServers = []string{
	"stun.l.google.com:19302",
	"stun.cloudflare.com:3478",
	"stun1.l.google.com:19302",
}

// NewTxID returns a random 12-byte STUN transaction id.
func NewTxID() [12]byte {
	var t [12]byte
	_, _ = rand.Read(t[:])
	return t
}

// BindingRequest builds a STUN Binding Request.
func BindingRequest(tx [12]byte) []byte {
	b := make([]byte, 20)
	binary.BigEndian.PutUint16(b[0:], 0x0001)
	binary.BigEndian.PutUint16(b[2:], 0)
	binary.BigEndian.PutUint32(b[4:], stunMagic)
	copy(b[8:], tx[:])
	return b
}

// IsSTUN reports whether a datagram looks like a STUN message.
func IsSTUN(b []byte) bool {
	return len(b) >= 20 && b[0]&0xC0 == 0 && binary.BigEndian.Uint32(b[4:]) == stunMagic
}

// ParseBindingResponse extracts the transaction id and mapped address.
func ParseBindingResponse(b []byte) (tx [12]byte, addr netip.AddrPort, err error) {
	if !IsSTUN(b) || binary.BigEndian.Uint16(b[0:]) != 0x0101 {
		return tx, addr, errors.New("not a binding success response")
	}
	copy(tx[:], b[8:20])
	n := int(binary.BigEndian.Uint16(b[2:]))
	if 20+n > len(b) {
		return tx, addr, errors.New("truncated")
	}
	attrs := b[20 : 20+n]
	var fallback netip.AddrPort
	for len(attrs) >= 4 {
		t := binary.BigEndian.Uint16(attrs[0:])
		l := int(binary.BigEndian.Uint16(attrs[2:]))
		if 4+l > len(attrs) {
			break
		}
		v := attrs[4 : 4+l]
		switch t {
		case 0x0020, 0x8020: // XOR-MAPPED-ADDRESS
			if a, ok := parseAddr(v, true, tx); ok {
				return tx, a, nil
			}
		case 0x0001: // MAPPED-ADDRESS
			if a, ok := parseAddr(v, false, tx); ok {
				fallback = a
			}
		}
		pad := (4 - l%4) % 4
		if 4+l+pad > len(attrs) {
			break
		}
		attrs = attrs[4+l+pad:]
	}
	if fallback.IsValid() {
		return tx, fallback, nil
	}
	return tx, addr, errors.New("no mapped address")
}

func parseAddr(v []byte, xor bool, tx [12]byte) (netip.AddrPort, bool) {
	if len(v) < 8 {
		return netip.AddrPort{}, false
	}
	fam := v[1]
	port := binary.BigEndian.Uint16(v[2:])
	if xor {
		port ^= uint16(stunMagic >> 16)
	}
	switch fam {
	case 0x01:
		var ip [4]byte
		copy(ip[:], v[4:8])
		if xor {
			var m [4]byte
			binary.BigEndian.PutUint32(m[:], stunMagic)
			for i := range ip {
				ip[i] ^= m[i]
			}
		}
		return netip.AddrPortFrom(netip.AddrFrom4(ip), port), true
	case 0x02:
		if len(v) < 20 {
			return netip.AddrPort{}, false
		}
		var ip [16]byte
		copy(ip[:], v[4:20])
		if xor {
			var m [16]byte
			binary.BigEndian.PutUint32(m[:], stunMagic)
			copy(m[4:], tx[:])
			for i := range ip {
				ip[i] ^= m[i]
			}
		}
		return netip.AddrPortFrom(netip.AddrFrom16(ip), port), true
	}
	return netip.AddrPort{}, false
}

// ResolveUDP resolves "host:port" for the given network ("udp4"/"udp6").
func ResolveUDP(network, hostport string) (netip.AddrPort, error) {
	a, err := net.ResolveUDPAddr(network, hostport)
	if err != nil {
		return netip.AddrPort{}, err
	}
	ap := a.AddrPort()
	return netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port()), nil
}
