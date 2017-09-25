package uaa_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/uaa"
	fakeuaa "github.com/cloudfoundry/bosh-cli/uaa/uaafakes"
)

var _ = Describe("AccessTokenSession", func() {
	var (
		initToken *fakeuaa.FakeAccessToken
		sess      *AccessTokenSession
	)

	BeforeEach(func() {
		initToken = &fakeuaa.FakeAccessToken{}
		sess = NewAccessTokenSession(initToken)
	})

	Describe("TokenFunc", func() {
		Context("on first call", func() {
			Context("when retrying is set", func() {
				It("returns an auth header with a new token", func() {
					firstToken := &fakeuaa.FakeAccessToken{
						TypeStub:  func() string { return "type1" },
						ValueStub: func() string { return "value1" },
					}
					initToken.RefreshReturns(firstToken, nil)

					header, err := sess.TokenFunc(true)
					Expect(err).ToNot(HaveOccurred())
					Expect(header).To(Equal("type1 value1"))
				})

				It("returns an error if obtaining first token fails", func() {
					firstToken := &fakeuaa.FakeAccessToken{}
					initToken.RefreshReturns(firstToken, errors.New("fake-err"))

					_, err := sess.TokenFunc(true)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when retrying is not set", func() {
				It("returns an auth header with a new token", func() {
					firstToken := &fakeuaa.FakeAccessToken{
						TypeStub:  func() string { return "type1" },
						ValueStub: func() string { return "value1" },
					}
					initToken.RefreshReturns(firstToken, nil)

					header, err := sess.TokenFunc(false)
					Expect(err).ToNot(HaveOccurred())
					Expect(header).To(Equal("type1 value1"))
				})

				It("returns an error if obtaining first token fails", func() {
					firstToken := &fakeuaa.FakeAccessToken{}
					initToken.RefreshReturns(firstToken, errors.New("fake-err"))

					_, err := sess.TokenFunc(false)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("on second call", func() {
			var (
				firstToken *fakeuaa.FakeAccessToken
			)

			BeforeEach(func() {
				firstToken = &fakeuaa.FakeAccessToken{
					TypeStub:  func() string { return "type1" },
					ValueStub: func() string { return "value1" },
				}
				initToken.RefreshReturns(firstToken, nil)

				_, err := sess.TokenFunc(false)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when retrying is not set", func() {
				It("returns an auth header of a first token", func() {
					header, err := sess.TokenFunc(false)
					Expect(err).ToNot(HaveOccurred())
					Expect(header).To(Equal("type1 value1"))
				})

				It("does not try to refresh any token", func() {
					Expect(initToken.RefreshCallCount()).To(Equal(1))
					Expect(firstToken.RefreshCallCount()).To(Equal(0))

					_, err := sess.TokenFunc(false)
					Expect(err).ToNot(HaveOccurred())
					Expect(initToken.RefreshCallCount()).To(Equal(1))
					Expect(firstToken.RefreshCallCount()).To(Equal(0))
				})
			})

			Context("when retrying is set", func() {
				It("returns an auth header with a new token", func() {
					secondToken := &fakeuaa.FakeAccessToken{
						TypeStub:  func() string { return "type2" },
						ValueStub: func() string { return "value2" },
					}
					firstToken.RefreshReturns(secondToken, nil)

					header, err := sess.TokenFunc(true)
					Expect(err).ToNot(HaveOccurred())
					Expect(header).To(Equal("type2 value2"))
				})

				It("returns an error if obtaining first token fails", func() {
					secondToken := &fakeuaa.FakeAccessToken{}
					firstToken.RefreshReturns(secondToken, errors.New("fake-err"))

					_, err := sess.TokenFunc(true)
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
