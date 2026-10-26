package obsdock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const url = "http://127.0.0.1:8765/?dock=1"

func write(t *testing.T, dir, name, text string) string {
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAddToExistingList(t *testing.T) {
	d := t.TempDir()
	p := write(t, d, "user.ini", "[General]\r\nFirstRun=true\r\n\r\n[BasicWindow]\r\nVerticalVolControl=false\r\nExtraBrowserDocks=[{\"title\":\"Chat\",\"url\":\"https://chat.example\",\"uuid\":\"abc\"}]\r\nDockState=xyz\r\n")
	if err := installIn(d, url); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	s := string(b)
	if !strings.Contains(s, `"title":"Chat"`) || !strings.Contains(s, `"title":"Van Link"`) || !strings.Contains(s, "DockState=xyz") || !strings.Contains(s, "\r\n") {
		t.Fatalf("bad result:\n%s", s)
	}
	if !installedIn(d, url) {
		t.Fatal("not detected as installed")
	}
	// second run does not duplicate
	installIn(d, url)
	b, _ = os.ReadFile(p)
	if strings.Count(string(b), "Van Link") != 1 {
		t.Fatalf("duplicated:\n%s", b)
	}
}

func TestAddWhenNoList(t *testing.T) {
	d := t.TempDir()
	p := write(t, d, "global.ini", "[General]\nFirstRun=true\n\n[BasicWindow]\nGeometry=AAA\n\n[Other]\nX=1\n")
	if err := installIn(d, url); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	docks, has, err := parse(string(b))
	if err != nil || !has || len(docks) != 1 || docks[0].URL != url {
		t.Fatalf("bad result:\n%s", b)
	}
	if !strings.Contains(string(b), "[Other]\nX=1") {
		t.Fatalf("other sections damaged:\n%s", b)
	}
}

func TestAddWhenNoSection(t *testing.T) {
	d := t.TempDir()
	p := write(t, d, "global.ini", "[General]\nFirstRun=true\n")
	if err := installIn(d, url); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if docks, has, _ := parse(string(b)); !has || len(docks) != 1 {
		t.Fatalf("bad result:\n%s", b)
	}
}

func TestNoOBS(t *testing.T) {
	if installIn(t.TempDir(), url) == nil {
		t.Fatal("expected an error when OBS has never run")
	}
}

func TestRemove(t *testing.T) {
	d := t.TempDir()
	p := write(t, d, "user.ini", "[BasicWindow]\nExtraBrowserDocks=[{\"title\":\"Chat\",\"url\":\"x\",\"uuid\":\"a\"}]\n")
	installIn(d, url)
	if err := removeIn(d); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if strings.Contains(string(b), "Van Link") || !strings.Contains(string(b), "Chat") {
		t.Fatalf("bad result:\n%s", b)
	}
}

func TestWebSocketEnable(t *testing.T) {
	d := t.TempDir()
	write(t, d, "user.ini", "[General]\n")
	os.MkdirAll(filepath.Join(d, "plugin_config", "obs-websocket"), 0o700)
	write(t, filepath.Join(d, "plugin_config", "obs-websocket"), "config.json",
		`{"alerts_enabled":false,"auth_required":true,"first_load":true,"server_enabled":false,"server_password":"keepMe123","server_port":4460}`)
	w, err := enableWS(d)
	if err != nil {
		t.Fatal(err)
	}
	if !w.Enabled || w.Password != "keepMe123" || w.Port != 4460 {
		t.Fatalf("bad: %+v", w)
	}
	r, err := readWS(d)
	if err != nil || r != w {
		t.Fatalf("read back: %+v %v", r, err)
	}
}

func TestWebSocketCreate(t *testing.T) {
	d := t.TempDir()
	write(t, d, "global.ini", "[General]\n")
	w, err := enableWS(d)
	if err != nil || !w.Enabled || len(w.Password) != 16 || w.Port != 4455 {
		t.Fatalf("bad: %+v %v", w, err)
	}
}

func TestWebSocketINI(t *testing.T) {
	dir := t.TempDir()
	ini := "\xef\xbb\xbf[General]\r\nFirstRun=true\r\n\r\n[BasicWindow]\r\nX=1\r\n"
	if err := os.WriteFile(filepath.Join(dir, "global.ini"), []byte(ini), 0o600); err != nil {
		t.Fatal(err)
	}
	w, err := enableWS(dir)
	if err != nil || !w.Enabled || !w.Auth || w.Port != 4455 || len(w.Password) < 6 {
		t.Fatalf("enable: %+v %v", w, err)
	}
	r, err := readWS(dir)
	if err != nil || r != w {
		t.Fatalf("read back %+v, want %+v (%v)", r, w, err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "global.ini"))
	s := string(b)
	if !strings.HasPrefix(s, "\xef\xbb\xbf[General]\r\n") || !strings.Contains(s, "[OBSWebSocket]\r\n") || !strings.Contains(s, "ServerEnabled=true\r\n") {
		t.Fatalf("bad ini:\n%q", s)
	}
	// An existing section keeps the user's port and password.
	ini2 := "[OBSWebSocket]\nServerEnabled=false\nServerPort=4470\nServerPassword=keepme123\nAuthRequired=true\n"
	os.WriteFile(filepath.Join(dir, "global.ini"), []byte(ini2), 0o600)
	w2, _ := enableWS(dir)
	if w2.Port != 4470 || w2.Password != "keepme123" || !w2.Enabled {
		t.Fatalf("existing section not kept: %+v", w2)
	}
}
