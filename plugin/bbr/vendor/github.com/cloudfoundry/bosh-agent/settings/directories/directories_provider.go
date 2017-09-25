package directories

import "path/filepath"

type Provider struct {
	baseDir string
}

func NewProvider(baseDir string) Provider {
	return Provider{baseDir: filepath.Clean(baseDir)}
}

func (p Provider) BaseDir() string {
	return p.baseDir
}

func (p Provider) BoshDir() string {
	return filepath.Join(p.BaseDir(), "bosh")
}

func (p Provider) BoshBinDir() string {
	return filepath.Join(p.BoshDir(), "bin")
}

func (p Provider) EtcDir() string {
	return filepath.Join(p.BoshDir(), "etc")
}

func (p Provider) StoreDir() string {
	return filepath.Join(p.BaseDir(), "store")
}

func (p Provider) DataDir() string {
	return filepath.Join(p.BaseDir(), "data")
}

func (p Provider) StoreMigrationDir() string {
	return filepath.Join(p.BaseDir(), "store_migration_target")
}

func (p Provider) PkgDir() string {
	return filepath.Join(p.DataDir(), "packages")
}

func (p Provider) CompileDir() string {
	return filepath.Join(p.DataDir(), "compile")
}

func (p Provider) MonitJobsDir() string {
	return filepath.Join(p.BaseDir(), "monit", "job")
}

func (p Provider) MonitDir() string {
	return filepath.Join(p.BaseDir(), "monit")
}

func (p Provider) JobsDir() string {
	return filepath.Join(p.BaseDir(), "jobs")
}

func (p Provider) JobBinDir(jobName string) string {
	return filepath.Join(p.JobsDir(), jobName, "bin")
}

func (p Provider) MicroStore() string {
	return filepath.Join(p.BaseDir(), "micro_bosh", "data", "cache")
}

func (p Provider) SettingsDir() string {
	return filepath.Join(p.BoshDir(), "settings")
}

func (p Provider) TmpDir() string {
	return filepath.Join(p.DataDir(), "tmp")
}

func (p Provider) LogsDir() string {
	return filepath.Join(p.BaseDir(), "sys", "log")
}

func (p Provider) AgentLogsDir() string {
	return filepath.Join(p.BaseDir(), "bosh", "log")
}

func (p Provider) InstanceDir() string {
	return filepath.Join(p.BaseDir(), "instance")
}

func (p Provider) DisksDir() string {
	return filepath.Join(p.InstanceDir(), "disks")
}

func (p Provider) InstanceDNSDir() string {
	return filepath.Join(p.InstanceDir(), "dns")
}

func (p Provider) BlobsDir() string {
	return filepath.Join(p.DataDir(), "blobs")
}
