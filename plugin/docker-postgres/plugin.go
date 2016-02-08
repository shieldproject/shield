// The `docker-postgres` plugin for SHIELD implements backup + restore functionality
// specific to the `docker-boshrelease`'s cf-containers-broker, in conjunction with
// Postgres docker containers. It is very specific to backing up postgres docker containers
// on this service broker, and will likely not work with other implementations.
//
// PLUGIN FEATURES
//
// This plugin implements functionality suitable for use with the following SHIELD
// Job components:
//
//    Target: yes
//    Store:  no
//
// PLUGIN CONFIGURATION
//
// No configuration is required for this plugin, as the docker deployments are self-contained
// on a single VM, and all the data for backing up + restoring services can be detected
// automatically. Your endpoint JSON should look something like this:
//
//    {}
//
// BACKUP DETAILS
//
// The `docker-postgres` plugin backs up by connecting to docker, finding all running
// containers, and loops through each, grabbing its connection info from docker, and runs
// a pg_dump on the docker database. Each dump is written into a tar archive, along with metadata
// about the service ID and connection info.
//
// RESTORE DETAILS
//
// During restore, `docker-postgres` iterates through each backup in the archive, parsing
// out service information + data. It then deletes any existing containers using the ID
// of the container to be restored, recreates its volume directories if necessary, and
// creates a new container with the original name, id, and port mappings. Finally, it
// restores the data to the new postgres instance via `psql`.
//
// Restores with `docker-postgres` are service-impacting. Postgres containers are potentially
// deleted and re-created, preventing apps from communicating with them during the backup.
//
// DEPENDENCIES
//
// This plugin relies on the `pg_dump` and `psql` commands. Please ensure that they
// are present on the system that will be running the backups + restores for postgres.
// If you are using shield-boshrelease to deploy SHIELD, these tools are provided, if you
// include the `agent-pgtools` job template along side your `shield-agent`.
//
package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"

	. "github.com/starkandwayne/shield/plugin"
)

func main() {
	p := DockerPostgresPlugin{
		Name:    "Dockerized PostgreSQL Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
	}

	Run(p)
}

type DockerPostgresPlugin PluginInfo

func (p DockerPostgresPlugin) Meta() PluginInfo {
	return PluginInfo(p)
}

func dockerClient(endpoint ShieldEndpoint) (*docker.Client, error) {
	socket, err := endpoint.StringValue("socket")
	if err != nil {
		socket = "unix:///var/vcap/sys/run/docker/docker.sock"
	}

	DEBUG("connecting to docker at %s", socket)
	c, err := docker.NewClient(socket)
	if err != nil {
		DEBUG("connection failed: %s", err)
		return nil, err
	}

	return c, nil
}

func (p DockerPostgresPlugin) Backup(endpoint ShieldEndpoint) error {
	c, err := dockerClient(endpoint)
	if err != nil {
		return err
	}

	// list running containers
	DEBUG("listing running containers")
	registry, err := listContainers(c, false)
	if err != nil {
		return err
	}
	DEBUG("found %d running containers to backup", len(registry))

	// start a tar stream
	archive := NewArchiveWriter(os.Stdout)

	// determine our working dir for backup buffer files
	tmpdir, err := endpoint.StringValue("/tmp")
	if err != nil {
		tmpdir = "/var/vcap/store/tmp"
	}
	os.Mkdir(tmpdir, 0755)

	fail := MultiError{Message: fmt.Sprintf("failed to backup all %d postgres containers", len(registry))}
	i := 0
	for _, info := range registry {
		i++
		DEBUG("[%s] attempting to backup container", info.Name)
		// extract the Postgres URI from the container environment and network settings
		uri, err := pgURI(info)
		if err != nil {
			fail.Appendf("[%s] failed to generate postgres URI: %s", info.Name, err)
			continue
		}
		DEBUG("[%s] connecting to %s", info.Name, uri)

		// dump the Postgres database to a temporary file
		data, err := ioutil.TempFile(tmpdir, "pgdump")
		if err != nil {
			fail.Appendf("[%s] failed to create a temporary file: %s", info.Name, err)
			continue
		}
		err = pgdump(uri, data)
		if err != nil {
			fail.Appendf("[%s] failed to dump the database: %s", info.Name, err)

			// remove the temp file
			data.Close()
			os.Remove(data.Name())
			continue
		}

		// write the metadata and the backup data to the archive
		err = archive.Write(info.Name, info, data)
		if err != nil {
			fail.Appendf("[%s] failed to write backup #%d to archive: %s", info.Name, i, err)

			// remove the temp file
			data.Close()
			os.Remove(data.Name())
			continue
		}

		// remove the temp file
		data.Close()
		os.Remove(data.Name())

		DEBUG("[%s] wrote backup #%d to archive", info.Name, i)
	}
	archive.Close()
	DEBUG("DONE")
	if fail.Valid() {
		return fail
	}
	return nil
}

func (p DockerPostgresPlugin) Restore(endpoint ShieldEndpoint) error {
	c, err := dockerClient(endpoint)
	if err != nil {
		return err
	}

	// list running containers
	DEBUG("listing all containers")
	registry, err := listContainers(c, true)
	if err != nil {
		return err
	}
	DEBUG("found %d running containers", len(registry))

	fail := MultiError{Message: "failed to restore all postgres containers"}

	// treat stdin as a tar stream
	archive := NewArchiveReader(os.Stdin)
	for {
		var info docker.Container
		data, err := archive.Next(&info)
		if err == io.EOF {
			DEBUG("end of archive reached.  all done!")
			break
		}
		if err != nil {
			fail.Appendf("[%s] failed to retrieve backup from archive: %s", info.Name, err)
			break
		}

		// destroy any existing containers with the same name
		if existing, ok := registry[info.Name]; ok {
			DEBUG("[%s] removing existing container %s", info.Name, existing.ID)
			err = c.RemoveContainer(docker.RemoveContainerOptions{
				ID:            existing.ID,
				RemoveVolumes: true,
				Force:         true,
			})
			if err != nil {
				fail.Appendf("[%s] error removing existing container [%s]: %s", info.Name, existing.ID, err)
				continue
			}
		}

		// recreate volume directories if they exist
		for _, bind := range info.HostConfig.Binds {
			parts := strings.Split(bind, ":")
			if len(parts) != 2 {
				fail.Appendf("[%s] volume %s seems malformed...", info.Name, bind)
				continue
			}

			DEBUG("[%s] removing volume %s (mapped to %s in-container)", info.Name, parts[0], parts[1])
			os.RemoveAll(parts[0])
			os.Mkdir(parts[0], os.FileMode(0755))
		}

		// deploy a new container with the correct image / ip / creds
		DEBUG("[%s] deploying new container", info.Name)
		newContainer, err := c.CreateContainer(docker.CreateContainerOptions{
			Name:       info.Name,
			Config:     info.Config,
			HostConfig: info.HostConfig,
		})
		if err != nil {
			fail.Appendf("[%s] deploy failed: %s", info.Name, err)
			continue
		}
		DEBUG("[%s] starting container", info.Name)
		err = c.StartContainer(newContainer.ID, info.HostConfig)
		if err != nil {
			fail.Appendf("[%s] start failed: %s", info.Name, err)
			continue
		}

		// read backup data, piping to pgrestore process
		uri, err := pgURI(&info)
		if err != nil {
			fail.Appendf("[%s] failed to generate pg URI: %s", info.Name, err)
			continue
		}
		DEBUG("[%s] connecting to %s", info.Name, uri)
		waitForPostgres(uri, 60)
		err = pgrestore(uri, data)
		if err != nil {
			fail.Appendf("[%s] restore failed: %s", info.Name, err)
			continue
		}
		DEBUG("[%s] successfully restored", info.Name)
	}
	DEBUG("DONE")
	if fail.Valid() {
		return fail
	}
	return nil
}

func (p DockerPostgresPlugin) Store(endpoint ShieldEndpoint) (string, error) {
	return "", UNIMPLEMENTED
}

func (p DockerPostgresPlugin) Retrieve(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

func (p DockerPostgresPlugin) Purge(endpoint ShieldEndpoint, file string) error {
	return UNIMPLEMENTED
}

func pgURI(container *docker.Container) (string, error) {
	re, err := regexp.Compile(`^POSTGRES_(.*?)=(.*)$`)
	if err != nil {
		return "", err
	}

	var user, pass, db string
	for _, env := range container.Config.Env {
		if m := re.FindStringSubmatch(env); m != nil {
			switch m[1] {
			case "USERNAME":
				user = m[2]

			case "PASSWORD":
				pass = m[2]

			case "DBNAME":
				db = m[2]
			}
		}
	}

	if user == "" {
		return "", fmt.Errorf("unable to determine POSTGRES_USERNAME from container information")
	}
	if pass == "" {
		return "", fmt.Errorf("unable to determine POSTGRES_PASSWORD from container information")
	}
	if db == "" {
		return "", fmt.Errorf("unable to determine POSTGRES_DBNAME from container information")
	}

	//ip := container.NetworkSettings.IPAddress
	ip := "127.0.0.1"
	binding, ok := container.NetworkSettings.Ports["5432/tcp"]
	if !ok {
		return "", fmt.Errorf("port 5432/tcp not found in Ports bound for this docker container")
	}
	if len(binding) != 1 {
		return "", fmt.Errorf("incorrect number of port bindings found for 5432/tcp (expected only one, got %d)", len(binding))
	}
	port := binding[0].HostPort

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, ip, port, db), nil
}

func waitForPostgres(uri string, seconds int) {
	DEBUG("waiting up to %d seconds for connection to %s to succeed", seconds, uri)
	DEBUG("  (running command `/var/vcap/packages/postgres-9.4/bin/psql %s`)", uri)
	for seconds > 0 {
		cmd := exec.Command("/var/vcap/packages/postgres-9.4/bin/psql", uri)
		err := cmd.Run()
		if err == nil {
			DEBUG("connection to %s succeeded!", uri)
			return
		}
		time.Sleep(time.Second)
		seconds--
	}
	DEBUG("connection to %s ultimately failed", uri)
}

func pgdump(uri string, file *os.File) error {
	// FIXME: make it possible to select what version of postgres (9.x, 8.x, etc.)
	DEBUG("  (running command `/var/vcap/packages/postgres-9.4/bin/pg_dump -cC --format p -d %s`)", uri)
	cmd := exec.Command("/var/vcap/packages/postgres-9.4/bin/pg_dump", "-cC", "--format", "p", "-d", uri)
	cmd.Stdout = file

	return cmd.Run()
}

func pgrestore(uri string, in io.Reader) error {
	// FIXME: make it possible to select what version of postgres (9.x, 8.x, etc.)
	DEBUG("  (running command `/var/vcap/packages/postgres-9.4/bin/psql %s`)", uri)
	cmd := exec.Command("/var/vcap/packages/postgres-9.4/bin/psql", uri)
	cmd.Stdin = in // what about the call to Close()?

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func listContainers(client *docker.Client, all bool) (map[string]*docker.Container, error) {
	var opts docker.ListContainersOptions
	if all {
		opts.All = true
	} else {
		opts.Filters = map[string][]string{"status": []string{"running"}}
	}
	l, err := client.ListContainers(opts)
	if err != nil {
		return nil, err
	}

	m := make(map[string]*docker.Container, 0)
	for _, c := range l {
		info, err := client.InspectContainer(c.ID)
		if err != nil {
			DEBUG("failed to inspect container: %s", err)
		}
		m[info.Name] = info
	}

	return m, nil
}
