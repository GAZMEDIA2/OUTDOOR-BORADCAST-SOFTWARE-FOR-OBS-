package link

import "time"

// Snapshot is everything the monitoring screen shows, for both ends.
type Snapshot struct {
	Role       Role    `json:"role"`
	Name       string  `json:"name"`
	Connected  bool    `json:"connected"`
	Route      string  `json:"route"`
	LinkSecs   int     `json:"linkSecs"`
	Drops      int     `json:"drops"`
	DelayMS    int     `json:"delayMs"`   // round trip between the two ends
	Kbps       float64 `json:"kbps"`      // video rate seen here
	RepairPct  float64 `json:"repairPct"` // video packets re-sent
	LinkKbps   float64 `json:"linkKbps"`  // estimated capacity (from SRT)
	Encrypted  bool    `json:"encrypted"`
	LiveHere   bool    `json:"liveHere"` // van: sending; studio: receiving
	OBSHere    bool    `json:"obsHere"`  // OBS is exchanging video with Van Link here
	Signal     int     `json:"signal"`
	SignalWord string  `json:"signalWord"`
	RecLatency int     `json:"recLatencyMs"`
	TotalMB    float64 `json:"totalMb"`

	Peer        Status `json:"peer"`
	PeerSignalW string `json:"peerSignalWord"`
	PeerLive    bool   `json:"peerLive"` // van: studio confirms it receives; studio: van is sending

	Candidates []string `json:"candidates"`
	Events     []Event  `json:"events"`
}

// Snapshot returns the current state.
func (s *Session) Snapshot() Snapshot {
	now := time.Now()
	v := s.meter.view()
	st := s.localStatus(now)
	s.mu.Lock()
	defer s.mu.Unlock()
	up := s.peer.IsValid() && now.Sub(s.peerSeen) < 4*time.Second
	rtt := s.rttLocked(v)
	snap := Snapshot{
		Role: s.cfg.Role, Name: s.cfg.Name, Connected: up, Route: s.route,
		Drops: s.drops, DelayMS: int(rtt.Milliseconds()),
		Kbps: v.Kbps, RepairPct: v.RepairPct, LinkKbps: v.LinkKbps,
		Encrypted: v.Encrypted, LiveHere: st.Live, OBSHere: st.OBS,
		Signal: st.Signal, SignalWord: SignalWord(st.Signal),
		RecLatency: RecommendedLatencyMS(float64(rtt.Milliseconds()), v.RepairPct),
		TotalMB:    v.TotalMB,
		Candidates: append([]string(nil), s.candidates...),
		Events:     append([]Event(nil), s.events...),
	}
	if up {
		snap.LinkSecs = int(now.Sub(s.connectedAt).Seconds())
		snap.Peer = s.peerStatus
		snap.PeerLive = s.peerStatus.Live
	} else {
		snap.Route = ""
	}
	snap.PeerSignalW = SignalWord(snap.Peer.Signal)
	return snap
}
