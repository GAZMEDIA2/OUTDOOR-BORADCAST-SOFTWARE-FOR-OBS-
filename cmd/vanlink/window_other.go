//go:build !windows

package main

// runUI on macOS and Linux opens the screen in an app-style browser window
// and waits until Van Link quits.
func runUI(url string, show <-chan struct{}, quit <-chan struct{}, background bool) {
	if !background {
		openBrowser(url)
	}
	for {
		select {
		case <-quit:
			return
		case <-show:
			openBrowser(url)
		}
	}
}

func startWithWindows() {}
