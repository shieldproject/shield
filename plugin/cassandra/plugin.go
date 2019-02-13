// The `cassandra` plugin for SHIELD implements backup + restore of one single
// keyspace on a Cassandra node.
//
// PLUGIN FEATURES
//
// This plugin implements functionality suitable for use with the following
// SHIELD Job components:
//
//   Target: yes
//   Store:  no
//
// PLUGIN CONFIGURATION
//
// The endpoint configuration passed to this plugin is used to identify which
// cassandra node to back up, and how to connect to it. Your endpoint JSON
// should look something like this:
//
//    {
//        "host"         : "10.244.67.61",
//        "port"         : "9042",             # native transport port
//        "user"         : "username",
//        "password"     : "password",
//        "keyspace"     : "ksXXXX",           # Required
//        "datadir"      : "/path/to/datadir"
//    }
//
// The plugin provides devault values for those configuration properties, as
// detailed below. When a default value suits your needs, you can just ommit
// it.
//
//    {
//        "host"     : "127.0.0.1",
//        "port"     : "9042",
//        "user"     : "cassandra",
//        "password" : "cassandra",
//        "datadir"  : "/var/vcap/store/cassandra/data"
//    }
//
// This plugin uses the SHIELD v8 `env.path` configuration property to find
// the `nodetool` and `sstableloader` wrapper scripts that don't need any
// exernal environment variables to run. Administrators deploying this plugin
// to for backuping the 'cassandra' BOSH release should take care of providing
// the `/var/vcap/cassandra/job/bin` path in this `env.path` configuration
// property, and not the `bin` directory of the 'cassandra' package.
//
// BACKUP DETAILS
//
// Backup is limited to one single keyspace, and is made against one single
// node. To completely backup the given keyspace, the backup operation needs
// to be performed on all cluster nodes.
//
// As a result of the backup strategy implemented by this plugin, extra space
// is required on the persistent disk. At backup time, this plugin relies on
// `nodetool snapshot` which can only create its immutable files inside the
// Cassandra data directory. At restore time, `sstableloader` requires the
// backup files to be entirely decompressed before proceeding. This plugins is
// opinionated towards extracting those files in the persistent storage
// because the extra space is already required for backups.
//
// Therefore, as a rule of the thumb, you should provide twice the persistent
// storage required for your data.
//
// RESTORE DETAILS
//
// Restore is limited to the single keyspace specified in the plugin config.
// When restoring, this keyspace config must be the same as the keyspace
// specified at backup time. Indeed, this plugin doesn't support restoring to
// a different keyspace.
//
// Restore should happen on the same node where the data has been backed up.
// To completely restore a keyspace, the restore operation should be performed
// on each node of the cluster, with the data that was backed up on that same
// node.
//
// DEPENDENCIES
//
// This plugin relies on some `nodetool` and `sstableloader` wrapper scripts
// that will run the regular `nodetool` and `sstableloader` utilities without
// requiring any environment variable to be provided (like JAVA_HOME or
// CASSANDRA_CONF). The 'cassandra' BOSH release typically provides thoses
// scripts in `/var/vcap/cassandra/job/bin`. Please ensure that this directory
// is added to the SHIELD v8 `env.path` configuration property.
//
// This plugin also relies on some `tar` utility that should be provided on
// its PATH. This will typically be the standard GNU Tar utility, as provided
// by BOSH stemcells.

package main

import (
	"os"
	"path/filepath"
	"strings"

	fmt "github.com/jhunt/go-ansi"

	"github.com/starkandwayne/shield/plugin"
)

const (
	DefaultHost     = "127.0.0.1"
	DefaultPort     = "9042"
	DefaultUser     = "cassandra"
	DefaultPassword = "cassandra"
	DefaultDataDir  = "/var/vcap/store/cassandra/data"

	VcapOwnership = "vcap:vcap"
	SnapshotName  = "shield-backup"
)

func main() {
	p := CassandraPlugin{
		Name:    "Cassandra Backup Plugin",
		Author:  "Orange",
		Version: "0.1.0",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
  "host"         : "127.0.0.1",      # optional
  "port"         : "9042",           # optional
  "user"         : "username",
  "password"     : "password",
  "keyspace"     : "db",
  "datadir"      : "/path/to/data"   # optional
}
`,
		Defaults: `
{
  "host"     : "127.0.0.1",
  "port"     : "9042",
  "user"     : "cassandra",
  "password" : "cassandra",
  "datadir"  : "/var/vcap/store/cassandra/data"
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "target",
				Name:     "host",
				Type:     "string",
				Title:    "Cassandra Host",
				Help:     "The hostname or IP address of your Cassandra server.",
				Default:  "127.0.0.1",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "port",
				Type:     "port",
				Title:    "Cassandra Port",
				Help:     "The 'native transport' TCP port that Cassandra server is bound to, listening for incoming connections.",
				Default:  "9042",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "user",
				Type:     "string",
				Title:    "Cassandra Username",
				Help:     "Username to authenticate to Cassandra as.",
				Default:  "cassandra",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "password",
				Type:     "password",
				Title:    "Cassandra Password",
				Help:     "Password to authenticate to Cassandra as.",
				Default:  "cassandra",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "keyspace",
				Type:     "string",
				Title:    "Keyspace to Backup",
				Help:     "The name of the keyspace to backup.",
				Example:  "system",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "datadir",
				Type:     "abspath",
				Title:    "Path to Cassandra data/ directory",
				Help:     "The absolute path to the data/ directory that contains the Cassandra database files.",
				Default:  "/var/vcap/store/cassandra/data",
				Required: true,
			},
		},
	}

	plugin.Run(p)
}

type CassandraPlugin plugin.PluginInfo

type CassandraInfo struct {
	Host     string
	Port     string
	User     string
	Password string
	Keyspace string
	DataDir  string
}

// This function should be used to return the plugin's PluginInfo, however you decide to implement it
func (p CassandraPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

// Called to validate endpoints from the command line
func (p CassandraPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValueDefault("host", "")
	if err != nil {
		fmt.Printf("@R{\u2717 host          %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 host}          using default node @C{%s}\n", DefaultHost)
	} else {
		fmt.Printf("@G{\u2713 host}          @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("port", "")
	if err != nil {
		fmt.Printf("@R{\u2717 port          %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 port}          using default port @C{%s}\n", DefaultPort)
	} else {
		fmt.Printf("@G{\u2713 port}          @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("user", "")
	if err != nil {
		fmt.Printf("@R{\u2717 user          %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 user}          using default user @C{%s}\n", DefaultUser)
	} else {
		fmt.Printf("@G{\u2713 user}          @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("password", "")
	if err != nil {
		fmt.Printf("@R{\u2717 password      %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 password}      using default password @C{%s}\n", DefaultPassword)
	} else {
		fmt.Printf("@G{\u2713 password}      @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("keyspace")
	if err != nil {
		fmt.Printf("@R{\u2717 keyspace      %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 keyspace}      @C{%s}\n", s)
	}

	s, err = endpoint.StringValueDefault("datadir", "")
	if err != nil {
		fmt.Printf("@R{\u2717 datadir       %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 datadir}       using default @C{%s}\n", DefaultDataDir)
	} else {
		fmt.Printf("@G{\u2713 datadir}       @C{%s}\n", s)
	}

	if fail {
		return fmt.Errorf("cassandra: invalid configuration")
	}
	return nil
}

// Backup one cassandra keyspace
func (p CassandraPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	cassandra, err := cassandraInfo(endpoint)
	if err != nil {
		return err
	}

	plugin.DEBUG("Cleaning any stale '%s' snapshot", SnapshotName)
	cmd := fmt.Sprintf("nodetool clearsnapshot -t %s \"%s\"", SnapshotName, cassandra.Keyspace)
	plugin.DEBUG("Executing: `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDIN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Clean any stale snapshot}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Clean any stale snapshot}\n")

	defer func() {
		plugin.DEBUG("Clearing snapshot '%s'", SnapshotName)
		cmd := fmt.Sprintf("nodetool clearsnapshot -t %s \"%s\"", SnapshotName, cassandra.Keyspace)
		plugin.DEBUG("Executing: `%s`", cmd)
		err := plugin.Exec(cmd, plugin.STDIN)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Clean snapshot}\n")
			return
		}
		fmt.Fprintf(os.Stderr, "@G{\u2713 Clean snapshot}\n")
	}()

	plugin.DEBUG("Creating a new '%s' snapshot", SnapshotName)
	cmd = fmt.Sprintf("nodetool snapshot -t %s \"%s\"", SnapshotName, cassandra.Keyspace)
	plugin.DEBUG("Executing: `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDIN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Create new snapshot}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Create new snapshot}\n")

	// Here we need to hard-link the snapshots/shield-backup directories into
	// a {keyspace}/{tablename} structure that we'll temporarily put in
	// /var/vcap/store/shield/cassandra. Then we can tar it all and stream
	// that to stdout.

	baseDir := "/var/vcap/store/shield/cassandra"
	plugin.DEBUG("Creating any missing directories for '%s', with 0755 permissions", baseDir)
	err = os.MkdirAll(baseDir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Create base temporary directory}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Create base temporary directory}\n")

	// Recursively remove /var/vcap/store/shield/cassandra/{keyspace}, if any
	tmpKeyspaceDir := filepath.Join(baseDir, cassandra.Keyspace)
	plugin.DEBUG("Removing any stale '%s' directory", tmpKeyspaceDir)
	cmd = fmt.Sprintf("rm -rf \"%s\"", tmpKeyspaceDir)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDOUT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Clear base temporary directory}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Clear base temporary directory}\n")

	defer func() {
		// Recursively remove /var/vcap/store/shield/cassandra/{keyspace}, if any
		plugin.DEBUG("Cleaning the '%s' directory up", tmpKeyspaceDir)
		cmd := fmt.Sprintf("rm -rf \"%s\"", tmpKeyspaceDir)
		plugin.DEBUG("Executing `%s`", cmd)
		err := plugin.Exec(cmd, plugin.STDOUT)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Clean base temporary directory}\n")
			return
		}
		fmt.Fprintf(os.Stderr, "@G{\u2713 Clean base temporary directory}\n")
	}()

	plugin.DEBUG("Creating directory '%s' with 0700 permissions", tmpKeyspaceDir)
	err = os.Mkdir(tmpKeyspaceDir, 0700)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Create temporary directory}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Create temporary directory}\n")

	// Iterate through {dataDir}/{keyspace}/{tablename}/snapshots/shield-backup/*
	// and for all the immutable files we find here, we hard-link them
	// to /var/vcap/store/shield/cassandra/{keyspace}/{tablename}
	//
	// We chose to hard-link because copying those immutable files is
	// unnecessary anyway. It could lead to performance issues and would
	// consume twice the disk space it should.

	info, err := os.Lstat(cassandra.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Recursive hard-link snapshot files}\n")
		return err
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Recursive hard-link snapshot files}\n")
		return fmt.Errorf("cassandra DataDir is not a directory")
	}

	srcKeyspaceDir := filepath.Join(cassandra.DataDir, cassandra.Keyspace)
	dir, err := os.Open(srcKeyspaceDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Recursive hard-link snapshot files}\n")
		return err
	}
	defer dir.Close()

	entries, err := dir.Readdir(-1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Recursive hard-link snapshot files}\n")
		return err
	}
	for _, tableDirInfo := range entries {
		if !tableDirInfo.IsDir() {
			continue
		}

		src_dir := filepath.Join(srcKeyspaceDir, tableDirInfo.Name(), "snapshots", SnapshotName)

		tableName := tableDirInfo.Name()
		if idx := strings.LastIndex(tableName, "-"); idx >= 0 {
			tableName = tableName[:idx]
		}

		dst_dir := filepath.Join(tmpKeyspaceDir, tableName)
		plugin.DEBUG("Creating destination directory '%s'", dst_dir)
		err = os.MkdirAll(dst_dir, 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Recursive hard-link snapshot files}\n")
			return err
		}

		plugin.DEBUG("Hard-linking all '%s/*' files to '%s/'", src_dir, dst_dir)
		err = hardLinkAll(src_dir, dst_dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Recursive hard-link snapshot files in temp dir}\n")
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Recursive hard-link snapshot files in temp dir}\n")

	plugin.DEBUG("Setting ownership of all backup files to '%s'", VcapOwnership)
	cmd = fmt.Sprintf("chown -R vcap:vcap \"%s\"", tmpKeyspaceDir)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDOUT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Set ownership of snapshot hard-links}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Set ownership of snapshot hard-links}\n")

	plugin.DEBUG("Streaming output tar file")
	cmd = fmt.Sprintf("tar -c -C /var/vcap/store/shield/cassandra -f - \"%s\"", cassandra.Keyspace)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDOUT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Stream tar of snapshots files}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Stream tar of snapshots files}\n")

	return nil
}

// Hard-link all files from 'src_dir' to the 'dst_dir'
func hardLinkAll(src_dir string, dst_dir string) (err error) {

	dir, err := os.Open(src_dir)
	if err != nil {
		return err
	}
	defer func() {
		dir.Close()
	}()

	entries, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	for _, tableDirInfo := range entries {
		if tableDirInfo.IsDir() {
			continue
		}
		src := filepath.Join(src_dir, tableDirInfo.Name())
		dst := filepath.Join(dst_dir, tableDirInfo.Name())

		err = os.Link(src, dst)
		if err != nil {
			return err
		}
	}
	return nil
}

// Restore one cassandra keyspace
func (p CassandraPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	cassandra, err := cassandraInfo(endpoint)
	if err != nil {
		return err
	}

	plugin.DEBUG("Creating directory '%s' with 0755 permissions", "/var/vcap/store/shield/cassandra")
	err = os.MkdirAll("/var/vcap/store/shield/cassandra", 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Create base temporary directory}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Create base temporary directory}\n")

	keyspaceDirPath := filepath.Join("/var/vcap/store/shield/cassandra", cassandra.Keyspace)

	// Recursively remove /var/vcap/store/shield/cassandra/{cassandra.Keyspace}, if any
	cmd := fmt.Sprintf("rm -rf \"%s\"", keyspaceDirPath)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDOUT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Clear base temporary directory}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Clear base temporary directory}\n")

	defer func() {
		// plugin.DEBUG("Skipping recursive deletion of directory '%s'", keyspaceDirPath)

		// Recursively remove /var/vcap/store/shield/cassandra/{cassandra.Keyspace}, if any
		cmd := fmt.Sprintf("rm -rf \"%s\"", keyspaceDirPath)
		plugin.DEBUG("Executing `%s`", cmd)
		err := plugin.Exec(cmd, plugin.STDOUT)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Clean base temporary directory}\n")
			return
		}
		fmt.Fprintf(os.Stderr, "@G{\u2713 Clean base temporary directory}\n")
	}()

	cmd = fmt.Sprintf("tar -x -C /var/vcap/store/shield/cassandra -f -")
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDIN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Extract tar to temporary directory}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Extract tar to temporary directory}\n")

	// Iterate through all table directories /var/vcap/store/shield/cassandra/{cassandra.Keyspace}/{tablename}
	dir, err := os.Open(keyspaceDirPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Load all tables data}\n")
		return err
	}
	defer dir.Close()

	entries, err := dir.Readdir(-1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Load all tables data}\n")
		return err
	}
	for _, tableDirInfo := range entries {
		if !tableDirInfo.IsDir() {
			continue
		}
		// Run sstableloader on each sub-directory found, assuming it is a table backup
		tableDirPath := filepath.Join(keyspaceDirPath, tableDirInfo.Name())
		cmd := fmt.Sprintf("sstableloader -u \"%s\" -pw \"%s\" -d \"%s\" \"%s\"", cassandra.User, cassandra.Password, cassandra.Host, tableDirPath)
		plugin.DEBUG("Executing: `%s`", cmd)
		err = plugin.Exec(cmd, plugin.STDIN)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Load all tables data}\n")
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Load all tables data}\n")

	return nil
}

func (p CassandraPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p CassandraPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p CassandraPlugin) Purge(endpoint plugin.ShieldEndpoint, key string) error {
	return plugin.UNIMPLEMENTED
}

func cassandraInfo(endpoint plugin.ShieldEndpoint) (*CassandraInfo, error) {
	host, err := endpoint.StringValueDefault("host", DefaultHost)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("HOST: '%s'", host)

	port, err := endpoint.StringValueDefault("port", DefaultPort)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("PORT: '%s'", port)

	user, err := endpoint.StringValueDefault("user", DefaultUser)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("USER: '%s'", user)

	password, err := endpoint.StringValueDefault("password", DefaultPassword)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("PWD: '%s'", password)

	keyspace, err := endpoint.StringValue("keyspace")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("KEYSPACE: '%s'", keyspace)

	datadir, err := endpoint.StringValueDefault("datadir", DefaultDataDir)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("DATADIR: '%s'", datadir)

	return &CassandraInfo{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Keyspace: keyspace,
		DataDir:  datadir,
	}, nil
}
