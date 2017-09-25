// +build !windows

// Package pathenv returns the OS specific PATH environment variable to use
// when shelling out to user scripts (e.g pre-start, drain).
package pathenv

// Path returns the PATH environement variable for scripts.
func Path() string { return "/usr/sbin:/usr/bin:/sbin:/bin" }
