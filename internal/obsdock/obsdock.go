// Package obsdock adds the Van Link panel to OBS Studio as a dock, by writing
// it into OBS's own list of panels (the same list "View > Docks > Custom
// Browser Docks" edits). OBS rewrites that file when it closes, so the
// panel can only be added while OBS is closed.
package obsdock

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const bom = "\xef\xbb\xbf"

// Title is the panel's name in OBS.
const Title = "Van Link"

type dock struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	UUID  string `json:"uuid"`
}

// ErrOBSRunning means OBS must be closed first.
var ErrOBSRunning = errors.New("close OBS first, then try again - OBS overwrites its panel list when it closes")

// obsConfigDir is where OBS keeps its settings.
func obsConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "obs-studio")
	case "darwin":
		h, _ := os.UserHomeDir()
		return filepath.Join(h, "Library", "Application Support", "obs-studio")
	default:
		d, _ := os.UserConfigDir()
		return filepath.Join(d, "obs-studio")
	}
}

// files OBS reads its panel list from: user.ini (OBS 31+) and global.ini (older).
func files(dir string) []string {
	var out []string
	for _, n := range []string{"user.ini", "global.ini"} {
		p := filepath.Join(dir, n)
		if _, err := os.Stat(p); err == nil {
			out = append(out, p)
		}
	}
	return out
}

// Installed reports whether the Van Link panel is already in OBS's list.
func Installed(url string) bool { return installedIn(obsConfigDir(), url) }

func installedIn(dir, url string) bool {
	for _, f := range files(dir) {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		docks, _, _ := parse(string(b))
		for _, d := range docks {
			if d.URL == url {
				return true
			}
		}
	}
	return false
}

// OBSFound reports whether OBS has been run on this computer.
func OBSFound() bool { return len(files(obsConfigDir())) > 0 }

// Install adds the panel. OBS must be closed.
func Install(url string) error {
	if OBSRunning() {
		return ErrOBSRunning
	}
	return installIn(obsConfigDir(), url)
}

func installIn(dir, url string) error {
	fs := files(dir)
	if len(fs) == 0 {
		return errors.New("OBS settings not found - open and close OBS once, then try again")
	}
	for _, f := range fs {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		text := strings.TrimPrefix(string(b), bom)
		docks, has, err := parse(text)
		if err != nil {
			return err
		}
		found := false
		for i := range docks {
			if docks[i].URL == url || docks[i].Title == Title {
				docks[i].URL, docks[i].Title = url, Title
				found = true
			}
		}
		if !found {
			docks = append(docks, dock{Title: Title, URL: url, UUID: newUUID()})
		}
		js, _ := json.Marshal(docks)
		out := setKey(text, has, string(js))
		if err := os.WriteFile(f+".vanlink-backup", b, 0o600); err != nil {
			return err
		}
		if err := os.WriteFile(f, []byte(out), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// Remove takes the Van Link panel out of OBS. OBS must be closed.
func Remove() error {
	if OBSRunning() {
		return ErrOBSRunning
	}
	return removeIn(obsConfigDir())
}

func removeIn(dir string) error {
	for _, f := range files(dir) {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		text := strings.TrimPrefix(string(b), bom)
		docks, has, err := parse(text)
		if err != nil || !has {
			continue
		}
		var keep []dock
		for _, d := range docks {
			if d.Title != Title {
				keep = append(keep, d)
			}
		}
		if keep == nil {
			keep = []dock{}
		}
		js, _ := json.Marshal(keep)
		if err := os.WriteFile(f, []byte(setKey(text, true, string(js))), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// parse finds ExtraBrowserDocks in the [BasicWindow] section.
func parse(text string) (docks []dock, has bool, err error) {
	section := ""
	for _, line := range strings.Split(text, "\n") {
		l := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if strings.HasPrefix(l, "[") && strings.HasSuffix(l, "]") {
			section = l
			continue
		}
		if section == "[BasicWindow]" && strings.HasPrefix(l, "ExtraBrowserDocks=") {
			v := strings.TrimPrefix(l, "ExtraBrowserDocks=")
			if strings.TrimSpace(v) == "" {
				return nil, true, nil
			}
			if err := json.Unmarshal([]byte(v), &docks); err != nil {
				return nil, true, fmt.Errorf("could not read OBS's panel list: %w", err)
			}
			return docks, true, nil
		}
	}
	return nil, false, nil
}

// setKey writes ExtraBrowserDocks=<json>, adding the section if needed.
func setKey(text string, has bool, js string) string {
	nl := "\n"
	if strings.Contains(text, "\r\n") {
		nl = "\r\n"
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	section := ""
	inserted := false
	var out []string
	for i, line := range lines {
		l := strings.TrimSpace(line)
		isHeader := strings.HasPrefix(l, "[") && strings.HasSuffix(l, "]")
		if isHeader {
			if section == "[BasicWindow]" && !has && !inserted {
				out = append(out, "ExtraBrowserDocks="+js)
				inserted = true
			}
			section = l
		}
		if section == "[BasicWindow]" && strings.HasPrefix(l, "ExtraBrowserDocks=") {
			out = append(out, "ExtraBrowserDocks="+js)
			inserted = true
			continue
		}
		out = append(out, line)
		if i == len(lines)-1 && section == "[BasicWindow]" && !inserted {
			// the section is last in the file
			if strings.TrimSpace(line) == "" {
				out = out[:len(out)-1]
				out = append(out, "ExtraBrowserDocks="+js, "")
			} else {
				out = append(out, "ExtraBrowserDocks="+js)
			}
			inserted = true
		}
	}
	if !inserted {
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		out = append(out, "[BasicWindow]", "ExtraBrowserDocks="+js, "")
	}
	return strings.Join(out, nl)
}

// OBSRunning reports whether OBS is open right now.
func OBSRunning() bool {
	switch runtime.GOOS {
	case "windows":
		out, err := hiddenCommand("tasklist", "/FI", "IMAGENAME eq obs64.exe", "/NH").Output()
		return err == nil && strings.Contains(strings.ToLower(string(out)), "obs64.exe")
	default:
		return exec.Command("pgrep", "-x", "obs").Run() == nil
	}
}

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
