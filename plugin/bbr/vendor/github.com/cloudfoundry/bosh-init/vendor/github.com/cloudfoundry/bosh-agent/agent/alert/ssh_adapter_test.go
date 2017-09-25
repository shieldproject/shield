package alert_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/alert"

	fakesettings "github.com/cloudfoundry/bosh-agent/settings/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	"github.com/pivotal-golang/clock/fakeclock"

	boshsyslog "github.com/cloudfoundry/bosh-agent/syslog"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("sshAdapter", func() {
	var (
		settingsService *fakesettings.FakeSettingsService
		timeService     *fakeclock.FakeClock
		logger          boshlog.Logger
		uuidGenerator   *fakeuuid.FakeGenerator
	)

	BeforeEach(func() {
		settingsService = &fakesettings.FakeSettingsService{}
		timeService = fakeclock.NewFakeClock(time.Now())
		logger = boshlog.NewLogger(boshlog.LevelNone)
		uuidGenerator = &fakeuuid.FakeGenerator{}
	})

	Describe("IsIgnorable", func() {

		itDoesNotIgnore := func(msgContent string) {
			sshMsg := boshsyslog.Msg{Content: msgContent}
			sshAdapter := NewSSHAdapter(
				sshMsg,
				settingsService,
				uuidGenerator,
				timeService,
				logger,
			)

			Expect(sshAdapter.IsIgnorable()).To(BeFalse())
		}

		It("Does not ignore user disconnects", func() {
			itDoesNotIgnore("Received disconnect from 9.9.9.9: 11: disconnected by user")
		})

		It("Does not ignore successful login (publickey)", func() {
			itDoesNotIgnore("Accepted publickey for vagrant from 9.9.9.9 port 58850 ssh2: RSA fake-rsa-key")
		})

		It("Does not ignore failed login (publickey)", func() {
			itDoesNotIgnore("Connection closed by 9.9.9.9 [preauth]")
		})

		It("Does not ignore successful login (password)", func() {
			itDoesNotIgnore("Accepted password for vcap from 172.16.79.1 port 63696 ssh2")
		})

		It("Does not ignore failed login (password)", func() {
			itDoesNotIgnore("Failed password for vcap from 172.16.79.1 port 63696 ssh2")
		})

		It("Ignores unknown messages", func() {
			msgContent := "ignorable unknown message"
			sshMsg := boshsyslog.Msg{Content: msgContent}
			sshAdapter := NewSSHAdapter(
				sshMsg,
				settingsService,
				uuidGenerator,
				timeService,
				logger,
			)

			Expect(sshAdapter.IsIgnorable()).To(BeTrue())
		})
	})

	Describe("Alert", func() {

		itAdaptsMessage := func(msgContent, expectedTitle string) {
			sshMsg := boshsyslog.Msg{Content: msgContent}
			sshAdapter := NewSSHAdapter(
				sshMsg,
				settingsService,
				uuidGenerator,
				timeService,
				logger,
			)

			uuidGenerator.GeneratedUUID = "fake-uuid"

			builtAlert, err := sshAdapter.Alert()
			Expect(err).ToNot(HaveOccurred())
			Expect(builtAlert.Title).To(Equal(expectedTitle))
			Expect(builtAlert.ID).To(Equal("fake-uuid"))
			Expect(builtAlert.Severity).To(Equal(SeverityWarning))
			Expect(builtAlert.Summary).To(Equal(msgContent))
			Expect(builtAlert.CreatedAt).To(Equal(timeService.Now().Unix()))
		}

		It("Returns logout when the user disconnects", func() {
			itAdaptsMessage("Received disconnect from 9.9.9.9: 11: disconnected by user", "SSH Logout")
		})

		It("Returns login when login is successful (publickey)", func() {
			itAdaptsMessage("Accepted publickey for vagrant from 9.9.9.9 port 58850 ssh2: RSA fake-rsa-key", "SSH Login")
		})

		It("Returns login when login is successful (password)", func() {
			itAdaptsMessage("Accepted password for vcap from 9.9.9.9 port 63696 ssh2", "SSH Login")
		})

		It("Returns preauth connection closed when the connection is closed during preauth", func() {
			itAdaptsMessage("Connection closed by 9.9.9.9 [preauth]", "SSH Access Denied")
		})

		It("Returns access denied when access is denied (password)", func() {
			itAdaptsMessage("Failed password for vcap from 9.9.9.9 port 63696 ssh2", "SSH Access Denied")
		})

		It("Defaults to SeverityWarning", func() {
			msgContent := "disconnected by user"
			sshMsg := boshsyslog.Msg{Content: msgContent}
			sshAdapter := NewSSHAdapter(
				sshMsg,
				settingsService,
				uuidGenerator,
				timeService,
				logger,
			)

			builtAlert, err := sshAdapter.Alert()
			Expect(err).ToNot(HaveOccurred())

			Expect(builtAlert.Severity).To(Equal(SeverityWarning))
		})

		It("CreatedAt is Now", func() {
			msgContent := "disconnected by user"
			sshMsg := boshsyslog.Msg{Content: msgContent}
			sshAdapter := NewSSHAdapter(
				sshMsg,
				settingsService,
				uuidGenerator,
				timeService,
				logger,
			)

			builtAlert, err := sshAdapter.Alert()
			Expect(err).ToNot(HaveOccurred())

			Expect(builtAlert.CreatedAt).To(Equal(timeService.Now().Unix()))
		})

		It("Sets the summary to the content of the message", func() {
			msgContent := "disconnected by user"
			sshMsg := boshsyslog.Msg{Content: msgContent}
			sshAdapter := NewSSHAdapter(
				sshMsg,
				settingsService,
				uuidGenerator,
				timeService,
				logger,
			)

			builtAlert, err := sshAdapter.Alert()
			Expect(err).ToNot(HaveOccurred())

			Expect(builtAlert.Summary).To(Equal(msgContent))
		})
	})
})
