package log

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var Logger = boshlog.NewLogger(boshlog.LevelWarn)
