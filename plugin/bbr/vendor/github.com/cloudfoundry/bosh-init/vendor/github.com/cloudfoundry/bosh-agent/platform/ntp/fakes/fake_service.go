package fakes

import boshntp "github.com/cloudfoundry/bosh-agent/platform/ntp"

type FakeService struct {
	GetOffsetNTPOffset boshntp.Info
}

func (oc *FakeService) GetInfo() (ntpInfo boshntp.Info) {
	ntpInfo = oc.GetOffsetNTPOffset
	return
}
