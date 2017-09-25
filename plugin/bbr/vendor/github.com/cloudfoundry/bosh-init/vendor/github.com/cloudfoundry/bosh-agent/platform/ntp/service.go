package ntp

import (
	"path"
	"regexp"
	"strings"

	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

var (
	offsetRegex    = regexp.MustCompile(`^(.+)\s+ntpdate.+offset\s+(-*\d+\.\d+)`)
	badServerRegex = regexp.MustCompile(`no server suitable for synchronization found`)
)

type Info struct {
	Offset    string `json:"offset,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Message   string `json:"message,omitempty"`
}

type Service interface {
	GetInfo() (ntpInfo Info)
}

type concreteService struct {
	fs          boshsys.FileSystem
	dirProvider boshdir.Provider
}

func NewConcreteService(fs boshsys.FileSystem, dirProvider boshdir.Provider) Service {
	return concreteService{
		fs:          fs,
		dirProvider: dirProvider,
	}
}

func (oc concreteService) GetInfo() Info {
	ntpPath := path.Join(oc.dirProvider.BaseDir(), "/bosh/log/ntpdate.out")
	content, err := oc.fs.ReadFileString(ntpPath)
	if err != nil {
		return Info{Message: "file missing"}
	}

	lines := strings.Split(strings.Trim(content, "\n"), "\n")
	lastLine := lines[len(lines)-1]

	matches := offsetRegex.FindAllStringSubmatch(lastLine, -1)

	if len(matches) > 0 && len(matches[0]) == 3 {
		return Info{
			Timestamp: matches[0][1],
			Offset:    matches[0][2],
		}
	} else if badServerRegex.MatchString(lastLine) {
		return Info{Message: "bad ntp server"}
	} else {
		return Info{Message: "bad file contents"}
	}
}
