# cf-webmock

Mocking library with a generic interface to mock sequential web requests. Provides DSL-ish wrappers for bosh and uaa apis.

###Usage

Initialize mock server
```go
BeforeEach(func() {
	boshDirector = mockbosh.New()
})
```

Setup mocks before interactions
```go
boshDirector.VerifyAndMock(
  mockbosh.Manifest(deploymentName(instanceID)).NotFound(),
  mockbosh.Tasks(deploymentName(instanceID)).RespondsWithNoTasks(),
  mockbosh.Deploy().WithManifest(manifestForFirstDeployment).RedirectsToTask(taskID),
)
```

Verify after test runs
``` go
AfterEach(func() {
  boshDirector.VerifyMocks()
})
```

