package http_test

import (
	"time"

	"github.com/cloudfoundry/bosh-agent/agentclient/http"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AgentClientFactory", func() {
	var (
		agentClientFactory http.AgentClientFactory
		caCert             string
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		agentClientFactory = http.NewAgentClientFactory(time.Second, logger)
	})

	Describe("NewAgentClient", func() {
		Context("with a valid CA", func() {
			BeforeEach(func() {
				caCert = `-----BEGIN CERTIFICATE-----
MIIE/zCCAumgAwIBAgIBATALBgkqhkiG9w0BAQswETEPMA0GA1UEAxMGcGVlckNB
MB4XDTE1MDcxNjEzMjQxOFoXDTI1MDcxNjEzMjQyM1owETEPMA0GA1UEAxMGcGVl
ckNBMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAuDFaTLJ//NLZUR8S
gnKRh0vdjOfSwLakRfmWp/midwDFILuGvHgRd3ItsmNthy2ECQ3mr+zETAQ/Q3vp
ba3P1hNMtCC1aHHnnF2KXqDCH9bYh7mqEhCUy3QXhJVWET2RgmWtvfXwPxr+hvxQ
tjXhb9YloKkm99HNwREqczSUTZMmxirLbKnm7ztHtrUqMpaWiyKablgoukJCpufQ
fOlKdxdX7fpQ5C2n+rYQWM2Xxu+KXeWv6E2MoZIYv+Gch2ZRWXC6fhQn7u8qSszZ
reVMGbqsaQG+powLMOlA9ZW3KbIrf+jeNY5YFBWcPnGDNBZYgzud4x0i1BwfA7Mp
T8fwjF1xEkmxB7Qf2gUZPEUDOgkDFszW2p9vEtqleMKJqSTMhxEMiwSB/CSVvGWI
SclUHJN7pqcX2bKbGFWxMNfI/ez9lSDH7mqfRDPz/pLAvXLf5Xlsnzat50PKpBWt
Wns1Z5KDeVMMn0MYu7gZ0GdA+/OotsP2r3BnmyPeiTQ0IlGz9Z7ikn/rZ+QfK8jf
WGkQZlaQuNBUvC5UEn+I9n/qrTw38jUUY+IDDWOLp9VzpLNWIkSMKqJnN1igCZ/D
QoW2rbqGwrv7UJywW1clglrS9nmOsGU9LtsU+KJeGRKK9lazkpujiKOLz306rIUU
NBtbB1DDyvLTaj7Ln8VMD6v2BPkCAwEAAaNmMGQwDgYDVR0PAQH/BAQDAgAGMBIG
A1UdEwEB/wQIMAYBAf8CAQAwHQYDVR0OBBYEFNixBensHx4NqEIf5jnCXZSXxnuH
MB8GA1UdIwQYMBaAFNixBensHx4NqEIf5jnCXZSXxnuHMAsGCSqGSIb3DQEBCwOC
AgEAhaHd/x1rAwkgIVEc+Y69vsrrpb2NOY6MB2ogLJnu8KaAcmvYsfku06Sc5GLn
tXpkoftknrbjVV+g+XUhCz18NUY7YAFbYmembkC8ZVP32nQ1rsUf9jx8yiNYkeLq
ZOYlnKbSram4/6Efg0ttxEgbIbwYPviApEH6DK26++vvxejgV+GdcMR9XXwEi/kN
j1+ZfkzVnlO5j5uPLZi8vgsalJvWcPygolTxL73pfNXHj9QilxpUdJiVOvxke4MA
VJOg8o02DN5QqRyT6oM1ivwbe7AYfZYRIjsJdSOXYvcBHk6iHZdPZeJcFnNjUOaE
jvG/d9ezdUHa3C4qtHvmqcl2AjN/o50VyCY9/Mkgn8/tDOvVt3l3uSh0O4SQaZA1
+KN7n0Jl0yiyv+3uGVWNOEX87SREcP0GbrsCdOGm3HmDTWw0UFidNJdzXkj2Iayv
/hOq0PTBwTFm8shSXiPsjh6WMBXkkmu5FB51ZQ4Ch0MZDtuvlw9sGX9/zFNwL3W8
Kqu6zV6ZSlv9RW9ChbHtDvs+DdqetU9WLYjglPcHfpV/BH1HRozfR1bStYm9Ljwy
P8ZEmoycBR/79PtVdkSiFB4PiSkLHr6ICDSQGO+9+mLNQubFS+czQon90bZ9GVfg
fvue6FeCS62q1lOmwKsNHi26szI5qY8b6Xj3cNjhDS5pIfg=
-----END CERTIFICATE-----
`
			})

			It("returns a valid agent client", func() {
				agentClient, err := agentClientFactory.NewAgentClient("director-id", "mbus-url", caCert)
				Expect(err).NotTo(HaveOccurred())
				Expect(agentClient).NotTo(BeNil()) // no accessible fields to check for insecure/secure
			})
		})

		Context("with no CA", func() {
			BeforeEach(func() {
				caCert = ""
			})

			It("returns an insecure client", func() {
				agentClient, err := agentClientFactory.NewAgentClient("director-id", "mbus-url", caCert)
				Expect(err).ToNot(HaveOccurred())
				Expect(agentClient).NotTo(BeNil()) // no accessible fields to check for insecure/secure
			})
		})

		Context("with an invalid CA", func() {
			BeforeEach(func() {
				caCert = "potato?"
			})

			It("returns an error", func() {
				agentClient, err := agentClientFactory.NewAgentClient("director-id", "mbus-url", caCert)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Missing PEM block"))
				Expect(agentClient).To(BeNil())
			})
		})
	})
})
