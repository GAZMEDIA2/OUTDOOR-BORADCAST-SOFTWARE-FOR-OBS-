package obsws

import "fmt"

// SceneName is the scene Van Link creates at the studio for incoming feeds.
const SceneName = "Van Link"

// SRTURL builds the SRT address OBS uses. Latency is given in milliseconds;
// OBS (like FFmpeg) expects microseconds in the URL.
func SRTURL(host string, port int, mode string, latencyMS int, passphrase string) string {
	return fmt.Sprintf("srt://%s:%d?mode=%s&latency=%d&passphrase=%s&pbkeylen=16",
		host, port, mode, latencyMS*1000, passphrase)
}

// SetStreamDestination points OBS's stream output at Van Link (van side).
func (c *Client) SetStreamDestination(server string) error {
	return c.Call("SetStreamServiceSettings", map[string]any{
		"streamServiceType":     "rtmp_custom",
		"streamServiceSettings": map[string]any{"server": server, "key": ""},
	}, nil)
}

// StartStream / StopStream control OBS's stream output.
func (c *Client) StartStream() error { return c.Call("StartStream", nil, nil) }
func (c *Client) StopStream() error  { return c.Call("StopStream", nil, nil) }

// StreamStatus reports whether OBS is streaming and how many frames it skipped.
type StreamStatus struct {
	Active       bool    `json:"outputActive"`
	Reconnecting bool    `json:"outputReconnecting"`
	Skipped      int     `json:"outputSkippedFrames"`
	Total        int     `json:"outputTotalFrames"`
	Congestion   float64 `json:"outputCongestion"`
	DurationMS   float64 `json:"outputDuration"`
}

func (c *Client) StreamStatus() (StreamStatus, error) {
	var s StreamStatus
	err := c.Call("GetStreamStatus", nil, &s)
	return s, err
}

// EnsureReceiver creates (or updates) the studio's media source for one van,
// inside the "Van Link" scene, listening for SRT on localhost only.
func (c *Client) EnsureReceiver(inputName, url string) error {
	settings := map[string]any{
		"is_local_file":       false,
		"input":               url,
		"input_format":        "mpegts",
		"hw_decode":           true,
		"restart_on_activate": false,
		"close_when_inactive": false,
		"clear_on_media_end":  true,
		"reconnect_delay_sec": 1,
		"buffering_mb":        0,                                 // no extra network buffer
		"ffmpeg_options":      "fflags=nobuffer flags=low_delay", // show frames as soon as they arrive (OBS 29.1+)
	}
	return c.ensureInput(inputName, "ffmpeg_source", settings)
}

// EnsurePhoneCamera creates (or updates) the studio's browser source for one
// phone camera, inside the "Van Link" scene. The page at url receives the
// phone's picture and sound; its sound goes to OBS's audio mixer.
func (c *Client) EnsurePhoneCamera(inputName, url string) error {
	return c.ensureInput(inputName, "browser_source", map[string]any{
		"url":                 url,
		"is_local_file":       false,
		"width":               1920,
		"height":              1080,
		"fps_custom":          true,
		"fps":                 30,
		"reroute_audio":       true,  // "Control audio via OBS"
		"shutdown":            false, // keep receiving when not shown
		"restart_when_active": false,
	})
}

// ensureInput creates inputName in the "Van Link" scene, or updates its
// settings if it already exists.
func (c *Client) ensureInput(inputName, kind string, settings map[string]any) error {
	var scenes struct {
		Scenes []struct {
			Name string `json:"sceneName"`
		} `json:"scenes"`
	}
	if err := c.Call("GetSceneList", nil, &scenes); err != nil {
		return err
	}
	have := false
	for _, s := range scenes.Scenes {
		if s.Name == SceneName {
			have = true
		}
	}
	if !have {
		if err := c.Call("CreateScene", map[string]any{"sceneName": SceneName}, nil); err != nil {
			return err
		}
	}
	var inputs struct {
		Inputs []struct {
			Name string `json:"inputName"`
		} `json:"inputs"`
	}
	if err := c.Call("GetInputList", nil, &inputs); err != nil {
		return err
	}
	for _, in := range inputs.Inputs {
		if in.Name == inputName {
			return c.Call("SetInputSettings", map[string]any{
				"inputName": inputName, "inputSettings": settings, "overlay": true,
			}, nil)
		}
	}
	return c.Call("CreateInput", map[string]any{
		"sceneName": SceneName, "inputName": inputName, "inputKind": kind,
		"inputSettings": settings, "sceneItemEnabled": true,
	}, nil)
}

// MediaState returns e.g. "OBS_MEDIA_STATE_PLAYING".
func (c *Client) MediaState(inputName string) (string, error) {
	var r struct {
		State string `json:"mediaState"`
	}
	err := c.Call("GetMediaInputStatus", map[string]any{"inputName": inputName}, &r)
	return r.State, err
}

// RestartMedia re-arms a media source (after the van reconnects).
func (c *Client) RestartMedia(inputName string) error {
	return c.Call("TriggerMediaInputAction", map[string]any{
		"inputName": inputName, "mediaAction": "OBS_WEBSOCKET_MEDIA_INPUT_ACTION_RESTART",
	}, nil)
}

// RemoveInput deletes a receiver source.
func (c *Client) RemoveInput(inputName string) error {
	return c.Call("RemoveInput", map[string]any{"inputName": inputName}, nil)
}

// ProfileParameter reads one OBS profile setting, e.g. ("Output", "DelayEnable").
func (c *Client) ProfileParameter(category, name string) (string, error) {
	var r struct {
		Value *string `json:"parameterValue"`
	}
	err := c.Call("GetProfileParameter", map[string]any{"parameterCategory": category, "parameterName": name}, &r)
	if err != nil || r.Value == nil {
		return "", err
	}
	return *r.Value, nil
}

// SetProfileParameter changes one OBS profile setting.
func (c *Client) SetProfileParameter(category, name, value string) error {
	return c.Call("SetProfileParameter", map[string]any{
		"parameterCategory": category, "parameterName": name, "parameterValue": value}, nil)
}
