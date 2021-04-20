package dns

// run a dns server for STACKDOMAIN (as set in cirri container...)
// or maybe $hostname.ona.im
// and ensure that the host has it added to the system'd dns resolution...

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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
			//if strings.HasSuffix(domain, "ona.im.") {
			if strings.Contains(domain, "ona.im") {
				logger.Infof("DNS request for (%s) failed\n", domain)
			}
		}
	}
	w.WriteMsg(&msg)
}

var ipaddr = "127.0.0.98"
var port = 53
var logger service.Logger

func DnsServer(l service.Logger) {
	logger = l
	srv := &dns.Server{Addr: ipaddr + ":" + strconv.Itoa(port), Net: "udp"}
	srv.Handler = &handler{}
	logger.Infof("DNS listening on port %d\n", port)
	if err := srv.ListenAndServe(); err != nil {
		logger.Errorf("Failed to set udp listener %s\n", err.Error())
	}
}

func EnsureResolveConfigured(logger service.Logger) error {

	// method - interestingly, this doesn't cache it, so this might be excellent for dynamic label based answers
	// start this dns server
	// add lines to /etc/systemd/resolved.conf as per https://github.com/hashicorp/consul/issues/4155#issuecomment-394362651
	// [Resolve]
	// DNS=127.0.0.98
	// Domains=~host.ona.im
	// systemctl restart systemd-resolved
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	resolvedConf := "/etc/systemd/resolved.conf"
	file, err := os.Open(resolvedConf)
	if err != nil {
		logger.Errorf("failed to open")
		return err
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var text []string

	// Big nasty assumption that resolved.conf only contains a [Resolve] section
	dnsline := "DNS=127.0.0.98"
	//domainline := "Domains=~" + hostname + ".ona.im,~host.ona.im"
	domainline := "Domains=~ona.im"
	resolvedConfChanged := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "DNS=") {
			if strings.Compare(line, dnsline) != 0 {
				resolvedConfChanged = true
			}
			text = append(text, dnsline)
		} else if strings.HasPrefix(line, "Domains=") {
			if strings.Compare(line, domainline) != 0 {
				resolvedConfChanged = true
			}
			text = append(text, domainline)
		} else {
			text = append(text, line)
		}
	}
	text = append(text, "")

	file.Close()

	if resolvedConfChanged {
		logger.Infof("updating: %s to use %s\n", resolvedConf, domainline)

		linesToWrite := strings.Join(text, "\n")
		err = ioutil.WriteFile(resolvedConf, []byte(linesToWrite), 0644)
		if err != nil {
			log.Fatal(err)
		}
	}
	logger.Infof("Hostname: %s", hostname)
	ipAddress := "172.17.0.1"
	// get docker bridge's gateway address (linux only)
	out, stderr, err := util.RunLocally(util.Options{}, "docker", "network", "inspect", "bridge")
	//logger.Infof("%s\n", out)
	logger.Infof("STDERR: %s\n", stderr)
	if err != nil {
		logger.Infof("ERROR: %s\n", err)
		logger.Infof("using default IP: %s\n", ipAddress)
	} else {
		var result []interface{}
		json.Unmarshal([]byte(out), &result)
		cfg := result[0].(map[string]interface{})
		IPAM := cfg["IPAM"].(map[string]interface{})
		ConfigArr := IPAM["Config"].([]interface{})
		Config := ConfigArr[0].(map[string]interface{})
		Gateway := Config["Gateway"].(string)
		ipAddress = Gateway
		logger.Infof("using IP from docker bridge: (%s)\n", ipAddress)
	}

	domainsToAddresses[hostname+".ona.im."] = ipAddress
	domainsToAddresses["."+hostname+".ona.im."] = ipAddress

	return nil
}

func ResetHostServices(logger service.Logger) error {
	// needs sudo - TODO: should check
	out, stderr, err := util.RunLocally(util.Options{}, "systemctl", "restart", "systemd-resolved")
	logger.Infof("%s\n", out)
	logger.Infof("%s\n", stderr)
	if err != nil {
		logger.Infof("ERROR: %s\n", err)

		return err
	}

	// resolvectl flush-caches
	out, stderr, err = util.RunLocally(util.Options{}, "resolvectl", "flush-caches")
	logger.Infof("%s\n", out)
	logger.Infof("%s\n", stderr)
	if err != nil {
		logger.Infof("ERROR: %s\n", err)

		return err
	}

	// TODO: check if its in /etc/hosts...

	return nil
}
