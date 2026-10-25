package link

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
)

// Every Van Link datagram starts with this byte. STUN messages start with
// 0x00/0x01 and SRT never reaches the wire unwrapped, so they never collide.
const magic = 0xB7

const (
	tProbe  = 0x01 // "are you there?"   sid | nonce | mac
	tAck    = 0x02 // "yes, I hear you"  sid | nonce | mac
	tStatus = 0x03 // status, both ways sid | json  | mac
	tData   = 0x04 // SRT datagram       payload (accepted only from the paired peer)
	tBye    = 0x05 // "stopping"         sid | mac
)

const macLen = 16

func sign(key []byte, parts ...[]byte) []byte {
	m := hmac.New(sha256.New, key)
	for _, p := range parts {
		m.Write(p)
	}
	return m.Sum(nil)[:macLen]
}

// seal builds  magic | type | sid | body | mac(type|sid|body).
func seal(key, sid []byte, typ byte, body []byte) []byte {
	out := make([]byte, 0, 2+len(sid)+len(body)+macLen)
	out = append(out, magic, typ)
	out = append(out, sid...)
	out = append(out, body...)
	return append(out, sign(key, []byte{typ}, sid, body)...)
}

var errBadPacket = errors.New("bad packet")

// open checks an authenticated message and returns its body.
func open(key, sid []byte, pkt []byte) (typ byte, body []byte, err error) {
	if len(pkt) < 2+len(sid)+macLen || pkt[0] != magic {
		return 0, nil, errBadPacket
	}
	typ = pkt[1]
	if string(pkt[2:2+len(sid)]) != string(sid) {
		return 0, nil, errBadPacket
	}
	body = pkt[2+len(sid) : len(pkt)-macLen]
	if !hmac.Equal(pkt[len(pkt)-macLen:], sign(key, []byte{typ}, sid, body)) {
		return 0, nil, errBadPacket
	}
	return typ, body, nil
}

func wrapData(payload []byte) []byte {
	out := make([]byte, 2+len(payload))
	out[0], out[1] = magic, tData
	copy(out[2:], payload)
	return out
}
