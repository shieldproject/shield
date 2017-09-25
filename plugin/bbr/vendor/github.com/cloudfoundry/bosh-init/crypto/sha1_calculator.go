package crypto

import (
	"crypto/sha1"
	"fmt"
	"hash"
	"io"
	"os"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type SHA1Calculator interface {
	Calculate(filePath string) (string, error)
}

type sha1Calculator struct {
	fs boshsys.FileSystem
}

func NewSha1Calculator(fs boshsys.FileSystem) SHA1Calculator {
	return sha1Calculator{
		fs: fs,
	}
}

func (c sha1Calculator) Calculate(filePath string) (string, error) {
	file, err := c.fs.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Calculating sha1 of '%s'", filePath)
	}
	defer func() {
		_ = file.Close()
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Opening file '%s' for sha1 calculation", filePath)
	}

	h := sha1.New()

	if fileInfo.IsDir() {
		err = c.fs.Walk(filePath+"/", func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				err := c.populateSha1(path, h)
				if err != nil {
					return bosherr.WrapErrorf(err, "Calculating directory SHA1 for %s", path)
				}
			}
			return nil
		})
		if err != nil {
			return "", err
		}
	} else {
		err = c.populateSha1(filePath, h)
		if err != nil {
			return "", bosherr.WrapErrorf(err, "Calculating file SHA1 for %s", filePath)
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (c sha1Calculator) populateSha1(filePath string, hash hash.Hash) error {
	file, err := c.fs.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		return bosherr.WrapErrorf(err, "Opening file '%s' for sha1 calculation", filePath)
	}
	defer func() {
		_ = file.Close()
	}()

	_, err = io.Copy(hash, file)
	if err != nil {
		return bosherr.WrapError(err, "Copying file for sha1 calculation")
	}

	return nil
}
