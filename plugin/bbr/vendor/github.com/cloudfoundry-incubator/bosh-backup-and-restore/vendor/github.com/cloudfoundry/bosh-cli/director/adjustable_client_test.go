package director_test

import (
	"errors"
	"net/http"
	gourl "net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	"io"
	"io/ioutil"
	"strings"
)

type FakeIOReader struct{}

func (reader FakeIOReader) Read(p []byte) (n int, err error) {
	return 0, nil
}

var _ = Describe("AdjustableClient", func() {
	var (
		innerClient *fakedir.FakeAdjustedClient
		adjustment  *fakedir.FakeAdjustment
		client      AdjustableClient
	)

	BeforeEach(func() {
		innerClient = &fakedir.FakeAdjustedClient{}
		adjustment = &fakedir.FakeAdjustment{}
		client = NewAdjustableClient(innerClient, adjustment)
	})

	Describe("Do", func() {
		var (
			req *http.Request
		)

		BeforeEach(func() {
			req = &http.Request{
				URL:    &gourl.URL{},
				Header: http.Header(map[string][]string{}),
			}
		})

		Context("if body is not empty", func() {
			var nopCloser io.ReadCloser

			BeforeEach(func() {
				reader := FakeIOReader{}
				nopCloser = ioutil.NopCloser(reader)
				req.Body = nopCloser
			})

			It("adjusts with retried true", func() {
				adjustment.AdjustStub = func(reqToAdjust *http.Request, retried bool) error {
					Expect(retried).To(BeTrue())
					reqToAdjust.Header.Add("Adjustment", "1")
					return nil
				}

				innerClient.DoStub = func(reqToExec *http.Request) (*http.Response, error) {
					Expect(reqToExec.Header["Adjustment"]).To(Equal([]string{"1"}))
					Expect(reqToExec.Body).ToNot(BeNil())
					return nil, nil
				}
				client.Do(req)
			})

			Context("request body is type converted by innerclient when it needs adjusting", func() {
				It("Should reset request body to original before attempting request again", func() {
					adjustment.NeedsReadjustmentStub = func(respToCheck *http.Response) bool {
						return true
					}

					adjustment.AdjustStub = func(reqToAdjust *http.Request, retried bool) error {
						Expect(retried).To(BeTrue())
						return nil
					}

					innerClient.DoStub = func(reqToExec *http.Request) (*http.Response, error) {
						Expect(reqToExec.Body).To(BeIdenticalTo(nopCloser))

						newReader := strings.NewReader("changed_request_body")
						newNopCloser := ioutil.NopCloser(newReader)

						reqToExec.Body = newNopCloser
						return nil, nil
					}
					client.Do(req)
				})
			})
		})

		It("adjusts request once before executing it", func() {
			adjustment.AdjustStub = func(reqToAdjust *http.Request, retried bool) error {
				Expect(retried).To(BeFalse())
				reqToAdjust.Header.Add("Adjustment", "1")
				return nil
			}

			innerClient.DoStub = func(reqToExec *http.Request) (*http.Response, error) {
				Expect(reqToExec.Header["Adjustment"]).To(Equal([]string{"1"}))
				return nil, nil
			}

			_, err := client.Do(req)
			Expect(err).ToNot(HaveOccurred())

			Expect(adjustment.AdjustCallCount()).To(Equal(1))
			Expect(innerClient.DoCallCount()).To(Equal(1))
		})

		It("returns first adjustment error without making any requests if adjustment fails", func() {
			adjustment.AdjustReturns(errors.New("fake-err"))

			resp, err := client.Do(req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
			Expect(resp).To(BeNil())

			Expect(adjustment.AdjustCallCount()).To(Equal(1))
			Expect(innerClient.DoCallCount()).To(Equal(0))
		})

		It("returns request error without readjustment if request fails", func() {
			innerClient.DoStub = func(_ *http.Request) (*http.Response, error) {
				return nil, errors.New("fake-err")
			}

			_, err := client.Do(req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(adjustment.AdjustCallCount()).To(Equal(1))
			Expect(adjustment.NeedsReadjustmentCallCount()).To(Equal(0))
			Expect(innerClient.DoCallCount()).To(Equal(1))
		})

		It("adjusts and readjusts request when readjustment is necessary", func() {
			firstResp := &http.Response{}
			secondResp := &http.Response{}

			adjustment.AdjustStub = func(reqToAdjust *http.Request, retried bool) error {
				if adjustment.AdjustCallCount() == 1 {
					Expect(retried).To(BeFalse())
					reqToAdjust.Header.Add("Adjustment", "1")
					return nil
				}
				if adjustment.AdjustCallCount() == 2 {
					Expect(retried).To(BeTrue())
					reqToAdjust.Header.Add("Adjustment", "2")
					return nil
				}
				panic("Not expected")
			}

			adjustment.NeedsReadjustmentStub = func(respToCheck *http.Response) bool {
				Expect(respToCheck).To(Equal(firstResp))
				return true
			}

			innerClient.DoStub = func(reqToExec *http.Request) (*http.Response, error) {
				if innerClient.DoCallCount() == 1 {
					Expect(reqToExec.Header["Adjustment"]).To(Equal([]string{"1"}))
					return firstResp, nil
				}
				if innerClient.DoCallCount() == 2 {
					Expect(reqToExec.Header["Adjustment"]).To(Equal([]string{"1", "2"}))
					return secondResp, nil
				}
				panic("Not expected")
			}

			resp, err := client.Do(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(Equal(secondResp))

			Expect(adjustment.AdjustCallCount()).To(Equal(2))
			Expect(adjustment.NeedsReadjustmentCallCount()).To(Equal(1))
			Expect(innerClient.DoCallCount()).To(Equal(2))
		})

		It("adjusts and does not readjust request when readjustment is not necessary", func() {
			firstResp := &http.Response{}

			adjustment.AdjustStub = func(reqToAdjust *http.Request, retried bool) error {
				if adjustment.AdjustCallCount() == 1 {
					Expect(retried).To(BeFalse())
					reqToAdjust.Header.Add("Adjustment", "1")
					return nil
				}
				panic("Not expected")
			}

			adjustment.NeedsReadjustmentStub = func(respToCheck *http.Response) bool {
				Expect(respToCheck).To(Equal(firstResp))
				return false
			}

			innerClient.DoStub = func(reqToExec *http.Request) (*http.Response, error) {
				if innerClient.DoCallCount() == 1 {
					Expect(reqToExec.Header["Adjustment"]).To(Equal([]string{"1"}))
					return firstResp, nil
				}
				panic("Not expected")
			}

			resp, err := client.Do(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(Equal(firstResp))

			// Request is only executed once since there is no need for readjustment
			Expect(adjustment.AdjustCallCount()).To(Equal(1))
			Expect(adjustment.NeedsReadjustmentCallCount()).To(Equal(1))
			Expect(innerClient.DoCallCount()).To(Equal(1))
		})

		It("returns readjustment error if readjustment fails", func() {
			adjustment.AdjustStub = func(reqToAdjust *http.Request, retried bool) error {
				if adjustment.AdjustCallCount() == 2 {
					return errors.New("fake-err")
				}
				return nil
			}

			adjustment.NeedsReadjustmentReturns(true)

			_, err := client.Do(req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			// Request is only executed once since second readjustment fails
			Expect(innerClient.DoCallCount()).To(Equal(1))
		})

		It("returns request error after readjustment if second request fails", func() {
			innerClient.DoStub = func(req *http.Request) (*http.Response, error) {
				if adjustment.AdjustCallCount() == 1 {
					return nil, nil
				}
				if adjustment.AdjustCallCount() == 2 {
					return nil, errors.New("fake-err")
				}
				panic("Not expected")
			}

			adjustment.NeedsReadjustmentReturns(true)

			_, err := client.Do(req)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(adjustment.AdjustCallCount()).To(Equal(2))
			Expect(adjustment.NeedsReadjustmentCallCount()).To(Equal(1))
			Expect(innerClient.DoCallCount()).To(Equal(2))
		})
	})
})
