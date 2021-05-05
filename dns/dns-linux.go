// +build linux

package dns

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/kardianos/service"
	"github.com/onaci/cirrid/util"
)

func EnsureResolveConfigured(logger service.Logger) error {
	logger.Infof("EnsureResolveConfigured")
	// method - interestingly, this doesn't cache it, so this might be excellent for dynamic label based answers
	// start this dns server
	// add lines to /etc/systemd/resolved.conf as per https://github.com/hashicorp/consul/issues/4155#issuecomment-394362651
	// [Resolve]
	// DNS=127.0.0.98
	// Domains=~host.ona.im
	// systemctl restart systemd-resolved

	// TODO: ask cirri container what its STACKDOMAIN is... use that instead
	// docker inspect cirri | jq .[].Config.Env

	resolvedConf := "/etc/systemd/resolved.conf"
	file, err := os.Open(resolvedConf)
	if err != nil {
		logger.Errorf("failed to open")
		return err
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var text []string

	// yeah, deduping, and using that to get a list of ~hosts
	domainMap := make(map[string]bool)
	domains := []string{}
	for host, _ := range domainsToAddresses {
		logger.Infof("host: %s", host)
		host = strings.TrimPrefix(host, "*.")
		host = strings.TrimSuffix(host, ".")
		if _, ok := domainMap[host]; !ok {
			domains = append(domains, "~"+host)
		}
		domainMap[host] = true
	}

	// Big nasty assumption that resolved.conf only contains a [Resolve] section
	dnsline := "DNS=127.0.0.98"
	//domainline := "Domains=~" + hostname + ".ona.im,~host.ona.im"
	domainline := "Domains=" + strings.Join(domains, " ")
	logger.Infof("domainline: %s", domainline)
	//	domainline := "Domains=" + strings.Join([]string{"~" + hostname + ".ona.im", "~" + "t" + ".ona.im"}, " ")
	resolvedConfChanged := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "DNS=") {
			if strings.Compare(line, dnsline) != 0 {
				resolvedConfChanged = true
				logger.Warningf("Replacing %s line (%s) with (%s)\n", resolvedConf, line, dnsline)
			}
			text = append(text, dnsline)
			// use empty string as indicator that we've written it already
			dnsline = ""
		} else if strings.HasPrefix(line, "Domains=") {
			if strings.Compare(line, domainline) != 0 {
				resolvedConfChanged = true
				logger.Warningf("Replacing /etc/systemd/resolve.conf line (%s) with (%s)\n", line, domainline)
			}
			text = append(text, domainline)
			// use empty string as indicator that we've written it already
			domainline = ""
		} else {
			text = append(text, line)
		}
	}

	if dnsline != "" {
		resolvedConfChanged = true
		logger.Warningf("Adding to /etc/systemd/resolve.conf : (%s)\n", dnsline)
		text = append(text, dnsline)
	}
	if domainline != "" {
		resolvedConfChanged = true
		logger.Warningf("Adding to /etc/systemd/resolve.conf : (%s)\n", domainline)
		text = append(text, domainline)
	}
	text = append(text, "")

	file.Close()

	if resolvedConfChanged {
		logger.Infof("updating: %s to use %s\n", resolvedConf, domainline)

		linesToWrite := strings.Join(text, "\n")
		err = ioutil.WriteFile(resolvedConf, []byte(linesToWrite), 0644)
		if err != nil {
			logger.Error(err)
		}
	}

	return nil
}

func getIpAddress() string {
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
	return ipAddress
}

func ResetHostServices(logger service.Logger) error {
	logger.Infof("ResetHostServices")

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

func getDNSServerIPAddress() string {
	return "127.0.0.98"
}
