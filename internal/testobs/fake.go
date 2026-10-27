// Package testobs is a stand-in for OBS Studio used in tests. It answers
// obs-websocket requests and, like OBS, really sends video (with ffmpeg) when
// told to stream, and really receives it when a Van Link source is created.
package testobs

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
)

// OBS is a fake OBS instance.
type OBS struct {
	Addr string
	Out  string // file the studio "source" writes received video to

	ln        net.Listener
	mu        sync.Mutex
	Requests  []string
	server    string
	streaming *exec.Cmd
	receiving *exec.Cmd
	browser   map[string]string // browser sources created: name -> page address
}

// Start listens on a free local port. out = where received video is saved.
func Start(out string) (*OBS, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	o := &OBS{Addr: ln.Addr().String(), ln: ln, Out: out}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go o.serve(c)
		}
	}()
	return o, nil
}

// Close stops everything.
func (o *OBS) Close() {
	o.ln.Close()
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, c := range []*exec.Cmd{o.streaming, o.receiving} {
		if c != nil && c.Process != nil {
			c.Process.Kill()
			c.Wait()
		}
	}
}

// Streaming reports whether the fake stream output is running.
func (o *OBS) Streaming() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.streaming != nil
}

func (o *OBS) serve(conn net.Conn) {
	defer conn.Close()
	br := bufio.NewReader(conn)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	h := sha1.Sum([]byte(req.Header.Get("Sec-WebSocket-Key") + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " +
		base64.StdEncoding.EncodeToString(h[:]) + "\r\n\r\n"))
	var wmu sync.Mutex
	send := func(v any) {
		b, _ := json.Marshal(v)
		hdr := []byte{0x81}
		switch {
		case len(b) < 126:
			hdr = append(hdr, byte(len(b)))
		case len(b) < 65536:
			hdr = append(hdr, 126, byte(len(b)>>8), byte(len(b)))
		default:
			hdr = append(hdr, 127)
			var l [8]byte
			binary.BigEndian.PutUint64(l[:], uint64(len(b)))
			hdr = append(hdr, l[:]...)
		}
		wmu.Lock()
		conn.Write(append(hdr, b...))
		wmu.Unlock()
	}
	send(map[string]any{"op": 0, "d": map[string]any{"rpcVersion": 1}}) // no password
	if _, err := readFrame(br); err != nil {
		return
	}
	send(map[string]any{"op": 2, "d": map[string]any{"negotiatedRpcVersion": 1}})
	for {
		msg, err := readFrame(br)
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
		if json.Unmarshal(msg, &r) != nil {
			continue
		}
		out := o.handle(r.D.Type, r.D.Data)
		send(map[string]any{"op": 7, "d": map[string]any{"requestType": r.D.Type, "requestId": r.D.ID,
			"requestStatus": map[string]any{"result": true, "code": 100}, "responseData": out}})
	}
}

func (o *OBS) handle(typ string, d map[string]any) any {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.Requests = append(o.Requests, typ)
	switch typ {
	case "SetStreamServiceSettings":
		if s, ok := d["streamServiceSettings"].(map[string]any); ok {
			o.server, _ = s["server"].(string)
		}
	case "StartStream":
		if o.streaming == nil && o.server != "" {
			o.streaming = exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-re",
				"-f", "lavfi", "-i", "testsrc2=size=1280x720:rate=30", "-f", "lavfi", "-i", "sine=frequency=440",
				"-c:v", "libx264", "-preset", "veryfast", "-b:v", "3M", "-g", "60", "-c:a", "aac",
				"-f", "mpegts", o.server)
			if o.streaming.Start() != nil {
				o.streaming = nil
			}
		}
	case "StopStream":
		if o.streaming != nil {
			o.streaming.Process.Kill()
			o.streaming.Wait()
			o.streaming = nil
		}
	case "GetStreamStatus":
		return map[string]any{"outputActive": o.streaming != nil}
	case "GetSceneList":
		return map[string]any{"scenes": []map[string]string{{"sceneName": "Main"}}}
	case "GetInputList":
		return map[string]any{"inputs": []map[string]string{}}
	case "CreateInput", "SetInputSettings":
		if s, ok := d["inputSettings"].(map[string]any); ok && s["url"] != nil {
			if o.browser == nil {
				o.browser = map[string]string{}
			}
			name, _ := d["inputName"].(string)
			o.browser[name], _ = s["url"].(string)
		} else if ok && o.receiving == nil {
			url, _ := s["input"].(string)
			o.receiving = exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
				"-i", url, "-c", "copy", "-t", recvSeconds(), o.Out)
			if o.receiving.Start() != nil {
				o.receiving = nil
			}
		}
	case "GetProfileParameter":
		return map[string]any{"parameterValue": "true"} // pretend Stream Delay is on
	case "GetMediaInputStatus":
		return map[string]any{"mediaState": "OBS_MEDIA_STATE_PLAYING"}
	}
	return map[string]any{}
}

// BrowserURL is the page address of a browser source OBS was asked to create.
func (o *OBS) BrowserURL(name string) string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.browser[name]
}

// WaitReceived waits for the receiving ffmpeg to finish its clip.
func (o *OBS) WaitReceived() error {
	o.mu.Lock()
	c := o.receiving
	o.mu.Unlock()
	if c == nil {
		return io.EOF
	}
	return c.Wait()
}

func readFrame(br *bufio.Reader) ([]byte, error) {
	var msg []byte
	for {
		var h [2]byte
		if _, err := io.ReadFull(br, h[:]); err != nil {
			return nil, err
		}
		n := uint64(h[1] & 0x7F)
		switch n {
		case 126:
			var l [2]byte
			io.ReadFull(br, l[:])
			n = uint64(binary.BigEndian.Uint16(l[:]))
		case 127:
			var l [8]byte
			io.ReadFull(br, l[:])
			n = binary.BigEndian.Uint64(l[:])
		}
		var mask [4]byte
		if h[1]&0x80 != 0 {
			io.ReadFull(br, mask[:])
		}
		p := make([]byte, n)
		if _, err := io.ReadFull(br, p); err != nil {
			return nil, err
		}
		for i := range p {
			p[i] ^= mask[i%4]
		}
		if h[0]&0x0F == 0x8 {
			return nil, io.EOF
		}
		msg = append(msg, p...)
		if h[0]&0x80 != 0 {
			return msg, nil
		}
	}
}

// recvSeconds is how long the stand-in studio records (TESTOBS_SECONDS, default 8).
func recvSeconds() string {
	if v := os.Getenv("TESTOBS_SECONDS"); v != "" {
		return v
	}
	return "8"
}
