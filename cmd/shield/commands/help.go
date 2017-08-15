package commands

import (
	"fmt"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
)

//HelpInfo contains information and functions to display help dialogue
type HelpInfo struct {
	Flags      []FlagInfo
	Message    string
	JSONInput  string
	JSONOutput string
}

//FlagInfo contains attributes needed to display help for this command's flags
type FlagInfo struct {
	Name       string
	Desc       string
	Short      rune
	Mandatory  bool
	Positional bool
	Valued     bool
}

//HelpLines returns a slice of strings formatted as `shortflag, longflag
//summary`, where colwidth is how many spaces are taken up by the flags and the
//buffer spaces before the summary
func (f *FlagInfo) HelpLines(colwidth int) []string {
	flags := []string{}
	if f.Short != 0 {
		flags = append(flags, f.formatShort())
	}
	flags = append(flags, f.formatLong())

	for i := range flags { //Turn the flags blue
		flags[i] = ansi.Sprintf("@B{%s}", flags[i])
	}
	flagStr := strings.Join(flags, ", ")
	//Adjust the formatting column width to account for non-printing chars
	nonAnsiFlagLength := f.combinedFlagLength()
	ansiFlagLength := len(flagStr)
	numNonPrinting := ansiFlagLength - nonAnsiFlagLength
	const lineWidth = 78

	//Add line with actual flags
	descLine, remaining := splitTokensAfterLen(f.Desc, lineWidth-colwidth)
	lines := []string{fmt.Sprintf("%-*s  %s", colwidth+numNonPrinting, flagStr, descLine)}

	//If the summary is longer than the line width, make another line for it
	for remaining != "" {
		descLine, remaining = splitTokensAfterLen(remaining, lineWidth-colwidth)
		lines = append(lines, fmt.Sprintf("%-*s  %s", colwidth, "", descLine))
	}

	return lines
}

//Adds leading dash
func (f FlagInfo) formatShort() (formatted string) {
	if f.Short != 0 {
		formatted = fmt.Sprintf("-%c", f.Short)
	}
	return formatted
}

func (f FlagInfo) formatShortIfPresent() string {
	if f.Short != 0 {
		return f.formatShort()
	}
	return f.formatLong()
}

//Adds leading dashes or wraps in <> if is a positional argument
func (f FlagInfo) formatLong() (formatted string) {
	if f.Name == "" {
		panic("No name given for flag")
	}
	if f.Positional {
		return fmt.Sprintf("<%s>", f.Name)
	} else if f.Valued {
		return fmt.Sprintf("--%s=value", f.Name)
	}
	return fmt.Sprintf("--%s", f.Name)
}

//The sum of the short and long flag name lengths plus the
// necessary dashes and punctuation/whitespace, if necessary
func (f FlagInfo) combinedFlagLength() (length int) {
	shortLen := f.lenShort()
	longLen := f.lenLong()
	length = shortLen + longLen
	if shortLen > 0 {
		length = length + 2
	}
	return
}

func (f FlagInfo) lenShort() (length int) {
	if f.Short != 0 {
		length = 2
	}
	return
}

func (f FlagInfo) lenLong() (length int) {
	if f.Name != "" {
		return len(f.Name) + 2
	}
	panic("flag name not set")
}

//FlagHelp returns all of the contained flags' help lines, formatted into
//columns
func (h HelpInfo) FlagHelp() (lines []string) {
	columnWidth := h.maxFlagLength()
	for _, flag := range h.Flags {
		lines = append(lines, flag.HelpLines(columnWidth)...)
	}

	return lines
}

//Get the longest length of flags in this helpinfo, to be used to determine the
//buffer width in help formatting
func (h HelpInfo) maxFlagLength() (length int) {
	for _, flag := range h.Flags {
		thisLen := flag.combinedFlagLength()
		if thisLen > length {
			length = thisLen
		}
	}
	return
}

//HelpHeader returns the input string formatted for a help dialogue's header
func HelpHeader(text string) string {
	return ansi.Sprintf("\n@G{%s}", text)
}

//ByFlag sorts FlagInfo objects by their flag. Positional arguments come first.
// Short flags are only used for sorting if long flags aren't present
type ByFlag []FlagInfo

func (f ByFlag) Len() int      { return len(f) }
func (f ByFlag) Swap(i, j int) { f[i], f[j] = f[j], f[i] }
func (f ByFlag) Less(i, j int) bool {
	if f[i].Positional != f[j].Positional { //Positional arguments come first
		return f[i].Positional
	}
	return f[i].Name < f[j].Name
}

var (
	//UnusedFlag --unused
	UnusedFlag = FlagInfo{
		Name: "unused",
		Desc: "Only return objects which are not registered to a job",
	}
	//UsedFlag --used
	UsedFlag = FlagInfo{
		Name: "used",
		Desc: "Only return objects which are registered to a job",
	}
	//FuzzyFlag --fuzzy
	FuzzyFlag = FlagInfo{
		Name: "fuzzy",
		Desc: "In RAW mode, perform fuzzy (inexact) searching",
	}
	//TargetNameFlag <targetname>
	TargetNameFlag = FlagInfo{
		Name: "targetname", Positional: true, Mandatory: true,
		Desc: `A string partially matching the name of a single target
				or a UUID exactly matching the UUID of a target.`,
	}
	//ScheduleNameFlag <schedulename>
	ScheduleNameFlag = FlagInfo{
		Name: "schedulename", Positional: true, Mandatory: true,
		Desc: `A string partially matching the name of a single schedule
				or a UUID exactly matching the UUID of a schedule.`,
	}
	//PolicyNameFlag <policyname>
	PolicyNameFlag = FlagInfo{
		Name: "policyname", Positional: true, Mandatory: true,
		Desc: `A string partially matching the name of a single policy
				or a UUID exactly matching the UUID of a policy.`,
	}
	//StoreNameFlag <storename>
	StoreNameFlag = FlagInfo{
		Name: "storename", Positional: true, Mandatory: true,
		Desc: `A string partially matching the name of a single store
				or a UUID exactly matching the UUID of a store.`,
	}
	//JobNameFlag <jobname>
	JobNameFlag = FlagInfo{
		Name: "jobname", Positional: true, Mandatory: true,
		Desc: `A string partially matching the name of a single job
				or a UUID exactly matching the UUID of a job.`,
	}

	//UpdateIfExistsFlag --update-if-exists
	UpdateIfExistsFlag = FlagInfo{
		Name: "update-if-exists",
		Desc: "Update record if another exists with same name",
	}
)

func splitTokensAfterLen(input string, numChars int) (before, after string) {
	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		panic("No input to split tokens was given")
	}

	curLen := len(tokens[0])
	splitAt := 1
	for ; splitAt < len(tokens); splitAt++ {
		curLen += len(tokens[splitAt]) + 1
		if curLen > numChars {
			break
		}
	}
	return strings.Join(tokens[:splitAt], " "), strings.Join(tokens[splitAt:], " ")
}
