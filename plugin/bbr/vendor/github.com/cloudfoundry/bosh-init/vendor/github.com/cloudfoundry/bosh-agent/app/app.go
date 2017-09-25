package app

import (
	"fmt"
	"net"
	"path/filepath"
	"time"

	"github.com/pivotal-golang/clock"

	boshagent "github.com/cloudfoundry/bosh-agent/agent"
	boshaction "github.com/cloudfoundry/bosh-agent/agent/action"
	boshapplier "github.com/cloudfoundry/bosh-agent/agent/applier"
	boshas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
	boshbc "github.com/cloudfoundry/bosh-agent/agent/applier/bundlecollection"
	boshaj "github.com/cloudfoundry/bosh-agent/agent/applier/jobs"
	boshap "github.com/cloudfoundry/bosh-agent/agent/applier/packages"
	boshrunner "github.com/cloudfoundry/bosh-agent/agent/cmdrunner"
	boshcomp "github.com/cloudfoundry/bosh-agent/agent/compiler"
	boshscript "github.com/cloudfoundry/bosh-agent/agent/script"
	boshtask "github.com/cloudfoundry/bosh-agent/agent/task"
	boshinf "github.com/cloudfoundry/bosh-agent/infrastructure"
	boshjobsuper "github.com/cloudfoundry/bosh-agent/jobsupervisor"
	boshmonit "github.com/cloudfoundry/bosh-agent/jobsupervisor/monit"
	boshmbus "github.com/cloudfoundry/bosh-agent/mbus"
	boshnotif "github.com/cloudfoundry/bosh-agent/notification"
	boshplatform "github.com/cloudfoundry/bosh-agent/platform"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshdirs "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshsigar "github.com/cloudfoundry/bosh-agent/sigar"
	boshsyslog "github.com/cloudfoundry/bosh-agent/syslog"
	boshblob "github.com/cloudfoundry/bosh-utils/blobstore"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
	sigar "github.com/cloudfoundry/gosigar"
)

type App interface {
	Setup(args []string) error
	Run() error
	GetPlatform() boshplatform.Platform
}

type app struct {
	logger      boshlog.Logger
	agent       boshagent.Agent
	platform    boshplatform.Platform
	fs          boshsys.FileSystem
	logTag      string
	dirProvider boshdirs.Provider
}

func New(logger boshlog.Logger, fs boshsys.FileSystem) App {
	return &app{
		logger: logger,
		fs:     fs,
		logTag: "App",
	}
}

func (app *app) Setup(args []string) error {
	opts, err := ParseOptions(args)
	if err != nil {
		return bosherr.WrapError(err, "Parsing options")
	}

	config, err := app.loadConfig(opts.ConfigPath)
	if err != nil {
		return bosherr.WrapError(err, "Loading config")
	}

	app.dirProvider = boshdirs.NewProvider(opts.BaseDirectory)
	app.logStemcellInfo()

	statsCollector := boshsigar.NewSigarStatsCollector(&sigar.ConcreteSigar{})

	state, err := boshplatform.NewBootstrapState(app.fs, filepath.Join(app.dirProvider.BoshDir(), "agent_state.json"))
	if err != nil {
		return bosherr.WrapError(err, "Loading state")
	}

	timeService := clock.NewClock()
	platformProvider := boshplatform.NewProvider(app.logger, app.dirProvider, statsCollector, app.fs, config.Platform, state, timeService)

	app.platform, err = platformProvider.Get(opts.PlatformName)
	if err != nil {
		return bosherr.WrapError(err, "Getting platform")
	}

	settingsSourceFactory := boshinf.NewSettingsSourceFactory(config.Infrastructure.Settings, app.platform, app.logger)
	settingsSource, err := settingsSourceFactory.New()
	if err != nil {
		return bosherr.WrapError(err, "Getting Settings Source")
	}

	settingsService := boshsettings.NewService(
		app.platform.GetFs(),
		filepath.Join(app.dirProvider.BoshDir(), "settings.json"),
		settingsSource,
		app.platform,
		app.logger,
	)
	boot := boshagent.NewBootstrap(
		app.platform,
		app.dirProvider,
		settingsService,
		app.logger,
	)

	if err = boot.Run(); err != nil {
		return bosherr.WrapError(err, "Running bootstrap")
	}

	mbusHandlerProvider := boshmbus.NewHandlerProvider(settingsService, app.logger)

	mbusHandler, err := mbusHandlerProvider.Get(app.platform, app.dirProvider)
	if err != nil {
		return bosherr.WrapError(err, "Getting mbus handler")
	}

	blobstoreProvider := boshblob.NewProvider(app.platform.GetFs(), app.platform.GetRunner(), app.dirProvider.EtcDir(), app.logger)

	blobsettings := settingsService.GetSettings().Blobstore
	blobstore, err := blobstoreProvider.Get(blobsettings.Type, blobsettings.Options)
	if err != nil {
		return bosherr.WrapError(err, "Getting blobstore")
	}

	monitClientProvider := boshmonit.NewProvider(app.platform, app.logger)

	monitClient, err := monitClientProvider.Get()
	if err != nil {
		return bosherr.WrapError(err, "Getting monit client")
	}

	jobSupervisorProvider := boshjobsuper.NewProvider(
		app.platform,
		monitClient,
		app.logger,
		app.dirProvider,
		mbusHandler,
	)

	jobSupervisor, err := jobSupervisorProvider.Get(opts.JobSupervisor)
	if err != nil {
		return bosherr.WrapError(err, "Getting job supervisor")
	}

	notifier := boshnotif.NewNotifier(mbusHandler)

	applier, compiler := app.buildApplierAndCompiler(app.dirProvider, blobstore, jobSupervisor)

	uuidGen := boshuuid.NewGenerator()

	taskService := boshtask.NewAsyncTaskService(uuidGen, app.logger)

	taskManager := boshtask.NewManagerProvider().NewManager(
		app.logger,
		app.platform.GetFs(),
		app.dirProvider.BoshDir(),
	)

	specFilePath := filepath.Join(app.dirProvider.BoshDir(), "spec.json")
	specService := boshas.NewConcreteV1Service(
		app.platform.GetFs(),
		specFilePath,
	)

	jobScriptProvider := boshscript.NewConcreteJobScriptProvider(
		app.platform.GetRunner(),
		app.platform.GetFs(),
		app.platform.GetDirProvider(),
		timeService,
		app.logger,
	)

	actionFactory := boshaction.NewFactory(
		settingsService,
		app.platform,
		blobstore,
		taskService,
		notifier,
		applier,
		compiler,
		jobSupervisor,
		specService,
		jobScriptProvider,
		app.logger,
	)

	actionRunner := boshaction.NewRunner()

	actionDispatcher := boshagent.NewActionDispatcher(
		app.logger,
		taskService,
		taskManager,
		actionFactory,
		actionRunner,
	)

	syslogServer := boshsyslog.NewServer(33331, net.Listen, app.logger)

	app.agent = boshagent.New(
		app.logger,
		mbusHandler,
		app.platform,
		actionDispatcher,
		jobSupervisor,
		specService,
		syslogServer,
		time.Minute,
		settingsService,
		uuidGen,
		timeService,
	)

	return nil
}

func (app *app) Run() error {
	err := app.agent.Run()
	if err != nil {
		return bosherr.WrapError(err, "Running agent")
	}
	return nil
}

func (app *app) GetPlatform() boshplatform.Platform {
	return app.platform
}

func (app *app) buildApplierAndCompiler(
	dirProvider boshdirs.Provider,
	blobstore boshblob.Blobstore,
	jobSupervisor boshjobsuper.JobSupervisor,
) (boshapplier.Applier, boshcomp.Compiler) {
	jobsBc := boshbc.NewFileBundleCollection(
		dirProvider.DataDir(),
		dirProvider.BaseDir(),
		"jobs",
		app.platform.GetFs(),
		app.logger,
	)

	packageApplierProvider := boshap.NewCompiledPackageApplierProvider(
		dirProvider.DataDir(),
		dirProvider.BaseDir(),
		dirProvider.JobsDir(),
		"packages",
		blobstore,
		app.platform.GetCompressor(),
		app.platform.GetFs(),
		app.logger,
	)

	jobApplier := boshaj.NewRenderedJobApplier(
		jobsBc,
		jobSupervisor,
		packageApplierProvider,
		blobstore,
		app.platform.GetCompressor(),
		app.platform.GetFs(),
		app.logger,
	)

	applier := boshapplier.NewConcreteApplier(
		jobApplier,
		packageApplierProvider.Root(),
		app.platform,
		jobSupervisor,
		dirProvider,
	)

	platformRunner := app.platform.GetRunner()
	fileSystem := app.platform.GetFs()
	cmdRunner := boshrunner.NewFileLoggingCmdRunner(
		fileSystem,
		platformRunner,
		dirProvider.LogsDir(),
		10*1024, // 10 Kb
	)

	compiler := boshcomp.NewConcreteCompiler(
		app.platform.GetCompressor(),
		blobstore,
		fileSystem,
		cmdRunner,
		dirProvider,
		packageApplierProvider.Root(),
		packageApplierProvider.RootBundleCollection(),
	)

	return applier, compiler
}

func (app *app) loadConfig(path string) (Config, error) {
	// Use one off copy of file system to read configuration file
	fs := boshsys.NewOsFileSystem(app.logger)
	return LoadConfigFromPath(fs, path)
}

func (app *app) logStemcellInfo() {
	stemcellVersionFilePath := filepath.Join(app.dirProvider.EtcDir(), "stemcell_version")
	stemcellVersion := app.fileContents(stemcellVersionFilePath)
	stemcellSha1 := app.fileContents(filepath.Join(app.dirProvider.EtcDir(), "stemcell_git_sha1"))
	msg := fmt.Sprintf("Running on stemcell version '%s' (git: %s)", stemcellVersion, stemcellSha1)
	app.logger.Info(app.logTag, msg)
}

func (app *app) fileContents(path string) string {
	contents, err := app.fs.ReadFileString(path)
	if err != nil || len(contents) == 0 {
		contents = "?"
	}
	return contents
}
