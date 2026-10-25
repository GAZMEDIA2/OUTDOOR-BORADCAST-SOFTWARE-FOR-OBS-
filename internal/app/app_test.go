package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/GAZMEDIA2/vanlink/internal/testobs"
)

func waitFor(t *testing.T, what string, d time.Duration, ok func() bool) {
	t.Helper()
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		if ok() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for: %s", what)
}

// The whole journey an operator makes: studio adds a van, van pairs with the
// code, studio pastes the reply, van goes live, studio receives the picture.
func TestFullBroadcast(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	out := t.TempDir() + "/studio-received.ts"
	studioOBS, err := testobs.Start(out)
	if err != nil {
		t.Fatal(err)
	}
	defer studioOBS.Close()
	vanOBS, err := testobs.Start("")
	if err != nil {
		t.Fatal(err)
	}
	defer vanOBS.Close()

	no := false
	studio := New(t.TempDir())
	defer studio.Close()
	van := New(t.TempDir())
	defer van.Close()
	for _, x := range []struct {
		a   *App
		obs string
	}{{studio, studioOBS.Addr}, {van, vanOBS.Addr}} {
		addr := x.obs
		manual := true
		if err := x.a.UpdateSettings(SettingsUpdate{OBSAddr: &addr, UseSTUN: &no, OBSManual: &manual}); err != nil {
			t.Fatal(err)
		}
	}
	if err := studio.SetRole("studio"); err != nil {
		t.Fatal(err)
	}
	if err := van.SetRole("van"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "OBS connected", 6*time.Second, func() bool {
		return studio.State().OBS.Connected && van.State().OBS.Connected
	})

	code, err := studio.AddVan("Stadium van")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(code, "VL1.") {
		t.Fatalf("bad code %q", code)
	}
	reply, err := van.PairVan(code)
	if err != nil {
		t.Fatal(err)
	}
	if err := studio.StudioReply(studio.State().Vans[0].ID, reply); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "connected", 8*time.Second, func() bool {
		return van.State().Van.Snap.Connected && studio.State().Vans[0].Snap.Connected
	})
	waitFor(t, "studio OBS source created", 6*time.Second, func() bool { return studio.State().Vans[0].OBSReady })

	if err := van.GoLive(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "LIVE AT STUDIO on the van screen", 10*time.Second, func() bool {
		s := van.State()
		return s.Van.Streaming && s.Van.Snap.PeerLive
	})
	sv := studio.State().Vans[0]
	t.Logf("studio sees: live=%v %.0f kbps, signal=%s, route=%s, encrypted=%v, van name=%q",
		sv.Snap.LiveHere, sv.Snap.Kbps, sv.Snap.SignalWord, sv.Snap.Route, sv.Snap.Encrypted, sv.Snap.Peer.Name)
	if !sv.Snap.LiveHere || !sv.Snap.Encrypted {
		t.Fatal("studio is not receiving encrypted video")
	}

	// The screen's API answers locally and refuses other host names.
	srv := httptest.NewServer(studio.Handler())
	defer srv.Close()
	r, _ := http.Get(srv.URL + "/api/state")
	var st State
	json.NewDecoder(r.Body).Decode(&st)
	if st.Role != "studio" || len(st.Vans) != 1 {
		t.Fatalf("api state: %+v", st)
	}
	req, _ := http.NewRequest("GET", srv.URL+"/api/state", nil)
	req.Host = "evil.example"
	if r, _ := http.DefaultClient.Do(req); r.StatusCode != http.StatusForbidden {
		t.Fatal("foreign host name was not refused")
	}
	req, _ = http.NewRequest("POST", srv.URL+"/api/van/golive", nil)
	if r, _ := http.DefaultClient.Do(req); r.StatusCode != http.StatusForbidden {
		t.Fatal("POST without Van Link header was not refused")
	}

	studioOBS.WaitReceived()
	probe, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-count_frames",
		"-show_entries", "stream=width,height,nb_read_frames", "-of", "csv=p=0", out).Output()
	if err != nil {
		t.Fatalf("studio recording unreadable: %v", err)
	}
	t.Logf("studio OBS received: %s", strings.TrimSpace(string(probe)))
	if !strings.HasPrefix(string(probe), "1280,720,") {
		t.Fatalf("unexpected video: %s", probe)
	}

	if err := van.StopLive(); err != nil {
		t.Fatal(err)
	}
	if vanOBS.Streaming() {
		t.Fatal("OBS still streaming after stop")
	}
}
