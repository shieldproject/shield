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
	flagsToPrint   []flagInfo    = []flagInfo{}
	flagLen        int           = 0
	jsonToPrint    string        = ""
	messageToPrint string        = ""
	messageArgs    []interface{} = []interface{}{}
	inputHelpText  string        = ""
)

type flagInfo struct {
	flag     []string
	desc     string
	optional bool
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
	aliases := c.AliasesFor(input)
	aliasString := ""
	if len(aliases) > 1 {
		aliasString = strings.Join(aliases, ", ")
	}

	if aliasString != "" {
		Header("ALIASES")
		ansi.Fprintf(os.Stderr, "  %s\n", aliasString)
	}
}

//Prints out a Header of the given string in the help format.
func Header(text string) {
	ansi.Fprintf(os.Stderr, "\n@R{%s}\n", text)
}

//Sorts flagInfo objects by their... flag, dashes excluded. --Z > -a
type ByFlag []flagInfo

func (f ByFlag) Len() int      { return len(f) }
func (f ByFlag) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
func (f ByFlag) Less(i, j int) bool {
	return strings.ToLower(strings.Replace(f[i].flag[0], "-", "", 2)) <
		strings.ToLower(strings.Replace(f[j].flag[0], "-", "", 2))
}

//Adds a flag to be printed in a call to PrintFlagHelp
//The list of flags given will be printed comma-separated
//Also used for usage
func FlagHelp(desc string, optional bool, flags ...string) {
	if len(flags) == 0 {
		panic("No flag specified to FlagHelp")
	}
	flagsToPrint = append(flagsToPrint, flagInfo{flags, desc, optional})
	//Calc longest flag
	thisLen := len(strings.Join(flags, ", "))
	if thisLen > flagLen {
		flagLen = thisLen
	}
}

//Prints the queued list of flags in the help format
func PrintFlagHelp() {
	//TODO: Make description stay in column when extends past terminal width
	if len(flagsToPrint) == 0 {
		return
	}
	Header("FLAGS")
	sort.Sort(ByFlag(flagsToPrint))
	for _, flags := range flagsToPrint {
		//Parse out each newline separated flags
		//Wrap each of the flag entries in @M{} to highlight them blue
		printFlagHelper(flags.flag, flags.desc)
	}
}

func printFlagHelper(flags []string, desc string) {
	defer func() { ansi.Fprintf(os.Stderr, "\n") }()
	//Print the flag list
	flagString := "@M{" + strings.Join(flags, "}, @M{") + "}"
	space := flagLen - len(strings.Join(flags, ", ")) + 2
	lines := strings.Split(desc, "\n")
	for i, v := range lines {
		lines[i] = strings.Trim(v, " \t")
	}
	ansi.Fprintf(os.Stderr, "  "+flagString+"%s%s", strings.Repeat(" ", space), lines[0])
	if len(lines) <= 1 {
		return
	}
	for _, v := range lines[1:] {
		ansi.Fprintf(os.Stderr, "\n%s%s", strings.Repeat(" ", flagLen+4), v)
	}
}

//Sets the JSON object to be printed
func JSONHelp(j string) {
	jsonToPrint = j
}

//Prints the JSON previously queued in the help format.
func PrintJSONHelp() {
	if jsonToPrint != "" {
		Header("RAW OUTPUT")
		ansi.Fprintf(os.Stderr, "%s\n", PrettyJSON(jsonToPrint))
	}
}

func MessageHelp(mess string, args ...interface{}) {
	messageToPrint = mess
}

//If MessageHelp was not called, the default summary for the command will be//printed.
func PrintMessage(command string, c *Command) {
	if messageToPrint == "" {
		messageToPrint = c.summary[command]
	}
	if len(messageArgs) > 0 {
		ansi.Fprintf(os.Stderr, messageToPrint, messageArgs)
	} else {
		ansi.Fprintf(os.Stderr, messageToPrint)
	}
	ansi.Fprintf(os.Stderr, "\n")
}

func PrintUsage(c string) {
	ansi.Fprintf(os.Stderr, "@G{shield %s}", c)
	for _, f := range flagsToPrint {
		if f.optional {
			ansi.Fprintf(os.Stderr, " @G{[%s]}", f.flag[0])
		} else {
			ansi.Fprintf(os.Stderr, " @G{%s}", f.flag[0])
		}
	}
	fmt.Fprintf(os.Stderr, "\n")
}

func InputHelp(help string) {
	inputHelpText = help
}

func PrintInputHelp() {
	if inputHelpText == "" {
		return
	}
	Header("RAW INPUT")
	ansi.Fprintf(os.Stderr, "%s\n", PrettyJSON(inputHelpText))
}
