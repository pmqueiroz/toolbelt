package ui

import (
	"os"
	"time"

	"github.com/charmbracelet/log"
)

var Log = newLogger()

func newLogger() *log.Logger {
	return log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
		Level:           log.InfoLevel,
	})
}

func Debug(msg string, keyvals ...any) { Log.Debug(msg, keyvals...) }
func Info(msg string, keyvals ...any)  { Log.Info(msg, keyvals...) }
func Warn(msg string, keyvals ...any)  { Log.Warn(msg, keyvals...) }
func Error(msg string, keyvals ...any) { Log.Error(msg, keyvals...) }
func Fatal(msg string, keyvals ...any) { Log.Fatal(msg, keyvals...) }

func SetVerbose(verbose bool) {
	if verbose {
		Log.SetLevel(log.DebugLevel)
		return
	}
	Log.SetLevel(log.InfoLevel)
}
