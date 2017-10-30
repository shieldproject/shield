package access

import (
	"bufio"
	"os"
	"strings"

	fmt "github.com/jhunt/go-ansi"
	"github.com/mattn/go-isatty"
	"golang.org/x/crypto/ssh/terminal"
)

func NormalPrompt(label string, args ...interface{}) string {
	fmt.Fprintf(os.Stderr, label, args...)
	s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSuffix(s, "\n")
}

func SecurePrompt(label string, args ...interface{}) string {
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		return strings.TrimSuffix(s, "\n")
	}

	fmt.Fprintf(os.Stderr, label, args...)
	b, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintf(os.Stderr, "\n")
	return string(b)
}
