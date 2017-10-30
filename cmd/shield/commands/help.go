package commands

import (
	"strings"

	fmt "github.com/jhunt/go-ansi"
)

//FlagList contains information and functions to display flag help dialogue
type FlagList []FlagInfo

//HelpStrings returns all of the contained flags' help lines, formatted into
//columns
func (f FlagList) HelpStrings() (lines []string) {
	columnWidth := f.maxFlagLength()
	for _, flag := range f {
		lines = append(lines, flag.HelpLines(columnWidth)...)
	}

	return lines
}

//Get the longest length of flags in this helpinfo, to be used to determine the
//buffer width in help formatting
func (f FlagList) maxFlagLength() (length int) {
	for _, flag := range f {
		thisLen := flag.combinedFlagLength()
		if thisLen > length {
			length = thisLen
		}
	}
	return
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
		flags[i] = fmt.Sprintf("@B{%s}", flags[i])
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
		return fmt.Sprintf("--%s VALUE", f.Name)
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
		var additionalLength int
		if f.Valued {
			additionalLength = 6
		}
		return len(f.Name) + 2 + additionalLength
	}
	panic("flag name not set")
}

//HelpHeader returns the input string formatted for a help dialogue's header
func HelpHeader(text string) string {
	return fmt.Sprintf("\n@G{%s}", text)
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
	TenantNameFlag = FlagInfo{
		Name: "tenantname", Positional: true, Mandatory: true,
		Desc: `A string partially matching the name of a single tenant
				or a UUID exactly matching the UUID of a tenant.`,
	}
	UserNameFlag = FlagInfo{
		Name: "username", Positional: true, Mandatory: true,
		Desc: `A string partially matching the account of a single (local) user
				or a UUID exactly matching the UUID of a (local) user.`,
	}
	//AccountFlag <account>
	AccountFlag = FlagInfo{
		Name: "account", Positional: true, Mandatory: true,
		Desc: `A string partially matching the name of a single account
				or a UUID exactly matching the UUID of an account.`,
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
