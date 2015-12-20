/*
Package oplog provides a configured instance of a log package logger that is shared within an executable.
It is used to log operational events and instrumentation that will typically be aggregated and anaylyzed by
operational log analysis tools. Therefore, it should not be used for debug logging, etc.

Typically the executable will provide -oplog, -oplogprefix and -oplogflg command line switches containing respectively
the log file name, log prefix and log flag values.
The executable's init will parse these command line flags and then configure this log instance with them.

If Config is not called, the default is to log to stderr with no prefix and no flag.

See the golang log package for a definition of the oplogflg bits that are ore'ed to form a flag value.

Due to initialization order issues, this logger cannot be used in init() functions.

See standard go log package for more info.
*/
package oplog

import (
	golog "log"
	"os"
)

type (
	LoggerT struct {
		logger *golog.Logger
	}
)

var logger = new(LoggerT)

/*
Fatal delegates to the shared golang logger
*/
func (l *LoggerT) Fatal(v ...interface{}) {
	if l.logger == nil {
		Config("", "", 0)
	}
	l.logger.Fatal(v...)
}

/*
Fatalf delegates to the shared golang logger
*/
func (l *LoggerT) Fatalf(format string, v ...interface{}) {
	if l.logger == nil {
		Config("", "", 0)
	}
	l.logger.Fatalf(format, v...)
}

/*
Fatalln delegates to the shared golang logger
*/
func (l *LoggerT) Fatalln(v ...interface{}) {
	if l.logger == nil {
		Config("", "", 0)
	}
	l.logger.Fatalln(v...)
}

/*
Panic delegates to the shared golang logger
*/
func (l *LoggerT) Panic(v ...interface{}) {
	if l.logger == nil {
		Config("", "", 0)
	}
	l.logger.Panic(v...)
}

/*
Panicf delegates to the shared golang logger
*/
func (l *LoggerT) Panicf(format string, v ...interface{}) {
	if l.logger == nil {
		Config("", "", 0)
	}
	l.logger.Panicf(format, v...)
}

/*
Panicln delegates to the shared golang logger
*/
func (l *LoggerT) Panicln(v ...interface{}) {
	if l.logger == nil {
		Config("", "", 0)
	}
	l.logger.Panicln(v...)
}

/*
Print delegates to the shared golang logger
*/
func (l *LoggerT) Print(v ...interface{}) {
	if l.logger == nil {
		Config("", "", 0)
	}
	l.logger.Print(v...)
}

/*
Printf delegates to the shared golang logger
*/
func (l *LoggerT) Printf(format string, v ...interface{}) {
	if l.logger == nil {
		Config("", "", 0)
	}
	l.logger.Printf(format, v...)
}

/*
Logger returns the internal instance of the golang log.Logger so that it can be passed to http.Server
*/
func (l *LoggerT) Logger() *golog.Logger {
	if l.logger == nil {
		Config("", "", 0)
	}
	return (l.logger)
}

/*
Fatal delegates to the shared golang logger
*/
func (l *LoggerT) Println(v ...interface{}) {
	if l.logger == nil {
		Config("", "", 0)
	}
	l.logger.Println(v...)
}

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

	logger.logger = golog.New(logFile, logprefix, logflg)

	if openErr != nil {
		logger.Printf("Logging to stderr because opening log file with Name: %v failed with Error: %v\n", logname, openErr)
	}
}

/*
Logger returns the shared logger
*/
func Logger() *LoggerT {
	return logger
}
