package main

import (
	"fmt"
	"log"
	"time"

	"github.com/miekg/dns"
)

func main() {
	fmt.Println("Simple DNS Server Test")
	
	// Create a simple DNS handler
	dns.HandleFunc(".", handleDNS)
	
	// Start server on port 6666
	server := &dns.Server{Addr: ":6666", Net: "udp"}
	
	fmt.Println("Starting DNS server on :6666")
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	
	// Wait a bit for server to start
	time.Sleep(1 * time.Second)
	fmt.Println("DNS server running. Test with: dig @127.0.0.1 -p 6666 test.com")
	
	// Keep running for 30 seconds
	time.Sleep(30 * time.Second)
	fmt.Println("Stopping server")
}

func handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	
	for _, question := range r.Question {
		domain := question.Name
		fmt.Printf("DNS query for: %s\n", domain)
		
		if question.Qtype == dns.TypeA {
			// Return 127.0.0.1 for all A queries
			rr, _ := dns.NewRR(fmt.Sprintf("%s A 127.0.0.1", domain))
			msg.Answer = append(msg.Answer, rr)
		}
	}
	
	w.WriteMsg(&msg)
}