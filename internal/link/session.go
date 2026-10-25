// Package link is Van Link's connection engine. One Session joins one van to
// one studio: it finds a direct path (IPv6, IPv4 hole punching, LAN), keeps it
// open, carries OBS's SRT video through it untouched, and swaps status both
// ways once a second.
package link

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GAZMEDIA2/vanlink/internal/netx"
	"github.com/GAZMEDIA2/vanlink/internal/pairing"
)

// Role of this end.
type Role string

const (
	Studio Role = "studio"
	Van    Role = "van"
)

// Config sets up a Session.
type Config struct {
	Role Role
	Name string // shown to the other side, e.g. "Studio" or "Van 1"

	// StudioCode: the studio creates it; the van receives it from the studio.
	StudioCode pairing.Code

	// OBSAddr: van = where Van Link listens for OBS's SRT output;
	// studio = where OBS's Van Link source listens.
	OBSAddr netip.AddrPort

	ListenPort  int      // 0 = any free port
	UseSTUN     bool     // learn the public IPv4 address (needed for IPv4 hole punching)
	STUNServers []string // default netx.DefaultSTUNServers

	Logf func(format string, args ...any)
}

// Session is one van <-> studio link.
type Session struct {
	cfg  Config
	key  []byte
	sid  []byte
	conn *net.UDPConn // talks to the other end
	obs  *net.UDPConn // talks to OBS on this computer

	mu            sync.Mutex
	candidates    []string         // our addresses, put in our code
	peerCands     []netip.AddrPort // other end's addresses, from its code
	peer          netip.AddrPort   // chosen path
	peerSeen      time.Time
	connectedAt   time.Time
	everConnected bool
	route         string
	obsPeer       netip.AddrPort // van: OBS's sending socket
	obsSeen       time.Time
	peerStatus    Status
	lastPeerT     int64     // peer's timestamp to echo
	lastPeerAt    time.Time // when we got it
	rtt           time.Duration
	drops         int
	events        []Event
	stunWait      map[[12]byte]chan netip.AddrPort
	closed        bool

	meter    srtMeter
	disabled atomic.Bool // licence lock: video is not carried while set
	done     chan struct{}
	wg       sync.WaitGroup
}

// Status is what each end tells the other every second.
type Status struct {
	Name      string  `json:"n"`
	Role      Role    `json:"r"`
	Live      bool    `json:"live"` // van: sending video; studio: receiving it
	Kbps      float64 `json:"kbps"` // video rate seen at this end
	RepairPct float64 `json:"rep"`  // share of video packets that had to be re-sent
	Signal    int     `json:"sig"`  // 0-4 bars, as computed at this end
	OBS       bool    `json:"obs"`  // OBS is connected to Van Link at this end
	T         int64   `json:"t"`    // sender clock, ms
	E         int64   `json:"e"`    // echo of the other side's last t
	D         int64   `json:"d"`    // how long we held that echo, ms
}

// Event is one line of the connection history.
type Event struct {
	At   time.Time `json:"at"`
	Text string    `json:"text"`
}

// New opens the sockets and gathers our addresses. Call Start to run.
func New(cfg Config) (*Session, error) {
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	if cfg.STUNServers == nil {
		cfg.STUNServers = netx.DefaultSTUNServers
	}
	c := cfg.StudioCode
	if len(c.KeyBytes()) != 32 || len(c.SessionBytes()) != 8 {
		return nil, errors.New("invalid studio code")
	}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: cfg.ListenPort})
	if err != nil {
		return nil, fmt.Errorf("cannot open network port: %w", err)
	}
	var obs *net.UDPConn
	if cfg.Role == Van {
		obs, err = net.ListenUDP("udp", net.UDPAddrFromAddrPort(cfg.OBSAddr))
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("port %d is busy - is another Van Link already running? (%w)", cfg.OBSAddr.Port(), err)
		}
	} else {
		obs, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
		if err != nil {
			conn.Close()
			return nil, err
		}
	}
	s := &Session{
		cfg: cfg, key: c.KeyBytes(), sid: c.SessionBytes(),
		conn: conn, obs: obs,
		stunWait: map[[12]byte]chan netip.AddrPort{},
		done:     make(chan struct{}),
	}
	return s, nil
}

// Start runs the engine and gathers addresses (including the public IPv4
// address when STUN is allowed). It returns once our code is ready.
func (s *Session) Start() {
	s.wg.Add(3)
	go s.readPeer()
	go s.readOBS()
	go s.loop()
	s.gather()
}

// Port is the local UDP port used to reach the other end.
func (s *Session) Port() int { return s.conn.LocalAddr().(*net.UDPAddr).Port }

// Candidates are this end's addresses for its pairing code.
func (s *Session) Candidates() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.candidates...)
}

func (s *Session) gather() {
	port := uint16(s.Port())
	var out []string
	seen := map[string]bool{}
	add := func(ap netip.AddrPort) {
		k := ap.String()
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	for _, a := range netx.LocalAddrs() {
		add(netip.AddrPortFrom(a, port))
	}
	if s.cfg.UseSTUN {
		if pub, ok := s.stun(3 * time.Second); ok {
			// put the public address first after IPv6
			add(pub)
			s.event("Public address found: " + pub.Addr().String())
		} else {
			s.event("Could not learn the public IPv4 address (no internet, or blocked)")
		}
	}
	// order: public IPv6, public IPv4, then local addresses
	sortCands(out)
	s.mu.Lock()
	s.candidates = out
	s.mu.Unlock()
}

func sortCands(c []string) {
	rank := func(s string) int {
		ap, err := netip.ParseAddrPort(s)
		if err != nil {
			return 9
		}
		a := ap.Addr()
		switch {
		case a.Is6() && !a.IsPrivate():
			return 0
		case a.Is4() && !a.IsPrivate():
			return 1
		case a.Is4():
			return 2
		}
		return 3
	}
	for i := 1; i < len(c); i++ {
		for j := i; j > 0 && rank(c[j]) < rank(c[j-1]); j-- {
			c[j], c[j-1] = c[j-1], c[j]
		}
	}
}

// stun asks public address mirrors which IPv4 address and port we appear
// from, using the same socket the video will use.
func (s *Session) stun(timeout time.Duration) (netip.AddrPort, bool) {
	ch := make(chan netip.AddrPort, 4)
	var txs [][12]byte
	for _, srv := range s.cfg.STUNServers {
		dst, err := netx.ResolveUDP("udp4", srv)
		if err != nil {
			continue
		}
		tx := netx.NewTxID()
		s.mu.Lock()
		s.stunWait[tx] = ch
		s.mu.Unlock()
		txs = append(txs, tx)
		s.writeTo(netx.BindingRequest(tx), dst)
	}
	defer func() {
		s.mu.Lock()
		for _, tx := range txs {
			delete(s.stunWait, tx)
		}
		s.mu.Unlock()
	}()
	if len(txs) == 0 {
		return netip.AddrPort{}, false
	}
	select {
	case a := <-ch:
		return a, true
	case <-time.After(timeout):
		return netip.AddrPort{}, false
	case <-s.done:
		return netip.AddrPort{}, false
	}
}

// SetPeerCode gives the engine the other end's code (the van's reply code
// at the studio). The van already has the studio's code from its config.
func (s *Session) SetPeerCandidates(cands []string) {
	var out []netip.AddrPort
	for _, c := range cands {
		if ap, err := netip.ParseAddrPort(c); err == nil {
			out = append(out, netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port()))
		}
	}
	s.mu.Lock()
	s.peerCands = out
	s.mu.Unlock()
	s.event(fmt.Sprintf("Dialling the other side (%d possible routes)", len(out)))
}

func (s *Session) writeTo(b []byte, dst netip.AddrPort) {
	if dst.Addr().Is4() && s.conn.LocalAddr().(*net.UDPAddr).IP.To4() == nil {
		dst = netip.AddrPortFrom(netip.AddrFrom16(dst.Addr().As16()), dst.Port())
	}
	_, _ = s.conn.WriteToUDPAddrPort(b, dst)
}

// readPeer handles everything arriving from the internet side.
func (s *Session) readPeer() {
	defer s.wg.Done()
	buf := make([]byte, 65536)
	for {
		n, from, err := s.conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			if s.isClosed() {
				return
			}
			continue
		}
		from = netip.AddrPortFrom(from.Addr().Unmap(), from.Port())
		pkt := buf[:n]
		now := time.Now()
		if netx.IsSTUN(pkt) {
			if tx, addr, err := netx.ParseBindingResponse(pkt); err == nil {
				s.mu.Lock()
				ch := s.stunWait[tx]
				s.mu.Unlock()
				if ch != nil {
					select {
					case ch <- addr:
					default:
					}
				}
			}
			continue
		}
		if len(pkt) < 2 || pkt[0] != magic {
			continue
		}
		if pkt[1] == tData {
			s.mu.Lock()
			ok := from == s.peer
			if ok {
				s.peerSeen = now
			}
			s.mu.Unlock()
			if ok {
				s.fromPeerData(pkt[2:], now)
			}
			continue
		}
		typ, body, err := open(s.key, s.sid, pkt)
		if err != nil {
			continue // not our session: ignore silently
		}
		switch typ {
		case tProbe:
			s.writeTo(seal(s.key, s.sid, tAck, body), from)
			s.adopt(from, now)
		case tAck:
			s.adopt(from, now)
		case tStatus:
			var st Status
			if json.Unmarshal(body, &st) == nil {
				s.mu.Lock()
				if from == s.peer {
					s.peerSeen = now
					s.peerStatus = st
					s.lastPeerT, s.lastPeerAt = st.T, now
					if st.E > 0 {
						r := time.Duration(now.UnixMilli()-st.E-st.D) * time.Millisecond
						if r >= 0 && r < 30*time.Second {
							if s.rtt == 0 {
								s.rtt = r
							} else {
								s.rtt = (s.rtt*7 + r*3) / 10
							}
						}
					}
				}
				s.mu.Unlock()
			}
		case tBye:
			s.mu.Lock()
			if from == s.peer {
				s.peer = netip.AddrPort{}
				s.connectedAt = time.Time{}
			}
			s.mu.Unlock()
			s.event("The other side stopped Van Link")
		}
	}
}

// adopt picks the path a verified packet came from. IPv6 beats IPv4, and a
// silent path is replaced by a live one (e.g. after a network change).
func (s *Session) adopt(from netip.AddrPort, now time.Time) {
	s.mu.Lock()
	cur := s.peer
	silent := now.Sub(s.peerSeen) > 3*time.Second
	switch {
	case !cur.IsValid() || silent:
		s.peer = from
	case from == cur:
	case from.Addr().Is6() && !from.Addr().IsPrivate() && cur.Addr().Is4():
		s.peer = from
	default:
		s.mu.Unlock()
		return
	}
	changed := s.peer != cur
	s.peerSeen = now
	if changed {
		s.route = routeName(s.peer.Addr())
		if s.connectedAt.IsZero() {
			s.connectedAt = now
		}
		if s.everConnected && !cur.IsValid() || s.everConnected && silent {
			s.mu.Unlock()
			s.event("Reconnected (" + routeName(from.Addr()) + ")")
			return
		}
		first := !s.everConnected
		s.everConnected = true
		s.mu.Unlock()
		if first {
			s.event("Connected directly (" + routeName(from.Addr()) + ")")
		} else {
			s.event("Switched to a better route (" + routeName(from.Addr()) + ")")
		}
		return
	}
	s.mu.Unlock()
}

func routeName(a netip.Addr) string {
	switch {
	case a.Is6() && !a.IsPrivate():
		return "IPv6 direct"
	case a.Is4() && a.IsPrivate():
		return "same local network"
	case a.IsPrivate():
		return "local network"
	default:
		return "IPv4 direct"
	}
}

// SetEnabled turns video forwarding on or off (used when the licence locks).
func (s *Session) SetEnabled(on bool) { s.disabled.Store(!on) }

// fromPeerData: SRT arriving over the internet.
func (s *Session) fromPeerData(b []byte, now time.Time) {
	if s.disabled.Load() {
		return
	}
	if s.cfg.Role == Studio {
		s.meter.media(b, now)
		_, _ = s.obs.WriteToUDPAddrPort(b, s.cfg.OBSAddr)
		return
	}
	s.meter.back(b)
	s.mu.Lock()
	dst := s.obsPeer
	s.mu.Unlock()
	if dst.IsValid() {
		_, _ = s.obs.WriteToUDPAddrPort(b, dst)
	}
}

// readOBS: SRT from OBS on this computer, sent on to the other end.
func (s *Session) readOBS() {
	defer s.wg.Done()
	buf := make([]byte, 65536)
	for {
		n, from, err := s.obs.ReadFromUDPAddrPort(buf)
		if err != nil {
			if s.isClosed() {
				return
			}
			continue
		}
		now := time.Now()
		pkt := buf[:n]
		s.mu.Lock()
		if s.cfg.Role == Van {
			s.obsPeer = from
		}
		s.obsSeen = now
		peer := s.peer
		s.mu.Unlock()
		if s.cfg.Role == Van {
			s.meter.media(pkt, now)
		} else {
			s.meter.back(pkt)
		}
		if peer.IsValid() && !s.disabled.Load() {
			s.writeTo(wrapData(pkt), peer)
		}
	}
}

// loop: dialling, keep-alive and status, several times a second.
func (s *Session) loop() {
	defer s.wg.Done()
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	nonce := make([]byte, 8)
	var tick int
	lastSTUN := time.Now()
	wasUp := false
	for {
		select {
		case <-s.done:
			return
		case now := <-t.C:
			tick++
			s.mu.Lock()
			up := s.peer.IsValid() && now.Sub(s.peerSeen) < 4*time.Second
			cands := append([]netip.AddrPort(nil), s.peerCands...)
			peer := s.peer
			s.mu.Unlock()

			if wasUp && !up {
				s.mu.Lock()
				s.drops++
				s.connectedAt = time.Time{}
				s.mu.Unlock()
				s.event("Connection lost - reconnecting...")
			}
			wasUp = up

			// Dial: every 250 ms until connected, then lightly every 2 s so a
			// better route (or a changed address) is still found.
			if !up || tick%8 == 0 {
				binary.BigEndian.PutUint64(nonce, uint64(now.UnixNano()))
				p := seal(s.key, s.sid, tProbe, nonce)
				for _, c := range cands {
					s.writeTo(p, c)
				}
				if peer.IsValid() {
					s.writeTo(p, peer)
				}
			}
			// Status every second.
			if tick%4 == 0 {
				s.meter.tick(now)
				if peer.IsValid() {
					b, _ := json.Marshal(s.localStatus(now))
					s.writeTo(seal(s.key, s.sid, tStatus, b), peer)
				}
			}
			// Keep our public mapping alive while we wait for the other side.
			if s.cfg.UseSTUN && !up && now.Sub(lastSTUN) > 20*time.Second {
				lastSTUN = now
				go s.stun(2 * time.Second)
			}
		}
	}
}

func (s *Session) localStatus(now time.Time) Status {
	v := s.meter.view()
	s.mu.Lock()
	defer s.mu.Unlock()
	live := !v.LastMedia.IsZero() && now.Sub(v.LastMedia) < 2*time.Second
	st := Status{
		Name: s.cfg.Name, Role: s.cfg.Role, Live: live,
		Kbps: v.Kbps, RepairPct: v.RepairPct,
		OBS: !s.obsSeen.IsZero() && now.Sub(s.obsSeen) < 5*time.Second,
		T:   now.UnixMilli(),
	}
	if s.lastPeerT > 0 {
		st.E = s.lastPeerT
		st.D = now.Sub(s.lastPeerAt).Milliseconds()
	}
	st.Signal = SignalScore(s.peer.IsValid() && now.Sub(s.peerSeen) < 4*time.Second,
		float64(s.rttLocked(v).Milliseconds()), v.RepairPct, v.Kbps, v.LinkKbps)
	return st
}

func (s *Session) rttLocked(v meterView) time.Duration {
	if v.SRTRTT > 0 {
		return v.SRTRTT
	}
	return s.rtt
}

func (s *Session) event(text string) {
	s.cfg.Logf("%s", text)
	s.mu.Lock()
	s.events = append(s.events, Event{At: time.Now(), Text: text})
	if len(s.events) > 50 {
		s.events = s.events[len(s.events)-50:]
	}
	s.mu.Unlock()
}

func (s *Session) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Note adds a line to the connection history shown on the screen.
func (s *Session) Note(text string) { s.event(text) }

// Close says goodbye to the other side and stops.
func (s *Session) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	peer := s.peer
	s.mu.Unlock()
	if peer.IsValid() {
		s.writeTo(seal(s.key, s.sid, tBye, nil), peer)
	}
	close(s.done)
	s.conn.Close()
	s.obs.Close()
	s.wg.Wait()
}
