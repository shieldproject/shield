package logger_test

import (
	"bytes"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/logger"
)

type intervalWriter struct {
	blockingWriter
	dur time.Duration
}

func (w *intervalWriter) Write(p []byte) (int, error) {
	w.Lock()
	time.Sleep(w.dur)
	n, err := w.buf.Write(p)
	w.Unlock()
	return n, err
}

type blockingWriter struct {
	buf bytes.Buffer
	sync.Mutex
}

func (w *blockingWriter) Write(p []byte) (int, error) {
	w.Lock()
	n, err := w.buf.Write(p)
	w.Unlock()
	return n, err
}

func (w *blockingWriter) Len() int {
	w.Lock()
	n := w.buf.Len()
	w.Unlock()
	return n
}

func (w *blockingWriter) String() string {
	w.Lock()
	s := w.buf.String()
	w.Unlock()
	return s
}

var _ = Describe("Logger", func() {
	var (
		outBuf = new(bytes.Buffer)
	)
	BeforeEach(func() {
		outBuf.Reset()
	})

	Describe("Async Logger", func() {
		It("logs the formatted message to Logger.err at the debug level", func() {
			logger := NewAsyncWriterLogger(LevelDebug, outBuf)
			logger.Debug("TAG", "some %s info to log", "awesome")
			logger.Flush()

			expectedContent := expectedLogFormat("TAG", "DEBUG - some awesome info to log")
			Expect(outBuf).To(MatchRegexp(expectedContent))
		})

		It("does not block when its writer is blocked", func() {
			out := new(blockingWriter)
			logger := NewAsyncWriterLogger(LevelDebug, out)

			out.Lock()
			ch := make(chan struct{}, 1)
			go func() {
				for i := 0; i < 10; i++ {
					logger.Info("TAG", "Make sure we are not just buffering bytes: %s", strings.Repeat("A", 4096))
					logger.Error("TAG", "Make sure we are not just buffering bytes: %s", strings.Repeat("A", 4096))
				}
				ch <- struct{}{}
			}()
			Eventually(ch).Should(Receive())
			Expect(out.buf.Len()).To(Equal(0))
		})

		It("copies queued log messages", func() {
			const s0 = "ABCDEFGHIJ"
			const s1 = "abcdefghij"

			outBuf := new(blockingWriter)
			logger := NewAsyncWriterLogger(LevelDebug, outBuf)

			outBuf.Lock()
			logger.Debug("TAG", s0)
			logger.Debug("TAG", s1)
			outBuf.Unlock()

			Expect(logger.Flush()).To(Succeed())

			lines := strings.Split(strings.TrimSpace(outBuf.buf.String()), "\n")
			Expect(lines).To(HaveLen(2))
			Expect(lines[0]).To(HaveSuffix(s0))
			Expect(lines[1]).To(HaveSuffix(s1))
		})

		It("continuously flushes queued log messages", func() {
			outBuf := new(blockingWriter)
			logger := NewAsyncWriterLogger(LevelDebug, outBuf)

			outBuf.Lock()
			for i := 0; i < 10; i++ {
				logger.Debug("TAG", "Queued log message")
			}
			Expect(outBuf.buf.Len()).To(Equal(0))
			outBuf.Unlock()
			Eventually(outBuf.Len).ShouldNot(Equal(0))
		})

		It("flushes with a timeout", func() {
			outBuf := new(blockingWriter)
			logger := NewAsyncWriterLogger(LevelDebug, outBuf)
			logger.Debug("TAG", "something")

			outBuf.Lock()
			Expect(logger.FlushTimeout(time.Millisecond * 10)).ToNot(Succeed())

			outBuf.Unlock()
			Expect(logger.FlushTimeout(time.Millisecond * 10)).To(Succeed())
			Expect(strings.TrimSpace(outBuf.buf.String())).To(HaveSuffix("something"))
		})

		It("flush doesn't block writes", func() {
			const (
				MessageCount  = 10
				WriteInterval = 10 * time.Millisecond
				FlushInterval = 10 * WriteInterval
			)

			out := &intervalWriter{dur: WriteInterval}
			logger := NewAsyncWriterLogger(LevelDebug, out)

			// add some messages to the queue
			out.Lock()
			for i := 0; i < MessageCount; i++ {
				logger.Debug("NEW", "message")
			}
			out.Unlock()

			go logger.Flush()

			ch := make(chan struct{}, 1)
			go func() {
				for i := 0; i < MessageCount; i++ {
					logger.Debug("NEW", "message")
				}
				ch <- struct{}{}
			}()
			Eventually(ch, time.Second).Should(Receive())
		})

		It("only flushes the current write queue", func() {
			const (
				MessageCount  = 10
				WriteInterval = 10 * time.Millisecond
				Timeout       = WriteInterval * MessageCount * 10
			)

			out := &intervalWriter{dur: WriteInterval}
			logger := NewAsyncWriterLogger(LevelDebug, out)

			// add some messages to the queue
			out.Lock()
			for i := 0; i < MessageCount; i++ {
				logger.Debug("QUEUED", "queued")
			}
			out.Unlock()

			// add messages faster than the queue can be drained
			tick := time.NewTicker(WriteInterval / 10)
			defer tick.Stop()
			go func() {
				for _ = range tick.C {
					logger.Debug("NEW", "new")
				}
			}()

			ch := make(chan struct{}, 1)
			go func() {
				logger.Flush()
				ch <- struct{}{}
			}()

			// we only care that flush returns
			Eventually(ch, Timeout).Should(Receive())
		})

		It("prints the correct prefix during concurrent writes", func() {
			ch := make(chan struct{}, 1)
			go func() {
				testConcurrentPrefix(NewAsyncWriterLogger)
				ch <- struct{}{}
			}()
			Eventually(ch, time.Second*5).Should(Receive())
		})
	})
})
