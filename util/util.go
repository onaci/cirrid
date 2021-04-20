package util

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-cmd/cmd"
)

// TODO: rewrite using https://github.com/go-cmd/cmd so I can stream too
type Options struct {
	Follow bool
}

// RunOn run a command on a remote host using shelled out ssh
func RunOn(o Options, hostname string, args ...string) (output, errout string, err error) {
	// Use execing ssh to use the .ssh/config file
	// reuse an exiting connection!
	// time ssh -t -o ControlPath=~/.ssh/master-%r@%h:%p -o ControlMaster=auto -o ControlPersist=60 hostname ls  -alh
	newArgs := append([]string{"ssh",
		"-4",
		"-t",
		"-o", "ControlPath=~/.ssh/master-%r@%h:%p",
		"-o", "ControlMaster=auto",
		"-o", "ControlPersist=60",
		hostname})
	for _, arg := range args {
		// Add quotes to enable args with spaces
		// TODO: watch https://github.com/AkihiroSuda/sshocker/issues/10 for more
		newArgs = append(newArgs, "\""+arg+"\"")
	}
	return RunLocally(o, newArgs...)
}

// RunLocally run a command on a remote host using shelled out ssh
func RunLocally(o Options, args ...string) (output, errout string, err error) {
	// log.Printf("[VERBOSE] ENV: %s\n", strings.Join(os.Environ(), " "))
	log.Printf("[VERBOSE] Exec: %s\n", strings.Join(args, " "))

	doneChan := make(chan struct{}) // only used ror follow
	cmdOptions := cmd.Options{
		Buffered:  true,
		Streaming: false,
	}
	if o.Follow {
		cmdOptions.Buffered = false
		cmdOptions.Streaming = true
	}

	// Create Cmd with options
	envCmd := cmd.NewCmdOptions(cmdOptions, args[0], args[1:]...)
	if !o.Follow {
		status := <-envCmd.Start()

		return strings.Join(status.Stdout, "\n"), strings.Join(status.Stderr, "\n"), status.Error
	}

	// Print STDOUT and STDERR lines streaming from Cmd
	go func() {
		defer close(doneChan)
		// Done when both channels have been closed
		// https://dave.cheney.net/2013/04/30/curious-channels
		for envCmd.Stdout != nil || envCmd.Stderr != nil {
			select {
			case line, open := <-envCmd.Stdout:
				if !open {
					envCmd.Stdout = nil
					continue
				}
				log.Println(line)
			case line, open := <-envCmd.Stderr:
				if !open {
					envCmd.Stderr = nil
					continue
				}
				fmt.Fprintln(os.Stderr, line)
			}
		}
	}()
	<-envCmd.Start()
	// Wait for goroutine to print everything
	<-doneChan
	return "", "", nil
}

// TODO: rewrite using cmdline type pointer so we're not re-creating the []string constantly
func AddStringArg(flag, value string, cmdline []string) []string {
	if value != "" {
		return append(
			[]string{
				fmt.Sprintf("%s=%s", flag, value),
			}, cmdline...)
	}
	return cmdline
}
func AddIntArg(flag string, value int, cmdline []string) []string {
	if value > 0 {
		return append(
			[]string{
				fmt.Sprintf("%s=%d", flag, value),
			}, cmdline...)
	}
	return cmdline
}
func AddBoolArg(flag string, value bool, cmdline []string) []string {
	if value {
		return append(
			[]string{
				fmt.Sprintf("%s", flag),
			}, cmdline...)
	}
	return cmdline
}
