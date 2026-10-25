// Package pairing creates and reads the codes the studio and van swap
// (by WhatsApp, SMS, etc.) so they can find each other without a server.
//
// Studio code: session id, shared secret, SRT passphrase, latency and the
// studio's candidate addresses. Van reply code: session id and the van's
// candidate addresses (no secrets).
package pairing

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const prefix = "VL1."

// Code is the content of a pairing code.
type Code struct {
	Role       string   `json:"r"`           // "studio" or "van"
	Session    string   `json:"s"`           // 16 hex chars
	Key        string   `json:"k,omitempty"` // 64 hex chars, studio code only
	Passphrase string   `json:"p,omitempty"` // SRT passphrase, studio code only
	LatencyMS  int      `json:"l,omitempty"` // SRT latency, studio code only
	Name       string   `json:"n,omitempty"` // friendly name, e.g. "Studio" or "Van 1"
	Candidates []string `json:"c"`           // "ip:port" / "[ipv6]:port"
}

// Encode turns a code into the short text people share.
func (c Code) Encode() (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// Decode reads a shared code. It tolerates spaces/newlines added by chat apps.
func Decode(s string) (Code, error) {
	var c Code
	s = strings.Join(strings.Fields(s), "")
	if !strings.HasPrefix(s, prefix) {
		return c, errors.New("this is not a Van Link code")
	}
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, prefix))
	if err != nil {
		return c, errors.New("the code is incomplete - copy the whole code and try again")
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, errors.New("the code is damaged - copy the whole code and try again")
	}
	if c.Role != "studio" && c.Role != "van" {
		return c, errors.New("unknown code type")
	}
	if len(c.Session) != 16 {
		return c, errors.New("the code is damaged")
	}
	if c.Role == "studio" && (len(c.Key) != 64 || c.Passphrase == "") {
		return c, errors.New("the studio code is damaged")
	}
	return c, nil
}

// KeyBytes returns the shared secret.
func (c Code) KeyBytes() []byte {
	b, _ := hex.DecodeString(c.Key)
	return b
}

// SessionBytes returns the 8-byte session id.
func (c Code) SessionBytes() []byte {
	b, _ := hex.DecodeString(c.Session)
	return b
}

// NewStudioSecrets creates a fresh session id, key and SRT passphrase.
func NewStudioSecrets() (session, key, passphrase string) {
	return randHex(8), randHex(32), randAlnum(24)
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("random source failed: %v", err))
	}
	return hex.EncodeToString(b)
}

func randAlnum(n int) string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}
