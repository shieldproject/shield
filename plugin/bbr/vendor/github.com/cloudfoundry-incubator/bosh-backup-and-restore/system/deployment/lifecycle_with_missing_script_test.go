package deployment

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
)

var _ = Describe("backup with a missing script", func() {
	It("fails with a meaningful message", func() {
		By("running the backup command")
		session := JumpboxInstance.RunCommandAs("vcap",
			fmt.Sprintf(
				`cd %s; BOSH_CLIENT_SECRET=%s ./bbr deployment --ca-cert bosh.crt --username %s --target %s --deployment %s backup`,
				workspaceDir,
				MustHaveEnv("BOSH_CLIENT_SECRET"),
				MustHaveEnv("BOSH_CLIENT"),
				MustHaveEnv("BOSH_URL"),
				RedisWithMissingScriptDeployment.Name,
			),
		)
		Eventually(session).Should(gexec.Exit(1))
		Expect(session).To(gbytes.Say(
			"The redis-server-with-restore-metadata restore script expects a backup script which produces custom-redis-backup artifact which is not present in the deployment",
		))
	})
})
