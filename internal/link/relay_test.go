package link

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/GAZMEDIA2/vanlink/internal/relay"
)

// testRelay starts a relay on localhost that trusts licences signed by the
// returned function.
func testRelay(t *testing.T) (addr string, sign func(plan string, expires int64) string) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	srv := &relay.Server{PublicKey: base64.StdEncoding.EncodeToString(pub), Logf: t.Logf}
	go srv.Serve(conn)
	sign = func(plan string, expires int64) string {
		body, _ := json.Marshal(map[string]any{"plan": plan, "expires": expires, "valid_until": time.Now().Add(time.Hour).Unix()})
		return base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, body))
	}
	return conn.LocalAddr().String(), sign
}

// Both ends can't reach each other (the van's only route leads nowhere and
// the studio has none): the link must come up through the relay.
func relayPair(t *testing.T, studioOBS, vanOBS int, relayAddr, lic string, vanCands []string) (*Session, *Session) {
	code := studioCode()
	st, err := New(Config{Role: Studio, Name: "Studio", StudioCode: code,
		OBSAddr: netip.MustParseAddrPort("127.0.0.1:" + strconv.Itoa(studioOBS)),
		Relay:   relayAddr, RelayLicence: func() string { return lic }, RelayAfter: 500 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	vn, err := New(Config{Role: Van, Name: "Van 1", StudioCode: code,
		OBSAddr: netip.MustParseAddrPort("127.0.0.1:" + strconv.Itoa(vanOBS)),
		Relay:   relayAddr, RelayLicence: func() string { return lic }, RelayAfter: 500 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	st.Start()
	vn.Start()
	vn.SetPeerCandidates(vanCands)
	t.Cleanup(func() { st.Close(); vn.Close() })
	return st, vn
}

func waitConnected(st, vn *Session, d time.Duration) bool {
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		if st.Snapshot().Connected && vn.Snapshot().Connected {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func TestRelayWhenNoDirectRoute(t *testing.T) {
	addr, sign := testRelay(t)
	lic := sign("standard", time.Now().Add(24*time.Hour).Unix())
	studioOBS, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer studioOBS.Close()
	vanPort := freePort(t)
	dead := fmt.Sprintf("127.0.0.1:%d", freePort(t)) // nobody there
	st, vn := relayPair(t, studioOBS.LocalAddr().(*net.UDPAddr).Port, vanPort, addr, lic, []string{dead})
	if !waitConnected(st, vn, 8*time.Second) {
		t.Fatalf("did not connect through the relay: %+v", vn.Snapshot().Events)
	}
	if r := vn.Snapshot().Route; r != "Van Link relay" {
		t.Fatalf("route %q, want the relay", r)
	}
	if r := st.Snapshot().Route; r != "Van Link relay" {
		t.Fatalf("studio route %q, want the relay", r)
	}

	// Video both ways through the relay, unchanged.
	vanOBS, _ := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: vanPort})
	defer vanOBS.Close()
	msg := bytes.Repeat([]byte{0x42}, 1316)
	vanOBS.Write(msg)
	buf := make([]byte, 2000)
	studioOBS.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, from, err := studioOBS.ReadFromUDP(buf)
	if err != nil || !bytes.Equal(buf[:n], msg) {
		t.Fatalf("studio OBS did not get the packet through the relay: %v", err)
	}
	studioOBS.WriteToUDP([]byte("ack"), from)
	vanOBS.SetReadDeadline(time.Now().Add(2 * time.Second))
	if n, err = vanOBS.Read(buf); err != nil || string(buf[:n]) != "ack" {
		t.Fatalf("van OBS did not get the reply through the relay: %v", err)
	}
}

// A direct route is used whenever there is one, even with the relay set up.
func TestDirectBeatsRelay(t *testing.T) {
	addr, sign := testRelay(t)
	lic := sign("pro", time.Now().Add(24*time.Hour).Unix())
	code := studioCode()
	st, _ := New(Config{Role: Studio, StudioCode: code, OBSAddr: netip.MustParseAddrPort("127.0.0.1:" + strconv.Itoa(freePort(t))),
		Relay: addr, RelayLicence: func() string { return lic }, RelayAfter: 300 * time.Millisecond})
	vn, _ := New(Config{Role: Van, StudioCode: code, OBSAddr: netip.MustParseAddrPort("127.0.0.1:" + strconv.Itoa(freePort(t))),
		Relay: addr, RelayLicence: func() string { return lic }, RelayAfter: 300 * time.Millisecond})
	st.Start()
	vn.Start()
	defer st.Close()
	defer vn.Close()
	vn.SetPeerCandidates([]string{fmt.Sprintf("127.0.0.1:%d", st.Port())})
	if !waitConnected(st, vn, 5*time.Second) {
		t.Fatal("did not connect")
	}
	time.Sleep(2 * time.Second)
	if r := vn.Snapshot().Route; r == "Van Link relay" || r == "" {
		t.Fatalf("route %q, want a direct route", r)
	}
}

// The relay only serves computers with a valid licence.
func TestRelayRefusesExpiredLicence(t *testing.T) {
	addr, sign := testRelay(t)
	lic := sign("standard", time.Now().Add(-time.Hour).Unix()) // ended an hour ago
	st, vn := relayPair(t, freePort(t), freePort(t), addr, lic, []string{fmt.Sprintf("127.0.0.1:%d", freePort(t))})
	if waitConnected(st, vn, 3*time.Second) {
		t.Fatal("connected through the relay with an expired licence")
	}
}

// Real SRT video through the relay, as in TestRealSRTVideo.
func TestRealSRTVideoThroughRelay(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	addr, sign := testRelay(t)
	lic := sign("owner", 0)
	studioOBS, vanOBS := freePort(t), freePort(t)
	st, vn := relayPair(t, studioOBS, vanOBS, addr, lic, []string{fmt.Sprintf("127.0.0.1:%d", freePort(t))})
	if !waitConnected(st, vn, 8*time.Second) {
		t.Fatal("did not connect through the relay")
	}
	pp := "TestPassphrase123456"
	out := t.TempDir() + "/received.ts"
	recv := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-i", fmt.Sprintf("srt://127.0.0.1:%d?mode=listener&latency=300000&passphrase=%s&pbkeylen=16", studioOBS, pp),
		"-c", "copy", "-t", "5", out)
	if err := recv.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	send := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-re",
		"-f", "lavfi", "-i", "testsrc2=size=1280x720:rate=30", "-c:v", "libx264", "-preset", "veryfast",
		"-b:v", "4M", "-g", "60", "-t", "7", "-f", "mpegts",
		fmt.Sprintf("srt://127.0.0.1:%d?mode=caller&latency=300000&passphrase=%s&pbkeylen=16", vanOBS, pp))
	if err := send.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(4 * time.Second)
	a := st.Snapshot()
	t.Logf("studio via %s: live=%v %.0f kbps encrypted=%v", a.Route, a.LiveHere, a.Kbps, a.Encrypted)
	if !a.LiveHere || !a.Encrypted {
		t.Errorf("studio should show encrypted video arriving")
	}
	send.Wait()
	recv.Wait()
	probe, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-count_frames",
		"-show_entries", "stream=width,height,nb_read_frames", "-of", "csv=p=0", out).Output()
	if err != nil || !bytes.HasPrefix(probe, []byte("1280,720,")) {
		t.Fatalf("video through the relay: %s %v", probe, err)
	}
	t.Logf("received through the relay: %s", bytes.TrimSpace(probe))
}
