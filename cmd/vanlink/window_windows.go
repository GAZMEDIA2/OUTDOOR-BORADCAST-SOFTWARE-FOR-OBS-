//go:build windows

package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	webview2 "github.com/jchv/go-webview2"
)

// The window must live on the program's main thread.
func init() { runtime.LockOSThread() }

// runUI shows Van Link in its own window. Closing the window keeps Van Link
// running (so the OBS panel and any broadcast carry on); opening Van Link
// again brings the window back. It returns when Van Link is quitting.
func runUI(url string, show <-chan struct{}, quit <-chan struct{}, background bool) {
	dataDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "VanLink", "WebView2")
	open := !background
	for {
		if open {
			w := webview2.NewWithOptions(webview2.WebViewOptions{
				AutoFocus: true,
				DataPath:  dataDir,
				WindowOptions: webview2.WindowOptions{
					Title:  "Van Link – Gaz Media Productions",
					Width:  1100,
					Height: 820,
					IconId: 1, // the icon built into VanLink.exe
					Center: true,
				},
			})
			if w == nil {
				// WebView2 is missing (very old Windows 10): use Edge's app window.
				log.Printf("WebView2 not available, opening an Edge app window")
				edgeAppWindow(url)
			} else {
				w.SetSize(1100, 820, webview2.HintNone)
				w.Navigate(url)
				done := make(chan struct{})
				go func() {
					select {
					case <-quit:
						w.Dispatch(func() { w.Terminate() })
					case <-done:
					}
				}()
				w.Run() // until the window is closed or Van Link quits
				close(done)
				w.Destroy()
			}
		}
		select {
		case <-quit:
			return
		case <-show:
			open = true
		}
	}
}

func edgeAppWindow(url string) {
	for _, p := range []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(os.Getenv("ProgramFiles"), `Microsoft\Edge\Application\msedge.exe`),
	} {
		if _, err := os.Stat(p); err == nil {
			cmd := exec.Command(p, "--app="+url, "--window-size=1100,820")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false}
			if cmd.Start() == nil {
				return
			}
		}
	}
	openBrowser(url)
}

// startWithWindows makes Van Link start quietly when the user logs in, so
// its OBS panel is ready as soon as OBS opens.
func startWithWindows() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command("reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
		"/v", "VanLink", "/t", "REG_SZ", "/d", `"`+exe+`" -background`, "/f")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()
}
