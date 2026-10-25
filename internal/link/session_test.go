package link

import (
	"bytes"
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/GAZMEDIA2/vanlink/internal/pairing"
)

func studioCode() pairing.Code {
	sid, key, pp := pairing.NewStudioSecrets()
	return pairing.Code{Role: "studio", Session: sid, Key: key, Passphrase: pp, LatencyMS: 300}
}

func freePort(t *testing.T) int {
	c, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).Port
}

func pair(t *testing.T, studioOBS, vanOBS int) (*Session, *Session) {
	code := studioCode()
	st, err := New(Config{Role: Studio, Name: "Studio", StudioCode: code,
		OBSAddr: netip.MustParseAddrPort("127.0.0.1:" + strconv.Itoa(studioOBS))})
	if err != nil {
		t.Fatal(err)
	}
	vn, err := New(Config{Role: Van, Name: "Van 1", StudioCode: code,
		OBSAddr: netip.MustParseAddrPort("127.0.0.1:" + strconv.Itoa(vanOBS))})
	if err != nil {
		t.Fatal(err)
	}
	st.Start()
	vn.Start()
	// The van only knows the studio's code; the studio answers its dial.
	vn.SetPeerCandidates([]string{fmt.Sprintf("127.0.0.1:%d", st.Port())})
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if st.Snapshot().Connected && vn.Snapshot().Connected {
			return st, vn
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("did not connect")
	return nil, nil
}

func TestConnectAndStatus(t *testing.T) {
	st, vn := pair(t, freePort(t), freePort(t))
	defer st.Close()
	defer vn.Close()
	time.Sleep(2500 * time.Millisecond)
	a, b := st.Snapshot(), vn.Snapshot()
	if a.Peer.Name != "Van 1" || b.Peer.Name != "Studio" {
		t.Fatalf("status not exchanged: %+v / %+v", a.Peer, b.Peer)
	}
	if b.Route == "" {
		t.Fatal("no route")
	}
}

func TestWrongKeyIgnored(t *testing.T) {
	code := studioCode()
	st, _ := New(Config{Role: Studio, StudioCode: code, OBSAddr: netip.MustParseAddrPort("127.0.0.1:1")})
	other := studioCode()
	vn, _ := New(Config{Role: Van, StudioCode: other, OBSAddr: netip.MustParseAddrPort("127.0.0.1:" + strconv.Itoa(freePort(t)))})
	st.Start()
	vn.Start()
	defer st.Close()
	defer vn.Close()
	vn.SetPeerCandidates([]string{fmt.Sprintf("127.0.0.1:%d", st.Port())})
	time.Sleep(1500 * time.Millisecond)
	if st.Snapshot().Connected {
		t.Fatal("studio accepted a van with the wrong code")
	}
}

// Fake OBS on both ends: datagrams go van-OBS -> studio-OBS and back unchanged.
func TestForwarding(t *testing.T) {
	studioOBS, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer studioOBS.Close()
	vanPort := freePort(t)
	st, vn := pair(t, studioOBS.LocalAddr().(*net.UDPAddr).Port, vanPort)
	defer st.Close()
	defer vn.Close()

	vanOBS, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: vanPort})
	if err != nil {
		t.Fatal(err)
	}
	defer vanOBS.Close()
	msg := bytes.Repeat([]byte{0x01}, 1316)
	vanOBS.Write(msg)
	buf := make([]byte, 2000)
	studioOBS.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, from, err := studioOBS.ReadFromUDP(buf)
	if err != nil || !bytes.Equal(buf[:n], msg) {
		t.Fatalf("studio OBS did not get the packet: %v", err)
	}
	studioOBS.WriteToUDP([]byte("reply-from-studio-obs"), from)
	vanOBS.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err = vanOBS.Read(buf)
	if err != nil || string(buf[:n]) != "reply-from-studio-obs" {
		t.Fatalf("van OBS did not get the reply: %v", err)
	}
}

// Real video: ffmpeg sends SRT (as OBS would) through the van and studio
// engines to an ffmpeg SRT listener (as the studio's OBS source would).
func TestRealSRTVideo(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	studioOBS, vanOBS := freePort(t), freePort(t)
	st, vn := pair(t, studioOBS, vanOBS)
	defer st.Close()
	defer vn.Close()
	pp := "TestPassphrase123456"
	out := t.TempDir() + "/received.ts"
	recv := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-i", fmt.Sprintf("srt://127.0.0.1:%d?mode=listener&latency=300000&passphrase=%s&pbkeylen=16", studioOBS, pp),
		"-c", "copy", "-t", "6", out)
	if err := recv.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	send := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-re",
		"-f", "lavfi", "-i", "testsrc2=size=1280x720:rate=30",
		"-f", "lavfi", "-i", "sine=frequency=440",
		"-c:v", "libx264", "-preset", "veryfast", "-b:v", "3M", "-g", "60",
		"-c:a", "aac", "-t", "8", "-f", "mpegts",
		fmt.Sprintf("srt://127.0.0.1:%d?mode=caller&latency=300000&passphrase=%s&pbkeylen=16", vanOBS, pp))
	if err := send.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(4 * time.Second)
	a, b := st.Snapshot(), vn.Snapshot()
	t.Logf("studio: live=%v %.0f kbps encrypted=%v signal=%s | van: live=%v liveAtStudio=%v %.0f kbps delay=%dms",
		a.LiveHere, a.Kbps, a.Encrypted, a.SignalWord, b.LiveHere, b.PeerLive, b.Kbps, b.DelayMS)
	if !a.LiveHere || !b.PeerLive || !a.Encrypted {
		t.Errorf("expected studio to be receiving encrypted video and van to see it")
	}
	send.Wait()
	recv.Wait()
	probe, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-count_frames", "-show_entries", "stream=width,height,nb_read_frames", "-of", "csv=p=0", out).Output()
	if err != nil {
		t.Fatalf("received file unreadable: %v", err)
	}
	t.Logf("received video: %s", bytes.TrimSpace(probe))
	if !bytes.HasPrefix(probe, []byte("1280,720,")) {
		t.Fatalf("unexpected video: %s", probe)
	}
}
