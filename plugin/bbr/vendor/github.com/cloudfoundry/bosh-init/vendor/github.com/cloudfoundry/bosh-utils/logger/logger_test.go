package logger_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bytes"
	. "github.com/cloudfoundry/bosh-utils/logger"
)

func expectedLogFormat(tag, msg string) string {
	return fmt.Sprintf("\\[%s\\] [0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} %s\n", tag, msg)
}

var _ = Describe("Levelify", func() {
	It("converts strings into LogLevel constants", func() {
		level, err := Levelify("NONE")
		Expect(err).ToNot(HaveOccurred())
		Expect(level).To(Equal(LevelNone))

		level, err = Levelify("none")
		Expect(err).ToNot(HaveOccurred())
		Expect(level).To(Equal(LevelNone))

		level, err = Levelify("DEBUG")
		Expect(err).ToNot(HaveOccurred())
		Expect(level).To(Equal(LevelDebug))

		level, err = Levelify("debug")
		Expect(err).ToNot(HaveOccurred())
		Expect(level).To(Equal(LevelDebug))

		level, err = Levelify("INFO")
		Expect(err).ToNot(HaveOccurred())
		Expect(level).To(Equal(LevelInfo))

		level, err = Levelify("info")
		Expect(err).ToNot(HaveOccurred())
		Expect(level).To(Equal(LevelInfo))

		level, err = Levelify("WARN")
		Expect(err).ToNot(HaveOccurred())
		Expect(level).To(Equal(LevelWarn))

		level, err = Levelify("warn")
		Expect(err).ToNot(HaveOccurred())
		Expect(level).To(Equal(LevelWarn))

		level, err = Levelify("ERROR")
		Expect(err).ToNot(HaveOccurred())
		Expect(level).To(Equal(LevelError))

		level, err = Levelify("error")
		Expect(err).ToNot(HaveOccurred())
		Expect(level).To(Equal(LevelError))
	})

	It("errors on unknown input", func() {
		_, err := Levelify("unknown")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Unknown LogLevel string 'unknown', expected one of [DEBUG, INFO, WARN, ERROR, NONE]"))

		_, err = Levelify("")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Unknown LogLevel string '', expected one of [DEBUG, INFO, WARN, ERROR, NONE]"))
	})
})

var _ = Describe("Logger", func() {
	var (
		outBuf *bytes.Buffer
		errBuf *bytes.Buffer
	)
	BeforeEach(func() {
		outBuf = bytes.NewBufferString("")
		errBuf = bytes.NewBufferString("")
	})

	Describe("Debug", func() {
		It("logs the formatted message to Logger.out at the debug level", func() {
			logger := NewWriterLogger(LevelDebug, outBuf, errBuf)
			logger.Debug("TAG", "some %s info to log", "awesome")

			expectedContent := expectedLogFormat("TAG", "DEBUG - some awesome info to log")
			Expect(outBuf).To(MatchRegexp(expectedContent))
			Expect(errBuf).ToNot(MatchRegexp(expectedContent))
		})
	})

	Describe("DebugWithDetails", func() {
		It("logs the message to Logger.out at the debug level with specially formatted arguments", func() {
			logger := NewWriterLogger(LevelDebug, outBuf, errBuf)
			logger.DebugWithDetails("TAG", "some info to log", "awesome")
			expectedContent := expectedLogFormat("TAG", "DEBUG - some info to log")
			Expect(outBuf).To(MatchRegexp(expectedContent))
			Expect(errBuf).ToNot(MatchRegexp(expectedContent))

			expectedDetails := "\n********************\nawesome\n********************"
			Expect(outBuf).To(ContainSubstring(expectedDetails))
			Expect(errBuf).ToNot(ContainSubstring(expectedDetails))
		})
	})

	Describe("Info", func() {
		It("logs the formatted message to Logger.out at the info level", func() {
			logger := NewWriterLogger(LevelInfo, outBuf, errBuf)
			logger.Info("TAG", "some %s info to log", "awesome")

			expectedContent := expectedLogFormat("TAG", "INFO - some awesome info to log")
			Expect(outBuf).To(MatchRegexp(expectedContent))
			Expect(errBuf).ToNot(MatchRegexp(expectedContent))
		})
	})

	Describe("Warn", func() {
		It("logs the formatted message to Logger.err at the warn level", func() {
			logger := NewWriterLogger(LevelWarn, outBuf, errBuf)
			logger.Warn("TAG", "some %s info to log", "awesome")

			expectedContent := expectedLogFormat("TAG", "WARN - some awesome info to log")
			Expect(outBuf).ToNot(MatchRegexp(expectedContent))
			Expect(errBuf).To(MatchRegexp(expectedContent))
		})
	})

	Describe("Error", func() {
		It("logs the formatted message to Logger.err at the error level", func() {
			logger := NewWriterLogger(LevelError, outBuf, errBuf)
			logger.Error("TAG", "some %s info to log", "awesome")

			expectedContent := expectedLogFormat("TAG", "ERROR - some awesome info to log")
			Expect(outBuf).ToNot(MatchRegexp(expectedContent))
			Expect(errBuf).To(MatchRegexp(expectedContent))
		})
	})

	Describe("ErrorWithDetails", func() {
		It("logs the message to Logger.err at the error level with specially formatted arguments", func() {
			logger := NewWriterLogger(LevelError, outBuf, errBuf)

			logger.ErrorWithDetails("TAG", "some error to log", "awesome")

			expectedContent := expectedLogFormat("TAG", "ERROR - some error to log")
			Expect(outBuf).ToNot(MatchRegexp(expectedContent))
			Expect(errBuf).To(MatchRegexp(expectedContent))

			expectedDetails := "\n********************\nawesome\n********************"
			Expect(outBuf).ToNot(ContainSubstring(expectedDetails))
			Expect(errBuf).To(ContainSubstring(expectedDetails))
		})
	})

	It("log level debug", func() {
		logger := NewWriterLogger(LevelDebug, outBuf, errBuf)
		logger.Debug("DEBUG", "some debug log")
		logger.Info("INFO", "some info log")
		logger.Warn("WARN", "some warn log")
		logger.Error("ERROR", "some error log")

		Expect(outBuf).To(ContainSubstring("DEBUG"))
		Expect(outBuf).To(ContainSubstring("INFO"))
		Expect(errBuf).To(ContainSubstring("WARN"))
		Expect(errBuf).To(ContainSubstring("ERROR"))
	})

	It("log level info", func() {
		logger := NewWriterLogger(LevelInfo, outBuf, errBuf)

		logger.Debug("DEBUG", "some debug log")
		logger.Info("INFO", "some info log")
		logger.Warn("WARN", "some warn log")
		logger.Error("ERROR", "some error log")

		Expect(outBuf).ToNot(ContainSubstring("DEBUG"))
		Expect(outBuf).To(ContainSubstring("INFO"))
		Expect(errBuf).To(ContainSubstring("WARN"))
		Expect(errBuf).To(ContainSubstring("ERROR"))
	})

	It("log level warn", func() {
		logger := NewWriterLogger(LevelWarn, outBuf, errBuf)

		logger.Debug("DEBUG", "some debug log")
		logger.Info("INFO", "some info log")
		logger.Warn("WARN", "some warn log")
		logger.Error("ERROR", "some error log")

		Expect(outBuf).ToNot(ContainSubstring("DEBUG"))
		Expect(outBuf).ToNot(ContainSubstring("INFO"))
		Expect(errBuf).To(ContainSubstring("WARN"))
		Expect(errBuf).To(ContainSubstring("ERROR"))
	})

	It("log level error", func() {
		logger := NewWriterLogger(LevelError, outBuf, errBuf)

		logger.Debug("DEBUG", "some debug log")
		logger.Info("INFO", "some info log")
		logger.Warn("WARN", "some warn log")
		logger.Error("ERROR", "some error log")

		Expect(outBuf).ToNot(ContainSubstring("DEBUG"))
		Expect(outBuf).ToNot(ContainSubstring("INFO"))
		Expect(errBuf).ToNot(ContainSubstring("WARN"))
		Expect(errBuf).To(ContainSubstring("ERROR"))
	})

	Describe("Toggling forced debug", func() {
		Describe("when the log level is error", func() {
			It("outputs at debug level", func() {
				logger := NewWriterLogger(LevelError, outBuf, errBuf)

				logger.ToggleForcedDebug()
				logger.Debug("TOGGLED_DEBUG", "some debug log")
				logger.Info("TOGGLED_INFO", "some info log")
				logger.Warn("TOGGLED_WARN", "some warn log")
				logger.Error("TOGGLED_ERROR", "some error log")

				Expect(outBuf).To(ContainSubstring("TOGGLED_DEBUG"))
				Expect(outBuf).To(ContainSubstring("TOGGLED_INFO"))
				Expect(errBuf).To(ContainSubstring("TOGGLED_WARN"))
				Expect(errBuf).To(ContainSubstring("TOGGLED_ERROR"))
			})

			It("outputs at error level when toggled back", func() {
				logger := NewWriterLogger(LevelError, outBuf, errBuf)

				logger.ToggleForcedDebug()
				logger.ToggleForcedDebug()
				logger.Debug("STANDARD_DEBUG", "some debug log")
				logger.Info("STANDARD_INFO", "some info log")
				logger.Warn("STANDARD_WARN", "some warn log")
				logger.Error("STANDARD_ERROR", "some error log")

				Expect(outBuf).ToNot(ContainSubstring("STANDARD_DEBUG"))
				Expect(outBuf).ToNot(ContainSubstring("STANDARD_INFO"))
				Expect(errBuf).ToNot(ContainSubstring("STANDARD_WARN"))
				Expect(errBuf).To(ContainSubstring("STANDARD_ERROR"))
			})
		})
	})
})
