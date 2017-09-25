package alert

import (
	"regexp"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshsyslog "github.com/cloudfoundry/bosh-agent/syslog"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
	"github.com/pivotal-golang/clock"
)

var syslogMessageExpressions = map[*regexp.Regexp]string{
	regexp.MustCompile("disconnected by user"):                  "SSH Logout",
	regexp.MustCompile("Accepted publickey for"):                "SSH Login",
	regexp.MustCompile("Accepted password for"):                 "SSH Login",
	regexp.MustCompile("Failed password for"):                   "SSH Access Denied",
	regexp.MustCompile("Connection closed by .* \\[preauth\\]"): "SSH Access Denied",
}

type sshAdapter struct {
	message         boshsyslog.Msg
	settingsService boshsettings.Service
	uuidGenerator   boshuuid.Generator
	timeService     clock.Clock
	logger          boshlog.Logger
}

func NewSSHAdapter(
	message boshsyslog.Msg,
	settingsService boshsettings.Service,
	uuidGenerator boshuuid.Generator,
	timeService clock.Clock,
	logger boshlog.Logger,
) Adapter {
	return &sshAdapter{
		message:         message,
		settingsService: settingsService,
		uuidGenerator:   uuidGenerator,
		timeService:     timeService,
		logger:          logger,
	}
}

func (m *sshAdapter) IsIgnorable() bool {
	_, found := m.title()
	return !found
}

func (m *sshAdapter) Alert() (Alert, error) {
	title, found := m.title()
	if !found {
		return Alert{}, nil
	}

	uuid, err := m.uuidGenerator.Generate()
	if err != nil {
		return Alert{}, bosherr.WrapError(err, "Generating uuid")
	}

	return Alert{
		ID:        uuid,
		Severity:  SeverityWarning,
		Title:     title,
		Summary:   m.message.Content,
		CreatedAt: m.timeService.Now().Unix(),
	}, nil
}

func (m *sshAdapter) title() (title string, found bool) {
	for expression, title := range syslogMessageExpressions {
		if expression.MatchString(m.message.Content) {
			return title, true
		}
	}
	return "", false
}
