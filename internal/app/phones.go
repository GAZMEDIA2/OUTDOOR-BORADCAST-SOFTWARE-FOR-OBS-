package app

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Phone cameras: a phone opens camera.php on the Van Link website in its
// browser (iPhone or Android, nothing to install) and sends its camera
// straight to an OBS browser source at the studio. Van Link creates that
// source, shows the code to give the phone, and reports whether it is live.

type phoneCam struct {
	entry    PhoneEntry
	obsReady bool
	locked   bool // the source shows nothing because the licence is locked
	status   phoneStatus
}

type phoneStatus struct {
	Checked    bool   `json:"checked"`
	SourceOpen bool   `json:"sourceOpen"` // the OBS source has the page open
	State      string `json:"state"`      // "live", "waiting" or ""
	Phone      string `json:"phone"`      // name the phone gave itself
	OnAir      bool   `json:"onAir"`      // the source is on OBS's Program
	Error      string `json:"error"`
}

// PhoneView is one phone camera as seen at the studio.
type PhoneView struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Code      string      `json:"code"` // ABCD-EFGH, entered on the phone
	Link      string      `json:"link"` // opens the camera page with the code filled in
	OBSSource string      `json:"obsSource"`
	OBSReady  bool        `json:"obsReady"`
	Status    phoneStatus `json:"status"`
}

const camAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O, 1/I

func newPhoneSecrets() (room, key string) {
	b := make([]byte, 8)
	rand.Read(b)
	r := make([]byte, 8)
	for i, x := range b {
		r[i] = camAlphabet[int(x)%len(camAlphabet)]
	}
	k := make([]byte, 16)
	rand.Read(k)
	return string(r), hex.EncodeToString(k)
}

func phoneInputName(p PhoneEntry) string { return "Van Link - " + p.Name }

func prettyRoom(r string) string { return r[:4] + "-" + r[4:] }

func site(path string) string { return strings.TrimRight(LicenceServer, "/") + "/" + path }

// phoneSourceURL is the page the studio's OBS browser source opens.
func phoneSourceURL(p PhoneEntry) string {
	return site("studio-camera.php?room=" + p.Room + "&key=" + p.Key + "&name=" + url.QueryEscape(p.Name))
}

func phoneLink(p PhoneEntry) string { return site("camera.php?c=" + p.Room) }

func (a *App) startPhone(p PhoneEntry) {
	a.mu.Lock()
	a.phones[p.ID] = &phoneCam{entry: p}
	a.mu.Unlock()
	a.preparePhone(p.ID)
}

// AddPhone creates a phone camera at the studio and its OBS source.
func (a *App) AddPhone(name string) (PhoneView, error) {
	cfg := a.store.get()
	if name == "" {
		for n := 1; ; n++ {
			name = "Phone " + strconv.Itoa(n)
			if !nameTaken(cfg, name) {
				break
			}
		}
	}
	if nameTaken(cfg, name) {
		return PhoneView{}, errors.New("a van or phone with that name already exists")
	}
	if st := a.Licence(); !usable(st) {
		return PhoneView{}, errors.New(lockMessage(st))
	}
	room, key := newPhoneSecrets()
	p := PhoneEntry{ID: "cam" + strings.ToLower(room[:6]), Name: name, Room: room, Key: key}
	if err := a.store.update(func(c *Config) { c.Phones = append(c.Phones, p) }); err != nil {
		return PhoneView{}, err
	}
	a.startPhone(p)
	return PhoneView{ID: p.ID, Name: p.Name, Code: prettyRoom(p.Room), Link: phoneLink(p)}, nil
}

func nameTaken(cfg Config, name string) bool {
	for _, v := range cfg.Vans {
		if strings.EqualFold(v.Name, name) {
			return true
		}
	}
	for _, p := range cfg.Phones {
		if strings.EqualFold(p.Name, name) {
			return true
		}
	}
	return false
}

// RemovePhone deletes a phone camera and its OBS source.
func (a *App) RemovePhone(id string) error {
	a.mu.Lock()
	p := a.phones[id]
	delete(a.phones, id)
	obs := a.obs
	a.mu.Unlock()
	if p == nil {
		return errors.New("unknown phone camera")
	}
	if obs != nil {
		_ = obs.RemoveInput(phoneInputName(p.entry))
	}
	return a.store.update(func(c *Config) {
		out := c.Phones[:0]
		for _, e := range c.Phones {
			if e.ID != id {
				out = append(out, e)
			}
		}
		c.Phones = out
	})
}

// preparePhone creates or updates the OBS browser source for a phone. While
// the licence is locked the source shows an empty page instead.
func (a *App) preparePhone(id string) {
	a.mu.Lock()
	p := a.phones[id]
	obs := a.obs
	a.mu.Unlock()
	if p == nil || obs == nil || !obs.Alive() {
		return
	}
	locked := !usable(a.Licence())
	u := phoneSourceURL(p.entry)
	if locked {
		u = "about:blank"
	}
	if err := obs.EnsurePhoneCamera(phoneInputName(p.entry), u); err != nil {
		a.logf("OBS source for %s: %v", p.entry.Name, err)
		return
	}
	a.mu.Lock()
	p.obsReady, p.locked = true, locked
	a.mu.Unlock()
}

// checkPhones keeps the phone cameras' OBS sources in place and asks the
// website whether each phone is live (called from the watchdog).
func (a *App) checkPhones(askWebsite bool) {
	a.mu.Lock()
	obsOK := a.obs != nil && a.obs.Alive()
	list := make([]*phoneCam, 0, len(a.phones))
	for _, p := range a.phones {
		list = append(list, p)
	}
	a.mu.Unlock()
	locked := !usable(a.Licence())
	for _, p := range list {
		a.mu.Lock()
		redo := obsOK && (!p.obsReady || p.locked != locked)
		a.mu.Unlock()
		if redo {
			a.preparePhone(p.entry.ID)
		}
	}
	if askWebsite && len(list) > 0 && a.phoneAsking.CompareAndSwap(false, true) {
		go func() {
			defer a.phoneAsking.Store(false)
			for _, p := range list {
				st := phoneState(p.entry)
				a.mu.Lock()
				p.status = st
				a.mu.Unlock()
			}
		}()
	}
}

func phoneState(p PhoneEntry) phoneStatus {
	b, _ := json.Marshal(map[string]string{"action": "status", "room": p.Room, "key": p.Key})
	c := &http.Client{Timeout: 8 * time.Second}
	resp, err := c.Post(site("api/cam.php"), "application/json", bytes.NewReader(b))
	if err != nil {
		return phoneStatus{Checked: true, Error: "Can't reach the Van Link website"}
	}
	defer resp.Body.Close()
	var r struct {
		OK         bool   `json:"ok"`
		Error      string `json:"error"`
		SourceOpen bool   `json:"sourceOpen"`
		State      string `json:"state"`
		Name       string `json:"name"`
		OnAir      bool   `json:"onAir"`
	}
	if json.NewDecoder(resp.Body).Decode(&r) != nil || !r.OK {
		return phoneStatus{Checked: true, Error: nonEmpty(r.Error, "The Van Link website gave an unexpected answer")}
	}
	return phoneStatus{Checked: true, SourceOpen: r.SourceOpen, State: r.State, Phone: r.Name, OnAir: r.OnAir}
}

func (a *App) phoneViews() []PhoneView {
	var out []PhoneView
	for _, p := range a.phones {
		out = append(out, PhoneView{ID: p.entry.ID, Name: p.entry.Name, Code: prettyRoom(p.entry.Room),
			Link: phoneLink(p.entry), OBSSource: phoneInputName(p.entry), OBSReady: p.obsReady, Status: p.status})
	}
	return out
}
