package commands

//Options contains all the possible command line options that commands may
//possibly use
type Options struct {
	Used     *bool
	Unused   *bool
	Paused   *bool
	Unpaused *bool
	All      *bool

	Debug             *bool
	Trace             *bool
	Raw               *bool
	ShowUUID          *bool
	UpdateIfExists    *bool
	Fuzzy             *bool
	SkipSSLValidation *bool
	Version           *bool
	Help              *bool
	CACert            *string

	Status *string

	Target    *string
	Store     *string
	Retention *string

	Plugin *string

	After  *string
	Before *string

	To *string

	Limit *string

	Full *bool

	Config   *string
	User     *string
	Password *string

	Backend  *string
	SysRole  *string
	Account  *string
	Provider *string
	Token    *string

	APIVersion int
}

//Opts is the options flag struct to be used by all commands
var Opts *Options
