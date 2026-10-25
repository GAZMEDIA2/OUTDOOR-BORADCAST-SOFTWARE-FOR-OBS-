// Package licence checks the signed licence the Van Link website issues and
// decides whether the app may be used - online or offline.
package licence

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// PublicKey verifies licences signed by vanlink.gazgoc.com.ng. Only the
// website holds the matching secret key, so licences cannot be forged.
var PublicKey = "YoapNMRwLFAJmRy0FuNK0RFheASh2ZjnFgvsRNoddlU="

// Payload is what the website signs.
type Payload struct {
	Email              string `json:"email"`
	Name               string `json:"name"`
	Plan               string `json:"plan"` // trial, standard, pro, owner, none
	Expires            int64  `json:"expires"`
	MaxVans            int    `json:"max_vans"`
	TrialMinutesPerDay int    `json:"trial_minutes_per_day"`
	TrialSecondsLeft   *int   `json:"trial_seconds_left_today"`
	Device             string `json:"device"`
	Issued             int64  `json:"issued"`
	ValidUntil         int64  `json:"valid_until"`
}

// Parse verifies the signature and returns the payload.
func Parse(lic string, publicKeyB64 string) (Payload, error) {
	var p Payload
	parts := strings.Split(lic, ".")
	if len(parts) != 2 {
		return p, errors.New("damaged licence")
	}
	body, err1 := base64.RawURLEncoding.DecodeString(parts[0])
	sig, err2 := base64.RawURLEncoding.DecodeString(parts[1])
	pk, err3 := base64.StdEncoding.DecodeString(publicKeyB64)
	if err1 != nil || err2 != nil || err3 != nil || len(pk) != ed25519.PublicKeySize {
		return p, errors.New("damaged licence")
	}
	if !ed25519.Verify(ed25519.PublicKey(pk), body, sig) {
		return p, errors.New("licence signature is not valid")
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return p, errors.New("damaged licence")
	}
	return p, nil
}

// Status is the app's decision about use.
type Status struct {
	OK       bool   `json:"ok"`
	Reason   string `json:"reason"` // signin, expired, offline, clock, device, invalid
	Message  string `json:"message"`
	Plan     string `json:"plan"`
	PlanName string `json:"planName"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Expires  int64  `json:"expires"`
	MaxVans  int    `json:"maxVans"`
	// Trial only: broadcast seconds left today (nil = unlimited).
	TrialLeft *int `json:"trialLeft"`
}

// PlanName is the name shown to people.
func PlanName(p string) string {
	switch p {
	case "trial":
		return "Free trial"
	case "standard":
		return "Standard"
	case "pro":
		return "Pro"
	case "owner":
		return "Owner (unlimited)"
	}
	return "No licence"
}

// Decide works out whether the app may be used right now. It needs no
// internet: the expiry date and signature are inside the licence.
//
//	maxSeen  - the latest time this computer has ever seen (to catch a clock wound back)
//	usedToday - trial broadcast seconds used locally since the licence was issued
func Decide(lic, deviceID string, now, maxSeen time.Time, usedSince int) Status {
	if lic == "" {
		return Status{Reason: "signin", Message: "Sign in to use Van Link."}
	}
	p, err := Parse(lic, PublicKey)
	if err != nil {
		return Status{Reason: "invalid", Message: "Your licence could not be checked. Sign in again."}
	}
	s := Status{Plan: p.Plan, PlanName: PlanName(p.Plan), Email: p.Email, Name: p.Name, Expires: p.Expires, MaxVans: p.MaxVans}
	switch {
	case p.Device != deviceID:
		s.Reason, s.Message = "device", "This licence belongs to another computer. Sign in again."
	case !maxSeen.IsZero() && now.Before(maxSeen.Add(-2*time.Hour)):
		s.Reason, s.Message = "clock", "This computer's date and time were set back. Correct the clock to keep using Van Link."
	case p.Plan == "none" || p.Expires <= 0:
		s.Reason, s.Message = "expired", "You have no active licence. Buy a plan or enter a registration code."
	case now.Unix() >= p.Expires:
		if p.Plan == "trial" {
			s.Reason, s.Message = "expired", "Your free trial has ended. Buy Standard or Pro, or enter a registration code."
		} else {
			s.Reason, s.Message = "expired", "Your "+PlanName(p.Plan)+" subscription has ended. Renew it, or enter a registration code."
		}
	case now.Unix() >= p.ValidUntil:
		s.Reason, s.Message = "offline", "Van Link needs to check your licence. Connect this computer to the internet."
	default:
		s.OK = true
	}
	if p.Plan == "trial" && p.TrialSecondsLeft != nil {
		left := *p.TrialSecondsLeft - usedSince
		if left < 0 {
			left = 0
		}
		s.TrialLeft = &left
	}
	return s
}
