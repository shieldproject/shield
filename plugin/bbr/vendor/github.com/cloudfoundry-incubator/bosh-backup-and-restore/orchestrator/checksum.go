package orchestrator

type BackupChecksum map[string]string

func (self BackupChecksum) Match(other BackupChecksum) bool {
	if len(self) != len(other) {
		return false
	}
	for key, _ := range self {
		if self[key] != other[key] {
			return false
		}
	}
	return true

}
