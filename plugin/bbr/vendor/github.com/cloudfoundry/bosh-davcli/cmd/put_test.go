package cmd_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-davcli/cmd"
	testcmd "github.com/cloudfoundry/bosh-davcli/cmd/testing"
	davconf "github.com/cloudfoundry/bosh-davcli/config"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func runPut(config davconf.Config, args []string) error {
	logger := boshlog.NewLogger(boshlog.LevelNone)
	factory := NewFactory(logger)
	factory.SetConfig(config)

	cmd, err := factory.Create("put")
	Expect(err).ToNot(HaveOccurred())

	return cmd.Run(args)
}

func fileBytes(path string) []byte {
	file, err := os.Open(path)
	Expect(err).ToNot(HaveOccurred())

	content, err := ioutil.ReadAll(file)
	Expect(err).ToNot(HaveOccurred())

	return content
}

var _ = Describe("PutCmd", func() {
	Describe("Run", func() {
		It("uploads the blob with valid args", func() {
			pwd, err := os.Getwd()
			Expect(err).ToNot(HaveOccurred())

			sourceFilePath := filepath.Join(pwd, "../test_assets/cat.jpg")
			targetBlob := "some-other-awesome-guid"
			serverWasHit := false

			handler := func(w http.ResponseWriter, r *http.Request) {
				defer GinkgoRecover()
				serverWasHit = true
				req := testcmd.NewHTTPRequest(r)

				username, password, err := req.ExtractBasicAuth()
				Expect(err).ToNot(HaveOccurred())
				Expect(req.URL.Path).To(Equal("/d1/" + targetBlob))
				Expect(req.Method).To(Equal("PUT"))
				Expect(req.ContentLength).To(Equal(int64(1718186)))
				Expect(username).To(Equal("some user"))
				Expect(password).To(Equal("some pwd"))

				expectedBytes := fileBytes(sourceFilePath)
				actualBytes, _ := ioutil.ReadAll(r.Body)
				Expect(expectedBytes).To(Equal(actualBytes))

				w.WriteHeader(201)
			}

			ts := httptest.NewServer(http.HandlerFunc(handler))
			defer ts.Close()

			config := davconf.Config{
				User:     "some user",
				Password: "some pwd",
				Endpoint: ts.URL,
			}

			err = runPut(config, []string{sourceFilePath, targetBlob})
			Expect(err).ToNot(HaveOccurred())
			Expect(serverWasHit).To(BeTrue())
		})

		It("returns err with incorrect arg count", func() {
			err := runPut(davconf.Config{}, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Incorrect usage"))
		})
	})
})
