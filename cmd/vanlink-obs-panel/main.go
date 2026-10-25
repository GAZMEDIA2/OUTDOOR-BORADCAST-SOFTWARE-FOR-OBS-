// vanlink-obs-panel adds (or removes) the Van Link panel in OBS Studio.
// Used by the "Van Link OBS Panel" installer.
//
//	vanlink-obs-panel install   exit 0 = done, 2 = OBS is open, 3 = OBS not found, 1 = other error
//	vanlink-obs-panel remove
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/GAZMEDIA2/vanlink/internal/obsdock"
)

const dockURL = "http://127.0.0.1:8765/?dock=1"

func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: vanlink-obs-panel install|remove")
		os.Exit(1)
	}
	if obsdock.OBSRunning() {
		fmt.Println(obsdock.ErrOBSRunning)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "install":
		if !obsdock.OBSFound() {
			fmt.Println("OBS settings not found - open and close OBS once, then try again")
			os.Exit(3)
		}
		err = obsdock.Install(dockURL)
		if err == nil {
			_, err = obsdock.EnableWebSocket() // so Van Link can connect to OBS by itself
		}
	case "remove":
		err = obsdock.Remove()
	default:
		err = errors.New("unknown command")
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("done")
}
