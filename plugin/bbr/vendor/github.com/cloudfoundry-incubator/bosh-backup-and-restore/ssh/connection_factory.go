package ssh

import "golang.org/x/crypto/ssh"

//go:generate counterfeiter -o fakes/fake_ssh_connection_factory.go . SSHConnectionFactory
type SSHConnectionFactory func(host, user, privateKey string, publicKeyCallback ssh.HostKeyCallback, publicKeyAlgorithm []string, logger Logger) (SSHConnection, error)
