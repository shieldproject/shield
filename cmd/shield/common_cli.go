package main

var (
	// Options
	pluginFilter string
	unusedFilter bool
	usedFilter   bool
)

func BoolString(tf bool) string {
	if tf {
		return "yes"
	} else {
		return "no"
	}
}
