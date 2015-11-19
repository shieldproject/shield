// Daemons need to log. This package makes that easy, allowing you to
// configure logging to syslog, a file, or console (stdout), with syslog
// style log levels.
//
// Since this logging mechanism supports non-syslog output, filtering based
// on log levels is done in-application, so you'll want to accept debug messages
// and above in syslog configs.
//
// Simply call SetupLogging(), passing it a reference to a LogConfig struct and
// start logging! If you happen to log something prior to setting up logging,
// messages will print to stderr.
package log

import "fmt"
import "io"
import "os"
import "log/syslog"
import "strings"
import "time"

type LogConfig struct {
	Type     string    // logging mode to use - file, syslog, console
	Level    string    // Syslog level to log at (debug, info, notice, error, etc)
	Facility string    // Syslog facility to log to (daemon, misc, etc)
	File     string    // Path that will be logged to if in file mode
}

type logger struct {
	out io.Writer
	level syslog.Priority
	ltype string
}

var log *logger

func init() {
	SetupLogging(LogConfig{ Type: "console", Level: "warning" })
}
// Does the needful to set up the logging subsystem based on the passed configuration data.
func SetupLogging (cfg LogConfig) {
	var l logger

	if cfg.Type == "syslog" {
		facility := get_facility(cfg.Facility)
		logger, err := syslog.New(facility, "")
		if err != nil {
			l.out = os.Stdout
			os.Stderr.Write([]byte(fmt.Sprintf("Unable to hook up to syslog, using console for logging: %s", err.Error())))
		}
		l.out = logger
	} else if cfg.Type == "file" {
		f, err := os.OpenFile(cfg.File, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			l.out = os.Stdout
			os.Stderr.Write([]byte(fmt.Sprintf("Unable to log to %s - using console instead: %s", cfg.File, err.Error())))
		}
		l.out = f
	} else {
		l.out = os.Stdout
	}
	l.level = get_level(cfg.Level)
	l.ltype = cfg.Type
	log = &l
}

func write (msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	if log.ltype != "syslog" {
		msg = fmt.Sprintf("%s bmad: %s\n", time.Now().String(), msg)
	}
	if log != nil && log.out != nil {
		log.out.Write([]byte(msg))
	} else {
		os.Stderr.Write([]byte(msg))
	}
}

// Logs a Debug message.
// Supports fmt.Sprintf style arguments.
func Debug (msg string, args ...interface{}) {
	if log.level >= syslog.LOG_DEBUG {
		write("DEBUG: " + msg, args...)
	}
}

// Logs an Info message.
// Supports fmt.Sprintf style arguments.
func Info (msg string, args ...interface{}) {
	if log.level >= syslog.LOG_INFO {
		write("INFO: " + msg, args...)
	}
}

// Logs a Notice message.
// Supports fmt.Sprintf style arguments.
func Notice (msg string, args ...interface{}) {
	if log.level >= syslog.LOG_NOTICE {
		write("NOTICE: " + msg, args...)
	}
}

// Logs a Warning message.
// Supports fmt.Sprintf style arguments.
func Warn (msg string, args ...interface{}) {
	if log.level >= syslog.LOG_WARNING {
		write("WARNING: " + msg, args...)
	}
}

// Logs an Error message.
// Supports fmt.Sprintf style arguments.
func Error (msg string, args ...interface{}) {
	if log.level >= syslog.LOG_ERR {
		write("ERROR: " + msg, args...)
	}
}

// Logs a Crit message.
// Supports fmt.Sprintf style arguments.
func Crit (msg string, args ...interface{}) {
	if log.level >= syslog.LOG_CRIT {
		write("CRITICAL: " + msg, args...)
	}
}

// Logs an Alert message.
// Supports fmt.Sprintf style arguments.
func Alert (msg string, args ...interface{}) {
	if log.level >= syslog.LOG_ALERT {
		write("ALERT: " + msg, args...)
	}
}

// Logs an Emerg message.
// Supports fmt.Sprintf style arguments.
func Emerg (msg string, args ...interface{}) {
	if log.level >= syslog.LOG_EMERG {
		write("EMERGENCY: " + msg, args...)
	}
}

// Validates the log level based on config strings
func get_level (level string) (syslog.Priority) {
	var priority syslog.Priority
	switch strings.ToLower(level) {
	case "debug":
		priority = priority | syslog.LOG_DEBUG
	case "info":
		priority = priority | syslog.LOG_INFO
	case "notice":
		priority = priority | syslog.LOG_NOTICE
	case "warning":
		priority = priority | syslog.LOG_WARNING
	case "warn":
		priority = priority | syslog.LOG_WARNING
	case "error":
		priority = priority | syslog.LOG_ERR
	case "err":
		priority = priority | syslog.LOG_ERR
	case "crit":
		priority = priority | syslog.LOG_CRIT
	case "alert":
		priority = priority | syslog.LOG_ALERT
	case "emerg":
		priority = priority | syslog.LOG_EMERG
	default:
		panic(fmt.Sprintf("Unsupported logging priority %q", level))
	}

	return priority
}

// Validates the syslog priority, based on config strings
func get_facility (facility string) (syslog.Priority) {
	var priority syslog.Priority
	switch strings.ToLower(facility) {
	case "kern":
		priority = syslog.LOG_KERN
	case "user":
		priority = syslog.LOG_USER
	case "mail":
		priority = syslog.LOG_MAIL
	case "daemon":
		priority = syslog.LOG_DAEMON
	case "auth":
		priority = syslog.LOG_AUTH
	case "syslog":
		priority = syslog.LOG_SYSLOG
	case "lpr":
		priority = syslog.LOG_LPR
	case "news":
		priority = syslog.LOG_NEWS
	case "uucp":
		priority = syslog.LOG_UUCP
	case "cron":
		priority = syslog.LOG_CRON
	case "authpriv":
		priority = syslog.LOG_AUTHPRIV
	case "ftp":
		priority = syslog.LOG_FTP
	case "local0":
		priority = syslog.LOG_LOCAL0
	case "local1":
		priority = syslog.LOG_LOCAL1
	case "local2":
		priority = syslog.LOG_LOCAL2
	case "local3":
		priority = syslog.LOG_LOCAL3
	case "local4":
		priority = syslog.LOG_LOCAL4
	case "local5":
		priority = syslog.LOG_LOCAL5
	case "local6":
		priority = syslog.LOG_LOCAL6
	case "local7":
		priority = syslog.LOG_LOCAL7
	default:
		panic(fmt.Sprintf("Unsupported logging priority %q", facility))
	}

	return priority
}
