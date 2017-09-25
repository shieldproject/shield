package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakecmd "github.com/cloudfoundry/bosh-cli/cmd/cmdfakes"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
)

var _ = Describe("RunErrandCmd", func() {
	var (
		deployment *fakedir.FakeDeployment
		downloader *fakecmd.FakeDownloader
		ui         *fakeui.FakeUI
		command    RunErrandCmd
	)

	BeforeEach(func() {
		deployment = &fakedir.FakeDeployment{}
		downloader = &fakecmd.FakeDownloader{}
		ui = &fakeui.FakeUI{}
		command = NewRunErrandCmd(deployment, downloader, ui)
	})

	Describe("Run", func() {
		var (
			opts RunErrandOpts
		)

		BeforeEach(func() {
			opts = RunErrandOpts{
				Args:        RunErrandArgs{Name: "errand-name"},
				KeepAlive:   true,
				WhenChanged: true,
			}
		})

		act := func() error { return command.Run(opts) }

		Context("when errand succeeds", func() {
			Context("when multiple errands return", func() {
				It("downloads logs if requested", func() {
					opts.DownloadLogs = true
					opts.LogsDirectory = DirOrCWDArg{Path: "/fake-dir"}

					result := []boshdir.ErrandResult{{
						ExitCode:        0,
						LogsBlobstoreID: "logs-blob-id",
						LogsSHA1:        "logs-sha1",
					}, {
						ExitCode:        0,
						LogsBlobstoreID: "logs-blob-id2",
						LogsSHA1:        "logs-sha2",
					}}

					deployment.RunErrandReturns(result, nil)

					err := act()
					Expect(err).ToNot(HaveOccurred())

					Expect(downloader.DownloadCallCount()).To(Equal(2))

					blobID, sha1, prefix, dstDirPath := downloader.DownloadArgsForCall(0)
					Expect(blobID).To(Equal("logs-blob-id"))
					Expect(sha1).To(Equal("logs-sha1"))
					Expect(prefix).To(Equal("errand-name"))
					Expect(dstDirPath).To(Equal("/fake-dir"))

					blobID, sha1, prefix, dstDirPath = downloader.DownloadArgsForCall(1)
					Expect(blobID).To(Equal("logs-blob-id2"))
					Expect(sha1).To(Equal("logs-sha2"))
					Expect(prefix).To(Equal("errand-name"))
					Expect(dstDirPath).To(Equal("/fake-dir"))
				})

				It("does not download logs if not requested", func() {
					opts.DownloadLogs = false

					err := act()
					Expect(err).ToNot(HaveOccurred())

					Expect(downloader.DownloadCallCount()).To(Equal(0))
				})

				It("does not download logs if requested and not logs blob returned", func() {
					opts.DownloadLogs = true
					opts.LogsDirectory = DirOrCWDArg{Path: "/fake-dir"}

					result := []boshdir.ErrandResult{{ExitCode: 0}}

					deployment.RunErrandReturns(result, nil)

					err := act()
					Expect(err).ToNot(HaveOccurred())

					Expect(downloader.DownloadCallCount()).To(Equal(0))
				})

				It("runs errand and outputs both stdout and stderr", func() {
					result := []boshdir.ErrandResult{{
						ExitCode: 0,
						Stdout:   "stdout-content",
						Stderr:   "",
					}, {
						ExitCode: 129,
						Stdout:   "",
						Stderr:   "stderr-content",
					}}

					deployment.RunErrandReturns(result, nil)

					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Errand 'errand-name' was canceled (exit code 129)"))

					Expect(ui.Table).To(Equal(
						boshtbl.Table{
							Content: "errand(s)",

							Header: []boshtbl.Header{
								boshtbl.NewHeader("Exit Code"),
								boshtbl.NewHeader("Stdout"),
								boshtbl.NewHeader("Stderr"),
							},

							SortBy: []boshtbl.ColumnSort{
								{Column: 0, Asc: true},
							},

							Rows: [][]boshtbl.Value{
								{
									boshtbl.NewValueInt(0),
									boshtbl.NewValueString("stdout-content"),
									boshtbl.NewValueString(""),
								}, {
									boshtbl.NewValueInt(129),
									boshtbl.NewValueString(""),
									boshtbl.NewValueString("stderr-content"),
								},
							},

							Notes: []string{},

							FillFirstColumn: true,

							Transpose: true,
						}))
				})

			})

			It("runs errand with given name", func() {
				deployment.RunErrandReturns([]boshdir.ErrandResult{{ExitCode: 0}}, nil)

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(deployment.RunErrandCallCount()).To(Equal(1))

				name, keepAlive, whenChanged := deployment.RunErrandArgsForCall(0)
				Expect(name).To(Equal("errand-name"))
				Expect(keepAlive).To(BeTrue())
				Expect(whenChanged).To(BeTrue())
			})

			It("downloads logs if requested", func() {
				opts.DownloadLogs = true
				opts.LogsDirectory = DirOrCWDArg{Path: "/fake-dir"}

				result := []boshdir.ErrandResult{{
					ExitCode:        0,
					LogsBlobstoreID: "logs-blob-id",
					LogsSHA1:        "logs-sha1",
				}}

				deployment.RunErrandReturns(result, nil)

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(downloader.DownloadCallCount()).To(Equal(1))

				blobID, sha1, prefix, dstDirPath := downloader.DownloadArgsForCall(0)
				Expect(blobID).To(Equal("logs-blob-id"))
				Expect(sha1).To(Equal("logs-sha1"))
				Expect(prefix).To(Equal("errand-name"))
				Expect(dstDirPath).To(Equal("/fake-dir"))
			})

			It("does not download logs if not requested", func() {
				opts.DownloadLogs = false

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(downloader.DownloadCallCount()).To(Equal(0))
			})

			It("does not download logs if requested and not logs blob returned", func() {
				opts.DownloadLogs = true
				opts.LogsDirectory = DirOrCWDArg{Path: "/fake-dir"}

				result := []boshdir.ErrandResult{{ExitCode: 0}}

				deployment.RunErrandReturns(result, nil)

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(downloader.DownloadCallCount()).To(Equal(0))
			})
		})

		Context("when errand fails (exit code is non-0)", func() {
			It("returns error", func() {
				deployment.RunErrandReturns([]boshdir.ErrandResult{{ExitCode: 1}}, nil)

				err := act()
				Expect(err).To(HaveOccurred())

				Expect(ui.Table).To(Equal(
					boshtbl.Table{
						Content: "errand(s)",

						Header: []boshtbl.Header{
							boshtbl.NewHeader("Exit Code"),
							boshtbl.NewHeader("Stdout"),
							boshtbl.NewHeader("Stderr"),
						},

						SortBy: []boshtbl.ColumnSort{
							{Column: 0, Asc: true},
						},

						Rows: [][]boshtbl.Value{
							{
								boshtbl.NewValueInt(1),
								boshtbl.NewValueString(""),
								boshtbl.NewValueString(""),
							},
						},

						Notes: []string{},

						FillFirstColumn: true,

						Transpose: true,
					}))

				Expect(err.Error()).To(Equal("Errand 'errand-name' completed with error (exit code 1)"))
			})
		})

		Context("when errand is canceled (exit code > 128)", func() {
			It("returns error", func() {
				deployment.RunErrandReturns([]boshdir.ErrandResult{{ExitCode: 129}}, nil)

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Errand 'errand-name' was canceled (exit code 129)"))
			})
		})

		It("returns error if running errand failed", func() {
			deployment.RunErrandReturns([]boshdir.ErrandResult{{}}, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
