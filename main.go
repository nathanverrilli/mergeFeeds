package main

import (
	"os"
	"os/signal"
	"sync"

	misc "github.com/nathanverrilli/nlvMisc"
)

const DEFAULT_OUTPUT_DIR = ".output"

func init() {
	// initialize program boilerplate stuff
	// turn on logger
	initLog("mergeFeeds.log")
	misc.AtClose(closeLog)
	// get program options
	initFlags()
	// set these options in misc from flags
	_ = misc.OptionPrintf(safeLogPrintf)
	_ = misc.OptionFatal(myFatal)
	_ = misc.OptionVerbose(FlagVerbose)
	_ = misc.OptionDebug(FlagDebug)
	_ = misc.OptionOutputDir(DEFAULT_OUTPUT_DIR)

	// handle ctrl-c or kill
	signalChan := make(chan os.Signal) // important: this channel should not be buffered
	signal.Notify(signalChan, os.Interrupt, os.Kill)
	go misc.HandleSignal(signalChan)

}

func main() {

	var url = "den/cpo/1.0/locations/?limit=1000&offset=0"

	var err error
	var wgJson sync.WaitGroup
	var wgError sync.WaitGroup
	var wgFeeds sync.WaitGroup

	wgJson.Add(1)
	wgError.Add(1)
	wgFeeds.Add(1)

	outJson := make(chan []byte, 16) // close called from here
	outError := make(chan []byte, 4) // close called from outJson

	go misc.RecordBytes("error.log", outError, wgError.Done)
	go filterJsonPage(outJson, outError, wgJson.Done)

	// load endpoints
	_, productionEndpointUrls, ProductionEndpointTokens := loadEndpoints(FlagAuthTokenFile)

	go func() {
		err = mergeFeeds(productionEndpointUrls, ProductionEndpointTokens, url, outJson, outError, wgFeeds.Done)
		if err != nil {
			xLog.Printf("error merging feeds because %s", err.Error())
		}
	}()
	wgFeeds.Wait()

	close(outJson)
	wgJson.Wait()

	close(outError)
	wgError.Wait()

}
