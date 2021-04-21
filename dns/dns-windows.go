// +build windows

package dns

import (
	"github.com/kardianos/service"
)

func EnsureResolveConfigured(logger service.Logger) error {
	logger.Infof("EnsureResolveConfigured")
	return nil
}

func ResetHostServices(logger service.Logger) error {
	logger.Infof("ResetHostServices")

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
