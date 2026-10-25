package obsws

import (
	"bufio"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"
)

// fakeOBS speaks just enough obs-websocket v5 to test the client:
// hello with a password challenge, identify, and request/response.
func fakeOBS(t *testing.T, password string, handle func(typ string, data map[string]any) (bool, any)) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveFake(conn, password, handle)
		}
	}()
	return ln.Addr().String()
}

func serveFake(conn net.Conn, password string, handle func(string, map[string]any) (bool, any)) {
	defer conn.Close()
	br := bufio.NewReader(conn)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	h := sha1.Sum([]byte(req.Header.Get("Sec-WebSocket-Key") + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " +
		base64.StdEncoding.EncodeToString(h[:]) + "\r\n\r\n"))
	c := &Client{conn: conn, br: br}
	send := func(v any) {
		b, _ := json.Marshal(v)
		// server frames are unmasked
		hdr := []byte{0x81}
		if len(b) < 126 {
			hdr = append(hdr, byte(len(b)))
		} else {
			hdr = append(hdr, 126, byte(len(b)>>8), byte(len(b)))
		}
		conn.Write(append(hdr, b...))
	}
	const salt, challenge = "c2FsdA==", "Y2hhbGxlbmdl"
	send(map[string]any{"op": 0, "d": map[string]any{"rpcVersion": 1,
		"authentication": map[string]string{"salt": salt, "challenge": challenge}}})
	msg, err := c.readMessage()
	if err != nil {
		return
	}
	var ident struct {
		D struct {
			Auth string `json:"authentication"`
		} `json:"d"`
	}
	json.Unmarshal(msg, &ident)
	s := sha256.Sum256([]byte(password + salt))
	a := sha256.Sum256([]byte(base64.StdEncoding.EncodeToString(s[:]) + challenge))
	if ident.D.Auth != base64.StdEncoding.EncodeToString(a[:]) {
		return // like OBS: close on bad password
	}
	send(map[string]any{"op": 2, "d": map[string]any{"negotiatedRpcVersion": 1}})
	for {
		msg, err := c.readMessage()
		if err != nil {
			return
		}
		var r struct {
			D struct {
				Type string         `json:"requestType"`
				ID   string         `json:"requestId"`
				Data map[string]any `json:"requestData"`
			} `json:"d"`
		}
		json.Unmarshal(msg, &r)
		ok, out := handle(r.D.Type, r.D.Data)
		send(map[string]any{"op": 7, "d": map[string]any{"requestType": r.D.Type, "requestId": r.D.ID,
			"requestStatus": map[string]any{"result": ok, "code": 100}, "responseData": out}})
	}
}

func TestDialAndRequests(t *testing.T) {
	var got []string
	addr := fakeOBS(t, "secret", func(typ string, d map[string]any) (bool, any) {
		got = append(got, typ)
		switch typ {
		case "GetSceneList":
			return true, map[string]any{"scenes": []map[string]string{{"sceneName": "Main"}}}
		case "GetInputList":
			return true, map[string]any{"inputs": []map[string]string{}}
		case "GetStreamStatus":
			return true, map[string]any{"outputActive": true, "outputSkippedFrames": 3}
		}
		return true, map[string]any{}
	})
	c, err := Dial(addr, "secret", 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.SetStreamDestination(SRTURL("127.0.0.1", 9710, "caller", 1000, "pp")); err != nil {
		t.Fatal(err)
	}
	if err := c.EnsureReceiver("Van Link - Van 1", "srt://x"); err != nil {
		t.Fatal(err)
	}
	st, err := c.StreamStatus()
	if err != nil || !st.Active || st.Skipped != 3 {
		t.Fatalf("stream status: %+v %v", st, err)
	}
	want := []string{"SetStreamServiceSettings", "GetSceneList", "CreateScene", "GetInputList", "CreateInput", "GetStreamStatus"}
	if len(got) != len(want) {
		t.Fatalf("requests: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("requests: %v", got)
		}
	}
}

func TestWrongPassword(t *testing.T) {
	addr := fakeOBS(t, "secret", func(string, map[string]any) (bool, any) { return true, nil })
	if _, err := Dial(addr, "wrong", 2*time.Second); err != ErrAuth {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestSRTURL(t *testing.T) {
	u := SRTURL("127.0.0.1", 9711, "listener", 1000, "abc")
	if u != "srt://127.0.0.1:9711?mode=listener&latency=1000000&passphrase=abc&pbkeylen=16" {
		t.Fatal(u)
	}
}
