package pathenv

import "os"

// Path returns the PATH environement variable for scripts.
func Path() string { return os.Getenv("PATH") }
