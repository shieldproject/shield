package directories

import "path"

type Provider struct {
	baseDir string
}

func NewProvider(baseDir string) Provider {
	return Provider{baseDir}
}

func (p Provider) BaseDir() string {
	return p.baseDir
}

func (p Provider) BoshDir() string {
	return path.Join(p.BaseDir(), "bosh")
}

func (p Provider) BoshBinDir() string {
	return path.Join(p.BoshDir(), "bin")
}

func (p Provider) EtcDir() string {
	return path.Join(p.BoshDir(), "etc")
}

func (p Provider) StoreDir() string {
	return path.Join(p.BaseDir(), "store")
}

func (p Provider) DataDir() string {
	return path.Join(p.BaseDir(), "data")
}

func (p Provider) StoreMigrationDir() string {
	return path.Join(p.BaseDir(), "store_migration_target")
}

func (p Provider) PkgDir() string {
	return path.Join(p.DataDir(), "packages")
}

func (p Provider) CompileDir() string {
	return path.Join(p.DataDir(), "compile")
}

func (p Provider) MonitJobsDir() string {
	return path.Join(p.BaseDir(), "monit", "job")
}

func (p Provider) MonitDir() string {
	return path.Join(p.BaseDir(), "monit")
}

func (p Provider) JobsDir() string {
	return path.Join(p.BaseDir(), "jobs")
}

func (p Provider) JobBinDir(jobName string) string {
	return path.Join(p.JobsDir(), jobName, "bin")
}

func (p Provider) MicroStore() string {
	return path.Join(p.BaseDir(), "micro_bosh", "data", "cache")
}

func (p Provider) SettingsDir() string {
	return path.Join(p.BoshDir(), "settings")
}

func (p Provider) TmpDir() string {
	return path.Join(p.DataDir(), "tmp")
}

func (p Provider) LogsDir() string {
	return path.Join(p.BaseDir(), "sys", "log")
}

func (p Provider) AgentLogsDir() string {
	return path.Join(p.BaseDir(), "bosh", "log")
}

func (p Provider) InstanceDir() string {
	return path.Join(p.BaseDir(), "instance")
}
