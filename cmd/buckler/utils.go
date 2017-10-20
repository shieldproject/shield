package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	fmt "github.com/starkandwayne/goutils/ansi"
	"golang.org/x/crypto/ssh/terminal"
)

func fail(rc int, m string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, m, args...)
	os.Exit(rc)
}

func bail(err error) {
	if err != nil {
		if opts.JSON {
			fmt.Fprintf(os.Stderr, "%s\n", asJSON(struct {
				Error string `json:"error"`
			}{
				Error: err.Error(),
			}))
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "@R{!!! %s}\n", err)
		os.Exit(1)
	}
}

func required(ok bool, msg string) {
	if !ok {
		fmt.Fprintf(os.Stderr, "@Y{%s}\n", msg)
		os.Exit(3)
	}
}

func confirm(yes bool, msg string, args ...interface{}) bool {
	if yes {
		return true
	}

	switch prompt(msg+" [y/N] ", args...) {
	case "y", "Y", "yes":
		return true
	default:
		return false
	}
}

func prompt(label string, args ...interface{}) string {
	fmt.Fprintf(os.Stderr, label, args...)
	s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSuffix(s, "\n")
}

func secureprompt(label string, args ...interface{}) string {
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		return strings.TrimSuffix(s, "\n")
	}

	fmt.Fprintf(os.Stderr, label, args...)
	b, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintf(os.Stderr, "\n")
	return string(b)
}

func asJSON(x interface{}) string {
	var raw []byte
	if s, ok := x.(string); ok {
		raw = []byte(s)

	} else if b, ok := x.([]byte); ok {
		raw = b

	} else {
		b, err := json.Marshal(x)
		if err != nil {
			return ""
		}
		raw = b
	}

	tmp := bytes.Buffer{}
	if json.Indent(&tmp, raw, "", " ") != nil {
		return string(raw)
	}
	return tmp.String()
}

func dataConfig(data []string) (map[string]interface{}, error) {
	conf := make(map[string]interface{})
	for _, datum := range data {
		p := strings.SplitN(datum, "=", 2)
		if len(p) < 2 {
			return nil, fmt.Errorf("invalid --data item '%s' (should be key=value format)\n")
		}
		conf[p[0]] = p[1]
	}
	return conf, nil
}
