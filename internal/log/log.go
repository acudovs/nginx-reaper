// Package log provides an unstructured leveled logging wrapper around standard log.
package log

import (
	"fmt"
	"log"
	"strings"
	"sync/atomic"
)

// ParseLevel converts case-insensitive string to log Level. Returns error if invalid.
// E.g. "panic" becomes PanicLevel.
func ParseLevel(name string) (Level, error) {
	if l, ok := names[strings.ToLower(name)]; ok {
		return l, nil
	}
	return 0, fmt.Errorf("invalid log level: %q", name)
}

// SetLevel sets log Level. If invalid, uses DefaultLevel.
func SetLevel(l Level) {
	if l > DebugLevel {
		Errorf("Invalid log level %v, using default %v", l, DefaultLevel)
		l = DefaultLevel
	}
	atomic.StoreUint32((*uint32)(&level), uint32(l))
}

// Panic logs a panic message, then panics.
func Panic(v ...any) {
	log.Panic(fmt.Sprint(append([]any{prefixes[PanicLevel]}, v...)...))
}

// Panicf logs a formatted panic message, then panics.
func Panicf(format string, v ...any) {
	log.Panicf(fmt.Sprint(prefixes[PanicLevel], format), v...)
}

// Error logs an error message.
func Error(v ...any) {
	Log(ErrorLevel, v...)
}

// Errorf logs a formatted error message.
func Errorf(format string, v ...any) {
	Logf(ErrorLevel, format, v...)
}

// Warning logs a warning message.
func Warning(v ...any) {
	Log(WarningLevel, v...)
}

// Warningf logs a formatted warning message.
func Warningf(format string, v ...any) {
	Logf(WarningLevel, format, v...)
}

// Info logs an info message.
func Info(v ...any) {
	Log(InfoLevel, v...)
}

// Infof logs a formatted info message.
func Infof(format string, v ...any) {
	Logf(InfoLevel, format, v...)
}

// Debug logs a debug message.
func Debug(v ...any) {
	Log(DebugLevel, v...)
}

// Debugf logs a formatted debug message.
func Debugf(format string, v ...any) {
	Logf(DebugLevel, format, v...)
}

// Log logs a message with a specified Level.
func Log(l Level, v ...any) {
	if l <= level {
		log.Print(fmt.Sprint(append([]any{prefixes[l]}, v...)...))
	}
}

// Logf logs a formatted message with a specified Level.
func Logf(l Level, format string, v ...any) {
	if l <= level {
		log.Printf(fmt.Sprint(prefixes[l], format), v...)
	}
}
