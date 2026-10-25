package app

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/GAZMEDIA2/vanlink/internal/licence"
)

// LicenceServer is the Van Link website the app signs in to.
var LicenceServer = "https://vanlink.gazgoc.com.ng"

// Account is this computer's sign-in, saved with the settings.
type Account struct {
	Email     string `json:"email"`
	Token     string `json:"token"`
	Licence   string `json:"licence"`
	DeviceID  string `json:"deviceId"`
	MaxSeen   int64  `json:"maxSeen"`   // latest clock time ever seen (catches a clock wound back)
	UsedSince int    `json:"usedSince"` // trial seconds broadcast since the licence was issued
	Unsent    int    `json:"unsent"`    // trial seconds not yet reported to the website
	LastCheck int64  `json:"lastCheck"`
}

func newDeviceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (a *App) ensureDeviceID() string {
	cfg := a.store.get()
	if len(cfg.Account.DeviceID) >= 16 {
		return cfg.Account.DeviceID
	}
	id := newDeviceID()
	_ = a.store.update(func(c *Config) { c.Account.DeviceID = id })
	return id
}

type apiReply struct {
	OK        *bool  `json:"ok"`
	Error     string `json:"error"`
	SignedOut bool   `json:"signed_out"`
	Token     string `json:"token"`
	Licence   string `json:"licence"`
}

func callServer(path string, body any) (apiReply, error) {
	var r apiReply
	b, _ := json.Marshal(body)
	c := &http.Client{Timeout: 20 * time.Second}
	resp, err := c.Post(strings.TrimRight(LicenceServer, "/")+"/api/"+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return r, fmt.Errorf("could not reach %s - check the internet connection", strings.TrimPrefix(LicenceServer, "https://"))
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return r, errors.New("unexpected reply from the Van Link website")
	}
	if r.OK != nil && !*r.OK {
		if r.Error == "" {
			r.Error = "request refused"
		}
		return r, errors.New(r.Error)
	}
	return r, nil
}

// SignIn signs this computer in with the customer's website account.
func (a *App) SignIn(email, password string) error {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return errors.New("enter your email and password")
	}
	dev := a.ensureDeviceID()
	host, _ := os.Hostname()
	r, err := callServer("login.php", map[string]any{"email": email, "password": password, "device_id": dev,
		"device_name": host, "role": a.store.get().Role})
	if err != nil {
		return err
	}
	if _, err := licence.Parse(r.Licence, licence.PublicKey); err != nil {
		return errors.New("the website sent a licence this version of Van Link cannot read - update Van Link")
	}
	return a.store.update(func(c *Config) {
		c.Account.Email, c.Account.Token, c.Account.Licence = email, r.Token, r.Licence
		c.Account.UsedSince, c.Account.Unsent, c.Account.LastCheck = 0, 0, time.Now().Unix()
	})
}

// SignOut signs this computer out (and frees its place on the plan).
func (a *App) SignOut() {
	cfg := a.store.get()
	if cfg.Account.Token != "" {
		_, _ = callServer("logout.php", map[string]any{"token": cfg.Account.Token})
	}
	_ = a.store.update(func(c *Config) {
		dev, seen := c.Account.DeviceID, c.Account.MaxSeen
		c.Account = Account{DeviceID: dev, MaxSeen: seen}
	})
	a.enforce()
}

// Redeem activates a registration code on the signed-in account.
func (a *App) Redeem(code string) error {
	cfg := a.store.get()
	if cfg.Account.Token == "" {
		return errors.New("sign in first")
	}
	r, err := callServer("redeem.php", map[string]any{"token": cfg.Account.Token, "code": code})
	if err != nil {
		return err
	}
	err = a.store.update(func(c *Config) {
		c.Account.Licence, c.Account.UsedSince, c.Account.LastCheck = r.Licence, 0, time.Now().Unix()
	})
	a.enforce()
	return err
}

// RefreshLicence checks in with the website: new licence, trial time reported.
func (a *App) RefreshLicence() error {
	cfg := a.store.get()
	if cfg.Account.Token == "" {
		return errors.New("not signed in")
	}
	sent := cfg.Account.Unsent
	r, err := callServer("check.php", map[string]any{"token": cfg.Account.Token, "live_seconds": sent, "role": cfg.Role})
	if err != nil {
		if r.SignedOut {
			_ = a.store.update(func(c *Config) { c.Account.Token, c.Account.Licence = "", "" })
			a.enforce()
		}
		return err
	}
	err = a.store.update(func(c *Config) {
		c.Account.Licence, c.Account.LastCheck = r.Licence, time.Now().Unix()
		c.Account.Unsent -= sent
		if c.Account.Unsent < 0 {
			c.Account.Unsent = 0
		}
		c.Account.UsedSince = c.Account.Unsent // the new licence already counts what was sent
	})
	a.enforce()
	return err
}

// LicenceRequired can be switched off for automated tests only.
var LicenceRequired = true

// Licence decides whether Van Link may be used right now (works offline).
func (a *App) Licence() licence.Status {
	if !LicenceRequired {
		return licence.Status{OK: true, Plan: "owner", PlanName: "Owner (unlimited)", MaxVans: 99}
	}
	cfg := a.store.get()
	return licence.Decide(cfg.Account.Licence, cfg.Account.DeviceID, time.Now(),
		time.Unix(cfg.Account.MaxSeen, 0), cfg.Account.UsedSince)
}

// usable is false when the licence is locked or today's trial time is used up.
func usable(s licence.Status) bool {
	return s.OK && (s.TrialLeft == nil || *s.TrialLeft > 0)
}

func lockMessage(s licence.Status) string {
	if !s.OK {
		return s.Message
	}
	return fmt.Sprintf("Today's %d free-trial minutes are used up. Broadcasting is available again tomorrow, or buy a plan to remove the limit.", 30)
}

// enforce applies the licence: stops sending and receiving when locked.
func (a *App) enforce() {
	st := a.Licence()
	ok := usable(st)
	a.mu.Lock()
	for _, v := range a.studio {
		v.sess.SetEnabled(ok)
	}
	van, obs, live := a.van, a.obs, a.vanLive
	a.mu.Unlock()
	if van != nil {
		van.SetEnabled(ok)
	}
	if !ok && live && obs != nil && obs.Alive() {
		_ = obs.StopStream()
		if van != nil {
			van.Note("Stopped: " + lockMessage(st))
		}
	}
}

// licenceTick runs every 2 seconds from the watchdog: counts trial time,
// remembers the clock, checks in with the website now and then.
func (a *App) licenceTick(sendingNow bool) {
	now := time.Now()
	cfg := a.store.get()
	st := a.Licence()
	changed := false
	acct := cfg.Account
	if now.Unix() > acct.MaxSeen {
		acct.MaxSeen = now.Unix()
		changed = true
	}
	if sendingNow && st.Plan == "trial" {
		acct.UsedSince += 2
		acct.Unsent += 2
		changed = true
	}
	if changed && (now.Unix()%30 < 2 || sendingNow) {
		_ = a.store.update(func(c *Config) {
			c.Account.MaxSeen, c.Account.UsedSince, c.Account.Unsent = acct.MaxSeen, acct.UsedSince, acct.Unsent
		})
	}
	every := int64(15 * 60)
	if st.Plan == "trial" && sendingNow {
		every = 120
	}
	if acct.Token != "" && now.Unix()-acct.LastCheck >= every {
		_ = a.store.update(func(c *Config) { c.Account.LastCheck = now.Unix() }) // don't retry every tick when offline
		go a.RefreshLicence()
	}
	if !usable(st) {
		a.enforce()
	} else {
		a.mu.Lock()
		for _, v := range a.studio {
			v.sess.SetEnabled(true)
		}
		if a.van != nil {
			a.van.SetEnabled(true)
		}
		a.mu.Unlock()
	}
}
