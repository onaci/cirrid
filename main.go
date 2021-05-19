package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onaci/cirrid/dns"
	"github.com/onaci/cirrid/install"

	"github.com/kardianos/service"

	"gopkg.in/ini.v1"
)

var logger service.Logger

// Program structures.
//  Define Start and Stop methods.
type program struct {
	exit chan struct{}
}

const globalCfgFile string = "/etc/cirrid.ini"
const defaultCfg = `
zone = "ona.im"

# ask the cirri container what host&stackdomain settings its listening to
ask_cirri = true

# also set current hostname + zone = magic
use_hostname = true

[hosts]
# list of hostname to IP address
# *.hostname.zone will be set to the same as hostname.zone, unless you also specify ".hostname=IP"
# instead of IP address, you can use the name of the network interface to use, or the string 'magic', which will try to "just work"
example = magic

`

func ensureCfgFile() (*ini.File, error) {
	cfg, err := ini.LooseLoad([]byte(defaultCfg), globalCfgFile)

	logger.Infof("Ensuring there's a cfg file at %s", globalCfgFile)

	// this lets us update the file in place
	err = cfg.SaveTo(globalCfgFile)
	if err != nil {
		return nil, err
	}
	err = cfg.Reload()
	return cfg, err
}

func (p *program) Start(s service.Service) error {
	if service.Interactive() {
		logger.Info("Running in terminal.")
	} else {
		logger.Info("Running under service manager.")
	}
	p.exit = make(chan struct{})

	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}
func (p *program) run() error {
	cfg, err := ensureCfgFile()
	if err != nil {
		fmt.Printf("Fail to read /etc/cirrid.ini file: %v", err)
		os.Exit(1)
	}
	logger.Infof("Zone set to %s\n", cfg.Section("").Key("zone").String())

	realPath, _ := os.Executable()
	realPath, _ = filepath.EvalSymlinks(realPath)

	logger.Infof("I'm running %v using exec: %s, which is actually file %s.", service.Platform(), os.Args[0], realPath)
	dns.SetLogger(logger)

	//dns.SetDNSValues()
	if cfg.Section("").Key("ask_cirri").MustBool(true) {
		stackdomain := dns.GetCirriStackdomain()
		if stackdomain != "" {
			dns.SetDNSValue(
				stackdomain,
				cfg.Section("").Key("zone").String(),
				"magic",
			)
		}
	}

	if cfg.Section("").Key("use_hostname").MustBool(true) {
		dns.SetDNSValue(
			dns.GetHostname(),
			cfg.Section("").Key("zone").String(),
			"magic",
		)
	}

	for _, key := range cfg.Section("hosts").Keys() {
		dns.SetDNSValue(
			key.Name(),
			cfg.Section("").Key("zone").String(),
			key.MustString("magic"),
		)
	}

	dns.EnsureWildCards()

	dns.EnsureResolveConfigured(logger)
	time.Sleep(100 * time.Millisecond)
	go dns.DnsServer(logger)
	time.Sleep(100 * time.Millisecond)
	dns.ResetHostServices(logger)

	ticker := time.NewTicker(6 * time.Hour)
	for {
		select {
		case tm := <-ticker.C:
			logger.Infof("Still running at %v...", tm)
		case <-p.exit:
			ticker.Stop()
			return nil
		}
	}
}
func (p *program) Stop(s service.Service) error {
	// Any work in Stop should be quick, usually a few seconds at most.
	logger.Info("I'm Stopping!")
	close(p.exit)
	return nil
}

// Service setup.
//   Define service config.
//   Create the service.
//   Setup the logger.
//   Handle service controls (optional).
//   Run the service.
func main() {
	if len(os.Args) < 2 {
		// TODO: if os.Arg[1] not in
		// TODO: add upgrade and version
		fmt.Printf("Valid cmdline: %s %q\n", os.Args[0], service.ControlAction)
		//		fmt.Printf("Valid cmdline: %q\n", append(service.ControlAction, "run", "upgrade", "version"))
		return
	}

	options := make(service.KeyValue)
	options["Restart"] = "on-success"
	options["SuccessExitStatus"] = "1 2 8 SIGKILL"
	svcConfig := &service.Config{
		Name:        "cirrid",
		DisplayName: "Cirri host Daemon",
		Description: "Cirri Daemon: https://github.com/onaci/cirri.",
		Executable:  "/usr/local/bin/cirrid",
		Arguments:   []string{"run"},
		Dependencies: []string{
			"Requires=network.target",
			"After=network-online.target syslog.target"},
		Option: options,
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	errs := make(chan error, 5)
	logger, err = s.Logger(errs)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: detect if its installed or not, and tell the user if that's why it failed to stop/start/restart
	go func() {
		for {
			err := <-errs
			if err != nil {
				log.Print(err)
			}
		}
	}()

	switch os.Args[1] {
	case "version":
		fmt.Printf("%s\n", install.Version)
	case "run":
		err = s.Run()
		if err != nil {
			logger.Error(err)
		}
	case "upgrade":
		log.Printf("UPGRADE: not implemented yet\n")
		// TODO: check for sudo / root
	case "install":
		log.Printf("installing:\n")
		// TODO: check for sudo / root
		// copy to /usr/local/bin/cirrid-VERSION
		// make softlink to /usr/local/bin/cirrid
		err := install.InstallBin()
		if err != nil {
			log.Fatal(err)
		}
		_, err = ensureCfgFile()
		if err != nil {
			fmt.Printf("Fail to read /etc/cirrid.ini file: %v", err)
			os.Exit(1)
		}
		// TODO: see if the service is already there, and if its definition is up to date...
		err = service.Control(s, "install")
		if err != nil && !strings.Contains(err.Error(), "Init already exists") {
			log.Fatal(err)
		}
		log.Printf("Start service:\n")
		err = service.Control(s, "restart")
		if err != nil {
			log.Fatal(err)
		}
	default:
		// TODO: check for sudo / root
		err := service.Control(s, os.Args[1])
		if err != nil {
			log.Printf("Valid actions: %q\n", service.ControlAction)
			log.Fatal(err)
		}
	}
}
