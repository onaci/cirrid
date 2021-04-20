package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/onaci/cirrid/dns"
	"github.com/onaci/cirrid/install"

	"github.com/kardianos/service"
)

var logger service.Logger

// Program structures.
//  Define Start and Stop methods.
type program struct {
	exit chan struct{}
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

	realPath, _ := os.Executable()
	realPath, _ = filepath.EvalSymlinks(realPath)

	logger.Infof("I'm running %v using exec: %s, which is actually file %s.", service.Platform(), os.Args[0], realPath)

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
		log.Printf("UPGRADE:\n")
	case "install":
		log.Printf("INSTALL:\n")
		// copy to /usr/local/bin/cirrid-VERSION
		// make softlink to /usr/local/bin/cirrid
		err := install.InstallBin()
		if err != nil {
			log.Fatal(err)
		}

		// TODO: see if the service is already there, and if its definition is up to date...
		err = service.Control(s, "install")
		if err != nil {
			log.Fatal(err)
		}
	default:
		err := service.Control(s, os.Args[1])
		if err != nil {
			log.Printf("Valid actions: %q\n", service.ControlAction)
			log.Fatal(err)
		}
	}
}
