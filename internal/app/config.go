package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Config is saved between runs so pairing codes stay valid after a restart.
type Config struct {
	Role        string `json:"role"` // "", "studio", "van"
	Name        string `json:"name"`
	OBSAddr     string `json:"obsAddr"`
	OBSPassword string `json:"obsPassword"`
	OBSManual   bool   `json:"obsManual"` // false = connect to OBS automatically
	LatencyMS   int    `json:"latencyMs"`
	UseSTUN     bool   `json:"useStun"`

	// Studio: one entry per van.
	Vans []VanEntry `json:"vans"`

	// Van: the studio code it was paired with, and its own port.
	StudioCode string `json:"studioCode"`
	VanPort    int    `json:"vanPort"`

	SetupDone bool `json:"setupDone"`
}

// VanEntry is one van paired with this studio.
type VanEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Session    string `json:"session"`
	Key        string `json:"key"`
	Passphrase string `json:"passphrase"`
	LinkPort   int    `json:"linkPort"` // fixed, so the studio's code keeps working
	OBSPort    int    `json:"obsPort"`  // where OBS's source listens (localhost)
	ReplyCode  string `json:"replyCode"`
}

const (
	defaultOBSAddr = "127.0.0.1:4455"
	vanOBSPort     = 9710  // van: OBS sends SRT here (localhost only)
	studioOBSBase  = 9711  // studio: OBS sources listen from here up (localhost)
	studioLinkBase = 47100 // studio: one internet port per van
	defaultLatency = 1000
)

type store struct {
	mu   sync.Mutex
	path string
	cfg  Config
}

func configDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		d = "."
	}
	return filepath.Join(d, "VanLink")
}

func loadStore(dir string) *store {
	if dir == "" {
		dir = configDir()
	}
	s := &store{path: filepath.Join(dir, "config.json")}
	s.cfg = Config{OBSAddr: defaultOBSAddr, LatencyMS: defaultLatency, UseSTUN: true}
	if b, err := os.ReadFile(s.path); err == nil {
		_ = json.Unmarshal(b, &s.cfg)
	}
	if s.cfg.OBSAddr == "" {
		s.cfg.OBSAddr = defaultOBSAddr
	}
	if s.cfg.LatencyMS == 0 {
		s.cfg.LatencyMS = defaultLatency
	}
	return s
}

func (s *store) get() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.cfg
	c.Vans = append([]VanEntry(nil), s.cfg.Vans...)
	return c
}

func (s *store) update(f func(*Config)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(&s.cfg)
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(s.cfg, "", "  ")
	return os.WriteFile(s.path, b, 0o600)
}
