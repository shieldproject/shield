package action

import (
	boshappl "github.com/cloudfoundry/bosh-agent/agent/applier"
	boshas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
	boshcomp "github.com/cloudfoundry/bosh-agent/agent/compiler"
	boshscript "github.com/cloudfoundry/bosh-agent/agent/script"
	boshtask "github.com/cloudfoundry/bosh-agent/agent/task"
	boshjobsuper "github.com/cloudfoundry/bosh-agent/jobsupervisor"
	boshnotif "github.com/cloudfoundry/bosh-agent/notification"
	boshplatform "github.com/cloudfoundry/bosh-agent/platform"
	boshntp "github.com/cloudfoundry/bosh-agent/platform/ntp"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshblob "github.com/cloudfoundry/bosh-utils/blobstore"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type concreteFactory struct {
	availableActions map[string]Action
}

func NewFactory(
	settingsService boshsettings.Service,
	platform boshplatform.Platform,
	blobstore boshblob.Blobstore,
	taskService boshtask.Service,
	notifier boshnotif.Notifier,
	applier boshappl.Applier,
	compiler boshcomp.Compiler,
	jobSupervisor boshjobsuper.JobSupervisor,
	specService boshas.V1Service,
	jobScriptProvider boshscript.JobScriptProvider,
	logger boshlog.Logger,
) (factory Factory) {
	compressor := platform.GetCompressor()
	copier := platform.GetCopier()
	dirProvider := platform.GetDirProvider()
	vitalsService := platform.GetVitalsService()
	certManager := platform.GetCertManager()
	ntpService := boshntp.NewConcreteService(platform.GetFs(), dirProvider)

	factory = concreteFactory{
		availableActions: map[string]Action{
			// Task management
			"ping":        NewPing(),
			"get_task":    NewGetTask(taskService),
			"cancel_task": NewCancelTask(taskService),

			// VM admin
			"ssh":             NewSSH(settingsService, platform, dirProvider, logger),
			"fetch_logs":      NewFetchLogs(compressor, copier, blobstore, dirProvider),
			"update_settings": NewUpdateSettings(certManager, logger),

			// Job management
			"prepare":    NewPrepare(applier),
			"apply":      NewApply(applier, specService, settingsService, dirProvider.InstanceDir(), platform.GetFs()),
			"start":      NewStart(jobSupervisor, applier, specService),
			"stop":       NewStop(jobSupervisor),
			"drain":      NewDrain(notifier, specService, jobScriptProvider, jobSupervisor, logger),
			"get_state":  NewGetState(settingsService, specService, jobSupervisor, vitalsService, ntpService),
			"run_errand": NewRunErrand(specService, dirProvider.JobsDir(), platform.GetRunner(), logger),
			"run_script": NewRunScript(jobScriptProvider, specService, logger),

			// Compilation
			"compile_package":    NewCompilePackage(compiler),
			"release_apply_spec": NewReleaseApplySpec(platform),

			// Disk management
			"list_disk":    NewListDisk(settingsService, platform, logger),
			"migrate_disk": NewMigrateDisk(platform, dirProvider),
			"mount_disk":   NewMountDisk(settingsService, platform, dirProvider, logger),
			"unmount_disk": NewUnmountDisk(settingsService, platform),

			// ARP cache management
			"delete_arp_entries": NewDeleteARPEntries(platform),

			// Networking
			"prepare_network_change":     NewPrepareNetworkChange(platform.GetFs(), settingsService, NewAgentKiller()),
			"prepare_configure_networks": NewPrepareConfigureNetworks(platform, settingsService),
			"configure_networks":         NewConfigureNetworks(NewAgentKiller()),

			// DNS
			"sync_dns": NewSyncDNS(blobstore, settingsService, platform, logger),
		},
	}
	return
}

func (f concreteFactory) Create(method string) (Action, error) {
	action, found := f.availableActions[method]
	if !found {
		return nil, bosherr.Errorf("Could not create action with method %s", method)
	}

	return action, nil
}
