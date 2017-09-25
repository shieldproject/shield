package integrationagentclient_test

import (
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/cloudfoundry/bosh-agent/agent/action"
	agentclienthttp "github.com/cloudfoundry/bosh-agent/agentclient/http"
	"github.com/cloudfoundry/bosh-agent/integration/integrationagentclient"
	"github.com/cloudfoundry/bosh-utils/httpclient"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("AgentClient", func() {
	var (
		server      *ghttp.Server
		agentClient *integrationagentclient.IntegrationAgentClient

		agentAddress        string
		replyToAddress      string
		toleratedErrorCount int
	)

	BeforeEach(func() {
		server = ghttp.NewServer()

		logger := boshlog.NewLogger(boshlog.LevelNone)

		agentAddress = server.URL()
		replyToAddress = "fake-reply-to-uuid"

		getTaskDelay := time.Duration(0)
		toleratedErrorCount = 2

		agentClient = integrationagentclient.NewIntegrationAgentClient(
			agentAddress,
			replyToAddress,
			getTaskDelay,
			toleratedErrorCount,
			httpclient.NewHTTPClient(httpclient.DefaultClient, logger),
			logger,
		)
	})

	Describe("SSH", func() {
		Context("when agent successfully executes ssh", func() {
			BeforeEach(func() {
				sshSuccess, err := json.Marshal(action.SSHResult{
					Command: "setup",
					Status:  "success",
				})
				Expect(err).ToNot(HaveOccurred())
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/agent"),
						ghttp.RespondWith(200, string(sshSuccess)),
						ghttp.VerifyJSONRepresenting(agentclienthttp.AgentRequestMessage{
							Method:    "ssh",
							Arguments: []interface{}{"setup", map[string]interface{}{"user_regex": "", "User": "username", "public_key": ""}},
							ReplyTo:   "fake-reply-to-uuid",
						}),
					),
				)
			})

			It("makes a POST request to the endpoint", func() {
				params := action.SSHParams{
					User: "username",
				}

				err := agentClient.SSH("setup", params)
				Expect(err).ToNot(HaveOccurred())
				Expect(server.ReceivedRequests()).To(HaveLen(10))
			})
		})

		Context("when POST to agent returns error", func() {
			BeforeEach(func() {
				server.AppendHandlers(ghttp.RespondWith(http.StatusInternalServerError, ""))
			})

			It("returns an error that wraps original error", func() {
				params := action.SSHParams{
					User: "username",
				}

				err := agentClient.SSH("setup", params)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("Performing request to agent")))
				Expect(err).To(MatchError(ContainSubstring("foo error")))
			})
		})
	})
})
