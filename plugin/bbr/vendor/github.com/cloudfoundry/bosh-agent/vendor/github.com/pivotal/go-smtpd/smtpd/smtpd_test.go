package smtpd

import (
	"testing"
	"bytes"
	"fmt"
	"net/smtp"
	"net"
	"time"
)

var newMailWasReceived bool

func TestSmtpdServer(t *testing.T) {
	newMailWasReceived = false
	srvPort := startServer()

	conn, err := smtp.Dial(fmt.Sprintf("localhost:%d", srvPort))
	for err != nil {
		conn, err = smtp.Dial(fmt.Sprintf("localhost:%d", srvPort))
	}
	defer conn.Close()

	conn.Mail("sender@example.org")
	conn.Rcpt("recipient@example.net")

	writeCloser, err := conn.Data()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer writeCloser.Close()

	buf := bytes.NewBufferString(`Hello World!`)
	buf.WriteTo(writeCloser)

	if newMailWasReceived != true {
		t.Fatalf("Email was not received")
	}
}

func TestSmtpdServerWithMonit(t *testing.T) {
	newMailWasReceived = false
	srvPort := startServer()

	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", srvPort))
	for err != nil {
		conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", srvPort))
	}
	defer conn.Close()

	emailBytes := [][]byte{
		[]byte("Hello\r\n"),
		[]byte("MAIL FROM: <monit@localhost>\r\n"),
		[]byte("RCPT TO: <recipient@localhost>\r\n"),
		[]byte("data\r\n"),
		[]byte("From: sender@example.org\r\n"),
		[]byte("Subject: test mail from command line\r\n"),
		[]byte("\r\n"),
		[]byte("Message-id: <1304319946.0@localhost>\r\n"),
		[]byte("Service: nats\r\n"),
		[]byte("Event: does not exist\r\n"),
		[]byte("Action: restart\r\n"),
		[]byte("Date: Sun, 22 May 2011 20:07:41 +0500\r\n"),
		[]byte("Description: process is not running\r\n"),
		[]byte(".\r\n"),
	}

	for _, b := range emailBytes {
		_, err = conn.Write(b)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	}

	time.Sleep(10 * time.Millisecond)

	if newMailWasReceived != true {
		t.Fatalf("Email was not received")
	}
}

func startServer() (port int) {
	onNewMail := func(Connection, MailAddress) (env Envelope, err error) {
		newMailWasReceived = true
		env = new(BasicEnvelope)
		return
	}

	port = getTestServerPort()
	serv := &Server{
		Addr:      fmt.Sprintf(":%d", port),
		OnNewMail: onNewMail,
	}

	go serv.ListenAndServe()
	return
}

var testServerPort int = 2500

func getTestServerPort() int {
	testServerPort++
	return testServerPort
}
