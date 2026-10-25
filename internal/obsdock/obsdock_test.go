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
