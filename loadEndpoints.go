package main

import (
	"encoding/json"
	"os"
)

type endPointData struct {
	Endpoints []endPoint `json:"endpoints"`
}
type endPoint struct {
	Region string `json:"region"`
	Base   string `json:"baseUrl"`
	Token  string `json:"token"`
}

func loadEndpoints(fn string) (regions, bases, tokens []string) {
	body, err := os.ReadFile(fn)
	if nil != err {
		xLog.Printf("error reading endpoints file: %s", err.Error())
		myFatal()
	}
	var ed endPointData
	err = json.Unmarshal(body, &ed)
	if nil != err {
		xLog.Printf("error parsing endpoints file %s: %s", fn, err.Error())
		myFatal()
	}
	regions = make([]string, len(ed.Endpoints))
	bases = make([]string, len(ed.Endpoints))
	tokens = make([]string, len(ed.Endpoints))
	for i, ep := range ed.Endpoints {
		regions[i] = ep.Region
		bases[i] = ep.Base
		tokens[i] = ep.Token
	}
	return regions, bases, tokens
}
