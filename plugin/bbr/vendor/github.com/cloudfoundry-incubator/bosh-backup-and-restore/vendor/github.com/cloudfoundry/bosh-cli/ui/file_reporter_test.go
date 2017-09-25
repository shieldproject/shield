package ui_test

import (
	. "github.com/cloudfoundry/bosh-cli/ui"

	"github.com/cloudfoundry/bosh-cli/ui/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type CallTracker struct {
	Seeks  []interface{}
	Closes int
}

type FakeSeekableReader struct {
	callTracker *CallTracker
}
type FakeReaderCloser struct{}

func (FakeSeekableReader) Read(p []byte) (n int, err error) {
	panic("should not call")
}

func (FakeReaderCloser) Read(p []byte) (n int, err error) {
	panic("should not call")
}

func (FakeReaderCloser) Close() error {
	panic("should not call")
}

func (r FakeSeekableReader) Seek(offset int64, whence int) (int64, error) {
	r.callTracker.Seeks = append(r.callTracker.Seeks, []interface{}{offset, whence})

	return 0, nil
}

func (r FakeSeekableReader) Close() error {
	r.callTracker.Closes++

	return nil
}

var _ = Describe("ReadCloserProxy", func() {
	Describe("Seek", func() {
		Context("when reader is seekable", func() {
			It("delegates to internal seeker", func() {
				seekerReader := FakeSeekableReader{
					callTracker: &CallTracker{},
				}
				fileReporter := NewFileReporter(&fakes.FakeUI{})
				readCloserProxy := fileReporter.TrackUpload(0, seekerReader)

				readCloserProxy.Seek(12, 42)
				Expect(seekerReader.callTracker.Seeks).To(ContainElement([]interface{}{int64(12), 42}))
			})
		})

		Context("when reader is NOT seekable", func() {
			It("does not complain and returns 0, nil", func() {
				reader := FakeReaderCloser{}
				fileReporter := NewFileReporter(&fakes.FakeUI{})
				readCloserProxy := fileReporter.TrackUpload(0, reader)

				bytes, err := readCloserProxy.Seek(12, 42)
				Expect(err).ToNot(HaveOccurred())
				Expect(bytes).To(Equal(int64(0)))
			})
		})
	})

	Describe("Close", func() {
		It("closes the reader, uses the ui for bar output, and prints a newline", func() {
			fakeUI := &fakes.FakeUI{}
			seekerReader := FakeSeekableReader{
				callTracker: &CallTracker{},
			}
			fileReporter := NewFileReporter(fakeUI)
			readCloserProxy := fileReporter.TrackUpload(0, seekerReader)

			err := readCloserProxy.Close()
			Expect(err).ToNot(HaveOccurred())
			Expect(seekerReader.callTracker.Closes).To(Equal(1))
			uiSaid := fakeUI.Said

			Expect(uiSaid).To(HaveLen(3))
			Expect(uiSaid[0]).To(MatchRegexp("^\\r\\s+#$"))
			Expect(uiSaid[1]).To(MatchRegexp("^\\r\\s+# 0s$"))
			Expect(uiSaid[2]).To(MatchRegexp(`^\n$`))
		})
	})
})
