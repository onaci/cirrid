// +build darwin

package dns

import (
	"bufio"
	//"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/kardianos/service"
	"github.com/onaci/cirrid/util"
)

func EnsureResolveConfigured(logger service.Logger) error {
	logger.Infof("EnsureResolveConfigured")
	for host, ip := range domainsToAddresses {
		logger.Infof("host: %s -> %s", host, ip)
		host = strings.TrimPrefix(host, "*.")
		host = strings.TrimSuffix(host, ".")
		host = strings.TrimPrefix(host, ".")
		createResolveFile(host)
	}
	return nil
}

func createResolveFile(host string) {
	resolvedConfChanged := false
	var text []string
	requiredLine := "nameserver " + getDNSServerIPAddress()

	// TODO: use the domainsToAddresses list
	resolvedConf := "/etc/resolver/" + host
	file, err := os.Open(resolvedConf)
	if err != nil {
		logger.Infof("no %s file: %s\n", resolvedConf, err)
		resolvedConfChanged = true
	} else {
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "nameserver ") {
				if strings.Compare(line, requiredLine) != 0 {
					resolvedConfChanged = true
					logger.Warningf("Replacing %s line (%s) with (%s)\n", resolvedConf, line, requiredLine)
				}
				text = append(text, requiredLine)
				// use empty string as indicator that we've written it already
				requiredLine = ""
			} else {
				text = append(text, line)
			}
		}
		file.Close()
	}
	if requiredLine != "" {
		resolvedConfChanged = true
		text = append(text, requiredLine)
	}
	text = append(text, "")

	if resolvedConfChanged {
		out, stderr, err := util.RunLocally(util.Options{}, "mkdir", "-p", "/etc/resolver")
		logger.Infof("%s\n", out)
		logger.Infof("STDERR: %s\n", stderr)
		if err != nil {
			logger.Infof("ERROR: %s\n", err)
		}

		logger.Infof("updating: %s to use %s\n", resolvedConf, requiredLine)

		linesToWrite := strings.Join(text, "\n")
		err = ioutil.WriteFile(resolvedConf, []byte(linesToWrite), 0644)
		if err != nil {
			logger.Error(err)
		}
	}
}

func ResetHostServices(logger service.Logger) error {
	logger.Infof("ResetHostServices")

	// sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder
	out, stderr, err := util.RunLocally(util.Options{}, "dscacheutil", "-flushcache")
	logger.Infof("%s\n", out)
	logger.Infof("STDERR: %s\n", stderr)
	if err != nil {
		logger.Infof("ERROR: %s\n", err)
	}
	out, stderr, err = util.RunLocally(util.Options{}, "killall", "-HUP", "mDNSResponder")
	logger.Infof("%s\n", out)
	logger.Infof("STDERR: %s\n", stderr)
	if err != nil {
		logger.Infof("ERROR: %s\n", err)
	}

	// sudo ifconfig lo0 alias 172.17.0.1
	out, stderr, err = util.RunLocally(util.Options{}, "ifconfig", "lo0", "alias", getIpAddress())
	logger.Infof("%s\n", out)
	logger.Infof("STDERR: %s\n", stderr)
	if err != nil {
		logger.Infof("ERROR: %s\n", err)
	}

	return nil
}

// the virtual IP address to talk to the local cirri container
func getIpAddress() string {
	ipAddress := "172.17.0.1"
	return ipAddress
}

func getDNSServerIPAddress() string {
	return "127.0.0.1"
}
