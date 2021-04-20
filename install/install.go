package install

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/onaci/cirrid/util"

	version "github.com/hashicorp/go-version"
)

var InstallDir = "/usr/local/bin"
var Commit = "DEVELOPMENT"
var BuildTime = "DEVELOPMENT"
var Version = "v0." + BuildTime + "+" + Commit
var cmdDryRun = false

func InstallBin() error {
	stat, err := os.Stat(InstallDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		err = os.MkdirAll(InstallDir, 0755)
		if err != nil {
			return err
		}
		stat, err = os.Lstat(InstallDir)
		if err != nil {
			return err
		}
	}
	//ensure we can write to installDir, or suggest sudo
	if !stat.IsDir() {
		return fmt.Errorf("InstallDir %s is not a directory", InstallDir)
	}
	// Check if the user bit is enabled in file permission
	if stat.Mode().Perm()&(1<<(uint(7))) == 0 {
		return fmt.Errorf("Please use 'sudo', you don't have permission to add files to InstallDir %s", InstallDir)
	}
	var sstat syscall.Stat_t
	if err = syscall.Stat(InstallDir, &sstat); err != nil {
		return err
	}

	err = nil
	if uint32(os.Geteuid()) != sstat.Uid {
		return fmt.Errorf("Please use 'sudo', you don't have permission to add files to InstallDir %s", InstallDir)
	}

	cirriRunPath, err := os.Executable()
	if err != nil {
		return err
	}
	versionForFileName := Version
	if strings.Contains(versionForFileName, "-dirty") {
		versionForFileName = "DEVELOPMENT"
	}
	cirriDestinationPath := filepath.Join(InstallDir, fmt.Sprintf("%s-%s", "cirrid", versionForFileName))
	if err := updateBinary(cirriRunPath, cirriDestinationPath, cmdDryRun); err != nil {
		return err
	}
	aliasPath := filepath.Join(InstallDir, "cirrid")
	if err := ensureSoftLink(cirriDestinationPath, aliasPath, cmdDryRun); err != nil {
		return err
	}
	return nil
}

func updateBinary(newBinary, destinationPath string, dryRun bool) error {
	if newBinary == destinationPath {
		log.Printf("Skipping %s, its the binary we're running\n", newBinary)
		return nil
	}

	InstallNeeded := ""
	if BuildTime == "DEVELOPMENT" || strings.Contains(Version, "-dirty") {
		InstallNeeded = BuildTime
	} else {
		// is there a possible old binary installed
		_, err := os.Stat(destinationPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			InstallNeeded = fmt.Sprintf("Install triggered: %s does not exist", destinationPath)
		} else {
			// get version of existing installed bin
			out, stderr, err := util.RunLocally(util.Options{}, destinationPath, "version", "--show-only=cirri", "--format={{.Version}}")
			if err != nil {
				log.Printf("%s\n", out)
				log.Printf("%s\n", stderr)
				log.Printf("%s\n", err)

				return err
			}
			installedVersion, err := version.NewVersion(strings.TrimPrefix(strings.TrimSpace(out), "v"))
			if err != nil {
				return err
			}
			binaryVersion, err := version.NewVersion(strings.TrimPrefix(Version, "v"))
			if err != nil {
				return err
			}

			if installedVersion.LessThan(binaryVersion) {
				InstallNeeded = fmt.Sprintf("Install triggered: %s is version %s, we have %s", destinationPath, installedVersion, Version)
			}
		}
	}
	// replace it if needed
	if InstallNeeded == "" {
		log.Printf("OK: %s is up to date with %s\n", destinationPath, newBinary)
		return nil
	}

	//copy the file!

	// TODO: determine if we have permission to write to the dir...
	// TODO: make sure the destination is in the path..

	if dryRun {
		log.Printf("DryRun - install %s to %s: %s\n", newBinary, destinationPath, InstallNeeded)
	} else {
		log.Printf("installing %s to %s: %s\n", newBinary, destinationPath, InstallNeeded)
		// do the copy using cmdline so we can use --sudo when needed...
		// TODO: --sudo...?
		out, stderr, err := util.RunLocally(util.Options{}, "rsync", newBinary, destinationPath)
		if err != nil {
			log.Printf("%s\n", out)
			log.Printf("%s\n", stderr)
			log.Printf("%s\n", err)

			return err
		}
	}

	return nil
}

func ensureSoftLink(sourcePath, destinationPath string, dryRun bool) error {
	if sourcePath == destinationPath {
		log.Printf("Skipping %s, its the binary we're running\n", sourcePath)
		return nil
	}

	InstallNeeded := ""
	rmNeeded := false
	// is there a possible old binary installed
	lstat, err := os.Lstat(destinationPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		InstallNeeded = fmt.Sprintf("%s does not exist", destinationPath)
	} else {
		// make sure we're a Link
		if lstat.Mode()&os.ModeSymlink == 0 {
			rmNeeded = true
			InstallNeeded = fmt.Sprintf("%s is not a softlink", destinationPath)
		} else {
			finalPath, err := os.Readlink(destinationPath)
			if err != nil {
				return err
			}
			if finalPath != sourcePath {
				rmNeeded = true
				InstallNeeded = fmt.Sprintf("%s points to %s, needs to link to %s", destinationPath, finalPath, sourcePath)
			}
		}
	}
	// replace it if needed
	if InstallNeeded == "" {
		log.Printf("OK: %s is a link to %s\n", destinationPath, sourcePath)
		return nil
	}

	//copy the file!

	// TODO: determine if we have permission to write to the dir...
	// TODO: make sure the destination is in the path..

	if dryRun {
		log.Printf("DryRun - link %s to %s: %s\n", destinationPath, sourcePath, InstallNeeded)
	} else {
		log.Printf("linking %s to %s: %s\n", destinationPath, sourcePath, InstallNeeded)
		// TODO: likely need to try to remove the link
		if rmNeeded {
			if err = os.Remove(destinationPath); err != nil {
				return err
			}
		}
		// do the copy using cmdline so we can use --sudo when needed...
		// TODO: --sudo...?
		if err = os.Symlink(sourcePath, destinationPath); err != nil {
			return nil
		}
	}

	return nil
}
