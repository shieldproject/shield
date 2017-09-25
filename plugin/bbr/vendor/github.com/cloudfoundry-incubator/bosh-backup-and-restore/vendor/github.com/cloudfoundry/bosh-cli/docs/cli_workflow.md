# Deployment Flow

This section describes how the CLI works. These steps are performed by the CLI.

For additional information see the [decision tree](init-cli-flow.png) of the deploy command.

## 1. Validating manifest, release and stemcell

The first step of the deploy process is validation. As part of that validation the CLI verifies if there are changes in either manifest, release or stemcell. In case there are no changes CLI will exit early with message `Skipping deploy`.

As part of manifest validation the CLI validates manifest properties and parses manifest for deploy. The CLI parses the deployment manifest into two parts: the deployment manifest, and the CPI configuration.

The deployment manifest is used arbitrary releases onto a single VM. The deployment manifest is defined by the `networks`, `resource_pools`, `disk_pools`, and `jobs` sections of the manifest. Currently only one job is allowed to be specified since the CLI will only create single VM.

The CPI configuration is used to install and configure the CPI locally. It is constructed from the `cloud_provider` section of the manifest.

## 2. Installing CPI Release

The provided CPI release is compiled on the machine where `bosh-init` is run, and is used locally to run the CPI commands necessary to create the VM.

The CPI release must contain a job specified by the `cloud_provider.template.job`. During CPI installation, all the packages that the CPI job depends on will be compiled and their templates rendered. CPI job templates have access to properties defined in the `cloud_provider -> properties` section of the manifest.

The compiled packages and rendered job templates are stored in a `~/.bosh/<installation_id>` folder for each deployment.

## 3. Uploading Stemcell

After the CPI is installed locally, the CLI calls the `create_stemcell` CPI method with the provided stemcell.

## 4. Starting Registry

Before creating a VM, the CLI starts the registry. The registry can be used by the CPI to store mutable data to be later accessed by the agent running on the VM. The registry is a service to store mutable data when the infrastructure's metadata service is immutable. This data is anything that is not known until after the CPI creates the VM that the agent will require. For example, information about any persistent disks that are attached to BOSH after the BOSH VM is created can be stored in the registry.

The CPI will store the registry URL in the infrastructure's metadata service. The agent on the VM will fetch registry settings from the provided URL.

Note: We are planning to eventually remove the registry to simplify how CPIs behave.

## 5. Deleting existing VM

In case the VM was previosly deployed, the CLI tries to connect to the agent on the existing VM. If the agent is responsive, the CLI stops services that are running on that VM and unmounts all disks that are attached to the VM. Eventually, the CLI deletes the existing VM and removes VM CID from deployment state file.

## 6. Creating new VM

Next, the CLI sends the `create_vm` command to the CPI with the properties parsed from the manifest. Additionally, the VM CID is persisted in deployment state file in the same folder as the deployment manifest.

## 7. Starting SSH Tunnel

The CLI creates a reverse SSH tunnel to the BOSH VM using the properties provided in the manifest. This allows the agent on the VM to access the registry, which is running on the machine where `bosh-init deploy` was run.

## 8. Waiting for Agent

Once the SSH tunnel is up the CLI uses the provided mbus URL to issue ping messages to the agent on the BOSH VM. Once the agent is ready it will respond to the ping.

## 9. Creating disk

The CLI will create and attach a disk to the VM if it is requested in the deployment manifest. There are two ways to request the disk:

1. Adding the `persistent_disk_pool` property on a job which references the disk pool in the list of `disk_pools` specified on the top level of the manifest.
2. Adding the `persistent_disk` property which specifies the size of persistent disk.

You should use `disk_pools` if you want to use disk `cloud_properties`.

In this case, the CLI calls the `create_disk` CPI method with the provided size. Additionally, the disk CID is persisted in deployment state file.

## 10. Attaching disk

After the disk is created, the CLI calls the `attach_disk` CPI method. After the disk is attached, the CLI issues a `mount_disk` request to the agent on the BOSH VM.

## 11. Sending stop message

Once the agent is listening on the mbus URL, the CLI sends a `stop` message to the agent. The agent is using `monit` to manage job states on VM. The `stop` is a preparation for the subsequent job update.

## 12. Sending apply message

Next the CLI sends an `apply` message with the list of packages and jobs that should be installed on the VM. The agent serves a blobstore at `<mbus URL>/blobs` endpoint.

For each of the templates specified, the CLI downloads the corresponding job template from the blobstore, renders the template with the properties specified for the job in the deployment manifest. Once all the templates are rendered, the CLI uploads the archive of all the rendered templates to the blobstore and generates an `apply` message. This `apply` message contains the list of all packages, spec of the templates archive with uploaded blob ID, networks spec parsed from deployment manifest and configuration hash which is a digest of all rendered job template files.

## 13. Sending start message

Once the `apply` task is finished the CLI sends a `start` message to the agent which starts installed jobs.
