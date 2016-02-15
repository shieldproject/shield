package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func BoolString(tf bool) string {
	if tf {
		return "Y"
	}
	return "N"
}

func CurrentUser() string {
	return fmt.Sprintf("%s@%s", os.Getenv("USER"), os.Getenv("HOSTNAME"))
}

func PrettyJSON(raw string) string {
	var v interface{}
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		DEBUG("json.Unmarshal failed with %s", err)
		return raw
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		DEBUG("json.MarshalIndent failed with %s", err)
		return raw
	}
	return string(b)
}

func DEBUG(format string, args ...interface{}) {
	if debug {
		content := fmt.Sprintf(format, args...)
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			lines[i] = "DEBUG> " + line
		}
		content = strings.Join(lines, "\n")
		fmt.Fprintf(os.Stderr, "%s\n", content)
	}
}
