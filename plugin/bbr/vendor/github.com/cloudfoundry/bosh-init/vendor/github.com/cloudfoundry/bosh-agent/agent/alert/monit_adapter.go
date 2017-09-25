package alert

import (
	"fmt"
	"sort"
	"strings"
	"time"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	"github.com/pivotal-golang/clock"
)

type MonitAdapter interface {
	IsIgnorable() bool
	Alert() (Alert, error)
	Severity() (severity SeverityLevel, found bool)
}

type monitAdapter struct {
	monitAlert      MonitAlert
	settingsService boshsettings.Service
	timeService     clock.Clock
}

func NewMonitAdapter(monitAlert MonitAlert, settingsService boshsettings.Service, timeService clock.Clock) MonitAdapter {
	return &monitAdapter{
		monitAlert:      monitAlert,
		settingsService: settingsService,
		timeService:     timeService,
	}
}

func (m *monitAdapter) IsIgnorable() bool {
	severity, _ := m.Severity()
	return severity == SeverityIgnored
}

func (m *monitAdapter) Alert() (Alert, error) {
	severity, _ := m.Severity()
	return Alert{
		ID:        m.monitAlert.ID,
		Severity:  severity,
		Title:     m.title(),
		Summary:   m.monitAlert.Description,
		CreatedAt: m.createdAt(),
	}, nil
}

func (m *monitAdapter) title() string {
	settings := m.settingsService.GetSettings()

	ips := settings.Networks.IPs()
	sort.Strings(ips)

	service := m.monitAlert.Service

	if len(ips) > 0 {
		service = fmt.Sprintf("%s (%s)", service, strings.Join(ips, ", "))
	}

	return fmt.Sprintf("%s - %s - %s", service, m.monitAlert.Event, m.monitAlert.Action)
}

func (m *monitAdapter) createdAt() int64 {
	createdAt, timeParseErr := time.Parse(time.RFC1123Z, m.monitAlert.Date)
	if timeParseErr != nil {
		createdAt = m.timeService.Now()
	}

	return createdAt.Unix()
}

func (m *monitAdapter) Severity() (severity SeverityLevel, found bool) {
	severity, found = eventToSeverity[strings.ToLower(m.monitAlert.Event)]
	if !found {
		severity = SeverityDefault
	}
	return severity, found
}

var eventToSeverity = map[string]SeverityLevel{
	"action done":                  SeverityIgnored,
	"checksum failed":              SeverityCritical,
	"checksum changed":             SeverityWarning,
	"checksum succeeded":           SeverityIgnored,
	"checksum not changed":         SeverityIgnored,
	"connection failed":            SeverityAlert,
	"connection succeeded":         SeverityIgnored,
	"connection changed":           SeverityError,
	"connection not changed":       SeverityIgnored,
	"content failed":               SeverityError,
	"content succeeded":            SeverityIgnored,
	"content match":                SeverityIgnored,
	"content doesn't match":        SeverityError,
	"data access error":            SeverityError,
	"data access succeeded":        SeverityIgnored,
	"data access changed":          SeverityWarning,
	"data access not changed":      SeverityIgnored,
	"execution failed":             SeverityAlert,
	"execution succeeded":          SeverityIgnored,
	"execution changed":            SeverityWarning,
	"execution not changed":        SeverityIgnored,
	"filesystem flags failed":      SeverityError,
	"filesystem flags succeeded":   SeverityIgnored,
	"filesystem flags changed":     SeverityWarning,
	"filesystem flags not changed": SeverityIgnored,
	"gid failed":                   SeverityError,
	"gid succeeded":                SeverityIgnored,
	"gid changed":                  SeverityWarning,
	"gid not changed":              SeverityIgnored,
	"heartbeat failed":             SeverityError,
	"heartbeat succeeded":          SeverityIgnored,
	"heartbeat changed":            SeverityWarning,
	"heartbeat not changed":        SeverityIgnored,
	"icmp failed":                  SeverityCritical,
	"icmp succeeded":               SeverityIgnored,
	"icmp changed":                 SeverityWarning,
	"icmp not changed":             SeverityIgnored,
	"monit instance failed":        SeverityAlert,
	"monit instance succeeded":     SeverityIgnored,
	"monit instance changed":       SeverityIgnored,
	"monit instance not changed":   SeverityIgnored,
	"invalid type":                 SeverityError,
	"type succeeded":               SeverityIgnored,
	"type changed":                 SeverityWarning,
	"type not changed":             SeverityIgnored,
	"does not exist":               SeverityAlert,
	"exists":                       SeverityIgnored,
	"existence changed":            SeverityWarning,
	"existence not changed":        SeverityIgnored,
	"permission failed":            SeverityError,
	"permission succeeded":         SeverityIgnored,
	"permission changed":           SeverityWarning,
	"permission not changed":       SeverityIgnored,
	"pid failed":                   SeverityCritical,
	"pid succeeded":                SeverityIgnored,
	"pid changed":                  SeverityWarning,
	"pid not changed":              SeverityIgnored,
	"ppid failed":                  SeverityCritical,
	"ppid succeeded":               SeverityIgnored,
	"ppid changed":                 SeverityWarning,
	"ppid not changed":             SeverityIgnored,
	"resource limit matched":       SeverityError,
	"resource limit succeeded":     SeverityIgnored,
	"resource limit changed":       SeverityWarning,
	"resource limit not changed":   SeverityIgnored,
	"size failed":                  SeverityError,
	"size succeeded":               SeverityIgnored,
	"size changed":                 SeverityError,
	"size not changed":             SeverityIgnored,
	"timeout":                      SeverityCritical,
	"timeout recovery":             SeverityIgnored,
	"timeout changed":              SeverityWarning,
	"timeout not changed":          SeverityIgnored,
	"timestamp failed":             SeverityError,
	"timestamp succeeded":          SeverityIgnored,
	"timestamp changed":            SeverityWarning,
	"timestamp not changed":        SeverityIgnored,
	"uid failed":                   SeverityCritical,
	"uid succeeded":                SeverityIgnored,
	"uid changed":                  SeverityWarning,
	"uid not changed":              SeverityIgnored,
}
