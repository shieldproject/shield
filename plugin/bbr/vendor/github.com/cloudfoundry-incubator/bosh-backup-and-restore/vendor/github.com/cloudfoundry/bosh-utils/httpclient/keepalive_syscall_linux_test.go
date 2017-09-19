package httpclient_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/httpclient"
	"github.com/cloudfoundry/bosh-utils/httpclient/fakes"
	"net"
	"os"
	"syscall"
)

var _ HTTPClient = &fakes.FakeHTTPClient{}

var _ = Describe("Linux-specific tests", func() {
	It("enables TCP (socket) keepalive with an appropriate interval", func() {
		// to test keepalive, we need a socket. A socket is an _active_ TCP connection to a server.
		// we make our own server, connect to it, and make our assertions against the socket
		laddr := "127.0.0.1:19642" // unlikely-to-be-used port number, unprivileged (1964, Feb, my birth)
		readyToAccept := make(chan bool, 1)

		go func() {
			defer GinkgoRecover()
			defer func() {
				readyToAccept <- true
			}()

			ln, err := net.Listen("tcp", laddr)
			Expect(err).ToNot(HaveOccurred())

			readyToAccept <- true

			_, err = ln.Accept()
			Expect(err).ToNot(HaveOccurred())
		}()

		<-readyToAccept

		client := CreateDefaultClient(nil)
		connection, err := client.Transport.(*http.Transport).Dial("tcp", laddr)
		Expect(err).ToNot(HaveOccurred())

		tcpConn, ok := connection.(*net.TCPConn)
		Expect(ok).To(BeTrue())

		f, err := tcpConn.File()
		Expect(err).ToNot(HaveOccurred())

		sockoptValue, err := syscall.GetsockoptInt(int(f.Fd()), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE)
		err = os.NewSyscallError("getsockopt", err)
		Expect(err).ToNot(HaveOccurred())
		Expect(sockoptValue).To(Equal(0x1))

		sockoptValue, err = syscall.GetsockoptInt(int(f.Fd()), syscall.IPPROTO_TCP, syscall.TCP_KEEPINTVL)
		err = os.NewSyscallError("getsockopt", err)
		Expect(err).ToNot(HaveOccurred())
		Expect(sockoptValue).To(Equal(30))
	})
})
