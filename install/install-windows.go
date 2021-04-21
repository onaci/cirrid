// +build windows

package install

import "io/fs"

// TODO: figure out how to help the user install
func checkPermission(stat fs.FileInfo, dir string) error {

	return nil
}
