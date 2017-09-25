set GOPATH=%CD%\gopath
set PATH=%GOPATH%\bin;%PATH%
set GO15VENDOREXPERIMENT=1

cd %GOPATH%\src\github.com\cloudfoundry\bosh-agent

go install github.com/cloudfoundry/bosh-agent/vendor/github.com/onsi/ginkgo/ginkgo

ginkgo -r -keepGoing -skipPackage="integration,vendor"
