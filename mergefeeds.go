package main

import (
	"fmt"
	"io"
	"sync"

	misc "github.com/nathanverrilli/nlvMisc"
)

// mergeFeeds combines multiple JSON location pages into a single output.
// Since output goes to a channel, this function is thread-safe.
func mergeFeeds(base []string, tokens []string, url string, out chan<- []byte, outError chan<- []byte, allDone func()) (err error) {
	var wg sync.WaitGroup
	var recordCount = make([]int, len(base))
	defer allDone()

	// sanity
	if len(base) <= 0 || len(base) != len(tokens) {
		return fmt.Errorf("mergefeeds(): base and token lists must have identical positive length")
	}
	if !misc.IsStringSet(&url) {
		return fmt.Errorf("mergefeeds(): url is not set")
	}
	if out == nil {
		return fmt.Errorf("mergefeeds(): output is not set")
	}
	if outError == nil {
		return fmt.Errorf("mergefeeds(): error output is not set")
	}

	for ix, baseStr := range base {
		wg.Add(1)
		go pullFeed(baseStr+url, tokens[ix], &recordCount[ix], out, outError, wg.Done)
	}
	wg.Wait()

	if FlagDebug {
		totalCount := 0
		// these counts are filled in by the pullFeed goroutines
		for ix := 0; ix < len(base); ix++ {
			totalCount += recordCount[ix]
			xLog.Printf("records from feed %s: %d", base[ix], recordCount[ix])
		}
		xLog.Printf("total records (all feeds): %d", totalCount)
	}
	return nil
}

// pullFeed fetches paginated JSON data, writing each page to the output and
// handling synchronization and errors. This thread-safe function runs as
// multiple goroutines, synchronizing using the provided mutex and waitgroup.Done()
// to signal completion. The sync pain is due to the annoying JSON comma,
// which forces mutex protection around writes.
func pullFeed(nextUrl string, token string, rc *int, out chan<- []byte, outError chan<- []byte, allDone func()) {
	var body []byte
	var err error

	defer allDone()

	pageCount := 0
	// loop until we get an empty string, which signals the end of the feed
	// or FlagMaxCalls is exceeded.
	for "" != nextUrl {

		if FlagDebug {
			xLog.Printf("Processing %s\n", nextUrl)
		}
		body, nextUrl, *rc, err = requestJsonObject(nextUrl, token)
		if err != nil {
			outError <- []byte(err.Error())
			outError <- body
		} else {
			out <- body
		}

		pageCount++
		// allow for each source to be tested up to FlagMaxCalls times
		// for easier testing (FlagMaxCalls calls rather than 100+).
		if FlagMaxCalls > 0 && pageCount >= FlagMaxCalls {
			break
		}
	}
}

// cleanWrite writes the provided text to the out writer and logs any
// errors encountered during the write operation.
// Errors are also logged to the general program log as well as the
// feed error log.
func cleanWrite(out io.Writer, outError io.Writer, text []byte) {
	_, err := out.Write(text)
	if nil != err {
		xLog.Println(err.Error())
		_, err = outError.Write([]byte(err.Error() + "\n"))
		if nil != err {
			xLog.Println(err.Error())
		}
	}
}
