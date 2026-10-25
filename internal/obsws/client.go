// Package obsws is a small client for OBS Studio's built-in remote control
// (obs-websocket, protocol v5, included in OBS 28 and newer). Van Link uses it
// to set the van's stream destination, start/stop streaming, and create the
// studio's receiver source - so nobody has to type addresses into OBS.
package obsws

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Client is a connection to OBS.
type Client struct {
	conn net.Conn
	br   *bufio.Reader
	wmu  sync.Mutex

	mu      sync.Mutex
	pending map[string]chan response
	nextID  int
	closed  chan struct{}
	err     error
}

type response struct {
	Status struct {
		Result  bool   `json:"result"`
		Code    int    `json:"code"`
		Comment string `json:"comment"`
	} `json:"requestStatus"`
	ID   string          `json:"requestId"`
	Data json.RawMessage `json:"responseData"`
}

// ErrAuth means OBS refused the password.
var ErrAuth = errors.New("OBS refused the password - check Tools > WebSocket Server Settings in OBS")

// Dial connects and identifies. addr is "host:port", usually "127.0.0.1:4455".
func Dial(addr, password string, timeout time.Duration) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("OBS is not reachable - is OBS open with its WebSocket server turned on? (%v)", err)
	}
	conn.SetDeadline(time.Now().Add(timeout))
	key := make([]byte, 16)
	rand.Read(key)
	k := base64.StdEncoding.EncodeToString(key)
	req := "GET / HTTP/1.1\r\nHost: " + addr + "\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + k + "\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Protocol: obswebsocket.json\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return nil, err
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	h := sha1.Sum([]byte(k + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	if resp.StatusCode != 101 || resp.Header.Get("Sec-WebSocket-Accept") != base64.StdEncoding.EncodeToString(h[:]) {
		conn.Close()
		return nil, errors.New("OBS did not accept the connection")
	}
	c := &Client{conn: conn, br: br, pending: map[string]chan response{}, closed: make(chan struct{})}

	// Hello -> Identify -> Identified
	var hello struct {
		Op int `json:"op"`
		D  struct {
			RPC  int `json:"rpcVersion"`
			Auth *struct {
				Challenge string `json:"challenge"`
				Salt      string `json:"salt"`
			} `json:"authentication"`
		} `json:"d"`
	}
	msg, err := c.readMessage()
	if err != nil || json.Unmarshal(msg, &hello) != nil || hello.Op != 0 {
		conn.Close()
		return nil, errors.New("unexpected reply from OBS")
	}
	ident := map[string]any{"rpcVersion": 1, "eventSubscriptions": 0}
	if hello.D.Auth != nil {
		if password == "" {
			conn.Close()
			return nil, ErrAuth
		}
		s := sha256.Sum256([]byte(password + hello.D.Auth.Salt))
		secret := base64.StdEncoding.EncodeToString(s[:])
		a := sha256.Sum256([]byte(secret + hello.D.Auth.Challenge))
		ident["authentication"] = base64.StdEncoding.EncodeToString(a[:])
	}
	if err := c.send(map[string]any{"op": 1, "d": ident}); err != nil {
		conn.Close()
		return nil, err
	}
	msg, err = c.readMessage()
	if err != nil {
		conn.Close()
		return nil, ErrAuth // OBS closes the socket on a bad password
	}
	var ok struct {
		Op int `json:"op"`
	}
	if json.Unmarshal(msg, &ok) != nil || ok.Op != 2 {
		conn.Close()
		return nil, ErrAuth
	}
	conn.SetDeadline(time.Time{})
	go c.readLoop()
	return c, nil
}

// Close disconnects.
func (c *Client) Close() { c.conn.Close() }

// Alive reports whether the connection is still up.
func (c *Client) Alive() bool {
	select {
	case <-c.closed:
		return false
	default:
		return true
	}
}

// Call sends one request and decodes its reply into out (may be nil).
func (c *Client) Call(typ string, data any, out any) error {
	c.mu.Lock()
	c.nextID++
	id := strconv.Itoa(c.nextID)
	ch := make(chan response, 1)
	c.pending[id] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()
	d := map[string]any{"requestType": typ, "requestId": id}
	if data != nil {
		d["requestData"] = data
	}
	if err := c.send(map[string]any{"op": 6, "d": d}); err != nil {
		return err
	}
	select {
	case r := <-ch:
		if !r.Status.Result {
			return fmt.Errorf("OBS: %s failed (%d %s)", typ, r.Status.Code, r.Status.Comment)
		}
		if out != nil && len(r.Data) > 0 {
			return json.Unmarshal(r.Data, out)
		}
		return nil
	case <-c.closed:
		return errors.New("OBS connection closed")
	case <-time.After(10 * time.Second):
		return errors.New("OBS did not answer")
	}
}

func (c *Client) readLoop() {
	defer close(c.closed)
	for {
		msg, err := c.readMessage()
		if err != nil {
			return
		}
		var env struct {
			Op int      `json:"op"`
			D  response `json:"d"`
		}
		if json.Unmarshal(msg, &env) != nil || env.Op != 7 {
			continue
		}
		c.mu.Lock()
		ch := c.pending[env.D.ID]
		c.mu.Unlock()
		if ch != nil {
			ch <- env.D
		}
	}
}

func (c *Client) send(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.writeFrame(0x1, b)
}

// --- minimal WebSocket framing (RFC 6455, client side) ---

func (c *Client) writeFrame(op byte, payload []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	h := []byte{0x80 | op}
	n := len(payload)
	switch {
	case n < 126:
		h = append(h, 0x80|byte(n))
	case n < 65536:
		h = append(h, 0x80|126, byte(n>>8), byte(n))
	default:
		h = append(h, 0x80|127)
		var l [8]byte
		binary.BigEndian.PutUint64(l[:], uint64(n))
		h = append(h, l[:]...)
	}
	var mask [4]byte
	rand.Read(mask[:])
	h = append(h, mask[:]...)
	m := make([]byte, n)
	for i := range payload {
		m[i] = payload[i] ^ mask[i%4]
	}
	_, err := c.conn.Write(append(h, m...))
	return err
}

func (c *Client) readMessage() ([]byte, error) {
	var msg []byte
	for {
		var h [2]byte
		if _, err := io.ReadFull(c.br, h[:]); err != nil {
			return nil, err
		}
		fin, op := h[0]&0x80 != 0, h[0]&0x0F
		n := uint64(h[1] & 0x7F)
		switch n {
		case 126:
			var l [2]byte
			if _, err := io.ReadFull(c.br, l[:]); err != nil {
				return nil, err
			}
			n = uint64(binary.BigEndian.Uint16(l[:]))
		case 127:
			var l [8]byte
			if _, err := io.ReadFull(c.br, l[:]); err != nil {
				return nil, err
			}
			n = binary.BigEndian.Uint64(l[:])
		}
		if n > 64<<20 {
			return nil, errors.New("message too large")
		}
		var mask [4]byte
		masked := h[1]&0x80 != 0
		if masked {
			if _, err := io.ReadFull(c.br, mask[:]); err != nil {
				return nil, err
			}
		}
		p := make([]byte, n)
		if _, err := io.ReadFull(c.br, p); err != nil {
			return nil, err
		}
		if masked {
			for i := range p {
				p[i] ^= mask[i%4]
			}
		}
		switch op {
		case 0x8:
			return nil, io.EOF
		case 0x9:
			_ = c.writeFrame(0xA, p)
			continue
		case 0xA:
			continue
		}
		msg = append(msg, p...)
		if fin {
			return msg, nil
		}
	}
}
