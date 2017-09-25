#!/bin/bash

set -e -x

VERSION=$(cat bosh-agent-zip-version/number)
COMPILED_AGENT_ZIP=$PWD/compiled-agent-zip

export PATH=/usr/local/ruby/bin:/usr/local/go/bin:$PATH
export GOPATH=$(pwd)/gopath

cd gopath/src/github.com/cloudfoundry/bosh-agent

GOOS=windows ./bin/go build -o bosh-agent.exe main/agent.go

git rev-parse HEAD > ./commit

curl http://repo.jenkins-ci.org/releases/com/sun/winsw/winsw/1.18/winsw-1.18-bin.exe -o ./service_wrapper.exe

cat > ./service_wrapper.xml <<EOF
<service>
  <id>bosh-agent</id>
  <name>BOSH Agent</name>
  <description>BOSH Agent</description>
  <executable>bosh-agent.exe</executable>
  <arguments>-P windows -C agent.json -M windows</arguments>
  <logpath>/var/vcap/bosh/log</logpath>
  <log mode="roll-by-size">
  	<sizeThreshold>10240</sizeThreshold>
  	<keepFiles>8</keepFiles>
  </log>
  <onfailure action="restart" delay="5 sec"/>
</service>
EOF

cat > ./service_wrapper.exe.config <<EOF
<configuration>
  <startup>
    <supportedRuntime version="v4.0" />
  </startup>
</configuration>
EOF

apt-get update
apt-get -y install zip

RELEASE_ZIP=$PWD/bosh-windows-integration-v$VERSION.zip
zip ${RELEASE_ZIP} ./commit ./bosh-agent.exe ./service_wrapper.exe ./service_wrapper.xml ./service_wrapper.exe.config
mv ${RELEASE_ZIP} ${COMPILED_AGENT_ZIP}
