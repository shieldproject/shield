package ssh

import (
	"fmt"
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
)

type SessionImpl struct {
	connOpts ConnectionOpts
	sessOpts SessionImplOpts
	result   boshdir.SSHResult

	privKeyFile    boshsys.File
	knownHostsFile boshsys.File

	fs boshsys.FileSystem
}

type SessionImplOpts struct {
	ForceTTY bool
}

func NewSessionImpl(
	connOpts ConnectionOpts,
	sessOpts SessionImplOpts,
	result boshdir.SSHResult,
	fs boshsys.FileSystem,
) *SessionImpl {
	return &SessionImpl{connOpts: connOpts, sessOpts: sessOpts, result: result, fs: fs}
}

func (r *SessionImpl) Start() ([]string, error) {
	var err error

	r.privKeyFile, err = r.makePrivKeyFile()
	if err != nil {
		return nil, err
	}

	r.knownHostsFile, err = r.makeKnownHostsFile()
	if err != nil {
		_ = r.fs.RemoveAll(r.privKeyFile.Name())
		return nil, err
	}

	// Options are used for both ssh and scp
	cmdOpts := []string{}

	if r.sessOpts.ForceTTY {
		cmdOpts = append(cmdOpts, "-tt")
	}

	cmdOpts = append(cmdOpts, []string{
		"-o", "ServerAliveInterval=30",
		"-o", "ForwardAgent=no",
		"-o", "PasswordAuthentication=no",
		"-o", "IdentitiesOnly=yes",
		"-o", "IdentityFile=" + r.privKeyFile.Name(),
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=" + r.knownHostsFile.Name(),
	}...)

	gwUsername, gwHost, gwPrivKeyPath := r.gwOpts(r.connOpts, r.result)

	if len(r.connOpts.SOCKS5Proxy) > 0 {
		proxyOpt := fmt.Sprintf(
			"ProxyCommand=nc -X 5 -x %s %%h %%p",
			strings.TrimPrefix(r.connOpts.SOCKS5Proxy, "socks5://"),
		)

		cmdOpts = append(cmdOpts, "-o", proxyOpt)

	} else if len(gwHost) > 0 {
		gwCmdOpts := []string{
			"-o", "ServerAliveInterval=30",
			"-o", "ForwardAgent=no",
			"-o", "ClearAllForwardings=yes",
			// Strict host key checking for a gateway is not necessary
			// since ProxyCommand is only used for forwarding TCP and
			// agent forwarding is disabled
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
		}

		if len(gwPrivKeyPath) > 0 {
			gwCmdOpts = append(
				gwCmdOpts,
				"-o", "PasswordAuthentication=no",
				"-o", "IdentitiesOnly=yes",
				"-o", "IdentityFile="+gwPrivKeyPath,
			)
		}

		proxyOpt := fmt.Sprintf(
			// Always force TTY for gateway ssh
			"ProxyCommand=ssh -tt -W %%h:%%p -l %s %s %s",
			gwUsername,
			gwHost,
			strings.Join(gwCmdOpts, " "),
		)

		cmdOpts = append(cmdOpts, "-o", proxyOpt)
	}

	cmdOpts = append(cmdOpts, r.connOpts.RawOpts...)

	return cmdOpts, nil
}

func (r *SessionImpl) Finish() error {
	// Make sure to try to delete all files regardless of errors
	privKeyErr := r.fs.RemoveAll(r.privKeyFile.Name())
	knownHostsErr := r.fs.RemoveAll(r.knownHostsFile.Name())

	if privKeyErr != nil {
		return privKeyErr
	}

	if knownHostsErr != nil {
		return knownHostsErr
	}

	return nil
}

func (r SessionImpl) makePrivKeyFile() (boshsys.File, error) {
	file, err := r.fs.TempFile("ssh-priv-key")
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Creating temp file for SSH private key")
	}

	_, err = file.Write([]byte(r.connOpts.PrivateKey))
	if err != nil {
		_ = r.fs.RemoveAll(file.Name())
		return nil, bosherr.WrapErrorf(err, "Writing SSH private key")
	}

	return file, nil
}

func (r SessionImpl) makeKnownHostsFile() (boshsys.File, error) {
	file, err := r.fs.TempFile("ssh-known-hosts")
	if err != nil {
		return nil, bosherr.WrapErrorf(err, "Creating temp file for SSH known hosts")
	}

	var content string

	for _, host := range r.result.Hosts {
		if len(host.HostPublicKey) > 0 {
			content += fmt.Sprintf("%s %s\n", host.Host, host.HostPublicKey)
		}
	}

	if len(content) > 0 {
		_, err := file.Write([]byte(content))
		if err != nil {
			_ = r.fs.RemoveAll(file.Name())
			return nil, bosherr.WrapErrorf(err, "Writing SSH known hosts")
		}
	}

	return file, nil
}

func (r SessionImpl) gwOpts(connOpts ConnectionOpts, result boshdir.SSHResult) (string, string, string) {
	if connOpts.GatewayDisable {
		return "", "", ""
	}

	// Take server provided gateway options
	username := result.GatewayUsername
	host := result.GatewayHost

	if len(connOpts.GatewayUsername) > 0 {
		username = connOpts.GatewayUsername
	}

	if len(connOpts.GatewayHost) > 0 {
		host = connOpts.GatewayHost
	}

	privKeyPath := connOpts.GatewayPrivateKeyPath

	return username, host, privKeyPath
}
