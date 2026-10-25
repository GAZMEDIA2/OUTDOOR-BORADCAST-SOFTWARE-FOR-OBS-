package licence

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func sign(t *testing.T, sk ed25519.PrivateKey, p map[string]any) string {
	b, _ := json.Marshal(p)
	return base64.RawURLEncoding.EncodeToString(b) + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(sk, b))
}

func TestDecide(t *testing.T) {
	pk, sk, _ := ed25519.GenerateKey(nil)
	old := PublicKey
	PublicKey = base64.StdEncoding.EncodeToString(pk)
	defer func() { PublicKey = old }()
	now := time.Unix(1_800_000_000, 0)
	lic := func(plan string, exp, valid int64, left any) string {
		return sign(t, sk, map[string]any{"plan": plan, "expires": exp, "valid_until": valid, "device": "dev1",
			"max_vans": 1, "trial_seconds_left_today": left, "email": "a@b.c"})
	}
	day := int64(86400)
	cases := []struct {
		name   string
		lic    string
		dev    string
		now    time.Time
		max    time.Time
		used   int
		ok     bool
		reason string
	}{
		{"valid standard", lic("standard", now.Unix()+day, now.Unix()+7*day, nil), "dev1", now, time.Time{}, 0, true, ""},
		{"expired subscription, even offline", lic("standard", now.Unix()-1, now.Unix()+7*day, nil), "dev1", now, time.Time{}, 0, false, "expired"},
		{"trial ended", lic("trial", now.Unix()-1, now.Unix()+7*day, 1800), "dev1", now, time.Time{}, 0, false, "expired"},
		{"offline too long", lic("pro", now.Unix()+30*day, now.Unix()-1, nil), "dev1", now, time.Time{}, 0, false, "offline"},
		{"clock wound back", lic("pro", now.Unix()+30*day, now.Unix()+7*day, nil), "dev1", now, now.Add(48 * time.Hour), 0, false, "clock"},
		{"copied to another computer", lic("pro", now.Unix()+30*day, now.Unix()+7*day, nil), "dev2", now, time.Time{}, 0, false, "device"},
		{"forged licence", lic("pro", now.Unix()+30*day, now.Unix()+7*day, nil)[:20] + "x" + lic("pro", now.Unix()+30*day, now.Unix()+7*day, nil)[21:], "dev1", now, time.Time{}, 0, false, "invalid"},
		{"not signed in", "", "dev1", now, time.Time{}, 0, false, "signin"},
		{"owner never expires", lic("owner", 4102444800, now.Unix()+7*day, nil), "dev1", now, time.Time{}, 0, true, ""},
	}
	for _, c := range cases {
		s := Decide(c.lic, c.dev, c.now, c.max, c.used)
		if s.OK != c.ok || (!c.ok && s.Reason != c.reason) {
			t.Errorf("%s: got ok=%v reason=%q (%s)", c.name, s.OK, s.Reason, s.Message)
		}
	}
	s := Decide(lic("trial", now.Unix()+day, now.Unix()+day, 1800), "dev1", now, time.Time{}, 600)
	if s.TrialLeft == nil || *s.TrialLeft != 1200 {
		t.Errorf("trial time left: %v", s.TrialLeft)
	}
}
