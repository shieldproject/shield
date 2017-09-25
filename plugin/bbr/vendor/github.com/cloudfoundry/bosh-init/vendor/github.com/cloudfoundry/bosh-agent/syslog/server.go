package syslog

import (
	"bufio"
	"net"
	"strconv"
	"sync"

	"github.com/jeromer/syslogparser/rfc3164"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

const concreteServerLogTag = "conreteServer"

type concreteServer struct {
	port   uint16
	logger boshlog.Logger

	listener         net.Listener
	lock             sync.Mutex
	listenerProvider func(protocol, address string) (net.Listener, error)
}

func NewServer(port uint16, listenerProvider func(protocol, address string) (net.Listener, error), logger boshlog.Logger) Server {
	return &concreteServer{port: port, logger: logger, listenerProvider: listenerProvider}
}

func (s *concreteServer) Start(callback CallbackFunc) error {
	var err error

	s.lock.Lock()

	s.listener, err = s.listenerProvider("tcp", "127.0.0.1:"+strconv.Itoa(int(s.port)))
	if err != nil {
		s.lock.Unlock()
		return bosherr.WrapErrorf(err, "Listening on port %d", s.port)
	}

	// Should not defer unlock since there is a long-running loop
	s.lock.Unlock()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}

		go s.handleConnection(conn, callback)
	}
}

func (s *concreteServer) Stop() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.listener != nil {
		return s.listener.Close()
	}

	return nil
}

func (s *concreteServer) handleConnection(conn net.Conn, callback CallbackFunc) {
	defer func() {
		if err := conn.Close(); err != nil {
			s.logger.Error(concreteServerLogTag, "Failed to close connection: %s", err.Error())
		}
	}()

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		bytes := scanner.Bytes()

		p := rfc3164.NewParser(bytes)

		err := p.Parse()
		if err != nil {
			s.logger.Error(
				concreteServerLogTag,
				"Failed to parse syslog message: %s error: %s",
				string(bytes), err.Error(),
			)
			continue
		}

		content, ok := p.Dump()["content"].(string)
		if !ok {
			s.logger.Error(
				concreteServerLogTag,
				"Failed to retrieve syslog message string content: %s",
				string(bytes),
			)
			continue
		}

		message := Msg{Content: content}

		callback(message)
	}

	err := scanner.Err()
	if err != nil {
		s.logger.Error(
			concreteServerLogTag,
			"Scanner error while parsing syslog message: %s",
			err.Error(),
		)
	}
}
