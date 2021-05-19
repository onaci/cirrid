package dns

// run a dns server for STACKDOMAIN (as set in cirri container...)
// or maybe $hostname.ona.im
// and ensure that the host has it added to the system'd dns resolution...

import (
	"encoding/json"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/kardianos/service"
	"github.com/miekg/dns"
	"github.com/onaci/cirrid/util"
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
			// TODO: is there a notMe answer?
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

func SetDNSValue(hostname, zone, ipAddress string) error {
	// TODO: maybe there's a dns name string manipulation module
	fullname := hostname
	if !strings.Contains(strings.TrimPrefix(hostname, "."), ".") {
		if !strings.HasPrefix(zone, ".") {
			zone = "." + zone
		}
		fullname = hostname + zone
	}

	if ipAddress == "magic" {
		ipAddress = getIpAddress()
	}

	domainsToAddresses[fullname+"."] = ipAddress

	return nil
}

func EnsureWildCards() {
	for host, ip := range domainsToAddresses {
		if strings.HasPrefix(host, ".") {
			continue
		}
		done := false
		for h, _ := range domainsToAddresses {
			if h == "."+host {
				done = true
				break
			}
		}
		if !done {
			domainsToAddresses["."+host] = ip
		}
	}
}

func DnsServer(l service.Logger) {
	logger = l

	srv := &dns.Server{Addr: getDNSServerIPAddress() + ":" + strconv.Itoa(port), Net: "udp"}
	srv.Handler = &handler{}
	logger.Infof("DNS listening on IP %s\n", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		logger.Errorf("Failed to set udp listener %s\n", err.Error())
	}
}

// TODO: get STACKDOMAIN from cirri container
func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		logger.Error(err)
	}
	logger.Infof("Hostname: %s", hostname)
	return hostname
}

func GetCirriStackdomain() string {
	stackdomain := ""
	// get docker bridge's gateway address (linux only)
	out, stderr, err := util.RunLocally(util.Options{}, "docker", "inspect", "cirri")
	//logger.Infof("%s\n", out)
	logger.Infof("STDERR: %s\n", stderr)
	if err != nil {
		logger.Infof("ERROR: %s\n", err)
		return stackdomain
	} else {
		var result []interface{}
		json.Unmarshal([]byte(out), &result)
		cfg := result[0].(map[string]interface{})
		Config := cfg["Config"].(map[string]interface{})
		Env := Config["Env"].([]interface{})

		stackdomainPrefix := "STACKDOMAIN="
		for _, r := range Env {
			e := r.(string)
			if strings.HasPrefix(e, stackdomainPrefix) {
				stackdomain = strings.TrimPrefix(e, stackdomainPrefix)
			}
		}

		logger.Infof("found stackdomain from cirri: (%s)\n", stackdomain)
	}
	return stackdomain
}
