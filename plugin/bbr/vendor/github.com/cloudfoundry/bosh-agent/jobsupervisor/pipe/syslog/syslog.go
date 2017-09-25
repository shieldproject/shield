// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syslog

import (
	"errors"
	"net"
	"os"
	"sync"
	"time"
)

// The Priority is a combination of the syslog facility and
// severity. For example, LOG_ALERT | LOG_FTP sends an alert severity
// message from the FTP facility. The default severity is LOG_EMERG;
// the default facility is LOG_KERN.
type Priority int

const severityMask = 0x07
const facilityMask = 0xf8

const (
	// Severity.

	// From /usr/include/sys/syslog.h.
	// These are the same on Linux, BSD, and OS X.
	LOG_EMERG Priority = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

const (
	// Facility.

	// From /usr/include/sys/syslog.h.
	// These are the same up to LOG_FTP on Linux, BSD, and OS X.
	LOG_KERN Priority = iota << 3
	LOG_USER
	LOG_MAIL
	LOG_DAEMON
	LOG_AUTH
	LOG_SYSLOG
	LOG_LPR
	LOG_NEWS
	LOG_UUCP
	LOG_CRON
	LOG_AUTHPRIV
	LOG_FTP
	_ // unused
	_ // unused
	_ // unused
	_ // unused
	LOG_LOCAL0
	LOG_LOCAL1
	LOG_LOCAL2
	LOG_LOCAL3
	LOG_LOCAL4
	LOG_LOCAL5
	LOG_LOCAL6
	LOG_LOCAL7
)

const maxBufSize = 1024 * 1024 * 20 // 20MB

// A Writer is a connection to a syslog server.
type Writer struct {
	priority Priority
	tag      string
	hostname string
	network  string
	raddr    string
	mu       sync.Mutex // guards conn
	conn     *netConn

	b   []byte // buffer for formatted messages
	pid int    // cached process pid
}

type netConn struct {
	local bool
	conn  net.Conn
}

// Dial establishes a connection to a log daemon by connecting to
// address raddr on the specified network. Each write to the returned
// writer sends a log message with the given facility, severity and
// tag.
func Dial(network, raddr string, priority Priority, tag string) (*Writer, error) {
	hostname, _ := os.Hostname()
	return DialHostname(network, raddr, priority, tag, hostname)
}

func DialHostname(network, raddr string, priority Priority, tag, hostname string) (*Writer, error) {
	if priority < 0 || priority > LOG_LOCAL7|LOG_DEBUG {
		return nil, errors.New("log/syslog: invalid priority")
	}

	if tag == "" {
		tag = os.Args[0]
	}

	w := &Writer{
		priority: priority,
		tag:      tag,
		hostname: hostname,
		network:  network,
		raddr:    raddr,
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	err := w.connect()
	if err != nil {
		return nil, err
	}
	return w, err
}

// connect makes a connection to the syslog server.
// It must be called with w.mu held.
func (w *Writer) connect() (err error) {
	if w.conn != nil {
		// ignore err from close, it makes sense to continue anyway
		w.conn.close()
		w.conn = nil
	}

	var c net.Conn
	c, err = net.Dial(w.network, w.raddr)
	if err == nil {
		w.conn = &netConn{conn: c}
		if w.hostname == "" {
			w.hostname = c.LocalAddr().String()
		}
	}
	return
}

// Write sends a log message to the syslog daemon.
func (w *Writer) Write(b []byte) (int, error) {
	return w.writeAndRetry(w.priority, string(b))
}

// Close closes a connection to the syslog daemon.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		err := w.conn.close()
		w.conn = nil
		return err
	}
	return nil
}

// Emerg logs a message with severity LOG_EMERG, ignoring the severity
// passed to New.
func (w *Writer) Emerg(m string) error {
	_, err := w.writeAndRetry(LOG_EMERG, m)
	return err
}

// Alert logs a message with severity LOG_ALERT, ignoring the severity
// passed to New.
func (w *Writer) Alert(m string) error {
	_, err := w.writeAndRetry(LOG_ALERT, m)
	return err
}

// Crit logs a message with severity LOG_CRIT, ignoring the severity
// passed to New.
func (w *Writer) Crit(m string) error {
	_, err := w.writeAndRetry(LOG_CRIT, m)
	return err
}

// Err logs a message with severity LOG_ERR, ignoring the severity
// passed to New.
func (w *Writer) Err(m string) error {
	_, err := w.writeAndRetry(LOG_ERR, m)
	return err
}

// Warning logs a message with severity LOG_WARNING, ignoring the
// severity passed to New.
func (w *Writer) Warning(m string) error {
	_, err := w.writeAndRetry(LOG_WARNING, m)
	return err
}

// Notice logs a message with severity LOG_NOTICE, ignoring the
// severity passed to New.
func (w *Writer) Notice(m string) error {
	_, err := w.writeAndRetry(LOG_NOTICE, m)
	return err
}

// Info logs a message with severity LOG_INFO, ignoring the severity
// passed to New.
func (w *Writer) Info(m string) error {
	_, err := w.writeAndRetry(LOG_INFO, m)
	return err
}

// Debug logs a message with severity LOG_DEBUG, ignoring the severity
// passed to New.
func (w *Writer) Debug(m string) error {
	_, err := w.writeAndRetry(LOG_DEBUG, m)
	return err
}

func (w *Writer) writeAndRetry(p Priority, s string) (n int, err error) {
	pr := (w.priority & facilityMask) | (p & severityMask)
	var msg []byte

	w.mu.Lock()
	if w.conn != nil {
		msg = w.format(pr, w.hostname, w.tag, s)
		if n, err = w.conn.write(msg); err == nil {
			w.mu.Unlock()
			return
		}
	}
	if err = w.connect(); err == nil {
		if msg == nil {
			msg = w.format(pr, w.hostname, w.tag, s)
		}
		n, err = w.conn.write(msg)
	}
	w.mu.Unlock()
	return
}

func (w *Writer) getpid() int {
	if w.pid == 0 {
		w.pid = os.Getpid()
	}
	return w.pid
}

// itoa, cheap integer to fixed-width decimal ASCII.
func itoa(dst []byte, n int) []byte {
	var a [20]byte
	i := len(a)
	us := uintptr(n)
	for us >= 10 {
		i--
		q := us / 10
		a[i] = byte(us - q*10 + '0')
		us = q
	}
	i--
	a[i] = byte(us + '0')
	return append(dst, a[i:]...)
}

// format generates a syslog formatted string. The format is as follows:
//   <PRI>TIMESTAMP HOSTNAME TAG[PID]: MSG
func (w *Writer) format(p Priority, hostname, tag, msg string) []byte {
	b := w.b[0:0]
	b = append(b, '<')
	b = itoa(b, int(p))
	b = append(b, '>')
	b = time.Now().AppendFormat(b, time.RFC3339)
	b = append(b, ' ')
	b = append(b, hostname...)
	b = append(b, ' ')
	b = append(b, tag...)
	b = append(b, '[')
	b = itoa(b, w.getpid())
	b = append(b, "]: "...)
	b = append(b, msg...)
	if b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}
	if len(b) < maxBufSize {
		w.b = b
	}
	return b
}

func (n *netConn) write(p []byte) (int, error) {
	return n.conn.Write(p)
}

func (n *netConn) close() error {
	return n.conn.Close()
}
