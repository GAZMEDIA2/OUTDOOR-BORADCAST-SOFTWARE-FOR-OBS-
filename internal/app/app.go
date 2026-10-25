// Package app ties Van Link together: the saved settings, the connection
// engine for each van, OBS remote control, and the screens (a local web page
// that also works as a dock inside OBS).
package app

import (
	"errors"
	"fmt"
	"log"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GAZMEDIA2/vanlink/internal/link"
	"github.com/GAZMEDIA2/vanlink/internal/netx"
	"github.com/GAZMEDIA2/vanlink/internal/obsdock"
	"github.com/GAZMEDIA2/vanlink/internal/obsws"
	"github.com/GAZMEDIA2/vanlink/internal/pairing"
	"github.com/GAZMEDIA2/vanlink/internal/sys"
)

// Version of Van Link.
var Version = "0.1.0"

// App is the running program.
type App struct {
	store *store

	mu       sync.Mutex
	obs      *obsws.Client
	obsErr   string
	studio   map[string]*studioVan
	van      *link.Session
	vanReply string
	vanLive  bool // OBS stream output active (van)
	netTest  *netx.TestResult
	testing  bool
	lastMsg  string
	quit     chan struct{}
	userQuit chan struct{}
	quitOnce sync.Once
	showReq  chan struct{}
	dockURL  string
}

// ShowRequested delivers "bring the window forward" requests.
func (a *App) ShowRequested() <-chan struct{} { return a.showReq }

// RequestShow asks for the window to be shown (when Van Link is opened again).
func (a *App) RequestShow() {
	select {
	case a.showReq <- struct{}{}:
	default:
	}
}

// SetDockURL tells the app the address of its OBS panel.
func (a *App) SetDockURL(u string) { a.mu.Lock(); a.dockURL = u; a.mu.Unlock() }

// InstallOBSPanel adds the Van Link panel to OBS.
func (a *App) InstallOBSPanel() error {
	a.mu.Lock()
	u := a.dockURL
	a.mu.Unlock()
	return obsdock.Install(u)
}

// QuitRequested is closed when the operator clicks Quit Van Link.
func (a *App) QuitRequested() <-chan struct{} { return a.userQuit }

// RequestQuit asks the program to stop (from the screen's Quit button).
func (a *App) RequestQuit() { a.quitOnce.Do(func() { close(a.userQuit) }) }

type studioVan struct {
	entry    VanEntry
	sess     *link.Session
	code     string
	obsReady bool
	lastFix  time.Time
}

// New loads settings and restarts any saved links.
func New(configDir string) *App {
	a := &App{store: loadStore(configDir), studio: map[string]*studioVan{}, quit: make(chan struct{}), userQuit: make(chan struct{}), showReq: make(chan struct{}, 1)}
	cfg := a.store.get()
	switch cfg.Role {
	case "studio":
		for _, v := range cfg.Vans {
			if err := a.startStudioVan(v); err != nil {
				log.Printf("could not restart %s: %v", v.Name, err)
			}
		}
	case "van":
		if cfg.StudioCode != "" {
			if _, err := a.pairVan(cfg.StudioCode); err != nil {
				log.Printf("could not restart van link: %v", err)
			}
		}
	}
	go a.watchdog()
	return a
}

// Close stops everything.
func (a *App) Close() {
	close(a.quit)
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, v := range a.studio {
		v.sess.Close()
	}
	if a.van != nil {
		a.van.Close()
	}
	if a.obs != nil {
		a.obs.Close()
	}
	sys.KeepAwake(false)
}

func (a *App) logf(format string, args ...any) { log.Printf(format, args...) }

// ---------------- studio ----------------

func inputName(v VanEntry) string { return "Van Link - " + v.Name }

func (a *App) startStudioVan(v VanEntry) error {
	cfg := a.store.get()
	code := pairing.Code{Role: "studio", Session: v.Session, Key: v.Key, Passphrase: v.Passphrase,
		LatencyMS: cfg.LatencyMS, Name: nonEmpty(cfg.Name, "Studio")}
	sess, err := link.New(link.Config{
		Role: link.Studio, Name: code.Name, StudioCode: code,
		OBSAddr:    netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), uint16(v.OBSPort)),
		ListenPort: v.LinkPort, UseSTUN: cfg.UseSTUN,
		Logf: func(f string, x ...any) { a.logf(v.Name+": "+f, x...) },
	})
	if err != nil {
		return err
	}
	sess.Start()
	code.Candidates = sess.Candidates()
	text, _ := code.Encode()
	if v.ReplyCode != "" {
		if rc, err := pairing.Decode(v.ReplyCode); err == nil {
			sess.SetPeerCandidates(rc.Candidates)
		}
	}
	a.mu.Lock()
	a.studio[v.ID] = &studioVan{entry: v, sess: sess, code: text}
	a.mu.Unlock()
	a.prepareReceiver(v.ID)
	return nil
}

// AddVan creates a new van slot at the studio and returns its pairing code.
func (a *App) AddVan(name string) (string, error) {
	cfg := a.store.get()
	used := map[int]bool{}
	for _, v := range cfg.Vans {
		used[v.LinkPort-studioLinkBase] = true
	}
	n := 0
	for used[n] {
		n++
	}
	if name == "" {
		name = "Van " + strconv.Itoa(n+1)
	}
	for _, v := range cfg.Vans {
		if strings.EqualFold(v.Name, name) {
			return "", errors.New("a van with that name already exists")
		}
	}
	sid, key, pp := pairing.NewStudioSecrets()
	v := VanEntry{ID: "van" + sid[:6], Name: name, Session: sid, Key: key, Passphrase: pp,
		LinkPort: studioLinkBase + n, OBSPort: studioOBSBase + n}
	if err := a.startStudioVan(v); err != nil {
		return "", err
	}
	_ = a.store.update(func(c *Config) { c.Vans = append(c.Vans, v) })
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.studio[v.ID].code, nil
}

// StudioReply takes the van's reply code.
func (a *App) StudioReply(id, text string) error {
	rc, err := pairing.Decode(text)
	if err != nil {
		return err
	}
	if rc.Role != "van" {
		return errors.New("this is a studio code - paste the code the VAN sent back")
	}
	a.mu.Lock()
	v := a.studio[id]
	a.mu.Unlock()
	if v == nil {
		return errors.New("unknown van")
	}
	if rc.Session != v.entry.Session {
		return errors.New("this reply code belongs to a different van or an older code")
	}
	v.sess.SetPeerCandidates(rc.Candidates)
	return a.store.update(func(c *Config) {
		for i := range c.Vans {
			if c.Vans[i].ID == id {
				c.Vans[i].ReplyCode = text
			}
		}
	})
}

// RemoveVan deletes a van slot and its OBS source.
func (a *App) RemoveVan(id string) error {
	a.mu.Lock()
	v := a.studio[id]
	delete(a.studio, id)
	obs := a.obs
	a.mu.Unlock()
	if v == nil {
		return errors.New("unknown van")
	}
	v.sess.Close()
	if obs != nil {
		_ = obs.RemoveInput(inputName(v.entry))
	}
	return a.store.update(func(c *Config) {
		out := c.Vans[:0]
		for _, e := range c.Vans {
			if e.ID != id {
				out = append(out, e)
			}
		}
		c.Vans = out
	})
}

func (a *App) prepareReceiver(id string) {
	a.mu.Lock()
	v := a.studio[id]
	obs := a.obs
	a.mu.Unlock()
	if v == nil || obs == nil || !obs.Alive() {
		return
	}
	cfg := a.store.get()
	url := obsws.SRTURL("127.0.0.1", v.entry.OBSPort, "listener", cfg.LatencyMS, v.entry.Passphrase)
	if err := obs.EnsureReceiver(inputName(v.entry), url); err != nil {
		a.logf("OBS source for %s: %v", v.entry.Name, err)
		return
	}
	a.mu.Lock()
	v.obsReady = true
	a.mu.Unlock()
}

// ---------------- van ----------------

func (a *App) pairVan(text string) (string, error) {
	sc, err := pairing.Decode(text)
	if err != nil {
		return "", err
	}
	if sc.Role != "studio" {
		return "", errors.New("this is a van code - paste the code the STUDIO sent you")
	}
	cfg := a.store.get()
	a.mu.Lock()
	old := a.van
	a.van = nil
	a.mu.Unlock()
	if old != nil {
		old.Close()
	}
	name := nonEmpty(cfg.Name, "Van")
	sess, err := link.New(link.Config{
		Role: link.Van, Name: name, StudioCode: sc,
		OBSAddr:    netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), vanOBSPort),
		ListenPort: cfg.VanPort, UseSTUN: cfg.UseSTUN,
		Logf: func(f string, x ...any) { a.logf("van: "+f, x...) },
	})
	if err != nil && cfg.VanPort != 0 { // saved port taken: use any
		sess, err = link.New(link.Config{
			Role: link.Van, Name: name, StudioCode: sc,
			OBSAddr: netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), vanOBSPort),
			UseSTUN: cfg.UseSTUN, Logf: func(f string, x ...any) { a.logf("van: "+f, x...) },
		})
	}
	if err != nil {
		return "", err
	}
	sess.Start()
	sess.SetPeerCandidates(sc.Candidates)
	reply := pairing.Code{Role: "van", Session: sc.Session, Name: name, Candidates: sess.Candidates()}
	rtext, _ := reply.Encode()
	a.mu.Lock()
	a.van, a.vanReply = sess, rtext
	a.mu.Unlock()
	_ = a.store.update(func(c *Config) { c.StudioCode = text; c.VanPort = sess.Port() })
	a.configureVanOBS()
	return rtext, nil
}

// PairVan is called when the van operator pastes the studio's code.
func (a *App) PairVan(text string) (string, error) { return a.pairVan(text) }

// UnpairVan forgets the studio.
func (a *App) UnpairVan() {
	a.mu.Lock()
	s := a.van
	a.van, a.vanReply = nil, ""
	a.mu.Unlock()
	if s != nil {
		s.Close()
	}
	_ = a.store.update(func(c *Config) { c.StudioCode = "" })
}

func (a *App) configureVanOBS() error {
	cfg := a.store.get()
	sc, err := pairing.Decode(cfg.StudioCode)
	if err != nil {
		return err
	}
	a.mu.Lock()
	obs := a.obs
	a.mu.Unlock()
	if obs == nil || !obs.Alive() {
		return errors.New("OBS is not connected")
	}
	lat := sc.LatencyMS
	if lat == 0 {
		lat = defaultLatency
	}
	return obs.SetStreamDestination(obsws.SRTURL("127.0.0.1", vanOBSPort, "caller", lat, sc.Passphrase))
}

// GoLive points OBS at Van Link and starts streaming.
func (a *App) GoLive() error {
	a.mu.Lock()
	obs, sess := a.obs, a.van
	a.mu.Unlock()
	if sess == nil {
		return errors.New("pair with the studio first")
	}
	if obs == nil || !obs.Alive() {
		return errors.New("OBS is not connected - open OBS and check Settings")
	}
	if err := a.configureVanOBS(); err != nil {
		return err
	}
	st, err := obs.StreamStatus()
	if err == nil && st.Active {
		return nil
	}
	return obs.StartStream()
}

// StopLive stops OBS streaming.
func (a *App) StopLive() error {
	a.mu.Lock()
	obs := a.obs
	a.mu.Unlock()
	if obs == nil || !obs.Alive() {
		return errors.New("OBS is not connected")
	}
	return obs.StopStream()
}

// ---------------- background ----------------

func (a *App) connectOBS() {
	a.mu.Lock()
	alive := a.obs != nil && a.obs.Alive()
	a.mu.Unlock()
	if alive {
		return
	}
	cfg := a.store.get()
	addr, pass := cfg.OBSAddr, cfg.OBSPassword
	friendly := ""
	if !cfg.OBSManual {
		// Automatic: use OBS's own remote-control settings, switching the
		// remote control on by ourselves when OBS is closed.
		ws, err := obsdock.ReadWebSocket()
		if (err != nil || !ws.Enabled) && obsdock.OBSFound() && !obsdock.OBSRunning() {
			if w, e := obsdock.EnableWebSocket(); e == nil {
				ws, err = w, nil
				a.logf("OBS remote control switched on automatically")
			}
		}
		switch {
		case err == nil && ws.Enabled:
			addr = "127.0.0.1:" + strconv.Itoa(ws.Port)
			pass = ""
			if ws.Auth {
				pass = ws.Password
			}
			friendly = "Open OBS - Van Link connects to it automatically."
		case !obsdock.OBSFound():
			friendly = "OBS Studio was not found on this computer. Install OBS Studio 28 or newer, open it once, then come back."
		default:
			friendly = "Close OBS and open it again - Van Link will then connect to it automatically."
		}
	}
	c, err := obsws.Dial(addr, pass, 3*time.Second)
	a.mu.Lock()
	if err != nil {
		msg := err.Error()
		if friendly != "" {
			msg = friendly
		}
		a.obs, a.obsErr = nil, msg
		a.mu.Unlock()
		return
	}
	a.obs, a.obsErr = c, ""
	ids := make([]string, 0, len(a.studio))
	for id := range a.studio {
		ids = append(ids, id)
		a.studio[id].obsReady = false
	}
	isVan := a.van != nil
	a.mu.Unlock()
	a.logf("connected to OBS")
	for _, id := range ids {
		a.prepareReceiver(id)
	}
	if isVan {
		_ = a.configureVanOBS()
	}
}

func (a *App) watchdog() {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-a.quit:
			return
		case <-t.C:
		}
		a.connectOBS()
		a.mu.Lock()
		obs := a.obs
		vans := make([]*studioVan, 0, len(a.studio))
		for _, v := range a.studio {
			vans = append(vans, v)
		}
		van := a.van
		a.mu.Unlock()

		onAir := false
		for _, v := range vans {
			snap := v.sess.Snapshot()
			if snap.LiveHere {
				onAir = true
			}
			if obs == nil || !obs.Alive() {
				continue
			}
			if !v.obsReady {
				a.prepareReceiver(v.entry.ID)
				continue
			}
			// The van is sending but OBS's source has given up: re-arm it.
			if snap.Connected && snap.PeerLive && time.Since(v.lastFix) > 6*time.Second {
				if st, err := obs.MediaState(inputName(v.entry)); err == nil &&
					st != "OBS_MEDIA_STATE_PLAYING" && st != "OBS_MEDIA_STATE_OPENING" && st != "OBS_MEDIA_STATE_BUFFERING" {
					v.lastFix = time.Now()
					_ = obs.RestartMedia(inputName(v.entry))
				}
			}
		}
		if van != nil && obs != nil && obs.Alive() {
			if st, err := obs.StreamStatus(); err == nil {
				a.mu.Lock()
				a.vanLive = st.Active
				a.mu.Unlock()
				onAir = onAir || st.Active
			}
		}
		sys.KeepAwake(onAir)
	}
}

// RunNetTest runs the network test in the background.
func (a *App) RunNetTest() {
	a.mu.Lock()
	if a.testing {
		a.mu.Unlock()
		return
	}
	a.testing = true
	a.mu.Unlock()
	go func() {
		r := netx.RunTest(a.store.get().UseSTUN, nil)
		a.mu.Lock()
		a.netTest, a.testing = &r, false
		a.mu.Unlock()
	}()
}

// ---------------- state for the screen ----------------

// VanView is one van as seen at the studio.
type VanView struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Code      string        `json:"code"`
	HasReply  bool          `json:"hasReply"`
	OBSSource string        `json:"obsSource"`
	OBSReady  bool          `json:"obsReady"`
	Snap      link.Snapshot `json:"snap"`
}

// State is everything the screen needs, refreshed every second.
type State struct {
	Version string `json:"version"`
	Role    string `json:"role"`
	Name    string `json:"name"`
	OBS     struct {
		Connected bool   `json:"connected"`
		Error     string `json:"error"`
		Addr      string `json:"addr"`
		HasPass   bool   `json:"hasPassword"`
		Manual    bool   `json:"manual"`
	} `json:"obs"`
	Settings struct {
		LatencyMS int  `json:"latencyMs"`
		UseSTUN   bool `json:"useStun"`
	} `json:"settings"`
	Setup struct {
		Supported bool `json:"supported"`
		Done      bool `json:"done"`
	} `json:"setup"`
	Vans []VanView `json:"vans"`
	Van  *struct {
		Reply     string        `json:"reply"`
		Streaming bool          `json:"streaming"`
		Snap      link.Snapshot `json:"snap"`
	} `json:"van"`
	NetTest  *netx.TestResult `json:"netTest"`
	Testing  bool             `json:"testing"`
	OBSPanel bool             `json:"obsPanel"`
}

// State builds the screen state.
func (a *App) State() State {
	cfg := a.store.get()
	var s State
	s.Version, s.Role, s.Name = Version, cfg.Role, cfg.Name
	s.Settings.LatencyMS, s.Settings.UseSTUN = cfg.LatencyMS, cfg.UseSTUN
	s.Setup.Supported, s.Setup.Done = sys.Supported(), cfg.SetupDone
	s.OBS.Addr, s.OBS.HasPass, s.OBS.Manual = cfg.OBSAddr, cfg.OBSPassword != "", cfg.OBSManual
	a.mu.Lock()
	defer a.mu.Unlock()
	s.OBS.Connected = a.obs != nil && a.obs.Alive()
	s.OBS.Error = a.obsErr
	for _, v := range a.studio {
		s.Vans = append(s.Vans, VanView{ID: v.entry.ID, Name: v.entry.Name, Code: v.code,
			HasReply: v.entry.ReplyCode != "", OBSSource: inputName(v.entry), OBSReady: v.obsReady,
			Snap: v.sess.Snapshot()})
	}
	sort.Slice(s.Vans, func(i, j int) bool { return s.Vans[i].Name < s.Vans[j].Name })
	if a.van != nil {
		s.Van = &struct {
			Reply     string        `json:"reply"`
			Streaming bool          `json:"streaming"`
			Snap      link.Snapshot `json:"snap"`
		}{a.vanReply, a.vanLive, a.van.Snapshot()}
	}
	s.NetTest, s.Testing = a.netTest, a.testing
	s.OBSPanel = a.dockURL != "" && obsdock.Installed(a.dockURL)
	return s
}

// SetRole chooses studio or van (and clears the other role's links).
func (a *App) SetRole(role string) error {
	if role != "studio" && role != "van" {
		return errors.New("unknown role")
	}
	a.mu.Lock()
	for id, v := range a.studio {
		v.sess.Close()
		delete(a.studio, id)
	}
	if a.van != nil {
		a.van.Close()
		a.van, a.vanReply = nil, ""
	}
	a.mu.Unlock()
	return a.store.update(func(c *Config) {
		if c.Role != role {
			c.Vans, c.StudioCode = nil, ""
		}
		c.Role = role
		if c.Name == "" {
			if role == "studio" {
				c.Name = "Studio"
			} else {
				c.Name = "Van"
			}
		}
	})
}

// SettingsUpdate is what the settings screen can change.
type SettingsUpdate struct {
	Name        *string `json:"name"`
	OBSAddr     *string `json:"obsAddr"`
	OBSPassword *string `json:"obsPassword"`
	OBSManual   *bool   `json:"obsManual"`
	LatencyMS   *int    `json:"latencyMs"`
	UseSTUN     *bool   `json:"useStun"`
}

// UpdateSettings saves settings; OBS reconnects with the new details.
func (a *App) UpdateSettings(u SettingsUpdate) error {
	if u.LatencyMS != nil && (*u.LatencyMS < 120 || *u.LatencyMS > 8000) {
		return fmt.Errorf("delay buffer must be between 120 and 8000 ms")
	}
	err := a.store.update(func(c *Config) {
		if u.Name != nil {
			c.Name = strings.TrimSpace(*u.Name)
		}
		if u.OBSAddr != nil && strings.TrimSpace(*u.OBSAddr) != "" {
			c.OBSAddr = strings.TrimSpace(*u.OBSAddr)
		}
		if u.OBSPassword != nil {
			c.OBSPassword = *u.OBSPassword
		}
		if u.OBSManual != nil {
			c.OBSManual = *u.OBSManual
		}
		if u.LatencyMS != nil {
			c.LatencyMS = *u.LatencyMS
		}
		if u.UseSTUN != nil {
			c.UseSTUN = *u.UseSTUN
		}
	})
	if u.OBSAddr != nil || u.OBSPassword != nil || u.OBSManual != nil {
		a.mu.Lock()
		if a.obs != nil {
			a.obs.Close()
			a.obs = nil
		}
		a.mu.Unlock()
		go a.connectOBS()
	}
	return err
}

// Setup runs the one-click system setup.
func (a *App) Setup() sys.SetupResult {
	r := sys.Setup()
	if r.Done {
		_ = a.store.update(func(c *Config) { c.SetupDone = true })
	}
	return r
}

// Restore undoes the system setup.
func (a *App) Restore() sys.SetupResult {
	r := sys.Restore()
	if r.Done {
		_ = a.store.update(func(c *Config) { c.SetupDone = false })
	}
	return r
}

func nonEmpty(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}
