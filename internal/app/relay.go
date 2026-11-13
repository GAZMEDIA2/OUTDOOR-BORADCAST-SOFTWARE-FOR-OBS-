package app

import (
	"os"
	"strconv"

	"github.com/GAZMEDIA2/vanlink/internal/relay"
)

// RelayServer is the Van Link relay, used only when the van and the studio
// cannot reach each other directly. VANLINK_RELAY overrides it (tests).
var RelayServer = "relay.vanlink.gazgoc.com.ng:" + strconv.Itoa(relay.DefaultPort)

func (a *App) relayAddr() string {
	if a.store.get().NoRelay {
		return ""
	}
	if v := os.Getenv("VANLINK_RELAY"); v != "" {
		return v
	}
	return RelayServer
}

// relayLicence is the licence the relay checks ("" = not signed in).
func (a *App) relayLicence() string { return a.store.get().Account.Licence }
