// vanlink-relay is the optional relay server for Van Link: it carries the
// video between a van and its studio when they cannot reach each other
// directly. Run it on a server with a public IP address and UDP port 47800 open.
package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/GAZMEDIA2/vanlink/internal/relay"
)

var version = "dev"

func main() {
	addr := flag.String("listen", ":47800", "UDP address to listen on")
	flag.Parse()
	conn, err := net.ListenUDP("udp", mustResolve(*addr))
	if err != nil {
		log.Fatal(err)
	}
	_ = conn.SetReadBuffer(8 << 20)
	_ = conn.SetWriteBuffer(8 << 20)
	srv := &relay.Server{Logf: log.Printf}
	log.Printf("Van Link relay %s listening on %s (UDP)", version, conn.LocalAddr())
	go func() {
		for range time.Tick(5 * time.Minute) {
			p, w := srv.Sessions()
			log.Printf("sessions: %d connected, %d waiting", p, w)
		}
	}()
	log.Fatal(srv.Serve(conn))
}

func mustResolve(a string) *net.UDPAddr {
	u, err := net.ResolveUDPAddr("udp", a)
	if err != nil {
		log.Fatal(err)
	}
	return u
}
