package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
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
	ansi.Fprintf(os.Stderr, "\n@G{%s}\n", text)
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
	if len(flagsToPrint) == 0 {
		return
	}
	Header("FLAGS")
	sort.Sort(ByFlag(flagsToPrint))
	for i, flags := range flagsToPrint {
		//Parse out each newline separated flags
		//Wrap each of the flag entries in @M{} to highlight them blue
		printFlagHelper(flags.flag, flags.desc, i == len(flagsToPrint)-1)
	}
}

func printFlagHelper(flags []string, desc string, last bool) {
	defer func() { ansi.Fprintf(os.Stderr, "\n") }()
	//Print the flag list
	flagString := "@B{" + strings.Join(flags, "}, @B{") + "}"
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
	if len(lines) > 1 && !last {
		ansi.Fprintf(os.Stderr, "\n")
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
	messageArgs = args
}

//If MessageHelp was not called, the default summary for the command will be
//printed. Otherwise, prints the string given to MessageHelp in its place
func PrintMessage(command string, c *Command) {
	if messageToPrint == "" {
		messageToPrint = c.summary[command]
	}
	if len(messageArgs) > 0 {
		ansi.Fprintf(os.Stderr, messageToPrint, messageArgs...)
	} else {
		ansi.Fprintf(os.Stderr, messageToPrint)
	}
	ansi.Fprintf(os.Stderr, "\n")
}

func PrintUsage(c string) {
	ansi.Fprintf(os.Stderr, "@G{shield %s}", c)
	sort.Sort(ByFlag(flagsToPrint))
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

func HelpListMacro(singular, plural string) {
	if singular != "job" && singular != "archive" {
		FlagHelp(fmt.Sprintf("Only show %s which are NOT in use by a job", plural), true, "--unused")
		FlagHelp(fmt.Sprintf("Only show %s which are in use by a job", plural), true, "--used")
	}
	FlagHelp("Outputs information as a JSON object", true, "--raw")

	if singular != "archive" {
		FlagHelp(fmt.Sprintf("A string partially matching the names of the %s to return", plural),
			true,
			fmt.Sprintf("<%sname>", singular))
	}
	HelpKMacro()
}

func HelpShowMacro(singular, plural string) {
	FlagHelp(fmt.Sprintf(`A string partially matching the name of a single %[1]s
				or a UUID exactly matching the UUID of a %[1]s.
				Not setting this value explicitly will default it to the empty string.`, singular),
		false, fmt.Sprintf("<%s>", singular))
	FlagHelp("Returns information as a JSON object", true, "--raw")
	HelpKMacro()
}

func HelpCreateMacro(singular, plural string) {
	FlagHelp(`Takes input as a JSON object from standard input
				Outputs the resultant target info as a JSON object`, true, "--raw")
	HelpKMacro()
}

func HelpEditMacro(singular, plural string) {
	FlagHelp(fmt.Sprintf(`Takes input as a JSON object from standard input
				Outputs the resultant %[1]s info as a JSON object.
				Suppresses interactive dialogues and confirmation.`, singular),
		true, "--raw")
	FlagHelp(fmt.Sprintf(`A string partially matching the name of a single %[1]s 
				or a UUID exactly matching the UUID of a %[1]s.
				Not setting this value explicitly will default it to the empty string.`, singular),
		false, fmt.Sprintf("<%s>", singular))
	HelpKMacro()
	MessageHelp("Modify an existing backup %[1]s. The UUID of the %[1]s will remain the same after modification.", singular)
}

func HelpDeleteMacro(singular, plural string) {
	FlagHelp(`Outputs the result as a JSON object.
				The cli will not prompt for confirmation in raw mode.`, true, "--raw")
	HelpKMacro()
	FlagHelp(fmt.Sprintf(`A string partially matching the name of a single %[1]s
				or a UUID exactly matching the UUID of a %[1]s.
				Not setting this value explicitly will default it to the empty string.`, singular),
		false, fmt.Sprintf("<%s>", singular))
	JSONHelp(fmt.Sprintf(`{"ok":"Deleted %s"}`, singular))
}

func HelpKMacro() {
	FlagHelp("Disable SSL certificate validation", true, "-k", "--skip-ssl-validation")
}
