package platform

import (
	"crypto/rand"
	"encoding/ascii85"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/cloudfoundry/bosh-agent/jobsupervisor/winsvc"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

var (
	userenv  = windows.NewLazySystemDLL("userenv.dll")
	netapi32 = windows.NewLazySystemDLL("Netapi32.dll")

	procCreateProfile        = userenv.NewProc("CreateProfile")
	procDeleteProfile        = userenv.NewProc("DeleteProfileW")
	procGetProfilesDirectory = userenv.NewProc("GetProfilesDirectoryW")
	procNetUserEnum          = netapi32.NewProc("NetUserEnum")
)

// createProfile, creates the profile and home directory of the user identified
// by Security Identifier sid.
func createProfile(sid, username string) (string, error) {
	const S_OK = 0x00000000
	if err := procCreateProfile.Find(); err != nil {
		return "", err
	}
	psid, err := syscall.UTF16PtrFromString(sid)
	if err != nil {
		return "", err
	}
	pusername, err := syscall.UTF16PtrFromString(username)
	if err != nil {
		return "", err
	}
	var pathbuf [260]uint16
	r1, _, e1 := syscall.Syscall6(procCreateProfile.Addr(), 4,
		uintptr(unsafe.Pointer(psid)),        // _In_  LPCWSTR pszUserSid
		uintptr(unsafe.Pointer(pusername)),   // _In_  LPCWSTR pszUserName
		uintptr(unsafe.Pointer(&pathbuf[0])), // _Out_ LPWSTR  pszProfilePath
		uintptr(len(pathbuf)),                // _In_  DWORD   cchProfilePath
		0, // unused
		0, // unused
	)
	if r1 != S_OK {
		if e1 == 0 {
			return "", os.NewSyscallError("CreateProfile", syscall.EINVAL)
		}
		return "", os.NewSyscallError("CreateProfile", e1)
	}
	profilePath := syscall.UTF16ToString(pathbuf[0:])
	return profilePath, nil
}

// deleteProfile, deletes the profile and home directory of the user identified
// by Security Identifier sid.
func deleteProfile(sid string) error {
	if err := procDeleteProfile.Find(); err != nil {
		return err
	}
	psid, err := syscall.UTF16PtrFromString(sid)
	if err != nil {
		return err
	}
	r1, _, e1 := syscall.Syscall(procDeleteProfile.Addr(), 3,
		uintptr(unsafe.Pointer(psid)), // _In_     LPCTSTR lpSidString,
		0, // _In_opt_ LPCTSTR lpProfilePath,
		0, // _In_opt_ LPCTSTR lpComputerName
	)
	if r1 == 0 {
		if e1 == 0 {
			return os.NewSyscallError("DeleteProfile", syscall.EINVAL)
		}
		return os.NewSyscallError("DeleteProfile", e1)
	}
	return nil
}

// getProfilesDirectory, returns the path to the root directory where user
// profiles are stored (typically C:\Users).
func getProfilesDirectory() (string, error) {
	if err := procGetProfilesDirectory.Find(); err != nil {
		return "", err
	}
	var buf [syscall.MAX_PATH]uint16
	n := uint32(len(buf))
	r1, _, e1 := syscall.Syscall(procGetProfilesDirectory.Addr(), 2,
		uintptr(unsafe.Pointer(&buf[0])), // _Out_   LPTSTR  lpProfilesDir,
		uintptr(unsafe.Pointer(&n)),      // _Inout_ LPDWORD lpcchSize
		0,
	)
	if r1 == 0 {
		if e1 == 0 {
			return "", os.NewSyscallError("GetProfilesDirectory", syscall.EINVAL)
		}
		return "", os.NewSyscallError("GetProfilesDirectory", e1)
	}
	s := syscall.UTF16ToString(buf[0:])
	return s, nil
}

// userHomeDirectory returns the home directory for user username.  An error
// is returned is the user profiles directory cannot be found or if the home
// directory is invalid.
//
// This is a minimal implementation that relies upon Windows naming home
// directories after user names (i.e. the home directory of user "foo" is
// C:\Users\foo).  This is the typical behavior when creating local users
// but is not guaranteed.
//
// A more complete implementation may be possible with the LoadUserProfile
// syscall.
func userHomeDirectory(username string) (string, error) {
	path, err := getProfilesDirectory()
	if err != nil {
		return "", err
	}
	home := filepath.Join(path, username)
	fi, err := os.Stat(home) // safe to use os pkg here, len(home) < MAX_PATH
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("not a directory: %s", home)
	}
	return home, nil
}

func isSpecial(c byte) bool {
	return ('!' <= c && c <= '/') || (':' <= c && c <= '@') ||
		('[' <= c && c <= '`') || ('{' <= c && c <= '~')
}

// validPassword, checks if password s meets the Windows complexity
// requirements defined here:
//
//   https://technet.microsoft.com/en-us/library/hh994562(v=ws.11).aspx
//
func validPassword(s string) bool {
	var (
		digits    bool
		special   bool
		alphaLow  bool
		alphaHigh bool
	)
	if len(s) < 8 {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case '0' <= c && c <= '9':
			digits = true
		case 'a' <= c && c <= 'z':
			alphaLow = true
		case 'A' <= c && c <= 'Z':
			alphaHigh = true
		case isSpecial(c):
			special = true
		}
	}
	var n int
	if digits {
		n++
	}
	if special {
		n++
	}
	if alphaLow {
		n++
	}
	if alphaHigh {
		n++
	}
	return n >= 3
}

// generatePassword, returns a 14 char ascii85 encoded password.
//
// DO NOT CALL THIS DIRECTLY, use randomPassword instead as it
// returns a valid Windows password.
func generatePassword() (string, error) {
	const Length = 14

	in := make([]byte, ascii85.MaxEncodedLen(Length))
	if _, err := io.ReadFull(rand.Reader, in); err != nil {
		return "", err
	}
	out := make([]byte, ascii85.MaxEncodedLen(len(in)))
	if n := ascii85.Encode(out, in); n < Length {
		return "", errors.New("short password")
	}

	// replace forward slashes as NET USER does not like them

	var char byte // replacement char
	for _, c := range out {
		if c != '/' {
			char = c
			break
		}
	}
	for i, c := range out {
		if c == '/' {
			out[i] = char
		}
	}
	return string(out[:Length]), nil
}

// randomPassword, returns a ascii85 encoded 14 char password
// if the password is longer than 14 chars NET.exe will ask
// for confirmation due to backwards compatibility issues with
// Windows prior to Windows 2000.
func randomPassword() (string, error) {
	limit := 100
	for ; limit >= 0; limit-- {
		s, err := generatePassword()
		if err != nil {
			return "", err
		}
		if validPassword(s) {
			return s, nil
		}
	}
	return "", errors.New("failed to generate valid Windows password")
}

func userExists(name string) bool {
	_, _, t, err := syscall.LookupSID("", name)
	return err == nil && t == syscall.SidTypeUser
}

func createUserProfile(username string) error {
	if userExists(username) {
		return fmt.Errorf("user account already exists: %s", username)
	}

	// Create local user
	password, err := randomPassword()
	if err != nil {
		return err
	}
	createCmd := exec.Command("NET.exe", "USER", username, password, "/ADD")
	createOut, err := createCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error creating user (%s): %s", err, string(createOut))
	}

	// Add to Administrators group
	groupCmd := exec.Command("NET.exe", "LOCALGROUP", "Administrators", username, "/ADD")
	groupOut, err := groupCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error adding user to Administrator group (%s): %s",
			err, string(groupOut))
	}

	sid, _, _, err := syscall.LookupSID("", username)
	if err != nil {
		return err
	}
	ssid, err := sid.String()
	if err != nil {
		return err
	}
	_, err = createProfile(ssid, username)
	return err
}

func deleteUserProfile(username string) error {
	sid, _, _, err := syscall.LookupSID("", username)
	if err != nil {
		return err
	}
	ssid, err := sid.String()
	if err != nil {
		return err
	}
	if err := deleteProfile(ssid); err != nil {
		return err
	}

	cmd := exec.Command("NET.exe", "USER", username, "/DELETE")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error deleting user (%s): %s", err, string(out))
	}

	return nil
}

func localAccountNames() ([]string, error) {
	const MAX_PREFERRED_LENGTH = 0xffffffff
	const FILTER_NORMAL_ACCOUNT = 2

	if err := procNetUserEnum.Find(); err != nil {
		return nil, err
	}
	var buf *byte
	var (
		read   uint32
		total  uint32
		resume uint32
	)
	r1, _, e1 := syscall.Syscall9(procNetUserEnum.Addr(), 8,
		0, // local computer
		0, // user account names
		FILTER_NORMAL_ACCOUNT,
		uintptr(unsafe.Pointer(&buf)),
		MAX_PREFERRED_LENGTH,
		uintptr(unsafe.Pointer(&read)),
		uintptr(unsafe.Pointer(&total)),
		uintptr(unsafe.Pointer(&resume)),
		0,
	)
	if r1 != 0 {
		if e1 == syscall.ERROR_MORE_DATA {
			// This shouldn't happen, but in case
			// it does we need to free the buffer
			windows.NetApiBufferFree(buf)
		}
		if e1 == 0 {
			return nil, os.NewSyscallError("NetUserEnum", syscall.EINVAL)
		}
		return nil, os.NewSyscallError("NetUserEnum", e1)
	}
	defer windows.NetApiBufferFree(buf)

	type USER_INFO_0 struct {
		Name *uint16
	}
	type sliceHeader struct {
		Data uintptr
		Len  int
		Cap  int
	}
	us := *(*[]USER_INFO_0)(unsafe.Pointer(&sliceHeader{
		Data: uintptr(unsafe.Pointer(buf)),
		Len:  int(read),
		Cap:  int(read),
	}))
	names := make([]string, int(read))
	for i, u := range us {
		names[i] = toString(u.Name)
	}
	return names, nil
}

func toString(p *uint16) string {
	if p == nil {
		return ""
	}
	return syscall.UTF16ToString((*[4096]uint16)(unsafe.Pointer(p))[:])
}

func serviceDisabled(s *mgr.Service) bool {
	conf, err := s.Config()
	return err == nil && conf.StartType == mgr.StartDisabled
}

// Make the function called by GetHostPublicKey configurable for testing.
var sshEnabled func() error = checkSSH

// checkSSH checks if the sshd and ssh-agent services are installed and running.
//
// The services are installed during stemcell creation, but are disabled.  The
// job windows-utilities-release/enable_ssh job is used to enable ssh.
func checkSSH() error {
	const ERROR_SERVICE_DOES_NOT_EXIST syscall.Errno = 0x424

	const msgFmt = "%s service not running and start type is disabled.  " +
		"To enable ssh on Windows you must run the enable_ssh job from the " +
		"windows-utilities-release."

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("opening service control manager: %s", err)
	}
	defer m.Disconnect()

	sshd, err := m.OpenService("sshd")
	if err != nil {
		if err == ERROR_SERVICE_DOES_NOT_EXIST {
			return errors.New("sshd is not installed")
		}
		return fmt.Errorf("opening service sshd: %s", err)
	}
	defer sshd.Close()

	agent, err := m.OpenService("ssh-agent")
	if err != nil {
		if err == ERROR_SERVICE_DOES_NOT_EXIST {
			return errors.New("ssh-agent is not installed")
		}
		return fmt.Errorf("opening service ssh-agent: %s", err)
	}
	defer agent.Close()

	st, err := sshd.Query()
	if err != nil {
		return fmt.Errorf("querying status of service (sshd): %s", err)
	}
	if st.State != svc.Running {
		if serviceDisabled(sshd) {
			return fmt.Errorf(msgFmt, "sshd")
		}
		return errors.New("sshd service is not running")
	}

	// ssh-agent is a dependency of sshd so it should always
	// be running if sshd is running - check just to make sure.
	st, err = agent.Query()
	if err != nil {
		return fmt.Errorf("querying status of service ssh-agent: %s", err)
	}
	if st.State != svc.Running {
		if serviceDisabled(agent) {
			return fmt.Errorf(msgFmt, "ssh-agent")
		}
		return errors.New("ssh-agent service is not running")
	}

	return nil
}

func disableWindowsUpdates() error {
	const path2016 = `SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU`
	const path2012 = `SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update`

	// Stop and Disable Windows Update Service

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("opening service control manager: %s", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService("wuauserv")
	if err != nil {
		return fmt.Errorf("opening Windows Update service: %s", err)
	}
	defer s.Close()

	if err := winsvc.SetStartType(s, mgr.StartDisabled); err != nil {
		return fmt.Errorf("disabling Windows Update service: %s", err)
	}
	if err := winsvc.Stop(s); err != nil {
		return fmt.Errorf("stopping Windows Update service: %s", err)
	}

	// Turn off updates via registry keys

	values := map[string]uint32{
		"AUOptions": 1,
	}

	// Note: always try the 2016 path first as it does not exist on 2012R2,
	// but the 2012R2 path exists on 2016.
	//
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, path2016, registry.ALL_ACCESS)
	switch err {
	case nil:
		// 2016 specific values
		values["NoAutoUpdate"] = 1

	case registry.ErrNotExist:
		// Try 2012R2 key path
		key, err = registry.OpenKey(registry.LOCAL_MACHINE, path2012, registry.ALL_ACCESS)
		if err != nil {
			return fmt.Errorf("opening registry key (%s): %s", path2012, err)
		}

		// 2012R2 specific values
		values["EnableFeatureSoftware"] = 0
		values["IncludeRecommendedUpdates"] = 0

	default:
		return fmt.Errorf("opening registry key (%s): %s", path2016, err)
	}
	defer key.Close()

	for k, v := range values {
		if err := key.SetDWordValue(k, v); err != nil {
			return fmt.Errorf("setting registry key (%s): %s", k, err)
		}
	}

	return nil
}

func setupRuntimeConfiguration() error {
	if err := disableWindowsUpdates(); err != nil {
		return fmt.Errorf("disabling updates: %s", err)
	}
	return nil
}

func setRandomPassword(username string) error {
	if !userExists(username) {
		// Special case, if the Admin account does not exist
		// or is disabled there is no need to randomize it.
		if username == administratorUserName {
			return nil
		}
		return fmt.Errorf("user does not exist: %s", username)
	}
	passwd, err := randomPassword()
	if err != nil {
		return err
	}
	cmd := exec.Command("NET.exe", "USER", username, passwd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error setting password for user (%s): %s", err, string(out))
	}
	return nil
}
