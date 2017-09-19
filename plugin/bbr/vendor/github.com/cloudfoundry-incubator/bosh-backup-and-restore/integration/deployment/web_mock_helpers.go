package deployment

import (
	"fmt"
	"math/rand"

	"github.com/pivotal-cf-experimental/cf-webmock/mockbosh"
	"github.com/pivotal-cf-experimental/cf-webmock/mockhttp"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/testcluster"
)

func MockDirectorWith(director *mockhttp.Server, info mockhttp.MockedResponseBuilder, vmsResponse []mockhttp.MockedResponseBuilder, sshResponse []mockhttp.MockedResponseBuilder, cleanupResponse []mockhttp.MockedResponseBuilder) {
	director.VerifyAndMock(AppendBuilders(
		[]mockhttp.MockedResponseBuilder{info},
		vmsResponse,
		sshResponse,
		cleanupResponse,
	)...)
}

func InfoWithBasicAuth() []mockhttp.MockedResponseBuilder {
	return []mockhttp.MockedResponseBuilder{
		mockbosh.Info().WithAuthTypeBasic(),
	}
}

func VmsForDeployment(deploymentName string, responseInstances []mockbosh.VMsOutput) []mockhttp.MockedResponseBuilder {
	randomTaskID := generateTaskId()
	return []mockhttp.MockedResponseBuilder{
		mockbosh.VMsForDeployment(deploymentName).RedirectsToTask(randomTaskID),
		mockbosh.Task(randomTaskID).RespondsWithTaskContainingState(mockbosh.TaskDone),
		mockbosh.Task(randomTaskID).RespondsWithTaskContainingState(mockbosh.TaskDone),
		mockbosh.TaskEvent(randomTaskID).RespondsWithVMsOutput([]string{}),
		mockbosh.TaskOutput(randomTaskID).RespondsWithVMsOutput(responseInstances),
	}
}

func VmsForDeploymentFails(deploymentName string) []mockhttp.MockedResponseBuilder {
	return []mockhttp.MockedResponseBuilder{
		mockbosh.VMsForDeployment(deploymentName).Fails("director unreachable"),
	}
}

func DownloadManifest(deploymentName string, manifest string) []mockhttp.MockedResponseBuilder {
	return []mockhttp.MockedResponseBuilder{
		mockbosh.Manifest(deploymentName).RespondsWith([]byte(manifest)),
	}
}

func AppendBuilders(arrayOfArrayOfBuilders ...[]mockhttp.MockedResponseBuilder) []mockhttp.MockedResponseBuilder {
	var flattenedArrayOfBuilders []mockhttp.MockedResponseBuilder
	for _, arrayOfBuilders := range arrayOfArrayOfBuilders {
		flattenedArrayOfBuilders = append(flattenedArrayOfBuilders, arrayOfBuilders...)
	}
	return flattenedArrayOfBuilders
}

func SetupSSH(deploymentName, instanceGroup, instanceID string, instanceIndex int, instance *testcluster.Instance) []mockhttp.MockedResponseBuilder {
	randomTaskID := generateTaskId()
	return []mockhttp.MockedResponseBuilder{
		mockbosh.StartSSHSession(deploymentName).SetSSHResponseCallback(func(username, key string) {
			instance.CreateUser(username, key)
		}).ForInstanceGroup(instanceGroup).RedirectsToTask(randomTaskID),
		mockbosh.Task(randomTaskID).RespondsWithTaskContainingState(mockbosh.TaskDone),
		mockbosh.Task(randomTaskID).RespondsWithTaskContainingState(mockbosh.TaskDone),
		mockbosh.TaskEvent(randomTaskID).RespondsWith("{}"),
		mockbosh.TaskOutput(randomTaskID).RespondsWith(
			fmt.Sprintf(`[{"status":"success",
"ip":"%s",
"host_public_key":"%s",
"id":"%s",
"index":%d}]`,
				instance.Address(),
				instance.HostPublicKey(),
				instanceID,
				instanceIndex,
			),
		),
	}
}

func CleanupSSH(deploymentName, instanceGroup string) []mockhttp.MockedResponseBuilder {
	randomTaskID := generateTaskId()
	return []mockhttp.MockedResponseBuilder{
		mockbosh.CleanupSSHSession(deploymentName).ForInstanceGroup(instanceGroup).RedirectsToTask(randomTaskID),
		mockbosh.Task(randomTaskID).RespondsWithTaskContainingState(mockbosh.TaskDone),
	}
}

func CleanupSSHFails(deploymentName, instanceGroup, errorMessage string) []mockhttp.MockedResponseBuilder {
	return []mockhttp.MockedResponseBuilder{
		mockbosh.CleanupSSHSession(deploymentName).ForInstanceGroup(instanceGroup).Fails(errorMessage),
	}
}

func generateTaskId() int {
	return rand.Int()
}
