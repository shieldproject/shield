package access

import (
	"bufio"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/starkandwayne/goutils/ansi"
	"golang.org/x/crypto/ssh/terminal"
)

func NormalPrompt(label string, args ...interface{}) string {
	ansi.Fprintf(os.Stderr, label, args...)
	s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSuffix(s, "\n")
}

func SecurePrompt(label string, args ...interface{}) string {
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		return strings.TrimSuffix(s, "\n")
	}

	ansi.Fprintf(os.Stderr, label, args...)
	b, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
	ansi.Fprintf(os.Stderr, "\n")
	return string(b)
}
