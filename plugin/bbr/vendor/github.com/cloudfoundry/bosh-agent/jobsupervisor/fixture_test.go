package jobsupervisor_test

import (
	"io/ioutil"
	"path/filepath"

	. "github.com/onsi/gomega"
)

func readFixture(relativePath string) []byte {
	filePath, err := filepath.Abs(relativePath)
	Expect(err).ToNot(HaveOccurred())

	content, err := ioutil.ReadFile(filePath)
	Expect(err).ToNot(HaveOccurred())

	return content
}
