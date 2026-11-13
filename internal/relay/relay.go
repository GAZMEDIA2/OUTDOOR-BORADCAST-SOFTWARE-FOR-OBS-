// Package relay is Van Link's optional relay: when a van and its studio
// cannot reach each other directly (for example both behind carrier-grade
// NAT), each end registers with the relay and the relay passes datagrams
// between the two. It never decrypts anything: the video stays SRT-encrypted
// and the status messages stay signed with the pairing key, end to end.
//
// Only signed-in Van Link users can use it: every registration carries the
// computer's licence, which the relay checks against the website's public key.
package relay

import (
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/GAZMEDIA2/vanlink/internal/licence"
)

// Wire format (same first byte as every Van Link datagram):
//
//	join:   magic | TJoin | sid(8) | role ('s' or 'v') | licence
//	reply:  magic | TReply | sid(8) | result (ROK or RRefused)
//
// Any other datagram from a registered address is passed to the other end
// of the same session unchanged.
const (
	Magic    = 0xB7
	TJoin    = 0x10
	TReply   = 0x11
	ROK      = 0
	RRefused = 1

	SidLen = 8
)

// DefaultPort is the relay's UDP port.
const DefaultPort = 47800

// JoinPacket builds a registration for one end of a session.
func JoinPacket(sid []byte, studio bool, lic string) []byte {
	role := byte('v')
	if studio {
		role = 's'
	}
	out := append([]byte{Magic, TJoin}, sid...)
	out = append(out, role)
	return append(out, lic...)
}

// ParseReply reads the relay's answer to a join.
func ParseReply(pkt []byte, sid []byte) (ok, isReply bool) {
	if len(pkt) != 3+SidLen || pkt[0] != Magic || pkt[1] != TReply || string(pkt[2:2+SidLen]) != string(sid) {
		return false, false
	}
	return pkt[2+SidLen] == ROK, true
}

// Server passes datagrams between the two ends of each session.
type Server struct {
	PublicKey string           // licence signing key (default: the one built into Van Link)
	Now       func() time.Time // for tests
	Logf      func(string, ...any)
	MaxIdle   time.Duration // an end that stops registering is forgotten after this

	mu       sync.Mutex
	sessions map[string]*session
	byAddr   map[netip.AddrPort]*end
	checked  map[string]time.Time // licences already verified -> valid until
}

type session struct {
	ends [2]*end // 0 = studio, 1 = van
}

type end struct {
	sess *session
	slot int
	addr netip.AddrPort
	seen time.Time
}

func (s *Server) init() {
	if s.PublicKey == "" {
		s.PublicKey = licence.PublicKey
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.Logf == nil {
		s.Logf = func(string, ...any) {}
	}
	if s.MaxIdle == 0 {
		s.MaxIdle = 30 * time.Second
	}
	s.sessions = map[string]*session{}
	s.byAddr = map[netip.AddrPort]*end{}
	s.checked = map[string]time.Time{}
}

// Serve runs until the connection is closed.
func (s *Server) Serve(conn *net.UDPConn) error {
	s.init()
	go func() {
		for range time.Tick(10 * time.Second) {
			s.expire()
		}
	}()
	buf := make([]byte, 65536)
	for {
		n, from, err := conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			return err
		}
		from = netip.AddrPortFrom(from.Addr().Unmap(), from.Port())
		pkt := buf[:n]
		if n >= 3+SidLen && pkt[0] == Magic && pkt[1] == TJoin {
			res := s.join(pkt, from)
			reply := append([]byte{Magic, TReply}, pkt[2:2+SidLen]...)
			_, _ = conn.WriteToUDPAddrPort(append(reply, res), from)
			continue
		}
		if to, ok := s.route(from); ok {
			_, _ = conn.WriteToUDPAddrPort(pkt, to)
		}
	}
}

func (s *Server) licenceOK(lic string) bool {
	now := s.Now()
	s.mu.Lock()
	until, seen := s.checked[lic]
	s.mu.Unlock()
	if seen {
		return now.Before(until)
	}
	p, err := licence.Parse(lic, s.PublicKey)
	until = time.Time{}
	if err == nil && p.Plan != "none" && p.Plan != "" && (p.Plan == "owner" || p.Expires > now.Unix()) {
		until = time.Unix(p.ValidUntil, 0)
		if p.Plan != "owner" && p.Expires < p.ValidUntil {
			until = time.Unix(p.Expires, 0)
		}
	}
	s.mu.Lock()
	if len(s.checked) > 10000 {
		s.checked = map[string]time.Time{}
	}
	s.checked[lic] = until
	s.mu.Unlock()
	return now.Before(until)
}

func (s *Server) join(pkt []byte, from netip.AddrPort) byte {
	sid := string(pkt[2 : 2+SidLen])
	slot := 1
	switch pkt[2+SidLen] {
	case 's':
		slot = 0
	case 'v':
	default:
		return RRefused
	}
	if !s.licenceOK(string(pkt[3+SidLen:])) {
		return RRefused
	}
	now := s.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	ss := s.sessions[sid]
	if ss == nil {
		ss = &session{}
		s.sessions[sid] = ss
	}
	e := ss.ends[slot]
	if e != nil && e.addr != from {
		delete(s.byAddr, e.addr) // that end moved (new network or restart)
		s.Logf("session %x: %s moved", sid, roleName(slot))
		e = nil
	}
	if old := s.byAddr[from]; old != nil && old.sess != ss {
		old.sess.ends[old.slot] = nil // this address now belongs to another session
	}
	if e == nil {
		e = &end{sess: ss, slot: slot, addr: from}
		ss.ends[slot] = e
		s.byAddr[from] = e
		s.Logf("session %x: %s joined from %s", sid, roleName(slot), from)
	}
	e.seen = now
	return ROK
}

func (s *Server) route(from netip.AddrPort) (netip.AddrPort, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.byAddr[from]
	if e == nil {
		return netip.AddrPort{}, false
	}
	other := e.sess.ends[1-e.slot]
	if other == nil {
		return netip.AddrPort{}, false
	}
	return other.addr, true
}

func (s *Server) expire() {
	now := s.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for sid, ss := range s.sessions {
		for i, e := range ss.ends {
			if e != nil && now.Sub(e.seen) > s.MaxIdle {
				delete(s.byAddr, e.addr)
				ss.ends[i] = nil
			}
		}
		if ss.ends[0] == nil && ss.ends[1] == nil {
			delete(s.sessions, sid)
		}
	}
}

// Sessions reports how many sessions have both ends registered.
func (s *Server) Sessions() (paired, waiting int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ss := range s.sessions {
		if ss.ends[0] != nil && ss.ends[1] != nil {
			paired++
		} else {
			waiting++
		}
	}
	return
}

func roleName(slot int) string {
	if slot == 0 {
		return "studio"
	}
	return "van"
}
