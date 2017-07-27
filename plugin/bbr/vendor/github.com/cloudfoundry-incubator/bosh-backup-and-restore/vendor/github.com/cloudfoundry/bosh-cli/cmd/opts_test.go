package cmd_test

import (
	"fmt"
	"os"
	"reflect"
	"regexp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var (
	dupSpaces = regexp.MustCompile("\\s{2,}")
)

func getStructTagForName(field string, opts interface{}) string {
	st, _ := reflect.TypeOf(opts).Elem().FieldByName(field)
	return dupSpaces.ReplaceAllString(string(st.Tag), " ")
}

var _ = Describe("Opts", func() {
	Describe("BoshOpts", func() {
		var opts *BoshOpts

		BeforeEach(func() {
			opts = &BoshOpts{}
		})

		It("long or short command options do not shadow global opts", func() {
			globalLongOptNames := []string{}
			globalShortOptNames := []string{}
			cmdOptss := []reflect.Value{}

			optsStruct := reflect.TypeOf(opts).Elem()

			for i := 0; i < optsStruct.NumField(); i++ {
				field := optsStruct.Field(i)
				if field.Tag.Get("long") != "" {
					globalLongOptNames = append(globalLongOptNames, field.Tag.Get("long"))
				}
				if field.Tag.Get("short") != "" {
					globalShortOptNames = append(globalShortOptNames, field.Tag.Get("short"))
				}
				if field.Tag.Get("command") != "" {
					m := reflect.ValueOf(opts).Elem().Field(i).Addr()
					cmdOptss = append(cmdOptss, m)
				}
			}

			var errs []string

			for _, optName := range globalLongOptNames {
				for _, cmdOpts := range cmdOptss {
					cmdOptsStruct := reflect.TypeOf(cmdOpts.Interface()).Elem()

					for i := 0; i < cmdOptsStruct.NumField(); i++ {
						field := cmdOptsStruct.Field(i)

						if field.Tag.Get("long") == optName {
							errs = append(errs, fmt.Sprintf("Command '%s' shadows global long option '%s'", cmdOptsStruct.Name(), optName))
						}
					}
				}
			}

			for _, optName := range globalShortOptNames {
				for _, cmdOpts := range cmdOptss {
					cmdOptsStruct := reflect.TypeOf(cmdOpts.Interface()).Elem()

					for i := 0; i < cmdOptsStruct.NumField(); i++ {
						field := cmdOptsStruct.Field(i)

						if field.Tag.Get("short") == optName {
							errs = append(errs, fmt.Sprintf("Command '%s' shadows global short option '%s'", cmdOptsStruct.Name(), optName))
						}
					}
				}
			}

			// --version flag is a bit awkward so let's ignore conflicts
			Expect(errs).To(Equal([]string{
				"Command 'UploadStemcellOpts' shadows global long option 'version'",
				"Command 'RepackStemcellOpts' shadows global long option 'version'",
				"Command 'UploadReleaseOpts' shadows global long option 'version'",
				"Command 'CreateReleaseOpts' shadows global long option 'version'",
				"Command 'FinalizeReleaseOpts' shadows global long option 'version'",
			}))
		})

		Describe("VersionOpt", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("VersionOpt", opts)).To(Equal(
					`long:"version" short:"v" description:"Show CLI version"`,
				))
			})
		})

		Describe("ConfigPathOpt", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("ConfigPathOpt", opts)).To(Equal(
					`long:"config" description:"Config file path" env:"BOSH_CONFIG" default:"~/.bosh/config"`,
				))
			})
		})

		Describe("EnvironmentOpt", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("EnvironmentOpt", opts)).To(Equal(
					`long:"environment" short:"e" description:"Director environment name or URL" env:"BOSH_ENVIRONMENT"`,
				))
			})
		})

		Describe("Sha2", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Sha2", opts)).To(Equal(
					`long:"sha2" description:"Use SHA256 checksums"`,
				))
			})
		})

		Describe("CACertOpt", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("CACertOpt", opts)).To(Equal(
					`long:"ca-cert" description:"Director CA certificate path or value" env:"BOSH_CA_CERT"`,
				))
			})
		})

		Describe("UsernameOpt", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("UsernameOpt", opts)).To(Equal(
					`long:"user" hidden:"true" env:"BOSH_USER"`,
				))
			})
		})

		Describe("ClientOpt", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("ClientOpt", opts)).To(Equal(
					`long:"client" description:"Override username or UAA client" env:"BOSH_CLIENT"`,
				))
			})
		})

		Describe("ClientSecretOpt", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("ClientSecretOpt", opts)).To(Equal(
					`long:"client-secret" description:"Override password or UAA client secret" env:"BOSH_CLIENT_SECRET"`,
				))
			})
		})

		Describe("DeploymentOpt", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("DeploymentOpt", opts)).To(Equal(
					`long:"deployment" short:"d" description:"Deployment name" env:"BOSH_DEPLOYMENT"`,
				))
			})
		})

		Describe("JSONOpt", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("JSONOpt", opts)).To(Equal(
					`long:"json" description:"Output as JSON"`,
				))
			})
		})

		Describe("TTYOpt", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("TTYOpt", opts)).To(Equal(
					`long:"tty" description:"Force TTY-like output"`,
				))
			})
		})

		Describe("NoColorOpt", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("NoColorOpt", opts)).To(Equal(
					`long:"no-color" description:"Toggle colorized output"`,
				))
			})
		})

		Describe("NonInteractiveOpt", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("NonInteractiveOpt", opts)).To(Equal(
					`long:"non-interactive" short:"n" description:"Don't ask for user input" env:"BOSH_NON_INTERACTIVE"`,
				))
			})
		})

		Describe("CreateEnv", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("CreateEnv", opts)).To(Equal(
					`command:"create-env" description:"Create or update BOSH environment"`,
				))
			})
		})

		Describe("DeleteEnv", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("DeleteEnv", opts)).To(Equal(
					`command:"delete-env" description:"Delete BOSH environment"`,
				))
			})
		})

		Describe("Environment", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Environment", opts)).To(Equal(
					`command:"environment" alias:"env" description:"Show environment"`,
				))
			})
		})

		Describe("Environments", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Environments", opts)).To(Equal(
					`command:"environments" alias:"envs" description:"List environments"`,
				))
			})
		})

		Describe("LogIn", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("LogIn", opts)).To(Equal(
					`command:"log-in" alias:"l" alias:"login" description:"Log in"`,
				))
			})
		})

		Describe("LogOut", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("LogOut", opts)).To(Equal(
					`command:"log-out" alias:"logout" description:"Log out"`,
				))
			})
		})

		Describe("Task", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Task", opts)).To(Equal(
					`command:"task" alias:"t" description:"Show task status and start tracking its output"`,
				))
			})
		})

		Describe("Tasks", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Tasks", opts)).To(Equal(
					`command:"tasks" alias:"ts" description:"List running or recent tasks"`,
				))
			})
		})

		Describe("CancelTask", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("CancelTask", opts)).To(Equal(
					`command:"cancel-task" alias:"ct" description:"Cancel task at its next checkpoint"`,
				))
			})
		})

		Describe("Locks", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Locks", opts)).To(Equal(
					`command:"locks" description:"List current locks"`,
				))
			})
		})

		Describe("CleanUp", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("CleanUp", opts)).To(Equal(
					`command:"clean-up" description:"Clean up releases, stemcells, disks, etc."`,
				))
			})
		})

		Describe("Interpolate", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Interpolate", opts)).To(Equal(
					`command:"interpolate" alias:"int" description:"Interpolates variables into a manifest"`,
				))
			})
		})

		Describe("CloudConfig", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("CloudConfig", opts)).To(Equal(
					`command:"cloud-config" alias:"cc" description:"Show current cloud config"`,
				))
			})
		})

		Describe("UpdateCloudConfig", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("UpdateCloudConfig", opts)).To(Equal(
					`command:"update-cloud-config" alias:"ucc" description:"Update current cloud config"`,
				))
			})
		})

		Describe("CPIConfig", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("CPIConfig", opts)).To(Equal(
					`command:"cpi-config" description:"Show current CPI config"`,
				))
			})
		})

		Describe("UpdateCPIConfig", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("UpdateCPIConfig", opts)).To(Equal(
					`command:"update-cpi-config" description:"Update current CPI config"`,
				))
			})
		})

		Describe("RuntimeConfig", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("RuntimeConfig", opts)).To(Equal(
					`command:"runtime-config" alias:"rc" description:"Show current runtime config"`,
				))
			})
		})

		Describe("UpdateRuntimeConfig", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("UpdateRuntimeConfig", opts)).To(Equal(
					`command:"update-runtime-config" alias:"urc" description:"Update current runtime config"`,
				))
			})
		})

		Describe("Deployment", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Deployment", opts)).To(Equal(
					`command:"deployment" alias:"dep" description:"Show deployment information"`,
				))
			})
		})

		Describe("Deployments", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Deployments", opts)).To(Equal(
					`command:"deployments" alias:"ds" alias:"deps" description:"List deployments"`,
				))
			})
		})

		Describe("DeleteDeployment", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("DeleteDeployment", opts)).To(Equal(
					`command:"delete-deployment" alias:"deld" description:"Delete deployment"`,
				))
			})
		})

		Describe("Deploy", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Deploy", opts)).To(Equal(
					`command:"deploy" alias:"d" description:"Update deployment"`,
				))
			})
		})

		Describe("Manifest", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Manifest", opts)).To(Equal(
					`command:"manifest" alias:"man" description:"Show deployment manifest"`,
				))
			})
		})

		Describe("Stemcells", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Stemcells", opts)).To(Equal(
					`command:"stemcells" alias:"ss" description:"List stemcells"`,
				))
			})
		})

		Describe("UploadStemcell", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("UploadStemcell", opts)).To(Equal(
					`command:"upload-stemcell" alias:"us" description:"Upload stemcell"`,
				))
			})
		})

		Describe("DeleteStemcell", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("DeleteStemcell", opts)).To(Equal(
					`command:"delete-stemcell" alias:"dels" description:"Delete stemcell"`,
				))
			})
		})

		Describe("RepackStemcell", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("RepackStemcell", opts)).To(Equal(
					`command:"repack-stemcell" description:"Repack stemcell"`,
				))
			})
		})

		Describe("Releases", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Releases", opts)).To(Equal(
					`command:"releases" alias:"rs" description:"List releases"`,
				))
			})
		})

		Describe("UploadRelease", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("UploadRelease", opts)).To(Equal(
					`command:"upload-release" alias:"ur" description:"Upload release"`,
				))
			})
		})

		Describe("ExportRelease", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("ExportRelease", opts)).To(Equal(
					`command:"export-release" description:"Export the compiled release to a tarball"`,
				))
			})
		})

		Describe("InspectRelease", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("InspectRelease", opts)).To(Equal(
					`command:"inspect-release" description:"List release contents such as jobs"`,
				))
			})
		})

		Describe("DeleteRelease", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("DeleteRelease", opts)).To(Equal(
					`command:"delete-release" alias:"delr" description:"Delete release"`,
				))
			})
		})

		Describe("Errands", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Errands", opts)).To(Equal(
					`command:"errands" alias:"es" description:"List errands"`,
				))
			})
		})

		Describe("RunErrand", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("RunErrand", opts)).To(Equal(
					`command:"run-errand" description:"Run errand"`,
				))
			})
		})

		Describe("Disks", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Disks", opts)).To(Equal(
					`command:"disks" description:"List disks"`,
				))
			})
		})

		Describe("DeleteDisk", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("DeleteDisk", opts)).To(Equal(
					`command:"delete-disk" description:"Delete disk"`,
				))
			})
		})

		Describe("Snapshots", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Snapshots", opts)).To(Equal(
					`command:"snapshots" description:"List snapshots"`,
				))
			})
		})

		Describe("TakeSnapshot", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("TakeSnapshot", opts)).To(Equal(
					`command:"take-snapshot" description:"Take snapshot"`,
				))
			})
		})

		Describe("DeleteSnapshot", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("DeleteSnapshot", opts)).To(Equal(
					`command:"delete-snapshot" description:"Delete snapshot"`,
				))
			})
		})

		Describe("DeleteSnapshots", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("DeleteSnapshots", opts)).To(Equal(
					`command:"delete-snapshots" description:"Delete all snapshots in a deployment"`,
				))
			})
		})

		Describe("DeleteVM", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("DeleteVM", opts)).To(Equal(
					`command:"delete-vm" description:"Delete VM"`,
				))
			})
		})

		Describe("Instances", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Instances", opts)).To(Equal(
					`command:"instances" alias:"is" description:"List all instances in a deployment"`,
				))
			})
		})

		Describe("VMs", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("VMs", opts)).To(Equal(
					`command:"vms" description:"List all VMs in all deployments"`,
				))
			})
		})

		Describe("UpdateResurrection", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("UpdateResurrection", opts)).To(Equal(
					`command:"update-resurrection" description:"Enable/disable resurrection"`,
				))
			})
		})

		Describe("Ignore", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Ignore", opts)).To(Equal(
					`command:"ignore" description:"Ignore an instance"`,
				))
			})
		})

		Describe("Unignore", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Unignore", opts)).To(Equal(
					`command:"unignore" description:"Unignore an instance"`,
				))
			})
		})

		Describe("CloudCheck", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("CloudCheck", opts)).To(Equal(
					`command:"cloud-check" alias:"cck" alias:"cloudcheck" description:"Cloud consistency check and interactive repair"`,
				))
			})
		})

		Describe("Logs", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Logs", opts)).To(Equal(
					`command:"logs" description:"Fetch logs from instance(s)"`,
				))
			})
		})

		Describe("Start", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Start", opts)).To(Equal(
					`command:"start" description:"Start instance(s)"`,
				))
			})
		})

		Describe("Stop", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Stop", opts)).To(Equal(
					`command:"stop" description:"Stop instance(s)"`,
				))
			})
		})

		Describe("Restart", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Restart", opts)).To(Equal(
					`command:"restart" description:"Restart instance(s)"`,
				))
			})
		})

		Describe("Recreate", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Recreate", opts)).To(Equal(
					`command:"recreate" description:"Recreate instance(s)"`,
				))
			})
		})

		Describe("SSH", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("SSH", opts)).To(Equal(
					`command:"ssh" description:"SSH into instance(s)"`,
				))
			})
		})

		Describe("SCP", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("SCP", opts)).To(Equal(
					`command:"scp" description:"SCP to/from instance(s)"`,
				))
			})
		})

		Describe("InitRelease", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("InitRelease", opts)).To(Equal(
					`command:"init-release" description:"Initialize release"`,
				))
			})
		})

		Describe("ResetRelease", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("ResetRelease", opts)).To(Equal(
					`command:"reset-release" description:"Reset release"`,
				))
			})
		})

		Describe("GenerateJob", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("GenerateJob", opts)).To(Equal(
					`command:"generate-job" description:"Generate job"`,
				))
			})
		})

		Describe("GeneratePackage", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("GeneratePackage", opts)).To(Equal(
					`command:"generate-package" description:"Generate package"`,
				))
			})
		})

		Describe("CreateRelease", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("CreateRelease", opts)).To(Equal(
					`command:"create-release" alias:"cr" description:"Create release"`,
				))
			})
		})

		Describe("Sha2ifyRelease", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Sha2ifyRelease", opts)).To(Equal(
					`command:"sha2ify-release" hidden:"true" description:"Convert release tarball to use SHA256"`,
				))
			})
		})

		Describe("FinalizeRelease", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("FinalizeRelease", opts)).To(Equal(
					`command:"finalize-release" description:"Create final release from dev release tarball"`,
				))
			})
		})

		Describe("Blobs", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Blobs", opts)).To(Equal(
					`command:"blobs" description:"List blobs"`,
				))
			})
		})

		Describe("AddBlob", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("AddBlob", opts)).To(Equal(
					`command:"add-blob" description:"Add blob"`,
				))
			})
		})

		Describe("RemoveBlob", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("RemoveBlob", opts)).To(Equal(
					`command:"remove-blob" description:"Remove blob"`,
				))
			})
		})

		Describe("SyncBlobs", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("SyncBlobs", opts)).To(Equal(
					`command:"sync-blobs" description:"Sync blobs"`,
				))
			})
		})

		Describe("UploadBlobs", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("UploadBlobs", opts)).To(Equal(
					`command:"upload-blobs" description:"Upload blobs"`,
				))
			})
		})

		Describe("AttachDisk", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("AttachDisk", opts)).To(Equal(
					`command:"attach-disk" description:"Attaches disk to an instance"`,
				))
			})
		})
	})

	Describe("CreateEnvOpts", func() {
		var opts *CreateEnvOpts

		BeforeEach(func() {
			opts = &CreateEnvOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		It("has --state", func() {
			Expect(getStructTagForName("StatePath", opts)).To(Equal(
				`long:"state" value-name:"PATH" description:"State file path"`,
			))
		})
	})

	Describe("CreateEnvArgs", func() {
		var args *CreateEnvArgs

		BeforeEach(func() {
			args = &CreateEnvArgs{}
		})

		Describe("Manifest", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Manifest", args)).To(Equal(
					`positional-arg-name:"PATH" description:"Path to a manifest file"`,
				))
			})
		})
	})

	Describe("DeleteEnvOpts", func() {
		var opts *DeleteEnvOpts

		BeforeEach(func() {
			opts = &DeleteEnvOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		It("has --state", func() {
			Expect(getStructTagForName("StatePath", opts)).To(Equal(
				`long:"state" value-name:"PATH" description:"State file path"`,
			))
		})
	})

	Describe("DeleteEnvArgs", func() {
		var args *DeleteEnvArgs

		BeforeEach(func() {
			args = &DeleteEnvArgs{}
		})

		Describe("Manifest", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Manifest", args)).To(Equal(
					`positional-arg-name:"PATH" description:"Path to a manifest file"`,
				))
			})
		})
	})

	Describe("AliasEnvOpts", func() {
		var opts *AliasEnvOpts

		BeforeEach(func() {
			opts = &AliasEnvOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})
	})

	Describe("AliasEnvArgs", func() {
		var args *AliasEnvArgs

		BeforeEach(func() {
			args = &AliasEnvArgs{}
		})

		Describe("Alias", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Alias", args)).To(Equal(
					`positional-arg-name:"ALIAS" description:"Environment alias"`,
				))
			})
		})
	})

	Describe("TaskOpts", func() {
		var opts *TaskOpts

		BeforeEach(func() {
			opts = &TaskOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true"`))
			})
		})

		Describe("Event", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Event", opts)).To(Equal(
					`long:"event" description:"Track event log"`,
				))
			})
		})

		Describe("CPI", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("CPI", opts)).To(Equal(
					`long:"cpi" description:"Track CPI log"`,
				))
			})
		})

		Describe("Debug", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Debug", opts)).To(Equal(
					`long:"debug" description:"Track debug log"`,
				))
			})
		})

		Describe("Result", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Result", opts)).To(Equal(
					`long:"result" description:"Track result log"`,
				))
			})
		})

		Describe("All", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("All", opts)).To(Equal(
					`long:"all" short:"a" description:"Include all task types (ssh, logs, vms, etc)"`,
				))
			})
		})
	})

	Describe("TaskArgs", func() {
		var opts *TaskArgs

		BeforeEach(func() {
			opts = &TaskArgs{}
		})

		Describe("ID", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("ID", opts)).To(Equal(
					`positional-arg-name:"ID"`,
				))
			})
		})
	})

	Describe("TasksOpts", func() {
		var opts *TasksOpts

		BeforeEach(func() {
			opts = &TasksOpts{}
		})

		Describe("Recent", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Recent", opts)).To(Equal(
					`long:"recent" short:"r" description:"Number of tasks to show" optional:"true" optional-value:"30"`,
				))
			})
		})

		Describe("All", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("All", opts)).To(Equal(
					`long:"all" short:"a" description:"Include all task types (ssh, logs, vms, etc)"`,
				))
			})
		})
	})

	Describe("CancelTaskOpts", func() {
		var opts *CancelTaskOpts

		BeforeEach(func() {
			opts = &CancelTaskOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})
	})

	Describe("CleanUpOpts", func() {
		var opts *CleanUpOpts

		BeforeEach(func() {
			opts = &CleanUpOpts{}
		})

		Describe("All", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("All", opts)).To(Equal(
					`long:"all" description:"Remove all unused releases, stemcells, etc.; otherwise most recent resources will be kept"`,
				))
			})
		})
	})

	Describe("AttachDiskOpts", func() {
		var opts *AttachDiskOpts

		BeforeEach(func() {
			opts = &AttachDiskOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})
	})

	Describe("AttachDiskArgs", func() {
		var opts *AttachDiskArgs

		BeforeEach(func() {
			opts = &AttachDiskArgs{}
		})

		Describe("DiskId", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Slug", opts)).To(Equal(`positional-arg-name:"INSTANCE-GROUP/INSTANCE-ID"`))
				Expect(getStructTagForName("DiskCID", opts)).To(Equal(`positional-arg-name:"DISK-CID"`))
			})
		})
	})

	Describe("InterpolateOpts", func() {
		var opts InterpolateOpts

		It("has Args", func() {
			Expect(getStructTagForName("Args", &opts)).To(Equal(`positional-args:"true" required:"true"`))
		})

		It("has Path", func() {
			Expect(getStructTagForName("Path", &opts)).To(Equal(
				`long:"path" value-name:"OP-PATH" description:"Extract value out of template (e.g.: /private_key)"`,
			))
		})

		It("has VarErrors", func() {
			Expect(getStructTagForName("VarErrors", &opts)).To(Equal(
				`long:"var-errs" description:"Expect all variables to be found, otherwise error"`,
			))
		})

		It("has VarErrorsUnused", func() {
			Expect(getStructTagForName("VarErrorsUnused", &opts)).To(Equal(
				`long:"var-errs-unused" description:"Expect all variables to be used, otherwise error"`,
			))
		})
	})

	Describe("InterpolateArgs", func() {
		var opts *InterpolateArgs

		BeforeEach(func() {
			opts = &InterpolateArgs{}
		})

		Describe("Manifest", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Manifest", opts)).To(Equal(
					`positional-arg-name:"PATH" description:"Path to a template that will be interpolated"`,
				))
			})
		})
	})

	Describe("UpdateCloudConfigOpts", func() {
		var opts *UpdateCloudConfigOpts

		BeforeEach(func() {
			opts = &UpdateCloudConfigOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})
	})

	Describe("UpdateCloudConfigArgs", func() {
		var opts *UpdateCloudConfigArgs

		BeforeEach(func() {
			opts = &UpdateCloudConfigArgs{}
		})

		Describe("CloudConfig", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("CloudConfig", opts)).To(Equal(
					`positional-arg-name:"PATH" description:"Path to a cloud config file"`,
				))
			})
		})
	})

	Describe("RuntimeConfigOpts", func() {
		var opts *RuntimeConfigOpts

		BeforeEach(func() {
			opts = &RuntimeConfigOpts{}
		})

		Describe("Name", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Name", opts)).To(Equal(`long:"name" description:"Runtime-Config name (default: '')" default:""`))
			})
		})
	})

	Describe("UpdateRuntimeConfigOpts", func() {
		var opts *UpdateRuntimeConfigOpts

		BeforeEach(func() {
			opts = &UpdateRuntimeConfigOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		Describe("Name", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Name", opts)).To(Equal(`long:"name" description:"Runtime-Config name (default: '')" default:""`))
			})
		})
	})

	Describe("UpdateRuntimeConfigArgs", func() {
		var opts *UpdateRuntimeConfigArgs

		BeforeEach(func() {
			opts = &UpdateRuntimeConfigArgs{}
		})

		Describe("RuntimeConfig", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("RuntimeConfig", opts)).To(Equal(
					`positional-arg-name:"PATH" description:"Path to a runtime config file"`,
				))
			})
		})
	})

	Describe("DeployOpts", func() {
		var opts *DeployOpts

		BeforeEach(func() {
			opts = &DeployOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		Describe("Recreate", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Recreate", opts)).To(Equal(
					`long:"recreate" description:"Recreate all VMs in deployment"`,
				))
			})
		})

		Describe("NoRedact", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("NoRedact", opts)).To(Equal(
					`long:"no-redact" description:"Show non-redacted manifest diff"`,
				))
			})
		})

		Describe("SkipDrain", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("SkipDrain", opts)).To(Equal(
					`long:"skip-drain" value-name:"INSTANCE-GROUP" description:"Skip running drain scripts for specific instance groups" optional:"true" optional-value:"*"`,
				))
			})
		})

		Describe("Canaries", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Canaries", opts)).To(Equal(
					`long:"canaries" description:"Override manifest values for canaries"`,
				))
			})
		})

		Describe("MaxInFlight", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("MaxInFlight", opts)).To(Equal(
					`long:"max-in-flight" description:"Override manifest values for max_in_flight"`,
				))
			})
		})

		Describe("DryRun", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("DryRun", opts)).To(Equal(
					`long:"dry-run" description:"Renders job templates without altering deployment"`,
				))
			})
		})
	})

	Describe("DeployArgs", func() {
		var opts *DeployArgs

		BeforeEach(func() {
			opts = &DeployArgs{}
		})

		Describe("Manifest", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Manifest", opts)).To(Equal(
					`positional-arg-name:"PATH" description:"Path to a manifest file"`,
				))
			})
		})
	})

	Describe("DeleteDeploymentOpts", func() {
		var opts *DeleteDeploymentOpts

		BeforeEach(func() {
			opts = &DeleteDeploymentOpts{}
		})

		Describe("Force", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Force", opts)).To(Equal(
					`long:"force" description:"Ignore errors"`,
				))
			})
		})
	})

	Describe("UploadStemcellOpts", func() {
		var opts *UploadStemcellOpts

		BeforeEach(func() {
			opts = &UploadStemcellOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		Describe("Fix", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Fix", opts)).To(Equal(
					`long:"fix" description:"Replaces the stemcell if already exists"`,
				))
			})
		})

		Describe("Name", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Name", opts)).To(Equal(
					`long:"name" description:"Name used in existence check (is not used with local stemcell file)"`,
				))
			})
		})

		Describe("Version", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Version", opts)).To(Equal(
					`long:"version" description:"Version used in existence check (is not used with local stemcell file)"`,
				))
			})
		})

		Describe("SHA1", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("SHA1", opts)).To(Equal(
					`long:"sha1" description:"SHA1 of the remote stemcell (is not used with local files)"`,
				))
			})
		})
	})

	Describe("UploadStemcellArgs", func() {
		var opts *UploadStemcellArgs

		BeforeEach(func() {
			opts = &UploadStemcellArgs{}
		})

		Describe("URL", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("URL", opts)).To(Equal(
					`positional-arg-name:"URL" description:"Path to a local file or URL"`,
				))
			})
		})
	})

	Describe("DeleteStemcellOpts", func() {
		var opts *DeleteStemcellOpts

		BeforeEach(func() {
			opts = &DeleteStemcellOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		Describe("Force", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Force", opts)).To(Equal(
					`long:"force" description:"Ignore errors"`,
				))
			})
		})
	})

	Describe("DeleteStemcellArgs", func() {
		var opts *DeleteStemcellArgs

		BeforeEach(func() {
			opts = &DeleteStemcellArgs{}
		})

		Describe("Slug", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Slug", opts)).To(Equal(
					`positional-arg-name:"NAME/VERSION"`,
				))
			})
		})
	})

	Describe("RepackStemcellOpts", func() {
		var opts *RepackStemcellOpts

		BeforeEach(func() {
			opts = &RepackStemcellOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		It("has --version", func() {
			Expect(getStructTagForName("Version", opts)).To(Equal(
				`long:"version" description:"Repacked stemcell version"`,
			))
		})

		It("has --name", func() {
			Expect(getStructTagForName("Name", opts)).To(Equal(
				`long:"name" description:"Repacked stemcell name"`,
			))
		})

		It("has --cloud-properties", func() {
			Expect(getStructTagForName("CloudProperties", opts)).To(Equal(
				`long:"cloud-properties" description:"Repacked stemcell cloud properties"`,
			))
		})
	})

	Describe("RepackStemcellArgs", func() {
		var opts *RepackStemcellArgs

		BeforeEach(func() {
			opts = &RepackStemcellArgs{}
		})

		Describe("PathToStemcell", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("PathToStemcell", opts)).To(Equal(
					`positional-arg-name:"PATH-TO-STEMCELL" description:"Path to stemcell"`,
				))
			})
		})

		Describe("PathToResult", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("PathToResult", opts)).To(Equal(
					`positional-arg-name:"PATH-TO-RESULT" description:"Path to repacked stemcell"`,
				))
			})

			It("rejects paths that are directories", func() {
				opts.PathToResult.FS = fakesys.NewFakeFileSystem()
				opts.PathToResult.FS.MkdirAll("/some/dir", os.ModeDir)
				Expect(opts.PathToResult.UnmarshalFlag("/some/dir")).NotTo(Succeed())
			})
		})
	})

	Describe("UploadReleaseOpts", func() {
		var opts *UploadReleaseOpts

		BeforeEach(func() {
			opts = &UploadReleaseOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true"`))
			})
		})

		Describe("Directory", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Directory", opts)).To(Equal(
					`long:"dir" description:"Release directory path if not current working directory" default:"."`,
				))
			})
		})

		Describe("Rebase", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Rebase", opts)).To(Equal(
					`long:"rebase" description:"Rebases this release onto the latest version known by the Director"`,
				))
			})
		})

		Describe("Fix", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Fix", opts)).To(Equal(
					`long:"fix" description:"Replaces corrupt and missing jobs and packages"`,
				))
			})
		})

		Describe("Name", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Name", opts)).To(Equal(
					`long:"name" description:"Name used in existence check (is not used with local release file)"`,
				))
			})
		})

		Describe("Version", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Version", opts)).To(Equal(
					`long:"version" description:"Version used in existence check (is not used with local release file)"`,
				))
			})
		})

		Describe("SHA1", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("SHA1", opts)).To(Equal(
					`long:"sha1" description:"SHA1 of the remote release (is not used with local files)"`,
				))
			})
		})
	})

	Describe("UploadReleaseArgs", func() {
		var opts *UploadReleaseArgs

		BeforeEach(func() {
			opts = &UploadReleaseArgs{}
		})

		Describe("URL", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("URL", opts)).To(Equal(
					`positional-arg-name:"URL" description:"Path to a local file or URL"`,
				))
			})
		})
	})

	Describe("DeleteReleaseOpts", func() {
		var opts *DeleteReleaseOpts

		BeforeEach(func() {
			opts = &DeleteReleaseOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		Describe("Force", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Force", opts)).To(Equal(
					`long:"force" description:"Ignore errors"`,
				))
			})
		})
	})

	Describe("DeleteReleaseArgs", func() {
		var opts *DeleteReleaseArgs

		BeforeEach(func() {
			opts = &DeleteReleaseArgs{}
		})

		Describe("Slug", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Slug", opts)).To(Equal(
					`positional-arg-name:"NAME[/VERSION]"`,
				))
			})
		})
	})

	Describe("ExportReleaseOpts", func() {
		var opts *ExportReleaseOpts

		BeforeEach(func() {
			opts = &ExportReleaseOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		Describe("Directory", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Directory", opts)).To(Equal(
					`long:"dir" description:"Destination directory" default:"."`,
				))
			})
		})
	})

	Describe("ExportReleaseArgs", func() {
		var opts *ExportReleaseArgs

		BeforeEach(func() {
			opts = &ExportReleaseArgs{}
		})

		Describe("ReleaseSlug", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("ReleaseSlug", opts)).To(Equal(
					`positional-arg-name:"NAME/VERSION"`,
				))
			})
		})

		Describe("OSVersionSlug", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("OSVersionSlug", opts)).To(Equal(
					`positional-arg-name:"OS/VERSION"`,
				))
			})
		})
	})

	Describe("InspectReleaseOpts", func() {
		var opts *InspectReleaseOpts

		BeforeEach(func() {
			opts = &InspectReleaseOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})
	})

	Describe("InspectReleaseArgs", func() {
		var opts *InspectReleaseArgs

		BeforeEach(func() {
			opts = &InspectReleaseArgs{}
		})

		Describe("Slug", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Slug", opts)).To(Equal(
					`positional-arg-name:"NAME/VERSION"`,
				))
			})
		})
	})

	Describe("RunErrandOpts", func() {
		var opts *RunErrandOpts

		BeforeEach(func() {
			opts = &RunErrandOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		Describe("KeepAlive", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("KeepAlive", opts)).To(Equal(
					`long:"keep-alive" description:"Use existing VM to run an errand and keep it after completion"`,
				))
			})
		})

		Describe("WhenChanged", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("WhenChanged", opts)).To(Equal(
					`long:"when-changed" description:"Run errand only if errand configuration has changed or if the previous run was unsuccessful"`,
				))
			})
		})

		Describe("DownloadLogs", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("DownloadLogs", opts)).To(Equal(
					`long:"download-logs" description:"Download logs"`,
				))
			})
		})

		Describe("LogsDirectory", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("LogsDirectory", opts)).To(Equal(
					`long:"logs-dir" description:"Destination directory for logs" default:"."`,
				))
			})
		})
	})

	Describe("RunErrandArgs", func() {
		var opts *RunErrandArgs

		BeforeEach(func() {
			opts = &RunErrandArgs{}
		})

		Describe("Name", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Name", opts)).To(Equal(
					`positional-arg-name:"NAME"`,
				))
			})
		})
	})

	Describe("DisksOpts", func() {
		var opts *DisksOpts

		BeforeEach(func() {
			opts = &DisksOpts{}
		})

		Describe("Orphaned", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Orphaned", opts)).To(Equal(
					`long:"orphaned" short:"o" description:"List orphaned disks"`,
				))
			})
		})
	})

	Describe("DeleteDiskOpts", func() {
		var opts *DeleteDiskOpts

		BeforeEach(func() {
			opts = &DeleteDiskOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})
	})

	Describe("DeleteDiskArgs", func() {
		var opts *DeleteDiskArgs

		BeforeEach(func() {
			opts = &DeleteDiskArgs{}
		})

		Describe("CID", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("CID", opts)).To(Equal(
					`positional-arg-name:"CID"`,
				))
			})
		})
	})

	Describe("OrphanDiskOpts", func() {
		var opts *OrphanDiskOpts

		BeforeEach(func() {
			opts = &OrphanDiskOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})
	})

	Describe("OrphanDiskArgs", func() {
		var opts *OrphanDiskArgs

		BeforeEach(func() {
			opts = &OrphanDiskArgs{}
		})

		Describe("CID", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("CID", opts)).To(Equal(
					`positional-arg-name:"CID"`,
				))
			})
		})
	})

	Describe("SnapshotsOpts", func() {
		var opts *SnapshotsOpts

		BeforeEach(func() {
			opts = &SnapshotsOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true"`))
			})
		})
	})

	Describe("TakeSnapshotOpts", func() {
		var opts *TakeSnapshotOpts

		BeforeEach(func() {
			opts = &TakeSnapshotOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true"`))
			})
		})
	})

	Describe("DeleteSnapshotOpts", func() {
		var opts *DeleteSnapshotOpts

		BeforeEach(func() {
			opts = &DeleteSnapshotOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})
	})

	Describe("DeleteSnapshotArgs", func() {
		var opts *DeleteSnapshotArgs

		BeforeEach(func() {
			opts = &DeleteSnapshotArgs{}
		})

		Describe("CID", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("CID", opts)).To(Equal(
					`positional-arg-name:"CID"`,
				))
			})
		})
	})

	Describe("DeleteVMOpts", func() {
		var opts *DeleteVMOpts

		BeforeEach(func() {
			opts = &DeleteVMOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})
	})

	Describe("DeleteVMArgs", func() {
		var opts *DeleteVMArgs

		BeforeEach(func() {
			opts = &DeleteVMArgs{}
		})

		Describe("CID", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("CID", opts)).To(Equal(
					`positional-arg-name:"CID"`,
				))
			})
		})
	})

	Describe("InstanceSlugArgs", func() {
		var opts *InstanceSlugArgs

		BeforeEach(func() {
			opts = &InstanceSlugArgs{}
		})

		Describe("Slug", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Slug", opts)).To(Equal(
					`positional-arg-name:"INSTANCE-GROUP/INSTANCE-ID"`,
				))
			})
		})
	})

	Describe("InstancesOpts", func() {
		var opts *InstancesOpts

		BeforeEach(func() {
			opts = &InstancesOpts{}
		})

		Describe("Details", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Details", opts)).To(Equal(
					`long:"details" short:"i" description:"Show details including VM CID, persistent disk CID, etc."`,
				))
			})
		})

		Describe("DNS", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("DNS", opts)).To(Equal(
					`long:"dns" description:"Show DNS A records"`,
				))
			})
		})

		Describe("Vitals", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Vitals", opts)).To(Equal(
					`long:"vitals" description:"Show vitals"`,
				))
			})
		})

		Describe("Processes", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Processes", opts)).To(Equal(
					`long:"ps" short:"p" description:"Show processes"`,
				))
			})
		})

		Describe("Failing", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Failing", opts)).To(Equal(
					`long:"failing" short:"f" description:"Only show failing instances"`,
				))
			})
		})
	})

	Describe("VMsOpts", func() {
		var opts *VMsOpts

		BeforeEach(func() {
			opts = &VMsOpts{}
		})

		Describe("DNS", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("DNS", opts)).To(Equal(
					`long:"dns" description:"Show DNS A records"`,
				))
			})
		})

		Describe("Vitals", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Vitals", opts)).To(Equal(
					`long:"vitals" description:"Show vitals"`,
				))
			})
		})
	})

	Describe("CloudCheckOpts", func() {
		var opts *CloudCheckOpts

		BeforeEach(func() {
			opts = &CloudCheckOpts{}
		})

		Describe("Auto", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Auto", opts)).To(Equal(
					`long:"auto" short:"a" description:"Resolve problems automatically"`,
				))
			})
		})

		Describe("Resolution", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Resolutions", opts)).To(Equal(
					`long:"resolution" description:"Apply resolution of given type"`,
				))
			})
		})

		Describe("Report", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Report", opts)).To(Equal(
					`long:"report" short:"r" description:"Only generate report; don't attempt to resolve problems"`,
				))
			})
		})
	})

	Describe("UpdateResurrectionOpts", func() {
		var opts *UpdateResurrectionOpts

		BeforeEach(func() {
			opts = &UpdateResurrectionOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})
	})

	Describe("IgnoreOpts", func() {
		var opts *IgnoreOpts

		BeforeEach(func() {
			opts = &IgnoreOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})
	})

	Describe("UnignoreOpts", func() {
		var opts *UnignoreOpts

		BeforeEach(func() {
			opts = &UnignoreOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})
	})

	Describe("UpdateResurrectionArgs", func() {
		var opts *UpdateResurrectionArgs

		BeforeEach(func() {
			opts = &UpdateResurrectionArgs{}
		})

		Describe("Enabled", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Enabled", opts)).To(Equal(
					`positional-arg-name:"on|off"`,
				))
			})
		})
	})

	Describe("LogsOpts", func() {
		var opts *LogsOpts

		BeforeEach(func() {
			opts = &LogsOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true"`))
			})
		})

		Describe("Directory", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Directory", opts)).To(Equal(
					`long:"dir" description:"Destination directory" default:"."`,
				))
			})
		})

		Describe("Follow", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Follow", opts)).To(Equal(
					`long:"follow" short:"f" description:"Follow logs via SSH"`,
				))
			})
		})

		Describe("Num", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Num", opts)).To(Equal(
					`long:"num" description:"Last number of lines"`,
				))
			})
		})

		Describe("Quiet", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Quiet", opts)).To(Equal(
					`long:"quiet" short:"q" description:"Suppresses printing of headers when multiple files are being examined"`,
				))
			})
		})

		Describe("Jobs", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Jobs", opts)).To(Equal(
					`long:"job" description:"Limit to only specific jobs"`,
				))
			})
		})

		Describe("Filters", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Filters", opts)).To(Equal(
					`long:"only" description:"Filter logs (comma-separated)"`,
				))
			})
		})

		Describe("Agent", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Agent", opts)).To(Equal(
					`long:"agent" description:"Include only agent logs"`,
				))
			})
		})
	})

	Describe("StartOpts", func() {
		var opts *StartOpts

		BeforeEach(func() {
			opts = &StartOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true"`))
			})
		})

		Describe("Force", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Force", opts)).To(Equal(
					`long:"force" description:"No-op for backwards compatibility"`,
				))
			})
		})

		Describe("Canaries", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Canaries", opts)).To(Equal(
					`long:"canaries" description:"Override manifest values for canaries"`,
				))
			})
		})

		Describe("MaxInFlight", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("MaxInFlight", opts)).To(Equal(
					`long:"max-in-flight" description:"Override manifest values for max_in_flight"`,
				))
			})
		})
	})

	Describe("StopOpts", func() {
		var opts *StopOpts

		BeforeEach(func() {
			opts = &StopOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true"`))
			})
		})

		Describe("Soft", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Soft", opts)).To(Equal(
					`long:"soft" description:"Stop process only (default)"`,
				))
			})
		})

		Describe("Hard", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Hard", opts)).To(Equal(
					`long:"hard" description:"Delete VM (but keep persistent disk)"`,
				))
			})
		})

		Describe("SkipDrain", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("SkipDrain", opts)).To(Equal(
					`long:"skip-drain" description:"Skip running drain scripts"`,
				))
			})
		})

		Describe("Force", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Force", opts)).To(Equal(
					`long:"force" description:"No-op for backwards compatibility"`,
				))
			})
		})

		Describe("Canaries", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Canaries", opts)).To(Equal(
					`long:"canaries" description:"Override manifest values for canaries"`,
				))
			})
		})

		Describe("MaxInFlight", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("MaxInFlight", opts)).To(Equal(
					`long:"max-in-flight" description:"Override manifest values for max_in_flight"`,
				))
			})
		})
	})

	Describe("RestartOpts", func() {
		var opts *RestartOpts

		BeforeEach(func() {
			opts = &RestartOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true"`))
			})
		})

		Describe("SkipDrain", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("SkipDrain", opts)).To(Equal(
					`long:"skip-drain" description:"Skip running drain scripts"`,
				))
			})
		})

		Describe("Force", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Force", opts)).To(Equal(
					`long:"force" description:"No-op for backwards compatibility"`,
				))
			})
		})

		Describe("Canaries", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Canaries", opts)).To(Equal(
					`long:"canaries" description:"Override manifest values for canaries"`,
				))
			})
		})

		Describe("MaxInFlight", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("MaxInFlight", opts)).To(Equal(
					`long:"max-in-flight" description:"Override manifest values for max_in_flight"`,
				))
			})
		})
	})

	Describe("RecreateOpts", func() {
		var opts *RecreateOpts

		BeforeEach(func() {
			opts = &RecreateOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true"`))
			})
		})

		Describe("SkipDrain", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("SkipDrain", opts)).To(Equal(
					`long:"skip-drain" description:"Skip running drain scripts"`,
				))
			})
		})

		Describe("Force", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Force", opts)).To(Equal(
					`long:"force" description:"No-op for backwards compatibility"`,
				))
			})
		})

		Describe("Canaries", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Canaries", opts)).To(Equal(
					`long:"canaries" description:"Override manifest values for canaries"`,
				))
			})
		})

		Describe("MaxInFlight", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("MaxInFlight", opts)).To(Equal(
					`long:"max-in-flight" description:"Override manifest values for max_in_flight"`,
				))
			})
		})

		Describe("DryRun", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("DryRun", opts)).To(Equal(
					`long:"dry-run" description:"Renders job templates without altering deployment"`,
				))
			})
		})

		Describe("Fix", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Fix", opts)).To(Equal(
					`long:"fix" description:"Fix unresponsive VMs"`,
				))
			})
		})
	})

	Describe("SSHOpts", func() {
		var opts *SSHOpts

		BeforeEach(func() {
			opts = &SSHOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true"`))
			})
		})

		Describe("Command", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Command", opts)).To(Equal(
					`long:"command" short:"c" description:"Command"`,
				))
			})
		})

		Describe("RawOpts", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("RawOpts", opts)).To(Equal(
					`long:"opts" description:"Options to pass through to SSH"`,
				))
			})
		})

		Describe("Results", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Results", opts)).To(Equal(
					`long:"results" short:"r" description:"Collect results into a table instead of streaming"`,
				))
			})
		})
	})

	Describe("SCPOpts", func() {
		var opts *SCPOpts

		BeforeEach(func() {
			opts = &SCPOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		Describe("Recursive", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Recursive", opts)).To(Equal(
					`long:"recursive" short:"r" description:"Recursively copy entire directories. Note that symbolic links encountered are followed in the tree traversal"`,
				))
			})
		})
	})

	Describe("SCPArgs", func() {
		var opts *SCPArgs

		BeforeEach(func() {
			opts = &SCPArgs{}
		})

		Describe("Paths", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Paths", opts)).To(Equal(`positional-arg-name:"PATH"`))
			})
		})
	})

	Describe("GatewayFlags", func() {
		var opts *GatewayFlags

		BeforeEach(func() {
			opts = &GatewayFlags{}
		})

		It("Disable contains desired values", func() {
			Expect(getStructTagForName("Disable", opts)).To(Equal(
				`long:"gw-disable" description:"Disable usage of gateway connection" env:"BOSH_GW_DISABLE"`,
			))
		})

		It("Username contains desired values", func() {
			Expect(getStructTagForName("Username", opts)).To(Equal(
				`long:"gw-user" description:"Username for gateway connection" env:"BOSH_GW_USER"`,
			))
		})

		It("Host contains desired values", func() {
			Expect(getStructTagForName("Host", opts)).To(Equal(
				`long:"gw-host" description:"Host for gateway connection" env:"BOSH_GW_HOST"`,
			))
		})

		It("PrivateKeyPath contains desired values", func() {
			Expect(getStructTagForName("PrivateKeyPath", opts)).To(Equal(
				`long:"gw-private-key" description:"Private key path for gateway connection" env:"BOSH_GW_PRIVATE_KEY"`,
			))
		})

		It("SOCKS5Proxy contains desired values", func() {
			Expect(getStructTagForName("SOCKS5Proxy", opts)).To(Equal(
				`long:"gw-socks5" description:"SOCKS5 URL" env:"BOSH_ALL_PROXY"`,
			))
		})
	})

	Describe("InitReleaseOpts", func() {
		var opts *InitReleaseOpts

		BeforeEach(func() {
			opts = &InitReleaseOpts{}
		})

		Describe("Directory", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Directory", opts)).To(Equal(
					`long:"dir" description:"Release directory path if not current working directory" default:"."`,
				))
			})
		})

		Describe("Git", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Git", opts)).To(Equal(
					`long:"git" description:"Initialize git repository"`,
				))
			})
		})
	})

	Describe("ResetReleaseOpts", func() {
		var opts *ResetReleaseOpts

		BeforeEach(func() {
			opts = &ResetReleaseOpts{}
		})

		Describe("Directory", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Directory", opts)).To(Equal(
					`long:"dir" description:"Release directory path if not current working directory" default:"."`,
				))
			})
		})
	})

	Describe("GenerateJobOpts", func() {
		var opts *GenerateJobOpts

		BeforeEach(func() {
			opts = &GenerateJobOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		Describe("Directory", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Directory", opts)).To(Equal(
					`long:"dir" description:"Release directory path if not current working directory" default:"."`,
				))
			})
		})
	})

	Describe("GenerateJobArgs", func() {
		var opts *GenerateJobArgs

		BeforeEach(func() {
			opts = &GenerateJobArgs{}
		})

		Describe("Name", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Name", opts)).To(Equal(
					`positional-arg-name:"NAME"`,
				))
			})
		})
	})

	Describe("GeneratePackageOpts", func() {
		var opts *GeneratePackageOpts

		BeforeEach(func() {
			opts = &GeneratePackageOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		Describe("Directory", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Directory", opts)).To(Equal(
					`long:"dir" description:"Release directory path if not current working directory" default:"."`,
				))
			})
		})
	})

	Describe("GeneratePackageArgs", func() {
		var opts *GeneratePackageArgs

		BeforeEach(func() {
			opts = &GeneratePackageArgs{}
		})

		Describe("Name", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Name", opts)).To(Equal(
					`positional-arg-name:"NAME"`,
				))
			})
		})
	})

	Describe("CreateReleaseOpts", func() {
		var opts *CreateReleaseOpts

		BeforeEach(func() {
			opts = &CreateReleaseOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true"`))
			})
		})

		Describe("Directory", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Directory", opts)).To(Equal(
					`long:"dir" description:"Release directory path if not current working directory" default:"."`,
				))
			})
		})

		Describe("Name", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Name", opts)).To(Equal(
					`long:"name" description:"Custom release name"`,
				))
			})
		})

		Describe("Version", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Version", opts)).To(Equal(
					`long:"version" description:"Custom release version (e.g.: 1.0.0, 1.0-beta.2+dev.10)"`,
				))
			})
		})

		Describe("TimestampVersion", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("TimestampVersion", opts)).To(Equal(
					`long:"timestamp-version" description:"Create release with the timestamp as the dev version (e.g.: 1+dev.TIMESTAMP)"`,
				))
			})
		})

		Describe("Final", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Final", opts)).To(Equal(
					`long:"final" description:"Make it a final release"`,
				))
			})
		})

		Describe("Tarball", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Tarball", opts)).To(Equal(
					`long:"tarball" description:"Create release tarball at path (e.g. /tmp/release.tgz)"`,
				))
			})

			It("rejects paths that are directories", func() {
				opts.Tarball.FS = fakesys.NewFakeFileSystem()
				opts.Tarball.FS.MkdirAll("/some/dir", os.ModeDir)
				Expect(opts.Tarball.UnmarshalFlag("/some/dir")).NotTo(Succeed())
			})
		})

		Describe("Force", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Force", opts)).To(Equal(
					`long:"force" description:"Ignore Git dirty state check"`,
				))
			})
		})
	})

	Describe("Sha2ifyReleaseOpts", func() {
		var opts *Sha2ifyReleaseOpts

		BeforeEach(func() {
			opts = &Sha2ifyReleaseOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true"`))
			})
		})
	})

	Describe("Sha2ifyReleaseArgs", func() {
		var opts *Sha2ifyReleaseArgs

		BeforeEach(func() {
			opts = &Sha2ifyReleaseArgs{}
		})

		Describe("Positional args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Path", opts)).To(Equal(`positional-arg-name:"PATH"`))
				Expect(getStructTagForName("Destination", opts)).To(Equal(`positional-arg-name:"DESTINATION"`))
			})

			It("rejects destinations that are directories", func() {
				opts.Destination.FS = fakesys.NewFakeFileSystem()
				opts.Destination.FS.MkdirAll("/some/dir", os.ModeDir)
				Expect(opts.Destination.UnmarshalFlag("/some/dir")).NotTo(Succeed())
			})
		})
	})

	Describe("CreateReleaseArgs", func() {
		var opts *CreateReleaseArgs

		BeforeEach(func() {
			opts = &CreateReleaseArgs{}
		})

		Describe("Manifest", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Manifest", opts)).To(Equal(`positional-arg-name:"PATH"`))
			})
		})
	})

	Describe("FinalizeReleaseOpts", func() {
		var opts *FinalizeReleaseOpts

		BeforeEach(func() {
			opts = &FinalizeReleaseOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		Describe("Directory", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Directory", opts)).To(Equal(
					`long:"dir" description:"Release directory path if not current working directory" default:"."`,
				))
			})
		})

		Describe("Name", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Name", opts)).To(Equal(
					`long:"name" description:"Custom release name"`,
				))
			})
		})

		Describe("Version", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Version", opts)).To(Equal(
					`long:"version" description:"Custom release version (e.g.: 1.0.0, 1.0-beta.2+dev.10)"`,
				))
			})
		})

		Describe("Force", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Force", opts)).To(Equal(
					`long:"force" description:"Ignore Git dirty state check"`,
				))
			})
		})
	})

	Describe("FinalizeReleaseArgs", func() {
		var opts *FinalizeReleaseArgs

		BeforeEach(func() {
			opts = &FinalizeReleaseArgs{}
		})

		Describe("Path", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Path", opts)).To(Equal(`positional-arg-name:"PATH"`))
			})
		})
	})

	Describe("BlobsOpts", func() {
		var opts *BlobsOpts

		BeforeEach(func() {
			opts = &BlobsOpts{}
		})

		Describe("Directory", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Directory", opts)).To(Equal(
					`long:"dir" description:"Release directory path if not current working directory" default:"."`,
				))
			})
		})
	})

	Describe("AddBlobArgs", func() {
		var opts *AddBlobArgs

		BeforeEach(func() {
			opts = &AddBlobArgs{}
		})

		Describe("Path", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Path", opts)).To(Equal(`positional-arg-name:"PATH"`))
			})
		})

		Describe("BlobsPath", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("BlobsPath", opts)).To(Equal(
					`positional-arg-name:"BLOBS-PATH"`,
				))
			})
		})
	})

	Describe("RemoveBlobOpts", func() {
		var opts *RemoveBlobOpts

		BeforeEach(func() {
			opts = &RemoveBlobOpts{}
		})

		Describe("Args", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Args", opts)).To(Equal(`positional-args:"true" required:"true"`))
			})
		})

		Describe("Directory", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Directory", opts)).To(Equal(
					`long:"dir" description:"Release directory path if not current working directory" default:"."`,
				))
			})
		})
	})

	Describe("RemoveBlobArgs", func() {
		var opts *RemoveBlobArgs

		BeforeEach(func() {
			opts = &RemoveBlobArgs{}
		})

		Describe("BlobsPath", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("BlobsPath", opts)).To(Equal(
					`positional-arg-name:"BLOBS-PATH"`,
				))
			})
		})
	})

	Describe("SyncBlobsOpts", func() {
		var opts *SyncBlobsOpts

		BeforeEach(func() {
			opts = &SyncBlobsOpts{}
		})

		Describe("Directory", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Directory", opts)).To(Equal(
					`long:"dir" description:"Release directory path if not current working directory" default:"."`,
				))
			})
		})

		Describe("Parallel", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("ParallelOpt", opts)).To(Equal(
					`long:"parallel" description:"Sets the max number of parallel downloads" default:"5"`,
				))
			})
		})
	})

	Describe("UploadBlobsOpts", func() {
		var opts *UploadBlobsOpts

		BeforeEach(func() {
			opts = &UploadBlobsOpts{}
		})

		Describe("Directory", func() {
			It("contains desired values", func() {
				Expect(getStructTagForName("Directory", opts)).To(Equal(
					`long:"dir" description:"Release directory path if not current working directory" default:"."`,
				))
			})
		})
	})
})
