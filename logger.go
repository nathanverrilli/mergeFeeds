package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"

	misc "github.com/nathanverrilli/nlvMisc"
)

var xLogFile *os.File
var xLogBuffWriter *bufio.Writer
var xLog log.Logger

// flushLog flushes the log buffer if it is not nil.
// If flushing the buffer results in an error, it logs the error message to standard output.
// This function is typically used to ensure that all log messages are written before shutting down the logging service.
func flushCloseLog() (err error) {
	if nil != xLogBuffWriter {
		return xLogBuffWriter.Flush()
	}
	return nil
}

var closeLogMutex sync.Mutex

// closeLog shuts the logging service down
// cleanly, flushing buffers (and thus
// preserving the most likely error of
// interest)
func closeLog() {

	var errList = make([]error, 0, 2)

	closeLogMutex.Lock()
	{
		if nil != xLogBuffWriter {
			errList = append(errList, flushCloseLog())
			xLogBuffWriter = nil
		}
		if nil != xLogFile {
			errList = append(errList, xLogFile.Close())
			xLogFile = nil
		}
	}
	closeLogMutex.Unlock()

	for _, err := range errList {
		if nil != err {
			_, _ = safeLogPrintf("error: %s", err.Error())
		}
	}
}

// initLog initializes the log file and log buffer.
// It opens the log file with the specified name, creating it if it does not exist, and truncates it if it does exist.
// If opening the log file encounters an error, it logs the error message to standard output using safeLogPrintf.
// It creates a new bufio.Writer to be used as the log buffer and sets the log writers to the standard output and the log buffer.
// It sets the log flags to include the date, time, UTC, and short file.
// It resolves the absolute path of the log file and logs it using safeLogPrintf.
// This function is typically called at the initialization of the logging service.
// The log file name should be passed as the lfName argument.
func initLog(lfName string) {
	var err error
	if !misc.IsStringSet(&lfName) {
		_, _ = safeLogPrintf("huh? log filename is not set")
		myFatal()
	}
	var logWriters = make([]io.Writer, 0, 2)
	xLogFile, err = os.OpenFile(lfName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if nil != err {
		_, _ = safeLogPrintf("error opening log file %s because %s",
			lfName, err.Error())
	}

	xLogBuffWriter = bufio.NewWriter(xLogFile)
	logWriters = append(logWriters, os.Stdout)
	logWriters = append(logWriters, xLogBuffWriter)
	xLog.SetFlags(log.Ldate | log.Ltime | log.LUTC | log.Lshortfile)
	xLog.SetOutput(io.MultiWriter(logWriters...))

	logPath, err := filepath.Abs(xLogFile.Name())
	if nil != err {
		_, _ = safeLogPrintf("huh? could not resolve logfilename %s because %s",
			xLogFile.Name(), err.Error())
		myFatal()
	}
	_, _ = safeLogPrintf("Logfile set to %s", logPath)
}

var myFatalMutex sync.Mutex

// myFatal is meant to close the program, and close the
// log files properly. Go doesn't support optional arguments,
// but variadic arguments allow finessing this. myFatal() gets
// a default RC of -1, and that's overridden by the first int
// in the slice of integers argument (which is present
// even if the length is 0).
//
// Part of the implementation of generic at-close system with
// misc.AtClose, misc.AtCloseErr, and misc.FinishClose.
//
// Any subsequent calls to myFatal() block (because
// returning would probably cause more errors since
// calling myFatal means something failed
// catastrophically.
func myFatal(rcList ...int) {
	myFatalMutex.Lock()
	// never released, because only one call
	// to this is permitted. All others can
	// block until the program exits
	var rc = -1
	if len(rcList) > 0 {
		rc = rcList[0]
	}
	_, _ = safeLogPrintf("myFatal called")
	misc.FinishClose()
	os.Exit(rc)
}

func getCallerInfo(frm int) (callerInfo string) {
	var srcName string
	srcPtr, srcFile, srcLine, ok := runtime.Caller(frm + 1)
	if !ok {
		srcName = "unknownFunc"
		srcFile = "unknownFile"
		srcLine = 0
	} else {
		srcFile = path.Base(srcFile)
		srcName = runtime.FuncForPC(srcPtr).Name()
	}
	return fmt.Sprintf(" %s %s:%d ", srcName, srcFile, srcLine)
}

// safeLogPrintf may be called in lieu of xLog.Printf() if there
// is a possibility the log may not be open. If the log is
// available, well and good. Otherwise, print the message to
// STDERR.
// This function might be called from multiple threads and would
// conflict with closeLog(), thus the mutex
func safeLogPrintf(format string, a ...any) (n int, err error) {
	// append information from the *two* frames above this in the calling stack
	// as this might be called indirectly (so the frame above the calling frame
	// is the frame of interest).
	strPrefix := "\n\t" + getCallerInfo(2) + "\n\t\t" + getCallerInfo(1) + ":"
	closeLogMutex.Lock()
	defer closeLogMutex.Unlock()
	if nil != xLogBuffWriter && nil != xLogFile {
		// logger still open
		xLog.Printf(strPrefix+format, a...)
	} else {
		// log is NOT open. Print to STDERR.
		_, _ = fmt.Fprintf(os.Stderr, "*** SAFELOG ***"+strPrefix+format+"\n", a...)
		// if there's an error printing to STDERR, then something is truly wrong!
	}
	return 0, nil
}

// debugMapStringString is a function that takes a map of string keys and string values as input.
// It generates a formatted string representation of the map for debugging purposes.
// The output string includes the size of the map and all key-value pairs in a tabular format.
// Each key-value pair is displayed in a separate line, with the key and value left-aligned in columns of width 20.
// This function does not return any value, it simply writes the debug string to standard output.
// Example usage:
//
//	params := map[string]string{
//	    "key1": "value1",
//	    "key2": "value2",
//	    "key3": "value3",
//	}
//	debugMapStringString(params)
//
// Output:
//
//	got map[string]string size 3
//	[ key1               ][ value1             ]
//	[ key2               ][ value2             ]
//	[ key3               ][ value3             ]
/***
func debugMapStringString(params map[string]string) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n\t"+"got map[string]string size %d\n", len(params)))
	for k, v := range params {
		sb.WriteString(fmt.Sprintf("\t[ %-20s ][ %-20s ]\n", k, v))
	}
}
 ***/
