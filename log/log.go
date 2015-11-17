/*
Package log provides a configured instance of a log package logger that is shared within an executable.
Typically the executable will provide -log, -logprefix and -logflg command line switches containing respectively the log file name, log prefix and log flag values.
The executable's init will parse these command line flags and then configure this log instance with them.

See standard go log package for more info.
*/
package log

import (
	golog "log"
	"os"
)

var logger *golog.Logger

/*
Config initializes the shared log instance. It should be called from an executable's init function. If it is not called, a default log instance that logs to os.Stderr is created.
*/
func Config(logname, logprefix string, logflg int) {
	var (
		logFile *os.File
		openErr error
	)

	if logname != "" {
		logFile, openErr = os.Create(logname)
		if openErr != nil {
			logFile = os.Stderr
		}
	} else {
		logFile = os.Stderr
	}

	logger = golog.New(logFile, logprefix, logflg)

	if openErr != nil {
		logger.Printf("Logging to stderr because opening log file with Name: %v failed with Error: %v\n", logname, openErr)
	}
}

/*
Logger returns the shared logger
*/
func Logger() *golog.Logger {
	if logger == nil {
		Config("", "", 0)
	}
	return logger
}
