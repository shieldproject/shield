//+build !windows

package system

import "os"

func symlink(oldPath, newPath string) error {
	return os.Symlink(oldPath, newPath)
}
