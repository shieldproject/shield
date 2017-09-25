package mbus

import (
	"net/url"

	"github.com/cloudfoundry/yagnats"

	boshhandler "github.com/cloudfoundry/bosh-agent/handler"
	boshmicro "github.com/cloudfoundry/bosh-agent/micro"
	boshplatform "github.com/cloudfoundry/bosh-agent/platform"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type HandlerProvider struct {
	settingsService boshsettings.Service
	logger          boshlog.Logger
	handler         boshhandler.Handler
}

func NewHandlerProvider(
	settingsService boshsettings.Service,
	logger boshlog.Logger,
) (p HandlerProvider) {
	p.settingsService = settingsService
	p.logger = logger
	return
}

func (p HandlerProvider) Get(
	platform boshplatform.Platform,
	dirProvider boshdir.Provider,
) (handler boshhandler.Handler, err error) {
	if p.handler != nil {
		handler = p.handler
		return
	}

	mbusURL, err := url.Parse(p.settingsService.GetSettings().Mbus)
	if err != nil {
		err = bosherr.WrapError(err, "Parsing handler URL")
		return
	}

	switch mbusURL.Scheme {
	case "nats":
		handler = NewNatsHandler(p.settingsService, yagnats.NewClient(), p.logger, platform)
	case "https":
		handler = boshmicro.NewHTTPSHandler(mbusURL, p.logger, platform.GetFs(), dirProvider)
	default:
		err = bosherr.Errorf("Message Bus Handler with scheme %s could not be found", mbusURL.Scheme)
	}

	p.handler = handler

	return
}
