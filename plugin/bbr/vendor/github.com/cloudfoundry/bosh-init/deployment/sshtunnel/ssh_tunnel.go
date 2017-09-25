package sshtunnel

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/pivotal-golang/clock"
)

type SSHTunnel interface {
	Start(chan<- error, chan<- error)
}

type sshTunnel struct {
	connectionRefusedTimeout time.Duration
	authFailureTimeout       time.Duration
	timeService              clock.Clock
	startDialDelay           time.Duration
	options                  Options
	remoteListener           net.Listener
	logger                   boshlog.Logger
	logTag                   string
}

func (s *sshTunnel) Start(readyErrCh chan<- error, errCh chan<- error) {
	authMethods := []ssh.AuthMethod{}

	if s.options.PrivateKey != "" {
		s.logger.Debug(s.logTag, "Reading private key file '%s'", s.options.PrivateKey)
		keyContents, err := ioutil.ReadFile(s.options.PrivateKey)
		if err != nil {
			readyErrCh <- bosherr.WrapErrorf(err, "Reading private key file '%s'", s.options.PrivateKey)
			return
		}

		s.logger.Debug(s.logTag, "Parsing private key file '%s'", s.options.PrivateKey)
		signer, err := ssh.ParsePrivateKey(keyContents)
		if err != nil {
			readyErrCh <- bosherr.WrapErrorf(err, "Parsing private key file '%s'", s.options.PrivateKey)
			return
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if s.options.Password != "" {
		s.logger.Debug(s.logTag, "Adding password auth method to ssh tunnel config")

		keyboardInteractiveChallenge := func(
			user,
			instruction string,
			questions []string,
			echos []bool,
		) (answers []string, err error) {
			if len(questions) == 0 {
				return []string{}, nil
			}
			return []string{s.options.Password}, nil
		}
		authMethods = append(authMethods, ssh.KeyboardInteractive(keyboardInteractiveChallenge))
		authMethods = append(authMethods, ssh.Password(s.options.Password))
	}

	sshConfig := &ssh.ClientConfig{
		User: s.options.User,
		Auth: authMethods,
	}

	s.logger.Debug(s.logTag, "Dialing remote server at %s:%d", s.options.Host, s.options.Port)
	remoteAddr := fmt.Sprintf("%s:%d", s.options.Host, s.options.Port)

	retryStrategy := &SSHRetryStrategy{
		TimeService:              s.timeService,
		ConnectionRefusedTimeout: s.connectionRefusedTimeout,
		AuthFailureTimeout:       s.authFailureTimeout,
	}

	var conn *ssh.Client
	var err error
	for i := 0; ; i++ {
		s.logger.Debug(s.logTag, "Making attempt #%d", i)
		conn, err = ssh.Dial("tcp", remoteAddr, sshConfig)

		if err == nil {
			break
		}

		if !retryStrategy.IsRetryable(err) {
			readyErrCh <- bosherr.WrapError(err, "Failed to connect to remote server")
			return
		}

		s.logger.Debug(s.logTag, "Attempt failed #%d: Dialing remote server: %s", i, err.Error())

		time.Sleep(s.startDialDelay)
	}

	remoteListenAddr := fmt.Sprintf("127.0.0.1:%d", s.options.RemoteForwardPort)
	s.logger.Debug(s.logTag, "Listening on remote server %s", remoteListenAddr)
	s.remoteListener, err = conn.Listen("tcp", remoteListenAddr)
	if err != nil {
		readyErrCh <- bosherr.WrapError(err, "Listening on remote server")
		return
	}

	readyErrCh <- nil
	for {
		remoteConn, err := s.remoteListener.Accept()
		s.logger.Debug(s.logTag, "Received connection")
		if err != nil {
			errCh <- bosherr.WrapError(err, "Accepting connection on remote server")
		}
		defer func() {
			if err = remoteConn.Close(); err != nil {
				s.logger.Warn(s.logTag, "Failed to close remote listener connection: %s", err.Error())
			}
		}()

		s.logger.Debug(s.logTag, "Dialing local server")
		localDialAddr := fmt.Sprintf("127.0.0.1:%d", s.options.LocalForwardPort)
		localConn, err := net.Dial("tcp", localDialAddr)
		if err != nil {
			errCh <- bosherr.WrapError(err, "Dialing local server")
			return
		}

		go func() {
			bytesNum, err := io.Copy(remoteConn, localConn)
			defer func() {
				if err = localConn.Close(); err != nil {
					s.logger.Warn(s.logTag, "Failed to close local dial connection: %s", err.Error())
				}
			}()
			s.logger.Debug(s.logTag, "Copying bytes from local to remote %d", bytesNum)
			if err != nil {
				errCh <- bosherr.WrapError(err, "Copying bytes from local to remote")
			}
		}()

		go func() {
			bytesNum, err := io.Copy(localConn, remoteConn)
			defer func() {
				if err = localConn.Close(); err != nil {
					s.logger.Warn(s.logTag, "Failed to close local dial connection: %s", err.Error())
				}
			}()
			s.logger.Debug(s.logTag, "Copying bytes from remote to local %d", bytesNum)
			if err != nil {
				errCh <- bosherr.WrapError(err, "Copying bytes from remote to local")
			}
		}()
	}
}

type SSHRetryStrategy struct {
	ConnectionRefusedTimeout time.Duration
	AuthFailureTimeout       time.Duration
	TimeService              clock.Clock

	initialized   bool
	startTime     time.Time
	authStartTime time.Time
}

func (s *SSHRetryStrategy) IsRetryable(err error) bool {
	now := s.TimeService.Now()
	if !s.initialized {
		s.startTime = now
		s.authStartTime = now
		s.initialized = true
	}

	if strings.Contains(err.Error(), "no common algorithms") {
		return false
	}

	if strings.Contains(err.Error(), "unable to authenticate") {
		return now.Before(s.authStartTime.Add(s.AuthFailureTimeout))
	}

	s.authStartTime = now
	return now.Before(s.startTime.Add(s.ConnectionRefusedTimeout))
}
