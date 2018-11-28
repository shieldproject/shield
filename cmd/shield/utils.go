package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
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

func strptime(t string) int64 {
	f := os.Getenv("SHIELD_DATE_FORMAT")
	if f == "" {
		f = "2006-01-02 15:04:05-0700"
	}
	u, err := time.Parse(f, t)
	if err != nil {
		bail(err)
	}
	return u.Unix()
}

func strftimenil(t int64, ifnil string) string {
	if t == 0 {
		return ifnil
	}
	return strftime(t)
}

func parseBytes(in string) (int64, error) {
	if in == "" {
		return 0, nil
	}

	re := regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s?([kmgt])?b?`)
	m := re.FindStringSubmatch(in)
	if m == nil {
		return 0, fmt.Errorf("Invalid size spec '%s' (try something like '100M' or '1.5gb')", in)
	}
	f, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("Invalid size spec '%s' (%s)", in, err)
	}

	switch m[2] {
	case "":
		return int64(f), nil

	case "k", "K":
		return (int64)(f * 1024), nil

	case "m", "M":
		return (int64)(f * 1024 * 1024), nil

	case "g", "G":
		return (int64)(f * 1024 * 1024 * 1024), nil

	case "t", "T":
		return (int64)(f * 1024 * 1024 * 1024 * 1024), nil
	}

	return 0, fmt.Errorf("Invalid size spec '%s'", in)
}

func formatBytes(in int64) string {
	if in < 1024 {
		return fmt.Sprintf("%db", in)
	}
	if in < 1024*1024 {
		return fmt.Sprintf("%0.1fK", float64(in)/1024.0)
	}
	if in < 1024*1024*1024 {
		return fmt.Sprintf("%0.1fM", float64(in)/1024.0/1024.0)
	}
	if in < 1024*1024*1024*1024 {
		return fmt.Sprintf("%0.1fG", float64(in)/1024.0/1024.0/1024.0)
	}
	return fmt.Sprintf("%0.1fT", float64(in)/1024.0/1024.0/1024.0/1024.0)
}

func uuid8(s string) string {
	if len(s) < 8 {
		return s
	}
	return s[0:8]
}
