// +build !windows

package install

import (
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

func checkPermission(stat fs.FileInfo, dir string) error {
	if os.Geteuid() != 0 {
		// Check if the user bit is enabled in file permission
		if stat.Mode().Perm()&(1<<(uint(7))) == 0 {
			return fmt.Errorf("Please use 'sudo', you don't have permission to add files to dir %s 1", dir)
		}
		var sstat syscall.Stat_t
		if err := syscall.Stat(dir, &sstat); err != nil {
			return err
		}

		if uint32(os.Geteuid()) != sstat.Uid {
			return fmt.Errorf("Please use 'sudo', you don't have permission to add files to dir %s 2", dir)
		}
	}
	return nil
}
