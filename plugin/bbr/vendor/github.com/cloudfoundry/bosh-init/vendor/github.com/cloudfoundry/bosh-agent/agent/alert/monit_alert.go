package alert

type MonitAlert struct {
	ID          string
	Service     string
	Event       string
	Action      string
	Date        string // RFC1123Z formatted date string
	Description string
}
