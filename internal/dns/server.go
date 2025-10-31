package dns

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

type DNSServer struct {
	server         *dns.Server
	blockedDomains map[string]bool
	upstreamDNS    string
	mutex          sync.RWMutex
	running        bool
}

func NewDNSServer() *DNSServer {
	return &DNSServer{
		blockedDomains: make(map[string]bool),
		upstreamDNS:    "8.8.8.8:53", // Google DNS as upstream
		running:        false,
	}
}

func (ds *DNSServer) Start() error {
	if ds.running {
		return fmt.Errorf("DNS server already running")
	}

	mux := dns.NewServeMux()
	mux.HandleFunc(".", ds.handleDNSRequest)

	// Try ports in order: 6666, 5353 (avoid 53 for now)
	ports := []string{":6666", ":5353"}
	
	for _, port := range ports {
		ds.server = &dns.Server{
			Addr:    port,
			Net:     "udp",
			Handler: mux,
		}
		
		log.Printf("Trying to start DNS server on %s", port)
		
		// Try to start server synchronously first
		if err := ds.tryStartServer(); err == nil {
			ds.running = true
			log.Printf("DNS server started successfully on %s", port)
			return nil
		} else {
			log.Printf("Failed to bind to %s: %v", port, err)
		}
	}
	
	return fmt.Errorf("failed to start DNS server on any available port")
}

func (ds *DNSServer) tryStartServer() error {
	// Test if we can bind to the port
	conn, err := net.ListenPacket("udp", ds.server.Addr)
	if err != nil {
		return err
	}
	conn.Close()
	
	// Start server in background
	go func() {
		if err := ds.server.ListenAndServe(); err != nil {
			log.Printf("DNS server error: %v", err)
			ds.running = false
		}
	}()
	
	return nil
}

func (ds *DNSServer) Stop() error {
	if !ds.running {
		return nil
	}

	ds.running = false
	if ds.server != nil {
		return ds.server.Shutdown()
	}
	return nil
}

func (ds *DNSServer) BlockDomain(domain string) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	ds.blockedDomains[strings.ToLower(domain)] = true
	log.Printf("Blocked domain: %s", domain)
}

func (ds *DNSServer) UnblockDomain(domain string) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	delete(ds.blockedDomains, strings.ToLower(domain))
	log.Printf("Unblocked domain: %s", domain)
}

func (ds *DNSServer) UnblockAll() {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	ds.blockedDomains = make(map[string]bool)
	log.Println("Unblocked all domains")
}

func (ds *DNSServer) IsBlocked(domain string) bool {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()
	
	domain = strings.ToLower(domain)
	
	// Check exact match
	if ds.blockedDomains[domain] {
		return true
	}
	
	// Check subdomains
	for blockedDomain := range ds.blockedDomains {
		if strings.HasSuffix(domain, "."+blockedDomain) {
			return true
		}
	}
	
	return false
}

func (ds *DNSServer) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	msg.Authoritative = true

	for _, question := range r.Question {
		domain := strings.TrimSuffix(question.Name, ".")
		
		if ds.IsBlocked(domain) {
			// Return blocked response
			ds.handleBlockedDomain(&msg, question)
			log.Printf("Blocked DNS request for: %s", domain)
		} else {
			// Forward to upstream DNS
			ds.forwardToUpstream(&msg, question)
		}
	}

	w.WriteMsg(&msg)
}

func (ds *DNSServer) handleBlockedDomain(msg *dns.Msg, question dns.Question) {
	switch question.Qtype {
	case dns.TypeA:
		// Return localhost for blocked domains
		rr, _ := dns.NewRR(fmt.Sprintf("%s A 127.0.0.1", question.Name))
		msg.Answer = append(msg.Answer, rr)
	case dns.TypeAAAA:
		// Return IPv6 localhost for blocked domains
		rr, _ := dns.NewRR(fmt.Sprintf("%s AAAA ::1", question.Name))
		msg.Answer = append(msg.Answer, rr)
	default:
		// Return NXDOMAIN for other types
		msg.SetRcode(msg, dns.RcodeNameError)
	}
}

func (ds *DNSServer) forwardToUpstream(msg *dns.Msg, question dns.Question) {
	client := new(dns.Client)
	upstreamMsg := new(dns.Msg)
	upstreamMsg.SetQuestion(question.Name, question.Qtype)

	response, _, err := client.Exchange(upstreamMsg, ds.upstreamDNS)
	if err != nil {
		log.Printf("Failed to query upstream DNS: %v", err)
		msg.SetRcode(msg, dns.RcodeServerFailure)
		return
	}

	// Copy answers from upstream response
	msg.Answer = append(msg.Answer, response.Answer...)
	msg.Ns = append(msg.Ns, response.Ns...)
	msg.Extra = append(msg.Extra, response.Extra...)
}

func (ds *DNSServer) IsRunning() bool {
	return ds.running
}