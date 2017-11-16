package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"time"

	fmt "github.com/jhunt/go-ansi"
	"github.com/mattn/go-isatty"
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

func bailon(pre string, err error) {
	if err != nil {
		bail(fmt.Errorf("%s: %s\n", pre, err))
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

func strftime(t int64) string {
	f := os.Getenv("SHIELD_DATE_FORMAT")
	if f == "" {
		f = "2006-01-02 15:04:05-0700"
	}
	return time.Unix(t, 0).Format(f)
}

func strftimenil(t int64, ifnil string) string {
	if t == 0 {
		return ifnil
	}
	return strftime(t)
}
