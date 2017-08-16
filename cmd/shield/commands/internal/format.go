package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/starkandwayne/shield/cmd/shield/log"
)

//ErrCanceled is the error returned from commands if the user denied a
//confirmation prompt.
var ErrCanceled = fmt.Errorf("Canceling... ")

//IndentSlice takes a slice of strings and prepends two spaces to each
func IndentSlice(lines []string) []string {
	for i, line := range lines {
		lines[i] = fmt.Sprintf("  %s", line)
	}
	return lines
}

//IndentString appends two spaces to every line of the string and then returns the string
func IndentString(s string) string {
	lines := strings.Split(s, "\n")
	lines = IndentSlice(lines)
	return strings.Join(lines, "\n")
}

//PrettyJSON formats the input JSON with indentation
func PrettyJSON(raw string) string {
	tmpBuf := bytes.Buffer{}
	err := json.Indent(&tmpBuf, []byte(raw), "", "  ")
	if err != nil {
		log.DEBUG("json.Indent failed with %s", err)
		return raw
	}
	return tmpBuf.String()
}

//RawJSON converts the output to JSON and prints it to the screen
//Panics if cannot convert to JSON.
func RawJSON(raw interface{}) {
	b, err := json.Marshal(raw)
	if err != nil {
		panic("Could not convert interface to JSON")
	}

	fmt.Printf("%s\n", string(b))
}

//RawUUID prints the given UUID to stdout.
func RawUUID(uuid string) {
	fmt.Println(uuid)
}
