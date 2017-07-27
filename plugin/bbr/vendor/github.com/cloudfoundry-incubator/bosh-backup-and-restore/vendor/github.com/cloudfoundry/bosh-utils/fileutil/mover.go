package fileutil

import (
	"os"
	"syscall"

	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type fileMover struct {
	fs boshsys.FileSystem
}

func NewFileMover(fs boshsys.FileSystem) fileMover {
	return fileMover{fs: fs}
}

func (m fileMover) Move(oldPath, newPath string) error {
	err := m.fs.Rename(oldPath, newPath)

	le, ok := err.(*os.LinkError)
	if !ok {
		return err
	}

	switch le.Err {
	case syscall.Errno(0x12):
		err = m.fs.CopyFile(oldPath, newPath)
		if err != nil {
			return err
		}

		err = m.fs.RemoveAll(oldPath)
		if err != nil {
			return err
		}

		return nil
	default:
		return err
	}
}
