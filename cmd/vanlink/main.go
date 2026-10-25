// Van Link: live video from an outside broadcast van to the studio, directly
// over the internet, with OBS Studio. Run it on both computers; it opens its
// screen in the web browser (and can be docked inside OBS).
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/GAZMEDIA2/vanlink/internal/app"
	"github.com/GAZMEDIA2/vanlink/internal/obsdock"
	"github.com/GAZMEDIA2/vanlink/internal/sys"
)

var version = "dev"

func main() {
	// Elevated helper run (Windows permission prompt): do the change and exit.
	if len(os.Args) == 2 && (os.Args[1] == "firewall-add" || os.Args[1] == "firewall-remove") {
		if err := sys.RunElevated(os.Args[1]); err != nil {
			os.Exit(1)
		}
		return
	}

	port := flag.Int("port", 8765, "port for the Van Link screen (this computer only)")
	cfgDir := flag.String("config-dir", "", "where settings are kept (default: your user settings folder)")
	noBrowser := flag.Bool("no-browser", false, "do not open the screen automatically")
	background := flag.Bool("background", false, "start without showing the window (used when Windows starts)")
	installPanel := flag.Bool("install-obs-panel", false, "add the Van Link panel to OBS and exit")
	removePanel := flag.Bool("remove-obs-panel", false, "remove the Van Link panel from OBS and exit")
	flag.Parse()

	dockURL := "http://127.0.0.1:" + strconv.Itoa(*port) + "/?dock=1"
	if *installPanel || *removePanel {
		var err error
		if *installPanel {
			err = obsdock.Install(dockURL)
		} else {
			err = obsdock.Remove()
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if version != "dev" {
		app.Version = version
	}
	log.SetFlags(log.Ldate | log.Ltime)
	setupLogFile(*cfgDir)

	// An older version still running (e.g. after an update)? Ask it to quit.
	if url, v := runningInstance(*port); url != "" && v != app.Version {
		post(url + "api/quit")
		for i := 0; i < 30; i++ {
			time.Sleep(200 * time.Millisecond)
			if u, _ := runningInstance(*port); u == "" {
				break
			}
		}
	}
	// Already running? Bring its window forward instead of starting a second copy.
	if url, _ := runningInstance(*port); url != "" {
		if !*noBrowser && !*background {
			post(url + "api/show")
		}
		return
	}
	log.Printf("Van Link %s starting", app.Version)

	a := app.New(*cfgDir)
	defer a.Close()

	ln, p, err := listen(*port)
	if err != nil {
		log.Fatalf("cannot start the Van Link screen: %v", err)
	}
	url := "http://127.0.0.1:" + strconv.Itoa(p) + "/"
	log.Printf("Van Link screen: %s", url)
	a.SetDockURL(url + "?dock=1")
	startWithWindows()
	// First run: put the Van Link panel into OBS (only possible while OBS is closed).
	if !obsdock.Installed(url+"?dock=1") && obsdock.OBSFound() && !obsdock.OBSRunning() {
		if err := obsdock.Install(url + "?dock=1"); err == nil {
			log.Printf("Van Link panel added to OBS")
		}
	}

	srv := &http.Server{Handler: a.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	quit := make(chan struct{})
	go func() {
		select {
		case <-stop:
		case <-a.QuitRequested():
		}
		close(quit)
	}()
	// The window runs on the main thread until Van Link quits.
	runUI(url, a.ShowRequested(), quit, *noBrowser || *background)
	log.Printf("Van Link stopping")
	srv.Close()
}

// runningInstance returns the screen address of a Van Link that is already
// running on this computer, if any.
func runningInstance(port int) (url, version string) {
	c := &http.Client{Timeout: 700 * time.Millisecond}
	url = "http://127.0.0.1:" + strconv.Itoa(port) + "/"
	r, err := c.Get(url + "api/state")
	if err != nil {
		return "", ""
	}
	defer r.Body.Close()
	var st struct {
		Version string `json:"version"`
	}
	if r.StatusCode == 200 && json.NewDecoder(r.Body).Decode(&st) == nil && st.Version != "" {
		return url, st.Version
	}
	return "", ""
}

func post(u string) {
	req, _ := http.NewRequest("POST", u, nil)
	req.Header.Set("X-VanLink", "1")
	if r, err := (&http.Client{Timeout: 2 * time.Second}).Do(req); err == nil {
		r.Body.Close()
	}
}

// setupLogFile writes the log next to the settings, so problems can be
// looked at later (the Windows app has no console window).
func setupLogFile(cfgDir string) {
	if cfgDir == "" {
		d, err := os.UserConfigDir()
		if err != nil {
			return
		}
		cfgDir = filepath.Join(d, "VanLink")
	}
	if os.MkdirAll(cfgDir, 0o700) != nil {
		return
	}
	p := filepath.Join(cfgDir, "vanlink.log")
	if fi, err := os.Stat(p); err == nil && fi.Size() > 5<<20 {
		os.Rename(p, p+".old")
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	log.SetOutput(io.MultiWriter(f, os.Stderr))
}

// listen uses the requested port, or the next free one if it is taken
// (for example when a second copy is already running).
func listen(port int) (net.Listener, int, error) {
	var last error
	for p := port; p < port+20; p++ {
		ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(p))
		if err == nil {
			return ln, p, nil
		}
		last = err
	}
	return nil, 0, fmt.Errorf("no free port near %d: %w", port, last)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("open %s in your web browser", url)
	}
}
