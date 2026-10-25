package link

import (
	"encoding/binary"
	"sync"
	"time"
)

// srtMeter watches the SRT packets passing through Van Link and works out
// bitrate, delay, repaired data and encryption - without touching them.
type srtMeter struct {
	mu sync.Mutex

	dataBytes, dataPkts, retransPkts uint64 // video direction (van -> studio)
	naks                             uint64
	encrypted                        bool

	srtRTT    time.Duration // from SRT full ACKs
	linkCap   float64       // estimated link capacity, bytes/s (from ACKs)
	lastMedia time.Time

	// per-second snapshot
	snapT                            time.Time
	snapBytes, snapPkts, snapRetrans uint64
	kbps                             float64
	repairPct                        float64
}

const srtPayloadGuess = 1316 // typical SRT live payload, for capacity estimate

// media is called for every datagram travelling van -> studio.
func (m *srtMeter) media(b []byte, now time.Time) {
	if len(b) < 16 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastMedia = now
	if b[0]&0x80 != 0 { // control packet
		return
	}
	m.dataPkts++
	m.dataBytes += uint64(len(b))
	if (b[4]>>3)&0x03 != 0 {
		m.encrypted = true
	}
	if (b[4]>>2)&0x01 != 0 {
		m.retransPkts++
	}
}

// back is called for every datagram travelling studio -> van (ACKs, NAKs).
func (m *srtMeter) back(b []byte) {
	if len(b) < 16 || b[0]&0x80 == 0 {
		return
	}
	typ := binary.BigEndian.Uint16(b[0:]) & 0x7FFF
	m.mu.Lock()
	defer m.mu.Unlock()
	switch typ {
	case 0x0002: // ACK
		if len(b) >= 16+28 {
			rtt := binary.BigEndian.Uint32(b[20:])
			capPkts := binary.BigEndian.Uint32(b[36:])
			if rtt > 0 && rtt < 30_000_000 {
				m.srtRTT = time.Duration(rtt) * time.Microsecond
			}
			if capPkts > 0 {
				m.linkCap = float64(capPkts) * srtPayloadGuess
			}
		}
	case 0x0003: // NAK = data went missing and was asked for again
		m.naks++
	}
}

// tick recomputes the per-second figures.
func (m *srtMeter) tick(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.snapT.IsZero() {
		m.snapT, m.snapBytes, m.snapPkts, m.snapRetrans = now, m.dataBytes, m.dataPkts, m.retransPkts
		return
	}
	dt := now.Sub(m.snapT).Seconds()
	if dt < 0.5 {
		return
	}
	db := m.dataBytes - m.snapBytes
	dp := m.dataPkts - m.snapPkts
	dr := m.retransPkts - m.snapRetrans
	m.kbps = float64(db) * 8 / 1000 / dt
	if dp > 0 {
		// smooth so one bad second does not make the screen jump
		m.repairPct = 0.6*m.repairPct + 0.4*(float64(dr)*100/float64(dp))
	} else {
		m.repairPct *= 0.6
	}
	m.snapT, m.snapBytes, m.snapPkts, m.snapRetrans = now, m.dataBytes, m.dataPkts, m.retransPkts
}

type meterView struct {
	Kbps      float64
	RepairPct float64
	SRTRTT    time.Duration
	LinkKbps  float64
	Encrypted bool
	LastMedia time.Time
	TotalMB   float64
}

func (m *srtMeter) view() meterView {
	m.mu.Lock()
	defer m.mu.Unlock()
	return meterView{
		Kbps: m.kbps, RepairPct: m.repairPct, SRTRTT: m.srtRTT,
		LinkKbps: m.linkCap * 8 / 1000, Encrypted: m.encrypted,
		LastMedia: m.lastMedia, TotalMB: float64(m.dataBytes) / 1e6,
	}
}

// SignalScore turns delay, repaired data and spare capacity into 0-4 bars.
func SignalScore(connected bool, rttMS, repairPct, sendKbps, capKbps float64) int {
	if !connected {
		return 0
	}
	s := 4
	switch {
	case repairPct > 8:
		s -= 3
	case repairPct > 3:
		s -= 2
	case repairPct > 1:
		s--
	}
	switch {
	case rttMS > 1200:
		s -= 2
	case rttMS > 600:
		s--
	}
	if capKbps > 0 && sendKbps > 0 {
		switch r := capKbps / sendKbps; {
		case r < 1.0:
			s -= 2
		case r < 1.3:
			s--
		}
	}
	if s < 1 {
		s = 1
	}
	return s
}

// SignalWord is the plain-language label for a score.
func SignalWord(s int) string {
	return [...]string{"No signal", "Poor", "Weak", "Good", "Excellent"}[s]
}

// RecommendedLatencyMS follows SRT's rule of thumb: about 4x the round trip,
// more when data is being lost, never below 120 ms.
func RecommendedLatencyMS(rttMS, repairPct float64) int {
	f := 4.0
	switch {
	case repairPct > 8:
		f = 8
	case repairPct > 3:
		f = 6
	case repairPct > 1:
		f = 5
	}
	v := int(rttMS*f/100+0.999) * 100
	if v < 120 {
		v = 120
	}
	if v > 8000 {
		v = 8000
	}
	return v
}
