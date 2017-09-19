package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("DeleteReleaseCmd", func() {
	var (
		ui       *fakeui.FakeUI
		director *fakedir.FakeDirector
		command  DeleteReleaseCmd
	)

	BeforeEach(func() {
		ui = &fakeui.FakeUI{}
		director = &fakedir.FakeDirector{}
		command = NewDeleteReleaseCmd(ui, director)
	})

	Describe("Run", func() {
		var (
			opts DeleteReleaseOpts
		)

		BeforeEach(func() {
			opts = DeleteReleaseOpts{}
		})

		act := func() error { return command.Run(opts) }

		Context("when release series is requested for deletion", func() {
			var (
				releaseSeries *fakedir.FakeReleaseSeries
			)

			BeforeEach(func() {
				opts.Args.Slug = boshdir.NewReleaseOrSeriesSlug("some-name", "")

				releaseSeries = &fakedir.FakeReleaseSeries{}
				director.FindReleaseSeriesReturns(releaseSeries, nil)
			})

			It("deletes release series", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.FindReleaseSeriesCallCount()).To(Equal(1))
				Expect(director.FindReleaseSeriesArgsForCall(0)).To(Equal(
					boshdir.NewReleaseSeriesSlug("some-name")))

				Expect(releaseSeries.DeleteCallCount()).To(Equal(1))
				Expect(releaseSeries.DeleteArgsForCall(0)).To(BeFalse())
			})

			It("deletes release series forcefully if requested", func() {
				opts.Force = true

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(releaseSeries.DeleteCallCount()).To(Equal(1))
				Expect(releaseSeries.DeleteArgsForCall(0)).To(BeTrue())
			})

			It("does not delete release series if confirmation is rejected", func() {
				ui.AskedConfirmationErr = errors.New("stop")

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("stop"))

				Expect(releaseSeries.DeleteCallCount()).To(Equal(0))
			})

			It("returns error if deleting release series failed", func() {
				releaseSeries.DeleteReturns(errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if finding release series failed", func() {
				director.FindReleaseSeriesReturns(nil, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))

				Expect(releaseSeries.DeleteCallCount()).To(Equal(0))
			})
		})

		Context("when release (not series) is requested for deletion", func() {
			var (
				release *fakedir.FakeRelease
			)

			BeforeEach(func() {
				opts.Args.Slug = boshdir.NewReleaseOrSeriesSlug("some-name", "some-version")

				release = &fakedir.FakeRelease{}
				director.FindReleaseReturns(release, nil)
			})

			It("deletes release", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(director.FindReleaseCallCount()).To(Equal(1))
				Expect(director.FindReleaseArgsForCall(0)).To(Equal(
					boshdir.NewReleaseSlug("some-name", "some-version")))

				Expect(release.DeleteCallCount()).To(Equal(1))
				Expect(release.DeleteArgsForCall(0)).To(BeFalse())
			})

			It("deletes release forcefully if requested", func() {
				opts.Force = true

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(release.DeleteCallCount()).To(Equal(1))
				Expect(release.DeleteArgsForCall(0)).To(BeTrue())
			})

			It("does not delete release if confirmation is rejected", func() {
				ui.AskedConfirmationErr = errors.New("stop")

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("stop"))

				Expect(release.DeleteCallCount()).To(Equal(0))
			})

			It("returns error if deleting release failed", func() {
				release.DeleteReturns(errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if finding release failed", func() {
				director.FindReleaseReturns(nil, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))

				Expect(release.DeleteCallCount()).To(Equal(0))
			})
		})
	})
})
