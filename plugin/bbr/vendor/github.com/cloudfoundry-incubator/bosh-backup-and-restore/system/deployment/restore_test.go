package deployment

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/system"
)

var _ = Describe("Restores a deployment", func() {
	var workspaceDir = "/var/vcap/store/restore_workspace"
	var backupMetadata = "../../fixtures/redis-backup/metadata"
	var instanceCollection = map[string][]string{
		"redis":       {"0", "1"},
		"other-redis": {"0"},
	}
	var backupName = "redis-backup_20170502T132536Z"

	It("restores", func() {
		By("setting up the jump box")
		Eventually(JumpboxInstance.RunCommand(
			fmt.Sprintf("sudo mkdir -p %s && sudo chown -R vcap:vcap %s && sudo chmod -R 0777 %s",
				workspaceDir+"/"+backupName, workspaceDir, workspaceDir),
		)).Should(gexec.Exit(0))

		JumpboxInstance.Copy( MustHaveEnv("BOSH_CERT_PATH"), workspaceDir+"/bosh.crt")
		JumpboxInstance.Copy( commandPath, workspaceDir)
		JumpboxInstance.Copy( backupMetadata, workspaceDir+"/"+backupName+"/metadata")
		runOnInstances(instanceCollection, func(in, ii string) {
			fileName := fmt.Sprintf("%s-%s-redis-server.tar", in, ii)
			JumpboxInstance.Copy(
				fixturesPath+fileName,
				fmt.Sprintf("%s/%s/%s", workspaceDir, backupName, fileName),
			)
		})

		By("running the restore command")
		Eventually(JumpboxInstance.RunCommand(
			fmt.Sprintf(`cd %s;
			BOSH_CLIENT_SECRET=%s ./bbr \
			  deployment --debug \
			  --ca-cert bosh.crt \
			  --username %s \
			  --target %s \
			  --deployment %s \
			  restore \
			  --artifact-path %s`,
				workspaceDir,
				MustHaveEnv("BOSH_CLIENT_SECRET"),
				MustHaveEnv("BOSH_CLIENT"),
				MustHaveEnv("BOSH_URL"),
				RedisDeployment.Name,
				backupName,
			),
		)).Should(gexec.Exit(0))

		By("running the post restore unlock script")
		runOnInstances(instanceCollection, func(instName, instIndex string) {
			session := RedisDeployment.Instance(instName, instIndex).RunCommand(
				"cat /tmp/post-restore-unlock.out",
			)

			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out.Contents()).Should(ContainSubstring("output from post-restore-unlock"))
		})

		By("cleaning up artifacts from the remote instances")
		runOnInstances(instanceCollection, func(instName, instIndex string) {
			session := RedisDeployment.Instance(instName, instIndex).RunCommand(
				"ls -l /var/vcap/store/bbr-backup",
			)
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).To(Equal(1))
			Expect(session.Out).To(gbytes.Say("No such file or directory"))
		})

		By("ensuring data is restored")
		runOnInstances(instanceCollection, func(instName, instIndex string) {
			Eventually(RedisDeployment.Instance(instName, instIndex).RunCommand(
				fmt.Sprintf("sudo ls -la /var/vcap/store/redis-server"),
			)).Should(gexec.Exit(0))

			redisSession := RedisDeployment.Instance(instName, instIndex).RunCommand(
				"/var/vcap/packages/redis/bin/redis-cli -a redis get FOO23",
			)

			Eventually(redisSession.Out).Should(gbytes.Say("BAR23"))
		})
	})
})
