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
	"sort"
	"strings"

	fmt "github.com/jhunt/go-ansi" 

	"github.com/starkandwayne/shield/plugin"
)

// Default configuration values for the plugin
const (
	DefaultHost      = "127.0.0.1"
	DefaultPort      = "9042"
	DefaultUser      = "cassandra"
	DefaultPassword  = "cassandra"
	DefaultSaveUsers = true
	DefaultDataDir   = "/var/vcap/store/cassandra/deployment_name/data"
	DefaultBinDir    = "/var/vcap/jobs/cassandra/bin"
	VcapOwnership = "vcap:vcap"
	SnapshotName  = "shield-backup"
)

// Array or slices aren't immutable by nature; you can't make them constant
var (
	DefaultExcludeKeyspaces = []string{"system_schema", "system_distributed", "system_auth", "system", "system_traces"}
	SystemAuthTables        = []string{"roles", "role_permissions", "role_members", "resource_role_permissons_index"}
)

func main() {
	p := CassandraPlugin{
		Name:    "Cassandra Backup Plugin",
		Author:  "Orange",
		Version: "1.0.0",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
  "host"              : "127.0.0.1",      # optional
  "port"              : "9042",           # optional
  "user"              : "username",
  "password"          : "password",
  "include_keyspaces" : ["db"],
  "exclude_keyspaces" : ["system"],
  "save_users"        : true,
  "bindir"            : "/path/to/bin",   # optional
  "datadir"           : "/path/to/data",  # optional
}
`,
		Defaults: `
{
  "host"              : "cassandra_host", // "12.0.0.1"
  "port"              : "9042", // "cassandra_port"
  "user"              : "cassandra_user",     //cassandra
  "password"          : "cassandra_password", // "cassandra"
  "include_keyspaces" : [""],
  "exclude_keyspaces" : [ "system_schema", "system_distributed", "system_auth", "system", "system_traces" ],
  "save_users"        : true,
  "bindir"            : "/var/vcap/jobs/cassandra/bin", 
  "datadir"           : "/var/vcap/store/cassandra/data" 
}
`,
		Fields: []plugin.Field{
			{
				Mode:     "target",
				Name:     "host",
				Type:     "string",
				Title:    "Cassandra Host",
				Help:     "The hostname or IP address of your Cassandra server.",
				Default:  "127.0.0.1",
				Required: false,
			},
			{
				Mode:     "target",
				Name:     "port",
				Type:     "port",
				Title:    "Cassandra Port",
				Help:     "The 'native transport' TCP port that Cassandra server is bound to, listening for incoming connections.",
				Default:  "9042",
				Required: false,
			},
			{
				Mode:     "target",
				Name:     "user",
				Type:     "string",
				Title:    "Cassandra Username",
				Help:     "Username to authenticate to Cassandra as.",
				Default:  "cassandra",
				Required: false,
			},
			{
				Mode:     "target",
				Name:     "cassandra",
				Type:     "password",
				Title:    "Cassandra Password",
				Help:     "Password to auth_userenv",
				Default:  "cassandra",
				Required: false,
			},
			{
				Mode:     "target",
				Name:     "include_keyspaces",
				Type:     "array",
				Title:    "Keyspaces to Include in the Backup",
				Help:     "The name of the keyspace to include in the backup.",
				Example:  "['system']",
				Required: false,
			},
			{
				Mode:     "target",
				Name:     "exclude_keyspaces",
				Type:     "array",
				Title:    "Keyspaces to Exclude from Backup",
				Help:     "The name of the keyspace to exclude from backup.",
				Example:  "[system']",
				Default:  "['system_schema', 'system_distributed', 'system_auth', 'system', 'system_traces']",
				Required: false,
			},
			{
				Mode:     "target",
				Name:     "save_users",
				Type:     "boolean",
				Title:    "to restore users and permissions along with the keyspaces that are restored",
				Help:     "The name of the keyspace to exclude from backup.",
				Example:  "true or false",
				Required: false,
			},
			{
				Mode:     "target",
				Name:     "bindir",
				Type:     "abspath",
				Title:    "Path to binaries nodetool, sstableloader and cqlsh needed for the backup",
				Help:     "where  to find binaries nodetool, sstableloader and cqlsh",
				Example:  "true or false",
				Default:  "/var/vcap/jobs/cassandra/bin",
				Required: false,
			},
			{
				Mode:     "target",
				Name:     "datadir",
				Type:     "abspath",
				Title:    "Path to Cassandra data/ directory",
				Help:     "The absolute path to the data/ directory that contains the Cassandra database files.",
				Default:  "/var/vcap/store/cassandra/data",
				Required: false,
			},
		},
	}

	plugin.Run(p)
}

type CassandraPlugin plugin.PluginInfo

type CassandraInfo struct {
	Host             string
	Port             string
	User             string
	Password         string
	IncludeKeyspaces []string
	ExcludeKeyspaces []string
	SaveUsers        bool
	BinDir  string
	DataDir string
}

// This function should be used to return the plugin's PluginInfo, however you decide to implement it
func (p CassandraPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

// Called to validate endpoints from the command line
func (p CassandraPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		a    []string
		s    string
		err  error
		fail bool
		b    bool
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

	a, err = endpoint.ArrayValueDefault("include_keyspaces", nil)
	if err != nil {
		fmt.Printf("@R{\u2717 include_keyspaces      %s}\n", err)
		fail = true
	} else if a == nil {
		fmt.Printf("@G{\u2713 include_keyspaces}      backing up *all* keyspaces\n")
	} else {
		fmt.Printf("@G{\u2713 include_keyspaces}      @C{%v}\n", a)
	}

	a, err = endpoint.ArrayValueDefault("exclude_keyspace", DefaultExcludeKeyspaces)
	if err != nil {
		fmt.Printf("@R{\u2717 exclude_keyspaces      %s}\n", err)
		fail = true
	} else if len(a) == 0 {
		fmt.Printf("@G{\u2713 exclude_keyspaces}      including *all* keyspaces\n")
	} else {
		fmt.Printf("@G{\u2713 exclude_keyspaces}      @C{%v}\n", a)
	}

	b, err = endpoint.BooleanValueDefault("save_users", DefaultSaveUsers)
	if err != nil {
		fmt.Printf("@R{\u2717 save_users      %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 save_users}      @C{%t}\n", b)
	}

	s, err = endpoint.StringValueDefault("bindir", "")
	if err != nil {
		fmt.Printf("@R{\u2717 bindir          %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 bindir}          using default @C{%s}\n", DefaultBinDir)
	} else {
		fmt.Printf("@G{\u2713 bindir}          @C{%s}\n", s)
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

func computeSavedKeyspaces(includeKeyspaces, excludeKeyspaces []string) []string {
	if includeKeyspaces == nil {
		return nil
	}
	savedKeyspaces := []string{}

	sort.Strings(excludeKeyspaces)
	for _, keyspace := range includeKeyspaces {
		idx := sort.SearchStrings(excludeKeyspaces, keyspace)
		if idx < len(excludeKeyspaces) && excludeKeyspaces[idx] == keyspace {
			continue
		}
		savedKeyspaces = append(savedKeyspaces, keyspace)
	}
	sort.Strings(savedKeyspaces)

	return savedKeyspaces
}

// Backup one cassandra keyspace
func (p CassandraPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	cassandra, err := cassandraInfo(endpoint)
	if err != nil {
		return err
	}

	plugin.DEBUG("Cleaning any stale '%s' snapshot", SnapshotName)
	cmd := fmt.Sprintf("%s/nodetool clearsnapshot -t %s", DefaultBinDir, SnapshotName)
	plugin.DEBUG("Executing: `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDIN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Clean up any stale snapshot}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Clean any stale snapshot}\n")

	defer func() {
		plugin.DEBUG("Clearing snapshot '%s'", SnapshotName)
		cmd := fmt.Sprintf("%s/nodetool clearsnapshot -t %s", DefaultBinDir,SnapshotName)
		plugin.DEBUG("Executing: `%s`", cmd)
		err := plugin.Exec(cmd, plugin.STDIN)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Clear snapshot}\n")
			return
		}
		fmt.Fprintf(os.Stderr, "@G{\u2713 Clear snapshot}\n")
	}()

	sort.Strings(cassandra.ExcludeKeyspaces)
	savedKeyspaces := computeSavedKeyspaces(cassandra.IncludeKeyspaces, cassandra.ExcludeKeyspaces)

	plugin.DEBUG("Creating a new '%s' snapshot", SnapshotName)
	cmd = fmt.Sprintf("%s/nodetool snapshot -t %s", DefaultBinDir, SnapshotName)
	if savedKeyspaces != nil {
		for _, keyspace := range savedKeyspaces {
			cmd = fmt.Sprintf("%s \"%s\"", cmd, keyspace)
		}
	}
	plugin.DEBUG("Executing: `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDIN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Create new snapshot}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Create new snapshot}\n")

	// Here we need to copy the snapshots/shield-backup directories into a
	// {keyspace}/{tablename} structure that we'll temporarily put in
	// /var/vcap/store/shield/cassandra. Then we can tar it all and stream
	// that to stdout.

	baseDir := "/var/vcap/store/shield/cassandra"

	// Recursively remove /var/vcap/store/shield/cassandra, if any
	plugin.DEBUG("Removing any stale '%s' directory", baseDir)
	cmd = fmt.Sprintf("rm -rf \"%s\"", baseDir)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDOUT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Clean up any stale base temporary directory}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Clean up any stale base temporary directory}\n")

	plugin.DEBUG("Creating base directories for '%s', with 0755 permissions", baseDir)
	err = os.MkdirAll(baseDir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Create base temporary directory}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Create basemporary directory}\n")

	defer func() {
		// Recursively remove /var/vcap/store/shield/cassandra directory
		plugin.DEBUG("Cleaning the '%s' directory up", baseDir)
		cmd := fmt.Sprintf("rm -rf \"%s\"", baseDir)
		plugin.DEBUG("Executing `%s`", cmd)
		err := plugin.Exec(cmd, plugin.STDOUT)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Clear base temporary directory}\n")
			return
		}
		fmt.Fprintf(os.Stderr, "@G{\u2713 Clear base temporary directory}\n")
	}()

	// Iterate through {dataDir}/{keyspace}/{tablename}/snapshots/shield-backup/*
	// and for all the immutable files we find here, we hard-link them
	// to /var/vcap/store/shield/cassandra/{keyspace}/{tablename}
	//
	// We chose to hard-link because copying those immutable files is
	// unnecessary anyway. It could lead to performance issues and would
	// consume twice the disk space it should.

	info, err := os.Lstat(cassandra.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Recursive hard-link snapshot files in temp dir}\n")
		return err
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Recursive hard-link snapshot files in temp dir}\n")
		return fmt.Errorf("cassandra DataDir is not a directory")
	}

	dir, err := os.Open(cassandra.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Recursive hard-link snapshot files in temp dir}\n")
		return err
	}
	defer dir.Close()

	entries, err := dir.Readdir(-1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Recursive hard-link snapshot files in temp dir}\n")
		return err
	}
	for _, keyspaceDirInfo := range entries {
		if !keyspaceDirInfo.IsDir() {
			continue
		}
		keyspace := keyspaceDirInfo.Name()
		if savedKeyspaces == nil {
			idx := sort.SearchStrings(cassandra.ExcludeKeyspaces, keyspace)
			if idx < len(cassandra.ExcludeKeyspaces) && cassandra.ExcludeKeyspaces[idx] == keyspace {
				plugin.DEBUG("Excluding keyspace '%s'", keyspace)
				continue
			}
		} else {
			idx := sort.SearchStrings(savedKeyspaces, keyspace)
			if idx >= len(savedKeyspaces) || savedKeyspaces[idx] != keyspace {
				plugin.DEBUG("Excluding keyspace '%s'", keyspace)
				continue
			}
		}
		err = hardLinkKeyspace(cassandra.DataDir, baseDir, keyspace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Recursive hard-link snapshot files in temp dir}\n")
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Recursive hard-link snapshot files in temp dir}\n")

	if cassandra.SaveUsers {
		err = backupUsers(cassandra, baseDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2713 Backup users}\n")
			return err
		}
		fmt.Fprintf(os.Stderr, "@G{\u2713 Backup users}\n")
	}

	plugin.DEBUG("Setting ownership of all backup files to '%s'", VcapOwnership)
	cmd = fmt.Sprintf("chown -R vcap:vcap \"%s\"", baseDir)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDOUT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Set ownership of snapshot hard-links}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Set ownership of snapshot hard-links}\n")

	plugin.DEBUG("Streaming output tar file")
	cmd = fmt.Sprintf("tar -c -C %s -f - .", baseDir)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDOUT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Stream tar of snapshots files}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Stream tar of snapshots files}\n")

	return nil
}

func hardLinkKeyspace(srcDataDir string, dstBaseDir string, keyspace string) error {
	tmpKeyspaceDir := filepath.Join(dstBaseDir, keyspace)
	plugin.DEBUG("Creating destination keyspace directory '%s' with 0700 permissions", tmpKeyspaceDir)
	err := os.Mkdir(tmpKeyspaceDir, 0700)
	if err != nil {
		return err
	}

	srcKeyspaceDir := filepath.Join(srcDataDir, keyspace)
	dir, err := os.Open(srcKeyspaceDir)
	if err != nil {
		return err
	}
	defer dir.Close()

	entries, err := dir.Readdir(-1)
	if err != nil {
		return err
	}
	for _, tableDirInfo := range entries {
		if !tableDirInfo.IsDir() {
			continue
		}

		srcDir := filepath.Join(srcKeyspaceDir, tableDirInfo.Name(), "snapshots", SnapshotName)
		_, err = os.Lstat(srcDir)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}

		tableName := tableDirInfo.Name()
		if idx := strings.LastIndex(tableName, "-"); idx >= 0 {
			tableName = tableName[:idx]
		}

		dstDir := filepath.Join(tmpKeyspaceDir, tableName)
		plugin.DEBUG("Creating destination table directory '%s'", dstDir)
		err = os.MkdirAll(dstDir, 0755)
		if err != nil {
			return err
		}

		plugin.DEBUG("Hard-linking all '%s/*' files to '%s/'", srcDir, dstDir)
		err = hardLinkAll(srcDir, dstDir)
		if err != nil {
			return err
		}
	}
	return nil
}

// Hard-link all files from 'srcDir' to the 'dstDir'
func hardLinkAll(srcDir string, dstDir string) (err error) {

	dir, err := os.Open(srcDir)
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
		src := filepath.Join(srcDir, tableDirInfo.Name())
		dst := filepath.Join(dstDir, tableDirInfo.Name())

		err = os.Link(src, dst)
		if err != nil {
			return err
		}
	}
	return nil
}

func backupUsers(cassandra *CassandraInfo, baseDir string) error {
	for _, table := range SystemAuthTables {
		plugin.DEBUG("Saving cassandra %s", table)
		cmd := fmt.Sprintf("%s/cqlsh -u \"%s\" -p \"%s\" -e \"COPY system_auth.%s TO '%s/system_auth.%s.csv' WITH HEADER=true;\" \"%s\"",
			DefaultBinDir, cassandra.User, cassandra.Password, table, baseDir, table, cassandra.Host)
		plugin.DEBUG("Executing `%s`", cmd)
		err := plugin.Exec(cmd, plugin.NOPIPE)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Saving cassandra %s}\n", table)
			return err
		}
		fmt.Fprintf(os.Stderr, "@G{\u2713 Saving cassandra %s}\n", table)
	}
	return nil
}

// Restore one cassandra keyspace
func (p CassandraPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	cassandra, err := cassandraInfo(endpoint)
	if err != nil {
		return err
	}

	baseDir := "/var/vcap/store/shield/cassandra"

	// Recursively remove /var/vcap/store/shield/cassandra, if any
	cmd := fmt.Sprintf("rm -rf \"%s\"", baseDir)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDOUT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Clean up any stale base temporary directory}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Clean up any stale base temporary directory}\n")

	plugin.DEBUG("Creating directory '%s' with 0755 permissions", baseDir)
	err = os.MkdirAll(baseDir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Create base temporary directory}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Create base temporary directory}\n")

	defer func() {
		// Recursively remove /var/vcap/store/shield/cassandra, if any
		cmd := fmt.Sprintf("rm -rf \"%s\"", baseDir)
		plugin.DEBUG("Executing `%s`", cmd)
		err := plugin.Exec(cmd, plugin.STDOUT)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Clear base temporary directory}\n")
			return
		}
		fmt.Fprintf(os.Stderr, "@G{\u2713 Clear base temporary directory}\n")
	}()

	sort.Strings(cassandra.ExcludeKeyspaces)
	savedKeyspaces := computeSavedKeyspaces(cassandra.IncludeKeyspaces, cassandra.ExcludeKeyspaces)

	// TODO: here we should extract only the necessary keyspaces
	cmd = fmt.Sprintf("tar -x -C %s -f -", baseDir)
	plugin.DEBUG("Executing `%s`", cmd)
	err = plugin.Exec(cmd, plugin.STDIN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Extract tar to temporry directory}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Extract tar to temporary directory}\n")

	dir, err := os.Open(baseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Load tables data}\n")
		return err
	}
	defer dir.Close()

	entries, err := dir.Readdir(-1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Load tables data}\n")
		return err
	}
	for _, keyspaceDirInfo := range entries {
		if !keyspaceDirInfo.IsDir() {
			continue
		}
		keyspace := keyspaceDirInfo.Name()
		if savedKeyspaces == nil {
			idx := sort.SearchStrings(cassandra.ExcludeKeyspaces, keyspace)
			if idx < len(cassandra.ExcludeKeyspaces) && cassandra.ExcludeKeyspaces[idx] == keyspace {
				plugin.DEBUG("Excluding keyspace '%s'", keyspace)
				continue
			}
		} else {
			idx := sort.SearchStrings(savedKeyspaces, keyspace)
			if idx >= len(savedKeyspaces) || savedKeyspaces[idx] != keyspace {
				plugin.DEBUG("Excluding keyspace '%s'", keyspace)
				continue
			}
		}
		keyspaceDirPath := filepath.Join(baseDir, keyspace)
		err = restoreKeyspace(cassandra, keyspaceDirPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Load tables data for keyspace '%s'}\n", keyspace)
			return err
		}
		fmt.Fprintf(os.Stderr, "@G{\u2713 Load tables data for keyspace '%s'}\n", keyspace)
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Load tables data}\n")

	if cassandra.SaveUsers {
		err = restoreUsers(cassandra, baseDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Restore users}\n")
			return err
		}
		fmt.Fprintf(os.Stderr, "@G{\u2713 Restore users}\n")
	}

	return nil
}

func restoreKeyspace(cassandra *CassandraInfo, keyspaceDirPath string) error {
	// Iterate through all table directories /var/vcap/store/shield/cassandra/{cassandra.IncludeKeyspaces}/{tablename}
	dir, err := os.Open(keyspaceDirPath)
	if err != nil {
		return err
	}
	defer dir.Close()

	entries, err := dir.Readdir(-1)
	if err != nil {
		return err
	}
	for _, tableDirInfo := range entries {
		if !tableDirInfo.IsDir() {
			continue
		}
		// Run sstableloader on each sub-directory found, assuming it is a table backup
		tableDirPath := filepath.Join(keyspaceDirPath, tableDirInfo.Name())
		cmd := fmt.Sprintf("%s/sstableloader -u \"%s\" -pw \"%s\" -d \"%s\" \"%s\"", DefaultBinDir, cassandra.User, cassandra.Password, cassandra.Host, tableDirPath)
		plugin.DEBUG("Executing: `%s`", cmd)
		err = plugin.Exec(cmd, plugin.STDIN)
		if err != nil {
			return err
		}
	}
	return nil
}

func restoreUsers(cassandra *CassandraInfo, baseDir string) error {
	plugin.DEBUG("Excluding cassandra user from 'system_auth.roles' table content")
	cmd := fmt.Sprintf("sed -i -e '/^cassandra,/d' \"%s/system_auth.roles.csv\"", baseDir)
	plugin.DEBUG("Executing: `%s`", cmd)
	err := plugin.Exec(cmd, plugin.STDIN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{\u2717 Exclude cassandra user from 'system_auth.roles' table content}\n")
		return err
	}
	fmt.Fprintf(os.Stderr, "@G{\u2713 Exclude cassandra user from 'system_auth.roles' table content}\n")

	for _, table := range SystemAuthTables {
		plugin.DEBUG("Restoring 'system_auth.%s' table content", table)
		cmd := fmt.Sprintf("%s/cqlsh -u \"%s\" -p \"%s\" -e \"COPY system_auth.%s FROM '%s/system_auth.%s.csv' WITH HEADER=true;\" \"%s\"",
			DefaultBinDir, cassandra.User, cassandra.Password, table, baseDir, table, cassandra.Host)
		plugin.DEBUG("Executing: `%s`", cmd)
		err := plugin.Exec(cmd, plugin.STDIN)
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{\u2717 Restore 'system_auth.%s' table content}\n", table)
			return err
		}
		fmt.Fprintf(os.Stderr, "@G{\u2713 Restore 'system_auth.%s' table content}\n", table)
	}
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

	includeKeyspace, err := endpoint.ArrayValueDefault("include_keyspaces", nil)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("INCLUDE_KEYSPACES: [%v]", includeKeyspace)

	excludeKeyspace, err := endpoint.ArrayValueDefault("exclude_keyspaces", DefaultExcludeKeyspaces)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("EXCLUDE_KEYSPACES: [%v]", excludeKeyspace)

	saveUsers, err := endpoint.BooleanValueDefault("save_users", DefaultSaveUsers)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("SAVE_USERS: %t", saveUsers)

	datadir, err := endpoint.StringValueDefault("datadir", DefaultDataDir)
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("DATADIR: '%s'", datadir)

	return &CassandraInfo{
		Host:             host,
		Port:             port,
		User:             user,
		Password:         password,
		IncludeKeyspaces: includeKeyspace,
		ExcludeKeyspaces: excludeKeyspace,
		SaveUsers:        saveUsers,
		DataDir:          datadir,
	}, nil
}
