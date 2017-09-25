// +build windows

package platform

var (
	// Export for testing
	UserHomeDirectory    = userHomeDirectory
	RandomPassword       = randomPassword
	ValidWindowsPassword = validPassword
	LocalAccountNames    = localAccountNames

	// Export for test cleanup
	DeleteUserProfile = deleteUserProfile
)

// SetSSHEnabled sets the function called by GetHostPublicKey to determine if
// ssh is enabled.
func SetSSHEnabled(new func() error) (previous func() error) {
	previous = sshEnabled
	sshEnabled = new
	return previous
}

func SetAdministratorUserName(name string) (previous string) {
	previous = administratorUserName
	administratorUserName = name
	return previous
}
