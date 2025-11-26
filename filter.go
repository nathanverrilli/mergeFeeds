package main

import (
	"encoding/json"
	"strings"
	"sync"

	denjson "github.com/nathanverrilli/denJson"
	misc "github.com/nathanverrilli/nlvMisc"
)

// filterJsonPage processes JSON location data from an input channel, filters duplicates, and writes filtered data to an output.
// It handles JSON parsing, marshaling, and error handling, while managing synchronization with goroutines.
// The function writes filtered data to a JSON file and sends errors to an error channel.
// A callback function is called when the processing is complete. NOT THREAD SAFE, DO NOT MULTITHREAD
func filterJsonPage(jsonPage <-chan []byte, outError chan<- []byte, allDone func()) {
	var wg sync.WaitGroup
	var ld denjson.LocationData
	var needComma = false

	defer allDone()

	stationOut := make(chan []byte, 32)
	wg.Add(1)
	go misc.RecordBytes("stations.json", stationOut, wg.Done)

	stationOut <- []byte("{\"data\": [ ")

	for b := range jsonPage {
		err := json.Unmarshal(b, &ld)
		if nil != err {
			xLog.Printf("error parsing JSON: %s", err.Error())
			outError <- []byte(err.Error())
			outError <- b
			outError <- []byte("\n")
		}

		for _, loc := range ld.Data {
			ok := filterDuplicateStations(&loc)
			if ok {
				txt, err := json.Marshal(loc)
				if nil != err {
					xLog.Printf("error marshalling station: %s", err.Error())
				} else {
					if needComma {
						stationOut <- []byte(",\n")
					} else {
						needComma = true
					}
					stationOut <- txt
				}
			}
		}
	}
	stationOut <- []byte(" ] } ")
	close(stationOut)
	wg.Wait()
	if FlagDebug {
		printDuplicateStats()
	}

}

func printDuplicateStats() {
	var sb strings.Builder
	xLog.Printf("duplicate station count: %d\n\tduplicates:", len(dupStations))
	ix := 0
	for k := range dupStations {
		ix++
		if 0 == ix%5 {
			xLog.Printf("%s", sb.String())
			sb.Reset()
			sb.WriteRune(' ')
		}
		sb.WriteString(k)
		sb.WriteRune(' ')
	}
	xLog.Printf("%s", sb.String())
}

var stations = make(map[string]struct{}, 16384)
var dupStations = make(map[string]struct{}, 8)

// filterDuplicateStations checks if a station is already present
// in the stations map, filtering out duplicates. It adds new
// station IDs to the map and increments a duplicate counter for
// repeated entries.
func filterDuplicateStations(loc *denjson.Location) bool {
	_, ok := stations[loc.ID]
	if !ok {
		stations[loc.ID] = struct{}{}
		return true
	}
	dupStations[loc.ID] = struct{}{}
	return false
}
