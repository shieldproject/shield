package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jhunt/ansi"
)

var (
	flagsToPrint []flagInfo = []flagInfo{}
	flagLen      int        = 0
	jsonToPrint  string     = ""
)

type flagInfo struct {
	flag []string
	desc string
}

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
	tmpBuf := bytes.Buffer{}
	err := json.Indent(&tmpBuf, []byte(raw), "", "  ")
	if err != nil {
		DEBUG("json.Indent failed with %s", err)
		return raw
	}
	return tmpBuf.String()
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

//Prints out all the aliases for the given command in the help format.
func PrintAliasHelp(input string, c *Command) {
	aliasString := c.AliasesFor(input)
	if aliasString != "" {
		ansi.Fprintf(os.Stderr, "@R{ALIASES}\n")
		ansi.Fprintf(os.Stderr, "\t%s\n", aliasString)
	}
}

//Prints out a Header of the given string in the help format.
func Header(text string) {
	ansi.Fprintf(os.Stderr, "@R{%s}\n", text)
}

//Sorts flagInfo objects by their... flag, dashes excluded. --Z > -a
type ByFlag []flagInfo

func (f ByFlag) Len() int      { return len(f) }
func (f ByFlag) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
func (f ByFlag) Less(i, j int) bool {
	return strings.Replace(f[i].flag[0], "-", "", 2) < strings.Replace(f[j].flag[0], "-", "", 2)
}

//Adds a flag to be printed in a call to PrintFlagHelp
//The list of flags given will be printed comma-separated
func FlagHelp(desc string, flags ...string) {
	if len(flags) == 0 {
		panic("No flag specified to FlagHelp")
	}
	flagsToPrint = append(flagsToPrint, flagInfo{flags, desc})
	//Calc longest flag
	thisLen := len(strings.Join(flags, ", "))
	if thisLen > flagLen {
		flagLen = thisLen
	}
}

//Prints the queued list of flags in the help format
func PrintFlagHelp() {
	if len(flagsToPrint) == 0 {
		return
	}
	Header("FLAGS")
	totalSpacing := flagLen + 2

	sort.Sort(ByFlag(flagsToPrint))
	for _, flag := range flagsToPrint {
		//Parse out each newline separated flags
		//Wrap each of the flag entries in @M{} to highlight them blue
		flagString := "@M{" + strings.Join(flag.flag, "}, @M{") + "}"
		space := totalSpacing - len(strings.Join(flag.flag, ", "))
		ansi.Fprintf(os.Stderr, flagString+"%s%s\n", strings.Repeat(" ", space), flag.desc)
	}
}

//Sets the JSON object to be printed
func JSONHelp(j string) {
	jsonToPrint = j
}

//Prints the JSON previously queued in the help format.
func PrintJSONHelp() {
	if jsonToPrint != "" {
		Header("JSON")
		ansi.Fprintf(os.Stderr, "%s\n", PrettyJSON(jsonToPrint))
	}
}
