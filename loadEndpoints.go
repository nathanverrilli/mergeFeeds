package main

import (
	"encoding/json"
	"os"
)

type endPointData struct {
	Endpoints []endPoint `json:"endpoints"`
}
type endPoint struct {
	Base  string `json:"baseUrl"`
	Token string `json:"token"`
}

func loadEndpoints(fn string) (bases []string, tokens []string) {
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
	bases = make([]string, len(ed.Endpoints))
	tokens = make([]string, len(ed.Endpoints))
	for i, ep := range ed.Endpoints {
		bases[i] = ep.Base
		tokens[i] = ep.Token
	}
	return bases, tokens
}
