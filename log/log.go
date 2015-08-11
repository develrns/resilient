/*
Package log provides an logger that logs to the $LOG log file. If $LOG is not provided, stderr is used.
$LOGPREF is the logging prefix and $LOGFLG is the logging flag. If $LOGFLG isn't provided, log.LstdFlags is used.
See standard go log package for more info.
*/
package log

import (
	"fmt"
	golog "log"
	"os"
	"strconv"
)

var logger *golog.Logger

func init() {
	var (
		logFileName = os.Getenv("$LOG")
		logPref     = os.Getenv("$LOGPREF")
		logFlg      = os.Getenv("$LOGFLG")
		logFlgI     int
		logFile     *os.File
		openErr     error
		atoiErr     error
	)

	if logFileName != "" {
		logFile, openErr = os.Create(logFileName)
		if openErr != nil {
			logFile = os.Stderr
		}
	} else {
		logFile = os.Stderr
	}

	if logFlg != "" {
		logFlgI, atoiErr = strconv.Atoi(logFlg)
	}
	if logFlg == "" || atoiErr != nil {
		logFlgI = golog.LstdFlags
	}

	logger = golog.New(logFile, logPref, logFlgI)

	if openErr != nil {
		fmt.Printf("Error opening log file with Name: %v Error: %v\n", logFileName, openErr)
	}
	if atoiErr != nil {
		fmt.Printf("Bad Log Flag: %v Error: %v\n ", logFlg, atoiErr)
	}
}

/*
Logger returns the rns logger
*/
func Logger() *golog.Logger {
	return logger
}
