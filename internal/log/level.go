package log

// Level represents a log level.
type Level uint32

// Log levels.
const (
	PanicLevel Level = iota
	ErrorLevel
	WarningLevel
	InfoLevel
	DebugLevel
	DefaultLevel = DebugLevel
)

// Current log Level.
var level = DefaultLevel

// Log Level to prefix mapping.
var prefixes = map[Level]string{
	PanicLevel:   "PANIC ",
	ErrorLevel:   "ERROR ",
	WarningLevel: "WARNING ",
	InfoLevel:    "INFO ",
	DebugLevel:   "DEBUG ",
}

// Log level name to Level mapping.
var names = map[string]Level{
	"panic":   PanicLevel,
	"error":   ErrorLevel,
	"warning": WarningLevel,
	"info":    InfoLevel,
	"debug":   DebugLevel,
}
