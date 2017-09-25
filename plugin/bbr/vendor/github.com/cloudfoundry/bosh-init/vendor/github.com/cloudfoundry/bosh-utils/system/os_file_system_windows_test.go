package system_test

import (
	"bytes"
	"math/rand"
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"io/ioutil"

	fsWrapper "github.com/charlievieth/fs"
)

func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func makeLongPath() string {
	volume := os.TempDir() + string(filepath.Separator)
	buf := bytes.NewBufferString(volume)
	for i := 0; i < 2; i++ {
		for i := byte('A'); i <= 'Z'; i++ {
			buf.Write(bytes.Repeat([]byte{i}, 4))
			buf.WriteRune(filepath.Separator)
		}
	}
	buf.Write([]byte(randSeq(10)))
	buf.WriteRune(filepath.Separator)
	return filepath.Clean(buf.String())
}

var _ = Describe("OS FileSystem LongPaths", func() {
	It("remove all long path", func() {
		osFs := createOsFs()

		longPath := makeLongPath()
		err := fsWrapper.MkdirAll(longPath, 0755)
		defer fsWrapper.RemoveAll(longPath)
		Expect(err).ToNot(HaveOccurred())

		dstFile, err := ioutil.TempFile(`\\?\`+longPath, "")
		Expect(err).ToNot(HaveOccurred())

		dstPath := path.Join(longPath, filepath.Base(dstFile.Name()))
		defer os.Remove(dstPath)
		dstFile.Close()

		fileInfo, err := osFs.Stat(dstPath)
		Expect(fileInfo).ToNot(BeNil())
		Expect(os.IsNotExist(err)).To(BeFalse())

		err = osFs.RemoveAll(dstPath)
		Expect(err).ToNot(HaveOccurred())

		_, err = osFs.Stat(dstPath)
		Expect(os.IsNotExist(err)).To(BeTrue())
	})
})
