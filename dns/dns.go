package dns

// run a dns server for STACKDOMAIN (as set in cirri container...)
// or maybe $hostname.ona.im
// and ensure that the host has it added to the system'd dns resolution...

import (
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/kardianos/service"
	"github.com/miekg/dns"
)

// test using:
//      dig @127.0.0.1 -p 9856 host.ona.im

var domainsToAddresses map[string]string = map[string]string{
	"host.ona.im.":  "104.198.14.52",
	".host.ona.im.": "104.198.14.52", // wildcard
}

type handler struct{}

func (this *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	switch r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		domain := msg.Question[0].Name
		address, ok := domainsToAddresses[domain]
		if !ok {
			firstDot := strings.Index(domain, ".")
			domainSuffix := domain[firstDot:]
			address, ok = domainsToAddresses[domainSuffix]
		}
		if ok {
			logger.Infof("DNS request for %s answered with %s\n", domain, address)

			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
				A:   net.ParseIP(address),
			})
		} else {
			//if strings.HasSuffix(domain, "ona.im.") {
			if strings.Contains(domain, "ona.im") {
				logger.Infof("DNS request for (%s) failed\n", domain)
			}
		}
	}
	w.WriteMsg(&msg)
}

var port = 53
var logger service.Logger

func SetLogger(l service.Logger) {
	logger = l
}

func SetDNSValues() {
	hostname := getHostname()
	ipAddress := getIpAddress()
	domainsToAddresses[hostname+".ona.im."] = ipAddress
	domainsToAddresses["."+hostname+".ona.im."] = ipAddress

	// TODO: don't ship this!
	domainsToAddresses["t.ona.im."] = ipAddress
	domainsToAddresses[".t.ona.im."] = ipAddress
	domainsToAddresses["portal.ereefs.info."] = "203.100.31.16"
	domainsToAddresses["data.ereefs.info."] = "203.100.31.16"
}

func DnsServer(l service.Logger) {
	logger = l

	srv := &dns.Server{Addr: getDNSServerIPAddress() + ":" + strconv.Itoa(port), Net: "udp"}
	srv.Handler = &handler{}
	logger.Infof("DNS listening on port %d\n", port)
	if err := srv.ListenAndServe(); err != nil {
		logger.Errorf("Failed to set udp listener %s\n", err.Error())
	}
}

// TODO: get STACKDOMAIN from cirri container
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		logger.Error(err)
	}
	logger.Infof("Hostname: %s", hostname)
	return hostname
}
