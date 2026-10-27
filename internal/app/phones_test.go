package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GAZMEDIA2/vanlink/internal/testobs"
)

// The studio adds a phone camera: OBS gets a browser source for it, the
// screen shows the code and link for the phone, and whether it is live.
func TestPhoneCamera(t *testing.T) {
	LicenceRequired = false
	defer func() { LicenceRequired = true }()
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]string
		json.NewDecoder(r.Body).Decode(&in)
		if r.URL.Path != "/api/cam.php" || in["action"] != "status" || len(in["key"]) != 32 || len(in["room"]) != 8 {
			http.Error(w, "bad request", 400)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "sourceOpen": true, "state": "live", "name": "Touchline", "onAir": true})
	}))
	defer web.Close()
	old := LicenceServer
	LicenceServer = web.URL
	defer func() { LicenceServer = old }()

	obs, err := testobs.Start("")
	if err != nil {
		t.Fatal(err)
	}
	defer obs.Close()
	studio := New(t.TempDir())
	defer studio.Close()
	addr, manual := obs.Addr, true
	if err := studio.UpdateSettings(SettingsUpdate{OBSAddr: &addr, OBSManual: &manual}); err != nil {
		t.Fatal(err)
	}
	if err := studio.SetRole("studio"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "OBS connected", 10*time.Second, func() bool { return studio.State().OBS.Connected })

	p, err := studio.AddPhone("Touchline cam")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Code) != 9 || p.Code[4] != '-' || !strings.HasPrefix(p.Link, web.URL+"/camera.php?c=") {
		t.Fatalf("bad code or link: %+v", p)
	}
	if _, err := studio.AddPhone("touchline CAM"); err == nil {
		t.Fatal("a second camera with the same name was allowed")
	}
	waitFor(t, "browser source in OBS", 10*time.Second, func() bool {
		s := studio.State()
		return len(s.Phones) == 1 && s.Phones[0].OBSReady
	})
	url := obs.BrowserURL("Van Link - Touchline cam")
	room := strings.ReplaceAll(p.Code, "-", "")
	if !strings.HasPrefix(url, web.URL+"/studio-camera.php?room="+room+"&key=") {
		t.Fatalf("browser source address: %q", url)
	}
	waitFor(t, "phone status from the website", 15*time.Second, func() bool {
		st := studio.State().Phones[0].Status
		return st.State == "live" && st.OnAir && st.Phone == "Touchline"
	})

	// Settings survive a restart.
	if got := studio.store.get().Phones; len(got) != 1 || got[0].Room != room {
		t.Fatalf("saved phones: %+v", got)
	}
	if err := studio.RemovePhone(p.ID); err != nil {
		t.Fatal(err)
	}
	if len(studio.State().Phones) != 0 || len(studio.store.get().Phones) != 0 {
		t.Fatal("phone camera not removed")
	}
}
