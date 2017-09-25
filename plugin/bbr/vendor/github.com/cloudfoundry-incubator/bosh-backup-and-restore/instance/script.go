package instance

import (
	"path/filepath"
	"strings"
)

type Script string

const (
	backupScriptName            = "backup"
	restoreScriptName           = "restore"
	metadataScriptName          = "metadata"
	preBackupLockScriptName     = "pre-backup-lock"
	postBackupUnlockScriptName  = "post-backup-unlock"
	postRestoreUnlockScriptName = "post-restore-unlock"

	jobBaseDirectory               = "/var/vcap/jobs/"
	jobDirectoryMatcher            = jobBaseDirectory + "*/bin/bbr/"
	backupScriptMatcher            = jobDirectoryMatcher + backupScriptName
	restoreScriptMatcher           = jobDirectoryMatcher + restoreScriptName
	metadataScriptMatcher          = jobDirectoryMatcher + metadataScriptName
	preBackupLockScriptMatcher     = jobDirectoryMatcher + preBackupLockScriptName
	postBackupUnlockScriptMatcher  = jobDirectoryMatcher + postBackupUnlockScriptName
	postRestoreUnlockScriptMatcher = jobDirectoryMatcher + postRestoreUnlockScriptName
)

func (s Script) isBackup() bool {
	match, _ := filepath.Match(backupScriptMatcher, string(s))
	return match
}

func (s Script) isRestore() bool {
	match, _ := filepath.Match(restoreScriptMatcher, string(s))
	return match
}

func (s Script) isMetadata() bool {
	match, _ := filepath.Match(metadataScriptMatcher, string(s))
	return match
}

func (s Script) isPreBackupUnlock() bool {
	match, _ := filepath.Match(preBackupLockScriptMatcher, string(s))
	return match
}

func (s Script) isPostBackupUnlock() bool {
	match, _ := filepath.Match(postBackupUnlockScriptMatcher, string(s))
	return match
}

func (s Script) isPostRestoreUnlock() bool {
	match, _ := filepath.Match(postRestoreUnlockScriptMatcher, string(s))
	return match
}

func (s Script) isPlatformScript() bool {
	return s.isBackup() ||
		s.isRestore() ||
		s.isPreBackupUnlock() ||
		s.isPostBackupUnlock() ||
		s.isPostRestoreUnlock() ||
		s.isMetadata()
}

func (s Script) splitPath() []string {
	strippedPrefix := strings.TrimPrefix(string(s), jobBaseDirectory)
	splitFirstElement := strings.SplitN(strippedPrefix, "/", 4)
	return splitFirstElement
}

func (s Script) JobName() string {
	pathSplit := s.splitPath()
	return pathSplit[0]
}

func (script Script) Name() string {
	pathSplit := script.splitPath()
	return pathSplit[len(pathSplit)-1]
}
