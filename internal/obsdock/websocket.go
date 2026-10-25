package obsdock

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// WS is OBS's remote-control (obs-websocket) setting, as saved by OBS in
// plugin_config/obs-websocket/config.json.
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
	if len(pw) < 6 {
		pw = randomPassword(16)
	}
	port := 4455
	if v, ok := raw["server_port"].(float64); ok && v > 0 {
		port = int(v)
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
