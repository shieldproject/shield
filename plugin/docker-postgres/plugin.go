package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"time"

	. "github.com/starkandwayne/shield/plugin"

	docker "github.com/fsouza/go-dockerclient"
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

	i := 0
	for _, info := range registry {
		i++
		DEBUG("[%s] attempting to backup container", info.Name)
		// extract the Postgres URI from the container environment and network settings
		uri, err := pgURI(info)
		if err != nil {
			DEBUG("[%s] failed to generate pg URI: %s", info.Name, err)
			continue
		}
		DEBUG("[%s] connecting to %s", info.Name, uri)

		// dump the Postgres database to a temporary file
		data, err := pgdump(uri)
		if err != nil {
			DEBUG("[%s] failed to dump the database: %s", info.Name, err)
			continue
		}

		// write the metadata and the backup data to the archive
		err = archive.Write(info.Name, info, data)
		data.Close()
		if err != nil {
			DEBUG("[%s] failed to write backup #%d to archive: %s", info.Name, i, err)
			continue
		}

		DEBUG("[%s] wrote backup #%d to archive", info.Name, i)
	}
	archive.Close()
	DEBUG("DONE")
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

	// treat stdin as a tar stream
	//archive := tar.NewReader(os.Stdin)
	archive := NewArchiveReader(os.Stdin)
	for {
		var info docker.Container
		data, err := archive.Next(&info)
		if err == io.EOF {
			DEBUG("end of archive reached.  all done!")
			break
		}
		if err != nil {
			DEBUG("[%s] failed to retrieve backup from archive: %s", info.Name, err)
			break
		}

		// destroy any existing containers with the same name
		if existing, ok := registry[info.Name]; ok {
			DEBUG("[%s] %s: already exists (as [%s]); removing existing container first", info.ID, info.Name, existing.ID)
			err = c.RemoveContainer(docker.RemoveContainerOptions{
				ID:            existing.ID,
				RemoveVolumes: true,
				Force:         true,
			})
			if err != nil {
				DEBUG("[%s] error removing existing container [%s]: %s", info.Name, existing.ID, err)
				continue
			}
		}

		// deploy a new container with the correct image / ip / creds
		DEBUG("[%s] deploying new container", info.Name)
		newContainer, err := c.CreateContainer(docker.CreateContainerOptions{
			Name:       info.Name,
			Config:     info.Config,
			HostConfig: info.HostConfig,
		})
		if err != nil {
			DEBUG("[%s] deploy failed: %s", info.Name, err)
			continue
		}
		DEBUG("[%s] starting container", info.Name)
		err = c.StartContainer(newContainer.ID, info.HostConfig)
		if err != nil {
			DEBUG("[%s] start failed: %s", info.Name, err)
			continue
		}

		// read backup data, piping to pgrestore process
		uri, err := pgURI(&info)
		if err != nil {
			DEBUG("[%s] failed to generate pg URI: %s", info.Name, err)
			continue
		}
		DEBUG("[%s] connecting to %s", info.Name, uri)
		waitForPostgres(uri, 60)
		err = pgrestore(uri, data)
		if err != nil {
			DEBUG("[%s] restore failed: %s", info.Name, err)
			continue
		}
		DEBUG("[%s] successfully restored", info.Name)
	}
	DEBUG("DONE")
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

func pgdump(uri string) (*os.File, error) {
	file, err := ioutil.TempFile("", "pgdump")
	if err != nil {
		return nil, err
	}

	// FIXME: make it possible to select what version of postgres (9.x, 8.x, etc.)
	cmd := exec.Command("/var/vcap/packages/postgres-9.4/bin/pg_dump", "-cC", "--format", "p", "-d", uri)
	cmd.Stdout = file

	err = cmd.Run()
	return file, err
}

func pgrestore(uri string, in io.Reader) error {
	// FIXME: make it possible to select what version of postgres (9.x, 8.x, etc.)
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
