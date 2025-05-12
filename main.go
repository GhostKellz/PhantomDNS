// main.go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/miekg/dns"
)

const upstream = "1.1.1.1:53"

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true

	c := new(dns.Client)
	resp, _, err := c.Exchange(r, upstream)
	if err != nil {
		log.Printf("[error] upstream query failed: %v", err)
		msg.SetRcode(r, dns.RcodeServerFailure)
	} else {
		msg = resp
	}

	_ = w.WriteMsg(msg)
}

func main() {
	port := ":53"
	if os.Geteuid() != 0 {
		log.Fatal("PhantomDNS must be run as root to bind to port 53")
	}

	dns.HandleFunc(".", handleDNSRequest)

	srv := &dns.Server{Addr: port, Net: "udp"}
	fmt.Println("PhantomDNS listening on", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %s", err.Error())
	}
}
