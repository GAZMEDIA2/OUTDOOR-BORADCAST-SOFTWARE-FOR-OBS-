package obsdock

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// WS is OBS's remote-control (obs-websocket) setting. OBS 30 and newer keep it
// in the [OBSWebSocket] section of global.ini / user.ini; OBS 28-29 kept it in
// plugin_config/obs-websocket/config.json. Van Link reads and writes both.
type WS struct {
	Enabled  bool
	Auth     bool
	Port     int
	Password string
}

func wsFile(dir string) string {
	return filepath.Join(dir, "plugin_config", "obs-websocket", "config.json")
}

// ReadWebSocket reads OBS's remote-control settings. Works while OBS is open.
func ReadWebSocket() (WS, error) { return readWS(obsConfigDir()) }

func readWS(dir string) (WS, error) {
	for _, f := range files(dir) {
		if b, err := os.ReadFile(f); err == nil {
			if w, ok := wsFromINI(string(b)); ok {
				return w, nil
			}
		}
	}
	b, err := os.ReadFile(wsFile(dir))
	if err != nil {
		return WS{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return WS{}, err
	}
	w := WS{Port: 4455}
	w.Enabled, _ = raw["server_enabled"].(bool)
	w.Auth, _ = raw["auth_required"].(bool)
	w.Password, _ = raw["server_password"].(string)
	if p, ok := raw["server_port"].(float64); ok && p > 0 {
		w.Port = int(p)
	}
	return w, nil
}

// EnableWebSocket switches OBS's remote control on (keeping an existing
// password, or creating one). OBS must be closed: it reads this file when it
// starts.
func EnableWebSocket() (WS, error) {
	if OBSRunning() {
		return WS{}, ErrOBSRunning
	}
	return enableWS(obsConfigDir())
}

func enableWS(dir string) (WS, error) {
	if len(files(dir)) == 0 {
		return WS{}, errors.New("OBS settings not found - open and close OBS once")
	}
	p := wsFile(dir)
	raw := map[string]any{}
	if b, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(b, &raw)
		_ = os.WriteFile(p+".vanlink-backup", b, 0o600)
	}
	pw, _ := raw["server_password"].(string)
	port := 4455
	if v, ok := raw["server_port"].(float64); ok && v > 0 {
		port = int(v)
	}
	// OBS 30+: the settings that count are in the ini files.
	for _, f := range files(dir) {
		if b, err := os.ReadFile(f); err == nil {
			if w, ok := wsFromINI(string(b)); ok {
				if len(w.Password) >= 6 {
					pw = w.Password
				}
				port = w.Port
			}
		}
	}
	if len(pw) < 6 {
		pw = randomPassword(16)
	}
	for _, f := range files(dir) {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		_ = os.WriteFile(f+".vanlink-backup", b, 0o600)
		text := string(b)
		for _, kv := range [][2]string{{"FirstLoad", "false"}, {"ServerEnabled", "true"}, {"AuthRequired", "true"},
			{"ServerPassword", pw}, {"ServerPort", strconv.Itoa(port)}} {
			text = iniSet(text, "OBSWebSocket", kv[0], kv[1])
		}
		if !strings.Contains(text, "AlertsEnabled=") {
			text = iniSet(text, "OBSWebSocket", "AlertsEnabled", "false")
		}
		if err := os.WriteFile(f, []byte(text), 0o600); err != nil {
			return WS{}, err
		}
	}
	raw["server_enabled"] = true
	raw["auth_required"] = true
	raw["server_password"] = pw
	raw["server_port"] = port
	raw["first_load"] = false // otherwise OBS makes a new password on its next start
	if _, ok := raw["alerts_enabled"]; !ok {
		raw["alerts_enabled"] = false
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return WS{}, err
	}
	b, _ := json.MarshalIndent(raw, "", "    ")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		return WS{}, err
	}
	return WS{Enabled: true, Auth: true, Port: port, Password: pw}, nil
}

func randomPassword(n int) string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}

// wsFromINI reads the [OBSWebSocket] section OBS 30+ writes to its ini files.
func wsFromINI(text string) (WS, bool) {
	en, ok := iniGet(text, "OBSWebSocket", "ServerEnabled")
	if !ok {
		return WS{}, false
	}
	w := WS{Port: 4455, Enabled: en == "true"}
	if v, ok := iniGet(text, "OBSWebSocket", "AuthRequired"); ok {
		w.Auth = v == "true"
	}
	w.Password, _ = iniGet(text, "OBSWebSocket", "ServerPassword")
	if v, ok := iniGet(text, "OBSWebSocket", "ServerPort"); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			w.Port = n
		}
	}
	return w, true
}

func iniLines(text string) []string {
	return strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
}

func iniGet(text, section, key string) (string, bool) {
	in := false
	for _, l := range iniLines(strings.TrimPrefix(text, bom)) {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			in = t == "["+section+"]"
			continue
		}
		if in && strings.HasPrefix(t, key+"=") {
			return strings.TrimPrefix(t, key+"="), true
		}
	}
	return "", false
}

// iniSet sets key=value in [section], adding the key or the section if needed.
// Line endings and a leading BOM are kept.
func iniSet(text, section, key, value string) string {
	nl := "\n"
	if strings.Contains(text, "\r\n") {
		nl = "\r\n"
	}
	hasBOM := strings.HasPrefix(text, bom)
	lines := iniLines(strings.TrimPrefix(text, bom))
	in, secAt, lastAt := false, -1, -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			in = t == "["+section+"]"
			if in {
				secAt, lastAt = i, i
			}
			continue
		}
		if in {
			if strings.HasPrefix(t, key+"=") {
				lines[i] = key + "=" + value
				return joinINI(lines, nl, hasBOM)
			}
			if t != "" {
				lastAt = i
			}
		}
	}
	if secAt < 0 {
		for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		lines = append(lines, "", "["+section+"]", key+"="+value, "")
		return joinINI(lines, nl, hasBOM)
	}
	lines = append(lines[:lastAt+1], append([]string{key + "=" + value}, lines[lastAt+1:]...)...)
	return joinINI(lines, nl, hasBOM)
}

func joinINI(lines []string, nl string, hasBOM bool) string {
	s := strings.Join(lines, nl)
	if hasBOM {
		s = bom + s
	}
	return s
}
