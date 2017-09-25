package task_test

import (
	"bytes"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/ui"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshuit "github.com/cloudfoundry/bosh-cli/ui/task"
)

var _ = Describe("Reporter (not for events)", func() {
	var (
		outBuf, errBuf               *bytes.Buffer
		fakeUI                       *fakeui.FakeUI
		reporter, reporterWithFakeUI boshuit.Reporter
	)

	BeforeEach(func() {
		outBuf = bytes.NewBufferString("")
		errBuf = bytes.NewBufferString("")
		logger := boshlog.NewLogger(boshlog.LevelNone)

		ui := NewPaddingUI(NewWriterUI(outBuf, errBuf, logger))
		reporter = boshuit.NewReporter(ui, false)

		fakeUI = &fakeui.FakeUI{}
		reporterWithFakeUI = boshuit.NewReporter(fakeUI, false)
	})

	Describe("TaskStarted/TaskFinished/TaskOutputChunk", func() {
		It("prints task ending on the same line as beginning", func() {
			reporter.TaskStarted(123)
			reporter.TaskFinished(123, "state")
			Expect(outBuf.String()).To(Equal("Task 123. State\n"))
		})

		It("prints task ending on a new line if there was any task output", func() {
			reporter.TaskStarted(123)
			reporter.TaskOutputChunk(123, []byte("chunk\n"))
			reporter.TaskFinished(123, "state")
			Expect(outBuf.String()).To(Equal("Task 123\n\nchunk\n\nTask 123 state\n"))
		})

		It("only prints task output as a block", func() {
			reporterWithFakeUI.TaskStarted(123)
			reporterWithFakeUI.TaskOutputChunk(123, []byte("chunk\n"))
			reporterWithFakeUI.TaskFinished(123, "state")
			Expect(fakeUI.Blocks).To(Equal([]string{"chunk\n"}))
		})
	})
})

var _ = Describe("Reporter (for events)", func() {
	var (
		outBuf, errBuf               *bytes.Buffer
		fakeUI                       *fakeui.FakeUI
		reporter, reporterWithFakeUI boshuit.Reporter
	)

	BeforeEach(func() {
		outBuf = bytes.NewBufferString("")
		errBuf = bytes.NewBufferString("")
		logger := boshlog.NewLogger(boshlog.LevelNone)

		ui := NewPaddingUI(NewWriterUI(outBuf, errBuf, logger))
		reporter = boshuit.NewReporter(ui, true)

		fakeUI = &fakeui.FakeUI{}
		reporterWithFakeUI = boshuit.NewReporter(fakeUI, true)
	})

	Describe("TaskStarted/TaskFinished/TaskOutputChunk", func() {
		It("prints task ending on the same line as beginning", func() {
			reporter.TaskStarted(123)
			reporter.TaskFinished(123, "state")
			Expect(outBuf.String()).To(Equal("Task 123. State\n"))
		})

		It("prints task ending on a new line if there was any task output", func() {
			reporter.TaskStarted(123)
			reporter.TaskOutputChunk(123, []byte("{}\n"))
			reporter.TaskFinished(123, "state")
			Expect(outBuf.String()).To(Equal(`Task 123


Started  Thu Jan  1 00:00:00 UTC 1970
Finished Thu Jan  1 00:00:00 UTC 1970
Duration 00:00:00

Task 123 state
`))
		})

		It("does not print empty events", func() {
			reporterWithFakeUI.TaskStarted(123)
			reporterWithFakeUI.TaskOutputChunk(123, []byte("{}\n"))
			reporterWithFakeUI.TaskFinished(123, "state")
			Expect(fakeUI.Blocks).To(BeNil())
		})

		It("panics if cannot unmarshal event chunk", func() {
			reporterWithFakeUI.TaskStarted(123)
			Expect(func() {
				reporterWithFakeUI.TaskOutputChunk(123, []byte("-\n"))
			}).To(Panic())
		})

		It("prints content as blocks", func() {
			reporterWithFakeUI.TaskStarted(123)
			reporterWithFakeUI.TaskOutputChunk(123, []byte(
				`{"time":1454193505,"error":{"code":100,"message":"err-msg"}}`+"\n"))
			reporterWithFakeUI.TaskFinished(123, "state")
			Expect(fakeUI.Blocks).To(Equal([]string{"\n22:38:25 | ", "Error: err-msg"}))
		})

		It("renders events", func() {
			deployExample := `
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding releases","index":1,"state":"started","progress":0}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding releases","index":1,"state":"finished","progress":100}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding existing deployment","index":2,"state":"started","progress":0}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding existing deployment","index":2,"state":"finished","progress":100}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding resource pools","index":3,"state":"started","progress":0}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding resource pools","index":3,"state":"finished","progress":100}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding stemcells","index":4,"state":"started","progress":0}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding stemcells","index":4,"state":"finished","progress":100}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding templates","index":5,"state":"started","progress":0}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding templates","index":5,"state":"finished","progress":100}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding properties","index":6,"state":"started","progress":0}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding properties","index":6,"state":"finished","progress":100}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding unallocated VMs","index":7,"state":"started","progress":0}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding unallocated VMs","index":7,"state":"finished","progress":100}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding instance networks","index":8,"state":"started","progress":0}
{"time":7414830567,"stage":"Preparing deployment","tags":[],"total":9,"task":"Binding instance networks","index":8,"state":"finished","progress":100}
{"time":7414830567,"stage":"Preparing package compilation","tags":[],"total":1,"task":"Finding packages to compile","index":1,"state":"started","progress":0}
{"time":7414830567,"stage":"Preparing package compilation","tags":[],"total":1,"task":"Finding packages to compile","index":1,"state":"finished","progress":100}
{"time":7414830567,"stage":"Preparing DNS","tags":[],"total":1,"task":"Binding DNS","index":1,"state":"started","progress":0}
{"time":7414830567,"stage":"Preparing DNS","tags":[],"total":1,"task":"Binding DNS","index":1,"state":"finished","progress":100}
{"time":7414830567,"stage":"Preparing configuration","tags":[],"total":1,"task":"Binding configuration","index":1,"state":"started","progress":0}
{"time":7414830568,"stage":"Preparing configuration","tags":[],"total":1,"task":"Binding configuration","index":1,"state":"finished","progress":100}
{"time":7414830568,"stage":"Updating job","tags":["job"],"total":1,"task":"job/0 (canary)","index":1,"state":"started","progress":0}
{"time":7414830569,"stage":"Updating job","tags":["job"],"total":1,"task":"job/0 (canary)","index":1,"state":"in_progress","progress":8}
{"time":7414830574,"stage":"Updating job","tags":["job"],"total":1,"task":"job/0 (canary)","index":1,"state":"in_progress","progress":16}
{"time":7414830574,"stage":"Updating job","tags":["job"],"total":1,"task":"job/0 (canary)","index":1,"state":"in_progress","progress":25}
{"time":7414830574,"stage":"Doing something else","tags":["job"],"total":1,"task":"job/1 (canary)","index":1,"state":"in_progress","progress":0}
{"time":7414830574,"stage":"Doing something else","tags":["job"],"total":1,"task":"job/1 (canary)","index":1,"state":"in_progress","progress":10}
{"time":7414830574,"stage":"Doing something else","tags":["job"],"total":1,"task":"job/1 (canary)","index":1,"state":"in_progress","progress":100}
{"time":7414830574,"stage":"Updating job","tags":["job"],"total":1,"task":"job/0 (canary)","index":1,"state":"in_progress","progress":33}
{"time":7414830574,"stage":"Updating job","tags":["job"],"total":1,"task":"job/0 (canary)","index":1,"state":"in_progress","progress":41}
{"time":7414830574,"stage":"Updating job","tags":["job"],"total":1,"task":"job/0 (canary)","index":1,"state":"in_progress","progress":50}
{"time":7414830574,"stage":"Updating job","tags":["job"],"total":1,"task":"job/0 (canary)","index":1,"state":"in_progress","progress":58}
{"time":7414830574,"stage":"Updating job","tags":["job"],"total":1,"task":"job/0 (canary)","index":1,"state":"in_progress","progress":66}
{"time":7414830600,"stage":"Updating job","tags":["job"],"total":1,"task":"job/0 (canary)","index":1,"state":"in_progress","progress":75}
{"time":7414830600,"stage":"Updating job","tags":["job"],"total":1,"task":"job/0 (canary)","index":1,"state":"in_progress","progress":83}
{"time":7414830605,"stage":"Updating job","tags":["job"],"total":1,"task":"job/0 (canary)","index":1,"state":"in_progress","progress":91}
{"time":7414830635,"stage":"Updating job","tags":["job"],"total":1,"task":"job/0 (canary)","index":1,"state":"in_progress","progress":99}
{"time":7414830635,"stage":"Updating job","tags":["job"],"total":1,"task":"job/0 (canary)","index":1,"state":"failed","progress":100,"data":{"error":"'job/0' is not running after update"}}
{"time":7414830635,"error":{"code":400007,"message":"'job/0' is not running after update"}}
`

			reporter.TaskStarted(2663)
			reporter.TaskOutputChunk(2663, []byte(deployExample))
			reporter.TaskFinished(2663, "error")
			Expect(outBuf.String()).To(Equal(`Task 2663

19:09:27 | Preparing deployment: Binding releases (00:00:00)
19:09:27 | Preparing deployment: Binding existing deployment (00:00:00)
19:09:27 | Preparing deployment: Binding resource pools (00:00:00)
19:09:27 | Preparing deployment: Binding stemcells (00:00:00)
19:09:27 | Preparing deployment: Binding templates (00:00:00)
19:09:27 | Preparing deployment: Binding properties (00:00:00)
19:09:27 | Preparing deployment: Binding unallocated VMs (00:00:00)
19:09:27 | Preparing deployment: Binding instance networks (00:00:00)
19:09:27 | Preparing package compilation: Finding packages to compile (00:00:00)
19:09:27 | Preparing DNS: Binding DNS (00:00:00)
19:09:27 | Preparing configuration: Binding configuration (00:00:01)
19:09:28 | Updating job job: job/0 (canary) (00:01:07)
            L Error: 'job/0' is not running after update

19:10:35 | Error: 'job/0' is not running after update

Started  Wed Dec 19 19:09:27 UTC 2204
Finished Wed Dec 19 19:10:35 UTC 2204
Duration 00:01:08

Task 2663 error
`))
		})

		It("renders error events", func() {
			deployExample := `
{"time":1468888884,"type":"deprecation","message":"Ignoring cloud config. Manifest contains 'networks' section."}
{"time":1468888884,"error":{"code":100,"message":"Failed to find keys in the config server: bool, bool2"}}
{"time":1468888884,"error":{"code":100,"message":"Failed to wang chung tonite"}}
`

			reporter.TaskStarted(2663)
			reporter.TaskOutputChunk(2663, []byte(deployExample))
			reporter.TaskFinished(2663, "error")
			Expect(outBuf.String()).To(Equal(`Task 2663

00:41:24 | Deprecation: Ignoring cloud config. Manifest contains 'networks' section.

00:41:24 | Error: Failed to find keys in the config server: bool, bool2

00:41:24 | Error: Failed to wang chung tonite

Started  Tue Jul 19 00:41:24 UTC 2016
Finished Tue Jul 19 00:41:24 UTC 2016
Duration 00:00:00

Task 2663 error
`))
		})

		It("renders warning events", func() {
			deployExample := `
{"time":1478564798,"stage":"Preparing deployment","tags":[],"total":1,"task":"Preparing deployment","index":1,"state":"started","progress":0}
{"time":1478564798,"stage":"Preparing deployment","tags":[],"total":1,"task":"Preparing deployment","index":1,"state":"finished","progress":100}
{"time":1478564798,"type":"warning","message":"You have ignored instances. They will not be changed."}
{"time":1478564798,"stage":"Preparing package compilation","tags":[],"total":1,"task":"Finding packages to compile","index":1,"state":"started","progress":0}
{"time":1478564798,"stage":"Preparing package compilation","tags":[],"total":1,"task":"Finding packages to compile","index":1,"state":"finished","progress":100}
`

			reporter.TaskStarted(2663)
			reporter.TaskOutputChunk(2663, []byte(deployExample))
			reporter.TaskFinished(2663, "error")
			Expect(outBuf.String()).To(Equal(`Task 2663

00:26:38 | Preparing deployment: Preparing deployment (00:00:00)
00:26:38 | Warning: You have ignored instances. They will not be changed.
00:26:38 | Preparing package compilation: Finding packages to compile (00:00:00)

Started  Tue Nov  8 00:26:38 UTC 2016
Finished Tue Nov  8 00:26:38 UTC 2016
Duration 00:00:00

Task 2663 error
`))
		})
	})
})
