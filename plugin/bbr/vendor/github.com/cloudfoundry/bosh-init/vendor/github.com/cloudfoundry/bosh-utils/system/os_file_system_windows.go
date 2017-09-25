package system

import (
	"os"
	"path/filepath"
)

func symlink(oldPath, newPath string) error {
	oldAbs, err := filepath.Abs(oldPath)
	if err != nil {
		return err
	}
	newAbs, err := filepath.Abs(newPath)
	if err != nil {
		return err
	}
	return os.Symlink(oldAbs, newAbs)
}
